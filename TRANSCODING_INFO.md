# Video Transcoding Support (MKV to MP4)

## Current Implementation
Currently, the video streaming uses direct byte-range streaming without transcoding. This works well for MP4 files but may have compatibility issues with MKV containers in HTML5 video players.

## Why Transcoding MKV?
- **Browser Compatibility**: HTML5 video players have better support for MP4 containers
- **Seeking Issues**: MKV files may have seeking/scrubbing problems in browsers
- **Codec Support**: Not all browsers support all codecs in MKV containers

## Option 1: FFmpeg Transcoding (Recommended)

### Installation
**Windows:**
```powershell
# Download FFmpeg from https://ffmpeg.org/download.html
# Add to system PATH
```

**Linux:**
```bash
sudo apt-get install ffmpeg
# or
sudo yum install ffmpeg
```

### Implementation Approach

#### A. On-the-fly Transcoding (Real-time)
```go
package telegram

import (
    "io"
    "os/exec"
    "log"
)

func NewTranscodingReader(chatID int64, messageID int, mimeType string) (io.ReadSeeker, error) {
    // Check if file needs transcoding
    if !needsTranscoding(mimeType) {
        return NewTelegramFile(chatID, messageID)
    }
    
    // Create pipe for FFmpeg
    telegramFile, err := NewTelegramFile(chatID, messageID)
    if err != nil {
        return nil, err
    }
    
    // FFmpeg command for fast transcoding
    cmd := exec.Command("ffmpeg",
        "-i", "pipe:0",              // Input from stdin
        "-c:v", "copy",              // Copy video (no re-encode)
        "-c:a", "aac",               // Convert audio to AAC
        "-movflags", "frag_keyframe+empty_moov+faststart", // Streamable MP4
        "-f", "mp4",                 // Output format
        "pipe:1",                    // Output to stdout
    )
    
    // Connect pipes
    cmd.Stdin = telegramFile
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, err
    }
    
    // Start FFmpeg
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    
    return stdout, nil
}

func needsTranscoding(mimeType string) bool {
    return mimeType == "video/x-matroska" || 
           mimeType == "video/mkv" ||
           strings.HasSuffix(strings.ToLower(fileName), ".mkv")
}
```

**Pros:**
- No storage needed
- Real-time processing
- Works with streaming

**Cons:**
- CPU intensive
- Higher latency
- Seeking may be problematic

#### B. Pre-transcoding with Caching
```go
package telegram

import (
    "os"
    "os/exec"
    "path/filepath"
    "crypto/md5"
    "fmt"
)

type TranscodeCache struct {
    cacheDir string
}

func NewTranscodeCache(dir string) *TranscodeCache {
    os.MkdirAll(dir, 0755)
    return &TranscodeCache{cacheDir: dir}
}

func (tc *TranscodeCache) GetOrTranscode(chatID int64, messageID int) (string, error) {
    // Generate cache key
    cacheKey := fmt.Sprintf("%d_%d", chatID, messageID)
    hash := md5.Sum([]byte(cacheKey))
    cachePath := filepath.Join(tc.cacheDir, fmt.Sprintf("%x.mp4", hash))
    
    // Check if already transcoded
    if _, err := os.Stat(cachePath); err == nil {
        return cachePath, nil
    }
    
    // Download and transcode
    telegramFile, err := NewTelegramFile(chatID, messageID)
    if err != nil {
        return "", err
    }
    
    // Create temp file
    tmpPath := cachePath + ".tmp"
    tmpFile, err := os.Create(tmpPath)
    if err != nil {
        return "", err
    }
    defer tmpFile.Close()
    
    // Download file first
    if _, err := io.Copy(tmpFile, telegramFile); err != nil {
        os.Remove(tmpPath)
        return "", err
    }
    tmpFile.Close()
    
    // Transcode
    cmd := exec.Command("ffmpeg",
        "-i", tmpPath,
        "-c:v", "libx264",           // H.264 codec
        "-preset", "fast",           // Fast encoding
        "-crf", "23",                // Quality (lower = better)
        "-c:a", "aac",               // AAC audio
        "-b:a", "128k",              // Audio bitrate
        "-movflags", "+faststart",   // Web-optimized MP4
        cachePath,
    )
    
    if err := cmd.Run(); err != nil {
        os.Remove(tmpPath)
        return "", err
    }
    
    os.Remove(tmpPath)
    return cachePath, nil
}
```

**Pros:**
- Transcode once, stream many times
- Better seeking performance
- Lower CPU during streaming

**Cons:**
- Requires storage
- Initial delay for first user
- Storage management needed

## Option 2: Container Remuxing (Fast)

Instead of re-encoding, just change the container from MKV to MP4 (if codecs are compatible):

```bash
ffmpeg -i input.mkv -c copy -movflags +faststart output.mp4
```

This is **much faster** (seconds instead of minutes) because it doesn't re-encode the video.

```go
func RemuxToMP4(inputPath, outputPath string) error {
    cmd := exec.Command("ffmpeg",
        "-i", inputPath,
        "-c", "copy",                    // Copy all streams
        "-movflags", "+faststart",       // Web optimization
        outputPath,
    )
    return cmd.Run()
}
```

## Option 3: Client-side Approach

Let the browser handle MKV if possible:
```html
<video controls>
    <source src="/stream/token" type="video/mp4">
    <source src="/stream/token" type="video/x-matroska">
    Your browser does not support this video.
</video>
```

Modern browsers (Chrome, Firefox) can handle many MKV files directly.

## Recommended Solution

### For Production:
1. **Detect container type** from file metadata
2. **MKV files**: Use container remuxing (fast, no re-encoding)
3. **Cache remuxed files** for 24-48 hours
4. **Cleanup cache** periodically based on LRU or storage limits

### Implementation in handlers.go:
```go
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    token := vars["token"]
    
    req, err := telegram.ParseStreamToken(token)
    if err != nil {
        http.Error(w, "invalid token", http.StatusBadRequest)
        return
    }
    
    // Get file info first
    fileSize, mimeType, err := telegram.GetFileInfo(req.ChatID, req.MessageID)
    if err != nil {
        http.Error(w, "failed to get file info", http.StatusInternalServerError)
        return
    }
    
    // Check if transcoding needed
    if mimeType == "video/x-matroska" || strings.Contains(mimeType, "mkv") {
        // Serve transcoded version
        s.handleTranscodedStream(w, r, req)
        return
    }
    
    // Serve directly
    telegramFile, err := telegram.NewTelegramFile(req.ChatID, req.MessageID)
    if err != nil {
        http.Error(w, "failed to access file", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Cache-Control", "public, max-age=3600")
    http.ServeContent(w, r, "", time.Time{}, telegramFile)
}
```

## Performance Considerations

### Bandwidth:
- Direct streaming: ~5-50 Mbps (depending on quality)
- Transcoding: Same, but with CPU overhead

### CPU Usage:
- Copy codec (remux): ~5-10% CPU
- Re-encode: ~50-90% CPU per stream

### Storage:
- Cache 10 movies (2GB each): ~20GB
- Implement LRU eviction
- Monitor disk usage

## Testing Transcoding

```bash
# Test if FFmpeg is installed
ffmpeg -version

# Quick test transcode
ffmpeg -i sample.mkv -c copy -movflags +faststart sample.mp4

# Test streaming-optimized MP4
ffmpeg -i input.mkv -c:v libx264 -preset ultrafast -c:a aac -movflags +faststart output.mp4
```

## Next Steps

1. Install FFmpeg on server
2. Implement container detection
3. Add remuxing for MKV files
4. Create cache directory structure
5. Add cleanup job for old cached files
6. Monitor performance and adjust settings

