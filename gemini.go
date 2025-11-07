package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"strix/telegram"
	"time"
)

var ModelName = "gemini-flash-latest"

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type GeminiFileMetadata struct {
	Title   string `json:"title"`
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
	Year    int    `json:"year"`
	Quality string `json:"quality"`
}

func parseFilenameWithGemini(filename, apiKey string) (*telegram.FileMetadata, error) {
	fmt.Println("Parsing filename with Gemini:", filename)
	prompt := fmt.Sprintf(`Extract metadata from this filename: "%s"

Return ONLY a valid JSON object with these fields:
{
  "title": "extracted title without year, quality tags, or metadata",
  "year": 0,
  "season": 0,
  "episode": 0,
  "quality": "720p or 1080p or 2160p or empty string"
}

Rules:
- title: clean title only, remove year, quality, codecs, release groups, torrent sites
- title: remove usernames, like @Adrama_Lovers (case insensitive), and other usernames like that
- title: remove dots, underscores, hyphens; replace with spaces
- title: remove texts like remastered, extended, director's cut, dual audio, multi audio
- title: purpose is to search the title on TMDB/IMDB, so title should be clean
- title: remove E01, S01E01, 1080p, 720p, HDRip, WEB-DL, BluRay, x264, x265, etc.
- title: remove Episode numbers from title, Season numbers too
- title: remove everything that is not part of the actual title
- title:remove episode name also if present
- year: 0 if unknown or not present else return the year
- season/episode: 0 if it's a movie, actual numbers if it's a TV show
- quality: one of "2160p", "1080p", "720p", "480p", or ""
- title: remove everything inside brackets (), [], {}
- Return ONLY the JSON, no explanation`, filename)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", ModelName, apiKey)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini API error: %s", string(body))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, err
	}
	fmt.Println(geminiResp)

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from gemini")
	}

	responseText := geminiResp.Candidates[0].Content.Parts[0].Text

	responseText = cleanJSONResponse(responseText)

	var geminiMetadata GeminiFileMetadata
	if err := json.Unmarshal([]byte(responseText), &geminiMetadata); err != nil {
		return nil, fmt.Errorf("failed to parse gemini response: %w", err)
	}

	quality := validateQuality(geminiMetadata.Quality)

	return &telegram.FileMetadata{
		Title:   geminiMetadata.Title,
		Season:  geminiMetadata.Season,
		Episode: geminiMetadata.Episode,
		Quality: quality,
		Year:    geminiMetadata.Year,
	}, nil
}

func validateQuality(quality string) string {
	validQualities := []string{"2160p", "1080p", "720p", "480p", "360p", "240p", "144p"}

	quality = strings.TrimSpace(strings.ToLower(quality))

	for _, valid := range validQualities {
		if strings.ToLower(valid) == quality {
			return valid
		}
	}

	return "HD"
}

func cleanJSONResponse(text string) string {
	start := -1
	end := -1

	for i := 0; i < len(text); i++ {
		if text[i] == '{' && start == -1 {
			start = i
		}
		if text[i] == '}' {
			end = i + 1
		}
	}

	if start != -1 && end != -1 {
		return text[start:end]
	}

	return text
}
