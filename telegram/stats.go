package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type searchContext struct {
	Query      string
	UserID     int64
	TitlesList []*titleInfo
	Timestamp  time.Time
}

type titleInfo struct {
	Title     string
	MediaType string
	TMDBID    int
	Seasons   map[int][]int
	Qualities map[string]bool
	Files     []string
}

var (
	searchContextMap      = make(map[int]searchContext)
	searchContextMapMutex sync.RWMutex
)

func cleanOldSearchContexts() {
	searchContextMapMutex.Lock()
	defer searchContextMapMutex.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for msgID, ctx := range searchContextMap {
		if ctx.Timestamp.Before(cutoff) {
			delete(searchContextMap, msgID)
		}
	}
}

func init() {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			cleanOldSearchContexts()
		}
	}()
}

func HandleStats(m *tg.NewMessage) error {
	if !isAuthorized(m.Sender.ID) {
		m.Reply("<b>Access Denied</b>\n\nYou are not authorized to use this command.")
		return nil
	}

	stats, err := db.GetStats()
	if err != nil {
		m.Reply(fmt.Sprintf("<b>Error:</b> Failed to get stats: %v", err))
		return nil
	}

	message := fmt.Sprintf(
		"<b>DATABASE STATISTICS</b>\n\n"+
			"<b>Media Files:</b>\n"+
			"→ Movies: <code>%d</code>\n"+
			"→ TV Shows: <code>%d</code>\n"+
			"→ Total Files: <code>%d</code>\n\n"+
			"<b>Users:</b>\n"+
			"→ Total Users: <code>%d</code>\n\n"+
			"<b>Storage:</b>\n"+
			"→ Database Size: <code>%.2f MB</code>\n"+
			"→ Storage Size: <code>%.2f MB</code>\n"+
			"→ Free Space: <code>%.2f MB</code>",
		stats.TotalMovies,
		stats.TotalTV,
		stats.TotalFiles,
		stats.TotalUsers,
		stats.DBSizeMB,
		stats.StorageSizeMB,
		stats.FreeSpaceMB,
	)

	m.Reply(message)
	return nil
}

func HandleSearch(m *tg.NewMessage) error {
	if !canSearch(m.Sender.ID) {
		m.Reply("<b>Access Denied</b>\n\nYou are not authorized to use this command.")
		return nil
	}

	query := strings.TrimSpace(strings.TrimPrefix(m.Text(), "/search"))
	if query == "" {
		m.Reply("<b>Search Files</b>\n\n<b>Usage:</b> <code>/search query</code>\n\n<b>Example:</b>\n<code>/search avengers</code>")
		return nil
	}

	results, err := db.SearchMedia(query)
	if err != nil {
		m.Reply(fmt.Sprintf("<b>Error:</b> Failed to search: %v", err))
		return nil
	}

	if len(results) == 0 {
		m.Reply(fmt.Sprintf("<b>Search Results</b>\n\nNo results found for: <code>%s</code>", query))
		return nil
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("<b>Search Results</b> (<code>%d</code> found)\n", len(results)))
	response.WriteString(fmt.Sprintf("<b>Query:</b> <code>%s</code>\n\n", query))

	for i, media := range results {
		if i >= 20 {
			response.WriteString(fmt.Sprintf("\n<i>... and %d more results</i>", len(results)-20))
			break
		}

		response.WriteString(fmt.Sprintf("<b>%d.</b> ", i+1))

		if media.MediaType == "movie" {
			response.WriteString(fmt.Sprintf("<b>%s</b> [Movie]\n", media.Title))
		} else {
			response.WriteString(fmt.Sprintf("<b>%s</b> [S%02dE%02d]\n", media.Title, media.Season, media.Episode))
		}

		response.WriteString(fmt.Sprintf("   → Quality: <code>%s</code>\n", media.Quality))
		response.WriteString(fmt.Sprintf("   → Size: <code>%.2f GB</code>\n", float64(media.FileSize)/(1024*1024*1024)))
		response.WriteString(fmt.Sprintf("   → File: <code>%s</code>\n", media.FileName))

		token := GenerateStreamToken(media.ChatID, media.MessageID)
		streamURL := fmt.Sprintf("%s/play?token=%s&file=%s", config.BaseURL, token, media.FileName)
		response.WriteString(fmt.Sprintf("   → Stream: %s\n\n", streamURL))
	}

	m.Reply(response.String())
	return nil
}

func HandleSearchByTitle(m *tg.NewMessage) error {
	if !canSearch(m.Sender.ID) {
		m.Reply("You are not authorized to use this command.")
		return nil
	}

	query := m.Args()
	if query == "" {
		m.Reply("<b>Search by Title</b>\n\n<b>Usage:</b> <code>/s movie/series title</code>\n\n<b>Example:</b>\n<code>/s breaking bad</code>")
		return nil
	}

	results, err := db.SearchByTitle(query)
	if err != nil {
		m.Reply(fmt.Sprintf("<b>Error:</b> Failed to search: %v", err))
		return nil
	}

	if len(results) == 0 {
		m.Reply(fmt.Sprintf("<b>Search Results</b>\n\nNo results found for: <code>%s</code>", query))
		return nil
	}

	titleMap := make(map[string]*titleInfo)

	for _, media := range results {
		key := fmt.Sprintf("%s_%s_%d", media.Title, media.MediaType, media.TMDBID)
		if titleMap[key] == nil {
			titleMap[key] = &titleInfo{
				Title:     media.Title,
				MediaType: media.MediaType,
				TMDBID:    media.TMDBID,
				Seasons:   make(map[int][]int),
				Qualities: make(map[string]bool),
				Files:     []string{},
			}
		}

		info := titleMap[key]

		if media.MediaType == "tv" {
			if info.Seasons[media.Season] == nil {
				info.Seasons[media.Season] = []int{}
			}
			hasEpisode := slices.Contains(info.Seasons[media.Season], media.Episode)
			if !hasEpisode {
				info.Seasons[media.Season] = append(info.Seasons[media.Season], media.Episode)
			}
		}

		if media.Quality != "" {
			info.Qualities[media.Quality] = true
		}
	}

	keyboard := tg.NewKeyboard()
	var response strings.Builder
	response.WriteString(fmt.Sprintf("<b>Search Results</b>\n<b>Query:</b> <code>%s</code>\n\n", query))
	response.WriteString(fmt.Sprintf("Found <b>%d</b> title(s). Select one to view details:\n", len(titleMap)))

	count := 0
	titlesList := make([]*titleInfo, 0, len(titleMap))
	for _, info := range titleMap {
		titlesList = append(titlesList, info)
	}

	for i := 0; i < len(titlesList) && i < 10; i++ {
		info := titlesList[i]
		typeText := "Movie"
		if info.MediaType == "tv" {
			typeText = "Series"
		}

		callbackData := fmt.Sprintf("title_%d_%s", info.TMDBID, info.MediaType)
		buttonText := fmt.Sprintf("%s [%s]", info.Title, typeText)

		keyboard.AddRow(tg.Button.Data(buttonText, callbackData))
		count++
	}

	if len(titleMap) > 10 {
		response.WriteString(fmt.Sprintf("\n<i>Showing first 10 of %d results</i>", len(titleMap)))
	}

	sentMsg, _ := m.Reply(response.String(), tg.SendOptions{
		ReplyMarkup: keyboard.Build(),
	})

	if sentMsg != nil {
		titlesCopy := make([]*titleInfo, len(titlesList))
		copy(titlesCopy, titlesList)

		searchContextMapMutex.Lock()
		searchContextMap[int(sentMsg.ID)] = searchContext{
			Query:      query,
			UserID:     m.Sender.ID,
			TitlesList: titlesCopy,
			Timestamp:  time.Now(),
		}
		searchContextMapMutex.Unlock()
	}

	return nil
}

func HandleCallback(c *tg.CallbackQuery) error {
	data := c.DataString()
	msgID := int(c.MessageID)
	senderID := c.OriginalUpdate.UserID
	searchContextMapMutex.RLock()
	ctx, hasContext := searchContextMap[msgID]
	searchContextMapMutex.RUnlock()

	if hasContext && ctx.UserID != senderID {
		c.Answer("This search was initiated by another user. Please use /s to start your own search.")
		return nil
	}

	if strings.HasPrefix(data, "title_") {
		parts := strings.Split(data, "_")
		if len(parts) != 3 {
			c.Answer("Invalid selection")
			return nil
		}

		tmdbID, err := strconv.Atoi(parts[1])
		if err != nil {
			c.Answer("Invalid ID")
			return nil
		}

		mediaType := parts[2]

		results, err := db.GetMediaByTMDBID(tmdbID, mediaType)
		if err != nil {
			c.Answer("Error fetching details")
			return nil
		}

		if len(results) == 0 {
			c.Answer("No files found")
			return nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("<b>%s</b> [%s]\n\n", results[0].Title, strings.ToUpper(mediaType)))

		posterURL := getTMDBPoster(tmdbID, mediaType)
		if mediaType == "tv" {

			seasonMap := make(map[int][]int)
			for _, media := range results {
				if seasonMap[media.Season] == nil {
					seasonMap[media.Season] = []int{}
				}
				if !slices.Contains(seasonMap[media.Season], media.Episode) {
					seasonMap[media.Season] = append(seasonMap[media.Season], media.Episode)
				}
			}

			keyboard := tg.NewKeyboard()
			for season := range seasonMap {
				callbackData := fmt.Sprintf("season_%d_%d_%s", tmdbID, season, mediaType)
				buttonText := fmt.Sprintf("Season %d (%d episodes)", season, len(seasonMap[season]))
				keyboard.AddRow(tg.Button.Data(buttonText, callbackData))
			}

			keyboard.AddRow(tg.Button.Data("« Back to Search", "back_search"))

			response.WriteString(fmt.Sprintf("Available <b>%d</b> season(s). Select to view episodes:", len(seasonMap)))

			opts := &tg.SendOptions{
				ReplyMarkup: keyboard.Build(),
			}
			if posterURL != "" {
				opts.Media = &tg.InputMediaPhotoExternal{
					URL: posterURL,
				}
			}
			c.Edit(response.String(), opts)
		} else {

			qualityMap := make(map[string]int)
			for _, media := range results {
				qualityMap[media.Quality]++
			}

			keyboard := tg.NewKeyboard()

			for _, media := range results {
				hasDuplicates := qualityMap[media.Quality] > 1

				var buttonText string
				if hasDuplicates {
					codec := ExtractCodecFunc(media.FileName)
					if codec != "" {
						buttonText = fmt.Sprintf("%s %s (%.2f GB)", media.Quality, codec, float64(media.FileSize)/(1024*1024*1024))
					} else {
						buttonText = fmt.Sprintf("%s (%.2f GB)", media.Quality, float64(media.FileSize)/(1024*1024*1024))
					}
				} else {
					buttonText = fmt.Sprintf("%s (%.2f GB)", media.Quality, float64(media.FileSize)/(1024*1024*1024))
				}

				callbackData := fmt.Sprintf("movie_%d_%s", tmdbID, media.Quality)
				keyboard.AddRow(tg.Button.Data(buttonText, callbackData))
			}

			keyboard.AddRow(tg.Button.Data("« Back to Search", "back_search"))

			response.WriteString("Select quality to stream:")

			opts := &tg.SendOptions{
				ReplyMarkup: keyboard.Build(),
			}
			if posterURL != "" {
				opts.Media = &tg.InputMediaPhotoExternal{
					URL: posterURL,
				}
			}
			c.Edit(response.String(), opts)
		}

		return nil
	}

	if strings.HasPrefix(data, "season_") {
		parts := strings.Split(data, "_")
		if len(parts) != 4 {
			c.Answer("Invalid selection")
			return nil
		}

		tmdbID, _ := strconv.Atoi(parts[1])
		season, _ := strconv.Atoi(parts[2])
		mediaType := parts[3]

		episodes, err := db.GetEpisodesBySeason(tmdbID, season)
		if err != nil {
			c.Answer("Error fetching episodes")
			return nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("<b>%s - Season %d</b>\n\n", episodes[0].Title, season))
		response.WriteString("Select episode:\n")

		episodeQualityMap := make(map[string]int)
		for _, ep := range episodes {
			key := fmt.Sprintf("%d_%s", ep.Episode, ep.Quality)
			episodeQualityMap[key]++
		}

		keyboard := tg.NewKeyboard()
		for _, ep := range episodes {
			key := fmt.Sprintf("%d_%s", ep.Episode, ep.Quality)
			hasDuplicates := episodeQualityMap[key] > 1

			var buttonText string
			if hasDuplicates {
				codec := ExtractCodecFunc(ep.FileName)
				if codec != "" {
					buttonText = fmt.Sprintf("E%02d - %s %s (%.2f GB)", ep.Episode, ep.Quality, codec, float64(ep.FileSize)/(1024*1024*1024))
				} else {
					buttonText = fmt.Sprintf("E%02d - %s (%.2f GB)", ep.Episode, ep.Quality, float64(ep.FileSize)/(1024*1024*1024))
				}
			} else {
				buttonText = fmt.Sprintf("E%02d - %s (%.2f GB)", ep.Episode, ep.Quality, float64(ep.FileSize)/(1024*1024*1024))
			}

			callbackData := fmt.Sprintf("ep_%d_%d", ep.ChatID, ep.MessageID)
			keyboard.AddRow(tg.Button.Data(buttonText, callbackData))
		}

		keyboard.AddRow(tg.Button.Data("« Back to Seasons", fmt.Sprintf("title_%d_%s", tmdbID, mediaType)))

		c.Edit(response.String(), &tg.SendOptions{
			ReplyMarkup: keyboard.Build(),
		})

		return nil
	}

	if strings.HasPrefix(data, "ep_") {
		parts := strings.Split(data, "_")
		if len(parts) != 3 {
			c.Answer("Invalid selection")
			return nil
		}

		chatID, _ := strconv.ParseInt(parts[1], 10, 64)
		messageID, _ := strconv.Atoi(parts[2])

		media, err := db.GetMediaByChatMessage(chatID, messageID)
		if err != nil || media == nil {
			c.Answer("File not found")
			return nil
		}

		token := GenerateStreamToken(media.ChatID, media.MessageID)
		streamURL := fmt.Sprintf("%s/play?token=%s", config.BaseURL, token)
		var response strings.Builder
		response.WriteString(fmt.Sprintf("<b>%s</b>\n\n", media.Title))
		response.WriteString(fmt.Sprintf("S%02dE%02d • <code>%s</code> • <code>%.2f GB</code>\n\n", media.Season, media.Episode, media.Quality, float64(media.FileSize)/(1024*1024*1024)))
		response.WriteString(fmt.Sprintf("<b>File:</b> <code>%s</code>", media.FileName))

		keyboard := tg.NewKeyboard()
		keyboard.AddRow(tg.Button.URL("Stream File", streamURL))
		keyboard.AddRow(tg.Button.Data("Get File", fmt.Sprintf("filedata_%d_%d", chatID, messageID)))
		keyboard.AddRow(tg.Button.Data("« Back to Episodes", fmt.Sprintf("season_%d_%d_tv", media.TMDBID, media.Season)))

		c.Edit(response.String(), &tg.SendOptions{
			ReplyMarkup: keyboard.Build(),
		})
		return nil
	}

	if strings.HasPrefix(data, "movie_") {
		parts := strings.Split(data, "_")
		if len(parts) < 3 {
			c.Answer("Invalid selection")
			return nil
		}

		tmdbID, _ := strconv.Atoi(parts[1])
		quality := strings.Join(parts[2:], "_")

		media, err := db.GetMediaByQuality(tmdbID, "movie", 0, 0, quality)
		if err != nil || media == nil {
			c.Answer("File not found")
			return nil
		}

		token := GenerateStreamToken(media.ChatID, media.MessageID)
		streamURL := fmt.Sprintf("%s/play?token=%s&file=%s", config.BaseURL, token, media.FileName)

		var response strings.Builder
		response.WriteString(fmt.Sprintf("<b>%s</b>\n\n", media.Title))
		response.WriteString(fmt.Sprintf("<code>%s</code> • <code>%.2f GB</code>\n\n", quality, float64(media.FileSize)/(1024*1024*1024)))
		response.WriteString(fmt.Sprintf("<b>File:</b> <code>%s</code>", media.FileName))

		keyboard := tg.NewKeyboard()
		keyboard.AddRow(tg.Button.URL("Stream File", streamURL))
		keyboard.AddRow(tg.Button.Data("Get File", fmt.Sprintf("filedata_%d_movie_%s", tmdbID, quality)))
		keyboard.AddRow(tg.Button.Data("« Back to Qualities", fmt.Sprintf("title_%d_movie", tmdbID)))

		c.Edit(response.String(), &tg.SendOptions{
			ReplyMarkup: keyboard.Build(),
		})
		return nil
	}

	if strings.HasPrefix(data, "filedata_") {
		c.Answer("Feature coming soon!")
		return nil
	}

	if data == "back_search" {

		searchContextMapMutex.RLock()
		ctx, hasContext := searchContextMap[msgID]
		searchContextMapMutex.RUnlock()

		if !hasContext {
			c.Edit("<b>Search</b>\n\nUse <code>/s &lt;title&gt;</code> to search again.", &tg.SendOptions{})
			c.Answer("")
			return nil
		}

		keyboard := tg.NewKeyboard()
		var response strings.Builder
		response.WriteString(fmt.Sprintf("<b>Search Results</b>\n<b>Query:</b> <code>%s</code>\n\n", ctx.Query))
		response.WriteString(fmt.Sprintf("Found <b>%d</b> title(s). Select one to view details:\n", len(ctx.TitlesList)))

		for i := 0; i < len(ctx.TitlesList) && i < 10; i++ {
			info := ctx.TitlesList[i]
			typeText := "Movie"
			if info.MediaType == "tv" {
				typeText = "Series"
			}

			callbackData := fmt.Sprintf("title_%d_%s", info.TMDBID, info.MediaType)
			buttonText := fmt.Sprintf("%s [%s]", info.Title, typeText)
			keyboard.AddRow(tg.Button.Data(buttonText, callbackData))
		}

		if len(ctx.TitlesList) > 10 {
			response.WriteString(fmt.Sprintf("\n<i>Showing first 10 of %d results</i>", len(ctx.TitlesList)))
		}

		c.Edit(response.String(), &tg.SendOptions{
			ReplyMarkup: keyboard.Build(),
		})
		return nil
	}

	return nil
}

func HandleNewMessage(m *tg.NewMessage) error {
	if m.Sender == nil {
		return nil
	}

	userID := m.Sender.ID
	username := m.Sender.Username
	firstName := m.Sender.FirstName
	lastName := m.Sender.LastName

	err := db.AddUser(userID, username, firstName, lastName)
	if err != nil {
		return err
	}

	return nil
}

func getTMDBPoster(tmdbID int, mediaType string) string {
	endpoint := "movie"
	if mediaType == "tv" {
		endpoint = "tv"
	}

	url := fmt.Sprintf("https://api.themoviedb.org/3/%s/%d?api_key=%s", endpoint, tmdbID, config.TMDBAPIKey)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[TMDB] Error fetching poster: %v", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[TMDB] Error reading response: %v", err)
		return ""
	}

	var result struct {
		PosterPath string `json:"poster_path"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[TMDB] Error parsing JSON: %v", err)
		return ""
	}

	if result.PosterPath != "" {
		return "https://image.tmdb.org/t/p/w500" + result.PosterPath
	}

	return ""
}
