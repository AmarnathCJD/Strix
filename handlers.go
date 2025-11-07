package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"strix/telegram"

	"github.com/gorilla/mux"
)

// TMDB API Handlers

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf("%s/search/multi?api_key=%s&query=%s", tmdbBaseURL, s.config.TMDBAPIKey, url.QueryEscape(query))
	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (s *Server) handleTVDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	url := fmt.Sprintf("%s/tv/%s?api_key=%s&append_to_response=credits,videos,recommendations,external_ids",
		tmdbBaseURL, id, s.config.TMDBAPIKey)

	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (s *Server) handleSeasonDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	season := vars["season"]

	url := fmt.Sprintf("%s/tv/%s/season/%s?api_key=%s", tmdbBaseURL, id, season, s.config.TMDBAPIKey)

	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (s *Server) handleMovieDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	url := fmt.Sprintf("%s/movie/%s?api_key=%s&append_to_response=credits,videos,recommendations,external_ids",
		tmdbBaseURL, id, s.config.TMDBAPIKey)

	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (s *Server) handleTrending(w http.ResponseWriter, r *http.Request) {
	mediaType := r.URL.Query().Get("type")
	if mediaType == "" {
		mediaType = "all"
	}

	timeWindow := r.URL.Query().Get("time")
	if timeWindow == "" {
		timeWindow = "week"
	}

	url := fmt.Sprintf("%s/trending/%s/%s?api_key=%s", tmdbBaseURL, mediaType, timeWindow, s.config.TMDBAPIKey)

	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	// Parse limit from query params
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		fmt.Sscanf(limitParam, "%d", &limit)
	}

	media, err := s.db.GetAllMedia(limit, offset)
	if err != nil {
		http.Error(w, "Failed to list files", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(media)
}

func (s *Server) handleSearchFiles(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	results, err := s.db.SearchMedia(query)
	if err != nil {
		http.Error(w, "Failed to search files", http.StatusInternalServerError)
		return
	}

	type SearchResult struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		FileName    string `json:"file_name"`
		MediaType   string `json:"media_type"`
		Season      int    `json:"season,omitempty"`
		Episode     int    `json:"episode,omitempty"`
		Quality     string `json:"quality"`
		FileSize    int64  `json:"file_size"`
		StreamToken string `json:"stream_token"`
		StreamURL   string `json:"stream_url"`
		TMDBID      int    `json:"tmdb_id"`
	}

	searchResults := make([]SearchResult, 0, len(results))
	for _, media := range results {
		streamToken := telegram.GenerateStreamToken(media.ChatID, media.MessageID)
		searchResults = append(searchResults, SearchResult{
			ID:          media.ID.Hex(),
			Title:       media.Title,
			FileName:    media.FileName,
			MediaType:   media.MediaType,
			Season:      media.Season,
			Episode:     media.Episode,
			Quality:     media.Quality,
			FileSize:    media.FileSize,
			StreamToken: streamToken,
			StreamURL:   fmt.Sprintf("/stream/%s", streamToken),
			TMDBID:      media.TMDBID,
		})
	}

	response := map[string]interface{}{
		"query":   query,
		"count":   len(searchResults),
		"results": searchResults,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetMovieFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tmdbID := vars["tmdb_id"]

	var id int
	fmt.Sscanf(tmdbID, "%d", &id)

	media, err := s.db.GetMediaByTMDB(id, "movie", 0, 0)
	if err != nil {
		http.Error(w, "Failed to fetch media", http.StatusInternalServerError)
		return
	}

	if media == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"available": false,
			"message":   "Media not available",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"available":     true,
		"message_id":    media.MessageID,
		"chat_id":       media.ChatID,
		"cdn_bot_index": media.CDNBotIndex,
		"title":         media.Title,
		"quality":       media.Quality,
		"stream_token":  telegram.GenerateStreamToken(media.ChatID, media.MessageID),
	})
}

func (s *Server) handleGetSeasonFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tmdbID := vars["tmdb_id"]
	season := vars["season"]

	var id, seasonNum int
	fmt.Sscanf(tmdbID, "%d", &id)
	fmt.Sscanf(season, "%d", &seasonNum)

	episodes, err := s.db.GetSeasonEpisodes(id, seasonNum)
	if err != nil {
		http.Error(w, "Failed to fetch episodes", http.StatusInternalServerError)
		return
	}

	episodeMap := make(map[int]interface{})
	for _, ep := range episodes {
		episodeMap[ep.Episode] = map[string]interface{}{
			"available":     true,
			"message_id":    ep.MessageID,
			"chat_id":       ep.ChatID,
			"cdn_bot_index": ep.CDNBotIndex,
			"title":         ep.Title,
			"quality":       ep.Quality,
			"stream_token":  telegram.GenerateStreamToken(ep.ChatID, ep.MessageID),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(episodeMap)
}

func (s *Server) handleGetEpisodeFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tmdbID := vars["tmdb_id"]
	season := vars["season"]
	episode := vars["episode"]

	var id, seasonNum, episodeNum int
	fmt.Sscanf(tmdbID, "%d", &id)
	fmt.Sscanf(season, "%d", &seasonNum)
	fmt.Sscanf(episode, "%d", &episodeNum)

	media, err := s.db.GetMediaByTMDB(id, "tv", seasonNum, episodeNum)
	if err != nil {
		http.Error(w, "Failed to fetch episode", http.StatusInternalServerError)
		return
	}

	if media == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"available": false,
			"message":   "Episode not available",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"available":     true,
		"message_id":    media.MessageID,
		"chat_id":       media.ChatID,
		"cdn_bot_index": media.CDNBotIndex,
		"title":         media.Title,
		"quality":       media.Quality,
		"stream_token":  telegram.GenerateStreamToken(media.ChatID, media.MessageID),
	})
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	// log.Printf("[STREAM] Request - Token: %s, Range: %s, Method: %s",
	// 	token, r.Header.Get("Range"), r.Method)

	req, err := telegram.ParseStreamToken(token)
	if err != nil {
		log.Printf("[STREAM] ERROR: Invalid token: %v", err)
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}

	telegramFile, err := telegram.NewTelegramFile(req.ChatID, req.MessageID)
	if err != nil {
		log.Printf("[STREAM] ERROR: Failed to create TelegramFile: %v", err)
		http.Error(w, "failed to access file", http.StatusInternalServerError)
		return
	}

	// fileSize := telegramFile.GetSize()
	mimeType := telegramFile.GetMimeType()

	// log.Printf("[STREAM] File info - Size: %d bytes (%.2f MB), MimeType: %s",
	// 	fileSize, float64(fileSize)/(1024*1024), mimeType)

	// Set headers before ServeContent
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length,Content-Range")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	// Handle preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// ServeContent handles:
	// - Range request parsing
	// - HEAD requests
	// - Conditional requests (If-Modified-Since, etc.)
	// - Proper status codes (200, 206, 416)
	// - Content-Length and Content-Range headers
	// Behind the scenes, it will call Read() and Seek() on TelegramFile
	// which fetches chunks from Telegram on-demand
	http.ServeContent(w, r, "", time.Time{}, telegramFile)

	log.Printf("[STREAM] Completed")
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func (s *Server) handleTVPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Fetch TV details from TMDB
	url := fmt.Sprintf("%s/tv/%s?api_key=%s&append_to_response=credits,videos,recommendations,external_ids",
		tmdbBaseURL, id, s.config.TMDBAPIKey)

	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var tvDetails TVDetails
	if err := json.NewDecoder(resp.Body).Decode(&tvDetails); err != nil {
		http.Error(w, "Failed to parse data", http.StatusInternalServerError)
		return
	}

	// Render the template with TV details
	funcMap := template.FuncMap{
		"split": strings.Split,
	}

	tmpl, err := template.New("series.html").Funcs(funcMap).ParseFiles("templates/series.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, tvDetails); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleMoviePage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Fetch movie details from TMDB
	url := fmt.Sprintf("%s/movie/%s?api_key=%s&append_to_response=credits,videos,recommendations,external_ids",
		tmdbBaseURL, id, s.config.TMDBAPIKey)

	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var movieDetails MovieDetails
	if err := json.NewDecoder(resp.Body).Decode(&movieDetails); err != nil {
		http.Error(w, "Failed to parse data", http.StatusInternalServerError)
		return
	}

	// Render the template with movie details
	funcMap := template.FuncMap{
		"split": strings.Split,
	}

	tmpl, err := template.New("movie.html").Funcs(funcMap).ParseFiles("templates/movie.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, movieDetails); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleSearchPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func (s *Server) handleStreamPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/stream.html")
}

// Helper function to check if CORS headers should be added
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
