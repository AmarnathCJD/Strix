package telegram

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram"
	tg "github.com/amarnathcjd/gogram/telegram"
)

const (
	ChunkSize = 524288
)

var (
	exportedSenderCache = make(map[int]*gogram.MTProto)
	exportedSenderMutex sync.RWMutex
)

type TelegramFile struct {
	chatID      int64
	messageID   int
	fileSize    int64
	mimeType    string
	fileName    string
	position    int64
	requester   *gogram.MTProto
	location    *tg.InputDocumentFileLocation
	cache       map[int64][]byte
	cacheSize   int
	mu          sync.Mutex
	lastAccess  int64   // For tracking access patterns
	requestsLog []int64 // For debugging
}

func NewTelegramFile(chatID int64, messageID int) (*TelegramFile, error) {
	client := getRandomBot()
	var requester = client.MTProto

	message, err := client.GetMessageByID(chatID, int32(messageID))
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	doc := message.Document()
	if doc == nil {
		return nil, fmt.Errorf("message has no document")
	}

	mimeType := doc.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	location := &tg.InputDocumentFileLocation{
		ID:            doc.ID,
		AccessHash:    doc.AccessHash,
		FileReference: doc.FileReference,
		ThumbSize:     "",
	}

	if doc.DcID != int32(client.GetDC()) {
		dcID := int(doc.DcID)

		exportedSenderMutex.RLock()
		cached, exists := exportedSenderCache[dcID]
		exportedSenderMutex.RUnlock()

		if exists {
			requester = cached
		} else {
			exported, err := client.CreateExportedSender(dcID, false)
			if err != nil {
				return nil, fmt.Errorf("failed to create exported sender: %w", err)
			}
			requester = exported

			exportedSenderMutex.Lock()
			exportedSenderCache[dcID] = exported
			exportedSenderMutex.Unlock()
		}
	}

	return &TelegramFile{
		chatID:      chatID,
		messageID:   messageID,
		fileSize:    doc.Size,
		mimeType:    mimeType,
		fileName:    fmt.Sprintf("file_%d_%d", chatID, messageID),
		position:    0,
		requester:   requester,
		location:    location,
		cache:       make(map[int64][]byte),
		cacheSize:   20, // Increase cache size for better seeking performance
		lastAccess:  0,
		requestsLog: make([]int64, 0, 100),
	}, nil
}

// Edited Read/Seek/getChunk methods for TelegramFile to be robust
// Assumes TelegramFile has fields:
// position int64
// fileSize int64
// cache map[int64][]byte
// cacheSize int
// mu sync.Mutex
// lastAccess int64
// requester *tg.Client (or similar with MakeRequest)
// location any
// ChunkSize constant defined elsewhere (int64)

func (tf *TelegramFile) Read(p []byte) (n int, err error) {
	// Fast path: EOF
	if tf.position >= tf.fileSize {
		return 0, io.EOF
	}

	remaining := tf.fileSize - tf.position
	toRead := min(int64(len(p)), remaining)

	var bytesRead int64
	for bytesRead < toRead {
		chunkOffset := (tf.position / ChunkSize) * ChunkSize
		offsetInChunk := tf.position - chunkOffset

		// Attempt to fetch a chunk, retry once on transient errors or empty result
		var chunk []byte
		var fetchErr error
		for attempt := 0; attempt < 2; attempt++ {
			chunk, fetchErr = tf.getChunk(chunkOffset)
			if fetchErr == nil && len(chunk) > 0 {
				break
			}
			// small backoff on retry
			if attempt == 0 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		if fetchErr != nil {
			// If we already read some bytes, return them to caller (per io.Reader contract)
			if bytesRead > 0 {
				log.Printf("[TelegramFile] Read: transient error after partial read: %v", fetchErr)
				return int(bytesRead), nil
			}
			return 0, fetchErr
		}

		if len(chunk) == 0 {
			// No data available for this chunk. If we have read some bytes, return them.
			if bytesRead > 0 {
				return int(bytesRead), nil
			}
			// Nothing to read, signal EOF
			return 0, io.EOF
		}

		available := int64(len(chunk)) - offsetInChunk
		if available <= 0 {
			// This should not happen in normal flow; break to avoid infinite loop
			break
		}

		toCopy := toRead - bytesRead
		if toCopy > available {
			toCopy = available
		}

		// copy expects ints
		t := int(bytesRead)
		cc := int(toCopy)
		copy(p[t:t+cc], chunk[offsetInChunk:offsetInChunk+toCopy])
		bytesRead += toCopy

		// update position and lastAccess (protect with mutex)
		tf.mu.Lock()
		tf.position += toCopy
		tf.lastAccess = tf.position
		tf.mu.Unlock()
	}

	if bytesRead == 0 {
		return 0, io.EOF
	}

	// Log what was read (convert to int for readability)
	log.Printf("[TelegramFile] Read: %d for requested %d bytes at position %d",
		int(bytesRead), int(toRead), int(tf.position-bytesRead))

	return int(bytesRead), nil
}

func (tf *TelegramFile) Seek(offset int64, whence int) (int64, error) {
	// Lock around position mutation to avoid races if Seek is called concurrently
	tf.mu.Lock()
	defer tf.mu.Unlock()

	// Defensive check: ensure fileSize is known
	if tf.fileSize < 0 {
		return 0, errors.New("unknown file size")
	}

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = tf.position + offset
	case io.SeekEnd:
		newPos = tf.fileSize + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}

	if newPos < 0 {
		return 0, fmt.Errorf("negative position")
	}
	if newPos > tf.fileSize {
		newPos = tf.fileSize
	}

	// Log significant seeks for debugging
	if tf.position > 0 && abs(newPos-tf.position) > 1048576 {
		log.Printf("[TelegramFile] Seek: %d -> %d (%.2f MB jump)",
			tf.position, newPos, float64(abs(newPos-tf.position))/(1024*1024))
	}

	tf.position = newPos
	tf.lastAccess = newPos
	return newPos, nil
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func (tf *TelegramFile) getChunk(offset int64) ([]byte, error) {
	// Use mutex to protect cache maps and requester access
	// We keep critical sections small: fetch outside lock if possible
	alignedOffset := (offset / 1024) * 1024

	// Quick cache check under lock
	func() {
		// placeholder to keep the lock scope obvious in code; actual cache read below uses lock
	}()

	// Check cache
	tf.mu.Lock()
	if chunk, ok := tf.cache[alignedOffset]; ok {
		// update last access and return a copy (or the slice as-is if immutable)
		tf.lastAccess = alignedOffset
		tf.mu.Unlock()
		return chunk, nil
	}
	tf.mu.Unlock()

	remaining := tf.fileSize - alignedOffset
	if remaining <= 0 {
		return nil, io.EOF
	}

	limit := int64(ChunkSize)
	if limit > 1048576 {
		limit = 1048576
	}
	if limit > remaining {
		limit = remaining
	}
	// align limit to 1 KB
	limit = (limit / 1024) * 1024
	if limit < 1024 {
		if remaining >= 1024 {
			limit = 1024
		} else {
			limit = remaining
		}
	}

	// Perform request (do not hold tf.mu while making network call)
	result, err := tf.requester.MakeRequest(&tg.UploadGetFileParams{
		Location: tf.location,
		Offset:   alignedOffset,
		Limit:    int32(limit),
		Precise:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("telegram fetch failed at offset %d: %w", alignedOffset, err)
	}

	file, ok := result.(*tg.UploadFileObj)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", result)
	}

	if len(file.Bytes) == 0 {
		return nil, fmt.Errorf("empty chunk received at offset %d", alignedOffset)
	}

	actualBytes := file.Bytes
	maxBytes := tf.fileSize - alignedOffset
	if int64(len(actualBytes)) > maxBytes {
		actualBytes = actualBytes[:maxBytes]
	}

	// Cache under lock
	tf.mu.Lock()
	// store a copy to avoid accidental mutation
	copyBuf := make([]byte, len(actualBytes))
	copy(copyBuf, actualBytes)
	tf.cache[alignedOffset] = copyBuf
	// update lastAccess
	tf.lastAccess = alignedOffset

	// Simple LRU-ish eviction
	if len(tf.cache) > tf.cacheSize {
		maxDist := int64(-1)
		evictKey := int64(-1)
		for k := range tf.cache {
			dist := abs(k - tf.lastAccess)
			if dist > maxDist {
				maxDist = dist
				evictKey = k
			}
		}
		if evictKey >= 0 && evictKey != alignedOffset {
			delete(tf.cache, evictKey)
		}
	}
	cached := tf.cache[alignedOffset]
	tf.mu.Unlock()

	return cached, nil
}

// Utility min function
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func (tf *TelegramFile) GetMimeType() string {
	return tf.mimeType
}

func (tf *TelegramFile) GetSize() int64 {
	return tf.fileSize
}

type StreamRequest struct {
	ChatID    int64
	MessageID int
	Start     int64
	End       int64
}

func ParseStreamToken(token string) (*StreamRequest, error) {
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	parts := strings.Split(string(decoded), ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token format")
	}

	chatID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid chat ID")
	}

	messageID, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid message ID")
	}

	return &StreamRequest{
		ChatID:    chatID,
		MessageID: messageID,
	}, nil
}

func GenerateStreamToken(chatID int64, messageID int) string {
	data := fmt.Sprintf("%d:%d", chatID, messageID)
	return base64.URLEncoding.EncodeToString([]byte(data))
}

func getRandomBot() *tg.Client {
	if len(config.CDNBots) == 0 {
		return bot
	}
	idx := rand.Intn(len(config.CDNBots))
	if idx == 0 || config.CDNBots[idx] == "" {
		return bot
	}
	return bot
}

func GetFileInfo(chatID int64, messageID int) (int64, string, error) {
	client := getRandomBot()
	message, err := client.GetMessageByID(chatID, int32(messageID))
	if err != nil {
		return 0, "", fmt.Errorf("failed to get message: %w", err)
	}

	doc := message.Document()
	if doc == nil {
		return 0, "", fmt.Errorf("message has no document")
	}

	mimeType := doc.MimeType
	if mimeType == "" {
		mimeType = "video/mp4"
	}

	return doc.Size, mimeType, nil
}
