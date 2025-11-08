package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"strix/telegram"
)

var filenamePatterns = []struct {
	regex   *regexp.Regexp
	extract func(matches []string) *telegram.FileMetadata
}{
	{
		regex: regexp.MustCompile(`(?i)S(\d{1,2})E(\d{1,3})`),
		extract: func(m []string) *telegram.FileMetadata {
			season, _ := strconv.Atoi(m[1])
			episode, _ := strconv.Atoi(m[2])
			return &telegram.FileMetadata{Season: season, Episode: episode}
		},
	},
	{
		regex: regexp.MustCompile(`(?i)(\d{1,2})x(\d{1,3})`),
		extract: func(m []string) *telegram.FileMetadata {
			season, _ := strconv.Atoi(m[1])
			episode, _ := strconv.Atoi(m[2])
			return &telegram.FileMetadata{Season: season, Episode: episode}
		},
	},
	{
		regex: regexp.MustCompile(`(?i)Season[\s._-]*(\d{1,2})[\s._-]*Episode[\s._-]*(\d{1,3})`),
		extract: func(m []string) *telegram.FileMetadata {
			season, _ := strconv.Atoi(m[1])
			episode, _ := strconv.Atoi(m[2])
			return &telegram.FileMetadata{Season: season, Episode: episode}
		},
	},
	{
		regex: regexp.MustCompile(`(?i)S(\d{1,2})[\s._-]*E(\d{1,3})`),
		extract: func(m []string) *telegram.FileMetadata {
			season, _ := strconv.Atoi(m[1])
			episode, _ := strconv.Atoi(m[2])
			return &telegram.FileMetadata{Season: season, Episode: episode}
		},
	},
}

var qualityPatterns = []struct {
	pattern string
	quality string
}{
	{"2160p", "2160p"},
	{"4K", "2160p"},
	{"UHD", "2160p"},
	{"1080p", "1080p"},
	{"FHD", "1080p"},
	{"720p", "720p"},
	{"HD", "720p"},
	{"480p", "480p"},
	{"SD", "480p"},
	{"360p", "360p"},
	{"240p", "240p"},
	{"144p", "144p"},
}

var cleanPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^@[A-Za-z0-9_]+[-\s]`),
	regexp.MustCompile(`(?i)\.(mkv|mp4|avi|mov|wmv|flv|webm|m4v)$`),

	regexp.MustCompile(`(?i)\b(RARBG|YTS|YIFY|ETRG|TGx|GalaxyTV|PSA|SPARKS|AMZN|NF|HULU|DSNP|ATVP|AppleTV|Amazon|Netflix|Disney)\b`),
	regexp.MustCompile(`(?i)\b(TorrentGalaxy|1337x|ThePirateBay|Kickass|LimeTorrents|Torrentz2)\b`),
	regexp.MustCompile(`(?i)\[(.*?)\]$`),

	regexp.MustCompile(`(?i)\bS\d{1,2}E\d{1,3}\b`),
	regexp.MustCompile(`(?i)\b\d{1,2}x\d{1,3}\b`),

	regexp.MustCompile(`(?i)\b(HDR10\+?|HDR|DV|DOLBY\.?VISION|SDR)\b`),
	regexp.MustCompile(`(?i)\b\d+\.?\d*CH\b`),
	regexp.MustCompile(`(?i)\b\d+\.\d+\b`),

	regexp.MustCompile(`(?i)\b(2160p|1080p|720p|480p|360p|240p|144p)\b`),
	regexp.MustCompile(`(?i)\b(4K|UHD|FHD|SD)\b`),
	regexp.MustCompile(`(?i)\bHD\b`),

	regexp.MustCompile(`(?i)\b(BluRay|Blu-Ray|WEB-?DL|HDTV|WEBRip|BRRip|DVDRip)\b`),
	regexp.MustCompile(`(?i)\b(CAM|TC|HDTC|R5|R6)\b`),
	regexp.MustCompile(`(?i)\bHDCAM\b`),
	regexp.MustCompile(`(?i)\b(TELESYNC|TELECINE|WORKPRINT|SCREENER|SCR|DVDSCR)\b`),

	regexp.MustCompile(`(?i)\b(x264|x265|H\.?264|H\.?265|HEVC|AVC|10bit|10-bit|8bit|8-bit)\b`),

	regexp.MustCompile(`(?i)\b(PROPER|REPACK|EXTENDED|UNRATED|UNCUT|DIRECTORS?\.CUT|THEATRICAL|IMAX)\b`),
	regexp.MustCompile(`(?i)\b(INTERNAL|LIMITED|FESTIVAL|RERiP|REAL\.PROPER)\b`),

	regexp.MustCompile(`(?i)\b(AAC\.?\d*\.?\d*|AC3|DTS[-.]?X?|DTS[-.]?HD|TrueHD|FLAC|MP3|Atmos|DD\+?\.?\d*\.?\d*|EAC3?)\b`),
	regexp.MustCompile(`(?i)\b(DUAL|MULTI|LiNE|HC|MD|KORSUB|SUBBED|DUBBED|VOSTFR)\b`),
	regexp.MustCompile(`(?i)\b(REMUX|HYBRID|CONVERTED|RETAIL|SUBPACK)\b`),

	regexp.MustCompile(`[\[\(]?\b(19|20)\d{2}\b[\]\)]?`),

	regexp.MustCompile(`(?i)[-._]\d{3,4}MB[-._]`),
	regexp.MustCompile(`(?i)[-._]\d+\.\d+GB[-._]`),

	regexp.MustCompile(`(?i)-[A-Za-z0-9]+$`),
	regexp.MustCompile(`[\[\]\(\)]+`),
}

func extractTitleFromFilenameLegacy(filename string) string {
	title := strings.TrimSuffix(filename, filepath.Ext(filename))

	yearPattern := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	if yearLoc := yearPattern.FindStringIndex(title); yearLoc != nil {
		title = title[:yearLoc[0]]
	}

	for _, pattern := range cleanPatterns {
		title = pattern.ReplaceAllString(title, "")
	}

	replacer := strings.NewReplacer(
		".", " ",
		"_", " ",
		"-", " ",
		"@", "",
		"#", "",
		"<", "",
		">", "",
		"[", "",
		"]", "",
		"(", "",
		")", "",
	)
	title = replacer.Replace(title)

	title = strings.TrimSpace(title)
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	return title
}

func parseFilenameLegacy(filename string) *telegram.FileMetadata {
	metadata := &telegram.FileMetadata{}

	yearPattern := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	fileWithoutYear := filename
	if yearMatch := yearPattern.FindString(filename); yearMatch != "" {
		fileWithoutYear = strings.Replace(filename, yearMatch, " ", 1)
	}

	for _, pattern := range filenamePatterns {
		if matches := pattern.regex.FindStringSubmatch(fileWithoutYear); len(matches) > 0 {
			extracted := pattern.extract(matches)
			if extracted.Season > 0 {
				metadata.Season = extracted.Season
				metadata.Episode = extracted.Episode
				break
			}
		}
	}

	upperFilename := strings.ToUpper(filename)
	for _, qp := range qualityPatterns {
		if strings.Contains(upperFilename, strings.ToUpper(qp.pattern)) {
			metadata.Quality = qp.quality
			break
		}
	}

	metadata.Title = extractTitleFromFilenameLegacy(filename)

	return metadata
}

func parseFilename(filename string) *telegram.FileMetadata {
	if telegram.GeminiAPIKey != "" {
		metadata, err := parseFilenameWithGemini(filename, telegram.GeminiAPIKey)
		if err == nil && metadata.Title != "" {
			return metadata
		}
	}

	return parseFilenameLegacy(filename)
}

func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	videoExts := []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v"}

	for _, validExt := range videoExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func extractCodec(filename string) string {
	codecPatterns := []struct {
		pattern *regexp.Regexp
		name    string
	}{
		{regexp.MustCompile(`(?i)\b(x265|H\.?265|HEVC)\b`), "HEVC"},
		{regexp.MustCompile(`(?i)\b(x264|H\.?264|AVC)\b`), "AVC"},
		{regexp.MustCompile(`(?i)\b(10bit|10-bit)\b`), "10bit"},
		{regexp.MustCompile(`(?i)\b(8bit|8-bit)\b`), "8bit"},
		{regexp.MustCompile(`(?i)\b(VP9)\b`), "VP9"},
		{regexp.MustCompile(`(?i)\b(AV1)\b`), "AV1"},
	}

	codecs := []string{}
	for _, cp := range codecPatterns {
		if cp.pattern.MatchString(filename) {
			codecs = append(codecs, cp.name)
		}
	}

	if len(codecs) > 0 {
		return strings.Join(codecs, " ")
	}
	return ""
}
