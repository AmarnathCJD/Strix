package telegram

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram"
	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	exportedSenderCache = make(map[int]*gogram.MTProto)
	exportedSenderMutex sync.RWMutex
)

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

type MediaInfo struct {
	Size     int64
	FileName string
	MimeType string
}

func GetMediaInfo(chatID int64, messageID int) (*MediaInfo, error) {
	client := getRandomBot()
	message, err := client.GetMessageByID(chatID, int32(messageID))
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	var fileSize int64
	var fileName string
	var mimeType string

	doc := message.Document()
	if doc != nil {
		fileSize = doc.Size
		mimeType = doc.MimeType

		for _, attr := range doc.Attributes {
			if fileNameAttr, ok := attr.(*tg.DocumentAttributeFilename); ok {
				fileName = fileNameAttr.FileName
				break
			}
		}
	}

	if fileSize == 0 {
		return nil, fmt.Errorf("no media found in message")
	}

	if fileName == "" {
		fileName = fmt.Sprintf("file_%d_%d", chatID, messageID)
	}

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return &MediaInfo{
		Size:     fileSize,
		FileName: fileName,
		MimeType: mimeType,
	}, nil
}

func StreamMediaChunks(chatID int64, messageID int, startChunk int64, callback func([]byte) error) error {
	client := getRandomBot()
	var requester = client.MTProto

	message, err := client.GetMessageByID(chatID, int32(messageID))
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	doc := message.Document()
	if doc == nil {
		return fmt.Errorf("message has no document")
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
				return fmt.Errorf("failed to create exported sender: %w", err)
			}
			requester = exported

			exportedSenderMutex.Lock()
			exportedSenderCache[dcID] = exported
			exportedSenderMutex.Unlock()
		}
	}

	chunkSize := int64(1024 * 1024)
	offset := startChunk * chunkSize
	fileSize := doc.Size
	maxRetries := 5
	currentRetries := 0

	for offset < fileSize {
		limit := chunkSize
		remaining := fileSize - offset

		if remaining < limit {
			limit = remaining
		}

		alignedOffset := (offset / 1024) * 1024
		alignedLimit := max(min(((limit+1023)/1024)*1024, 1048576), 1024)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		result, err := requester.MakeRequestCtx(ctx, &tg.UploadGetFileParams{
			Location:     location,
			Offset:       alignedOffset,
			Limit:        int32(alignedLimit),
			Precise:      true,
			CdnSupported: false,
		})

		cancel()

		if err != nil {
			if handleIfFlood(err) {
				continue
			}
			currentRetries++
			if currentRetries > maxRetries {
				return fmt.Errorf("telegram fetch failed after %d retries at offset %d: %w", maxRetries, offset, err)
			}
			backoffDuration := time.Duration(100*(1<<(currentRetries-1))) * time.Millisecond
			time.Sleep(backoffDuration)
			continue
		}

		currentRetries = 0

		file, ok := result.(*tg.UploadFileObj)
		if !ok {
			return fmt.Errorf("unexpected response type: %T", result)
		}

		if len(file.Bytes) == 0 {
			break
		}

		chunkData := file.Bytes
		offsetDiff := offset - alignedOffset
		if offsetDiff > 0 && offsetDiff < int64(len(chunkData)) {
			chunkData = chunkData[offsetDiff:]
		}

		if int64(len(chunkData)) > limit {
			chunkData = chunkData[:limit]
		}

		if err := callback(chunkData); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		offset += int64(len(chunkData))
	}

	return nil
}

func handleIfFlood(err error) bool {
	if tg.MatchError(err, "FLOOD_WAIT_") || tg.MatchError(err, "FLOOD_PREMIUM_WAIT_") {
		if waitTime := tg.GetFloodWait(err); waitTime > 0 {
			time.Sleep(time.Duration(waitTime) * time.Second)
			return true
		}
	}

	return false
}
