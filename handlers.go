package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"strix/telegram"

	"github.com/gorilla/mux"
)

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

	req, err := telegram.ParseStreamToken(token)
	if err != nil {
		log.Printf("[STREAM] Invalid token")
		http.Error(w, "Invalid or missing token", http.StatusUnauthorized)
		return
	}

	fileInfo, err := telegram.GetMediaInfo(req.ChatID, req.MessageID)
	if err != nil {
		log.Printf("[STREAM] Media not found")
		http.Error(w, "No media found", http.StatusNotFound)
		return
	}

	fileSize := fileInfo.Size
	fileName := fileInfo.FileName
	mimeType := fileInfo.MimeType

	displayName := fileName
	if len(displayName) > 40 {
		displayName = displayName[:37] + "..."
	}

	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Range")
		w.WriteHeader(http.StatusOK)
		return
	}

	var start, end int64
	var status int
	rangeHeader := r.Header.Get("Range")

	if rangeHeader != "" {
		rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
		parts := strings.Split(rangeStr, "-")

		if parts[0] != "" {
			start, _ = strconv.ParseInt(parts[0], 10, 64)
		}

		if len(parts) > 1 && parts[1] != "" {
			end, _ = strconv.ParseInt(parts[1], 10, 64)
		} else {
			end = fileSize - 1
		}

		if start >= fileSize || end >= fileSize || start > end {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
			http.Error(w, "Requested Range Not Satisfiable", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		status = http.StatusPartialContent
	} else {
		start = 0
		end = fileSize - 1
		status = http.StatusOK
	}

	contentLength := end - start + 1

	rangeDisplay := rangeHeader
	if rangeDisplay == "" {
		rangeDisplay = "full"
	}
	log.Printf("[STREAM] %s | %s", displayName, rangeDisplay)

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, fileName))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length,Content-Range")

	if status == http.StatusPartialContent {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	}

	if r.Method == http.MethodHead {
		w.WriteHeader(status)
		return
	}

	w.WriteHeader(status)

	chunkSize := int64(1024 * 1024)
	startChunk := start / chunkSize
	endChunk := end / chunkSize
	offsetInFirstChunk := start % chunkSize
	bytesInLastChunk := (end % chunkSize) + 1

	bytesSent := int64(0)
	currentChunk := startChunk

	clientContext := r.Context()

	err = telegram.StreamMediaChunks(req.ChatID, req.MessageID, startChunk, func(chunkData []byte) error {
		select {
		case <-clientContext.Done():
			return io.EOF
		default:
		}

		chunkToSend := chunkData

		if currentChunk == startChunk {
			if int64(len(chunkToSend)) > offsetInFirstChunk {
				chunkToSend = chunkToSend[offsetInFirstChunk:]
			} else {
				currentChunk++
				return nil
			}
		}

		if currentChunk == endChunk {
			if currentChunk == startChunk {

				if int64(len(chunkToSend)) > bytesInLastChunk-offsetInFirstChunk {
					chunkToSend = chunkToSend[:bytesInLastChunk-offsetInFirstChunk]
				}
			} else {

				if int64(len(chunkToSend)) > bytesInLastChunk {
					chunkToSend = chunkToSend[:bytesInLastChunk]
				}
			}
		}

		if bytesSent+int64(len(chunkToSend)) > contentLength {
			chunkToSend = chunkToSend[:contentLength-bytesSent]
		}

		if len(chunkToSend) > 0 {
			n, err := w.Write(chunkToSend)
			if err != nil {
				return io.EOF
			}
			bytesSent += int64(n)

			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}

		currentChunk++

		if bytesSent >= contentLength || currentChunk > endChunk {
			return io.EOF
		}

		return nil
	})

	if err != nil && err != io.EOF {
		if !strings.Contains(err.Error(), "broken pipe") &&
			!strings.Contains(err.Error(), "connection reset") {
			log.Printf("[STREAM] ✗ Error streaming %s", displayName)
		}
		return
	}
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func (s *Server) handleTVPage(w http.ResponseWriter, r *http.Request) {
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

	var tvDetails TVDetails
	if err := json.NewDecoder(resp.Body).Decode(&tvDetails); err != nil {
		http.Error(w, "Failed to parse data", http.StatusInternalServerError)
		return
	}

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

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
