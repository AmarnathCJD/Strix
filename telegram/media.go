package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type MediaAddState struct {
	IMDBID      string
	TMDBID      int
	MediaType   string
	Title       string
	PosterPath  string
	Season      int
	Episode     int
	Quality     string
	CDNBotIndex int
}

func HandleAddMedia(m *tg.NewMessage) error {
	if !isAuthorized(m.Sender.ID) {
		m.Reply("⚠️ <b>Access Denied</b>\n\nYou are not authorized to use this command.")
		return nil
	}

	var fileOrURLResp *tg.NewMessage
	var err error

	if m.IsReply() {
		repliedMsg, err := m.GetReplyMessage()
		if err == nil && repliedMsg.IsMedia() {
			fileOrURLResp = repliedMsg
		}
	}

	if fileOrURLResp == nil {
		fileOrURLResp, err = m.Ask("<b>Media Upload</b>\n\n→ Send video file directly, or\n→ Send Telegram post URL\n\n<b>Format:</b> <code>https://t.me/username/messageID</code>")
		if err != nil {
			m.Reply("❌ <b>Error</b>\n\n" + err.Error())
			return nil
		}
	}

	var isForwardedFile bool
	var parsedMetadata *FileMetadata
	var analyzeMsg *tg.NewMessage
	var analyzeMsgText string

	if fileOrURLResp.Media() != nil {
		if fileOrURLResp.File == nil || fileOrURLResp.File.Name == "" {
			m.Reply("❌ <b>Invalid Media</b>\n\nThis is not a video file. Please send a video file.")
			return nil
		}

		if !IsVideoFileFunc(fileOrURLResp.File.Name) {
			m.Reply("❌ <b>Invalid File Type</b>\n\nPlease send a video file (mkv, mp4, avi, etc.).")
			return nil
		}

		analyzeMsg, _ = m.Reply("🔍 <b>Analyzing file for metadata...</b>")

		isForwardedFile = true
		parsedMetadata = ParseFilenameFunc(fileOrURLResp.File.Name)

		analyzeMsgText = fmt.Sprintf("✅ <b>Metadata Detected</b>\n\n→ <b>Title:</b> %s\n→ <b>Season:</b> %d | <b>Episode:</b> %d\n→ <b>Quality:</b> %s",
			parsedMetadata.Title, parsedMetadata.Season, parsedMetadata.Episode, parsedMetadata.Quality)

		if analyzeMsg != nil {
			analyzeMsg.Edit(analyzeMsgText)
		}
	} else {
		isForwardedFile = false
	}

	var tmdbID int
	var mediaType string
	var title string
	var posterPath string
	var searchQuery string

	if parsedMetadata != nil && parsedMetadata.Title != "" {
		if parsedMetadata.Year > 0 {
			searchQuery = fmt.Sprintf("%s %d", parsedMetadata.Title, parsedMetadata.Year)
		} else {
			searchQuery = parsedMetadata.Title
		}
	} else {
		imdbResp, err := m.Ask("<b>Media Identification</b>\n\n→ Enter IMDb ID (e.g., <code>tt1234567</code>), or\n→ Enter search query")
		if err != nil {
			m.Reply("❌ <b>Error</b>\n\n" + err.Error())
			return nil
		}
		searchQuery = strings.TrimSpace(imdbResp.Text())
	}

	if strings.HasPrefix(strings.ToLower(searchQuery), "tt") {
		tmdbID, mediaType, title, posterPath, err = getTMDBFromIMDB(searchQuery)
		if err != nil {
			m.Reply("❌ <b>TMDB Fetch Error</b>\n\n" + err.Error())
			return nil
		}
	} else {
		results, err := searchTMDB(searchQuery)
		if err != nil {
			m.Reply("❌ <b>TMDB Search Error</b>\n\n" + err.Error())
			return nil
		}

		if len(results) == 0 && parsedMetadata != nil && parsedMetadata.Year > 0 {
			results, err = searchTMDB(parsedMetadata.Title)
			if err != nil {
				m.Reply("❌ <b>TMDB Search Error</b>\n\n" + err.Error())
				return nil
			}
		}

		if len(results) == 0 {
			imdbResp, err := m.Ask("❌ <b>No Results Found</b>\n\n→ Enter IMDb ID (e.g., <code>tt1234567</code>), or\n→ Enter a different search query")
			if err != nil {
				m.Reply("❌ <b>Error</b>\n\n" + err.Error())
				return nil
			}
			searchQuery = strings.TrimSpace(imdbResp.Text())

			if strings.HasPrefix(strings.ToLower(searchQuery), "tt") {
				tmdbID, mediaType, title, posterPath, err = getTMDBFromIMDB(searchQuery)
				if err != nil {
					m.Reply("❌ <b>TMDB Fetch Error</b>\n\n" + err.Error())
					return nil
				}
			} else {
				results, err = searchTMDB(searchQuery)
				if err != nil {
					m.Reply("❌ <b>TMDB Search Error</b>\n\n" + err.Error())
					return nil
				}

				if len(results) == 0 {
					m.Reply("❌ <b>No Results Found</b>\n\nPlease run <code>/add</code> again.")
					return nil
				}
			}
		}

		if len(results) > 0 {
			var resultMsg strings.Builder
			resultMsg.WriteString("<b>Search Results</b>\n\n")
			keyboard := tg.NewKeyboard()
			for i, r := range results {
				if i >= 5 {
					break
				}
				keyboard.AddRow(
					tg.Button.Data(fmt.Sprintf("%d. %s - %s - %s", i+1, r.Title, r.Year, r.Type), fmt.Sprintf("select_%d", i+1)),
				)
			}
			resultMsg.WriteString(fmt.Sprintf("\n→ <b>Enter your choice:</b> %d-%d", 1, len(results)))

			m, _ := m.Respond(resultMsg.String(), tg.SendOptions{
				ReplyMarkup: keyboard.Build(),
			})
			defer m.Delete()
			upd, err := m.WaitClick(60)
			if err != nil {
				m.Reply("❌ <b>Error</b>\n\n" + err.Error())
				return nil
			}

			choiceStr := strings.TrimPrefix(upd.DataString(), "select_")
			choice, err := strconv.Atoi(choiceStr)
			if err != nil || choice < 1 || choice > len(results) || choice > 10 {
				m.Reply("❌ <b>Invalid Choice</b>\n\nPlease run <code>/add</code> again.")
				return nil
			}

			selected := results[choice-1]
			tmdbID = selected.ID
			mediaType = selected.Type
			title = selected.Title
			posterPath = selected.PosterPath
		}
	}

	state := &MediaAddState{
		IMDBID:     searchQuery,
		TMDBID:     tmdbID,
		MediaType:  mediaType,
		Title:      title,
		PosterPath: posterPath,
	}

	if mediaType == "tv" {
		if parsedMetadata != nil && parsedMetadata.Season > 0 {
			state.Season = parsedMetadata.Season
			state.Episode = parsedMetadata.Episode
		} else {
			seasonResp, err := m.Ask(fmt.Sprintf("<b>%s</b>\n<i>TV Series</i>\n\n→ Enter season number:", title))
			if err != nil {
				m.Reply("❌ <b>Error</b>\n\n" + err.Error())
				return nil
			}
			season, _ := strconv.Atoi(seasonResp.Text())
			state.Season = season

			episodeResp, err := m.Ask("→ Enter episode number:")
			if err != nil {
				m.Reply("❌ <b>Error</b>\n\n" + err.Error())
				return nil
			}
			episode, _ := strconv.Atoi(episodeResp.Text())
			state.Episode = episode
		}
	}

	if parsedMetadata != nil && parsedMetadata.Quality != "" {
		state.Quality = parsedMetadata.Quality
	} else {
		qualityResp, err := m.Ask("<b>Video Quality</b>\n\n→ Enter quality: <code>1080p</code>, <code>720p</code>, <code>480p</code>, etc.")
		if err != nil {
			m.Reply("❌ <b>Error</b>\n\n" + err.Error())
			return nil
		}
		state.Quality = qualityResp.Text()
	}

	var saveErr error
	if isForwardedFile {
		saveErr = saveMediaFromForwardedFile(fileOrURLResp, state)
	} else {
		saveErr = saveMediaFromURL(fileOrURLResp.Text(), state)
	}

	if saveErr != nil {
		m.Reply("❌ <b>Save Failed</b>\n\n" + saveErr.Error())
		return nil
	}

	if analyzeMsg != nil {
		finalMsg := analyzeMsgText + "\n\n✅ <b>Media Added Successfully</b>"
		analyzeMsg.Edit(finalMsg)
	} else {
		var successMsg string
		if state.MediaType == "tv" {
			successMsg = fmt.Sprintf("✅ <b>Media Added Successfully</b>\n\n<b>%s</b>\n→ S%02dE%02d • <code>%s</code>", state.Title, state.Season, state.Episode, state.Quality)
		} else {
			successMsg = fmt.Sprintf("✅ <b>Media Added Successfully</b>\n\n<b>%s</b> • <code>%s</code>", state.Title, state.Quality)
		}

		if state.PosterPath != "" {
			posterURL := fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", state.PosterPath)
			if _, err := m.ReplyMedia(posterURL, tg.MediaOptions{
				Caption: successMsg,
			}); err != nil {
				m.Reply(successMsg)
			}
		} else {
			m.Reply(successMsg)
		}
	}

	return nil
}

func HandleAddMulti(m *tg.NewMessage) error {
	if !isAuthorized(m.Sender.ID) {
		m.Reply("⚠️ <b>Access Denied</b>\n\nYou are not authorized to use this command.")
		return nil
	}

	imdbResp, err := m.Ask("<b>Batch Upload</b>\n\n→ Enter IMDb ID (e.g., <code>tt1234567</code>), or\n→ Enter search query")
	if err != nil {
		m.Reply("❌ <b>Error</b>\n\n" + err.Error())
		return nil
	}

	searchQuery := strings.TrimSpace(imdbResp.Text())
	var tmdbID int
	var mediaType string
	var title string
	var posterPath string

	if strings.HasPrefix(strings.ToLower(searchQuery), "tt") {
		tmdbID, mediaType, title, posterPath, err = getTMDBFromIMDB(searchQuery)
		if err != nil {
			m.Reply("❌ <b>TMDB Fetch Error</b>\n\n" + err.Error())
			return nil
		}
	} else {
		results, err := searchTMDB(searchQuery)
		if err != nil {
			m.Reply("❌ <b>TMDB Search Error</b>\n\n" + err.Error())
			return nil
		}

		if len(results) == 0 {
			imdbResp, err := m.Ask("❌ <b>No Results Found</b>\n\n→ Enter IMDb ID (e.g., <code>tt1234567</code>), or\n→ Enter a different search query")
			if err != nil {
				m.Reply("❌ <b>Error</b>\n\n" + err.Error())
				return nil
			}
			searchQuery = strings.TrimSpace(imdbResp.Text())

			if strings.HasPrefix(strings.ToLower(searchQuery), "tt") {
				tmdbID, mediaType, title, posterPath, err = getTMDBFromIMDB(searchQuery)
				if err != nil {
					m.Reply("❌ <b>TMDB Fetch Error</b>\n\n" + err.Error())
					return nil
				}
			} else {
				results, err = searchTMDB(searchQuery)
				if err != nil {
					m.Reply("❌ <b>TMDB Search Error</b>\n\n" + err.Error())
					return nil
				}

				if len(results) == 0 {
					m.Reply("❌ <b>No Results Found</b>\n\nPlease run <code>/addmulti</code> again.")
					return nil
				}
			}
		}

		if len(results) > 0 {
			var resultMsg strings.Builder
			resultMsg.WriteString("<b>Search Results</b>\n\n")
			keyboard := tg.NewKeyboard()
			for i, r := range results {
				if i >= 5 {
					break
				}
				keyboard.AddRow(
					tg.Button.Data(fmt.Sprintf("%d. %s - %s - %s", i+1, r.Title, r.Year, r.Type), fmt.Sprintf("select_multi_%d", i+1)),
				)
			}
			resultMsg.WriteString(fmt.Sprintf("\n→ <b>Enter your choice:</b> %d-%d", 1, len(results)))

			msg, _ := m.Respond(resultMsg.String(), tg.SendOptions{
				ReplyMarkup: keyboard.Build(),
			})
			defer msg.Delete()
			upd, err := msg.WaitClick(60)
			if err != nil {
				m.Reply("❌ <b>Error</b>\n\n" + err.Error())
				return nil
			}

			choiceStr := strings.TrimPrefix(upd.DataString(), "select_multi_")
			choice, err := strconv.Atoi(choiceStr)
			if err != nil || choice < 1 || choice > len(results) || choice > 10 {
				m.Reply("❌ <b>Invalid Choice</b>\n\nPlease run <code>/addmulti</code> again.")
				return nil
			}

			selected := results[choice-1]
			tmdbID = selected.ID
			mediaType = selected.Type
			title = selected.Title
			posterPath = selected.PosterPath
		}
	}

	state := &MediaAddState{
		IMDBID:     searchQuery,
		TMDBID:     tmdbID,
		MediaType:  mediaType,
		Title:      title,
		PosterPath: posterPath,
	}

	m.Reply(fmt.Sprintf("<b>Batch Mode Active</b>\n\n<b>Title:</b> %s\n<b>Type:</b> %s\n\n→ Send video files or Telegram URLs\n→ Send <code>/done</code> when finished", title, mediaType))

	for {
		fileMsg, err := m.Ask("→ Send file/URL <i>(or <code>/done</code> to finish)</i>:")
		if err != nil {
			m.Reply("❌ <b>Error</b>\n\n" + err.Error())
			break
		}

		if strings.ToLower(fileMsg.Text()) == "/done" {
			m.Reply("✅ <b>Batch Upload Complete</b>")
			break
		}

		var parsedMetadata *FileMetadata
		var isFile bool

		if fileMsg.Media() != nil {
			if fileMsg.IsGroup() {
				continue
			}

			if fileMsg.File == nil || fileMsg.File.Name == "" {
				m.Reply("❌ <b>Invalid Media</b>\n\nThis is not a video file. Please send a video file.")
				continue
			}

			if !IsVideoFileFunc(fileMsg.File.Name) {
				m.Reply("❌ <b>Invalid File Type</b>\n\nPlease send a video file (mkv, mp4, avi, etc.).")
				continue
			}

			m.Reply("🔍 <b>Analyzing file for metadata...</b>")

			isFile = true
			parsedMetadata = ParseFilenameFunc(fileMsg.File.Name)

			if parsedMetadata.Season > 0 || parsedMetadata.Episode > 0 || parsedMetadata.Quality != "" {
				m.Reply(fmt.Sprintf("✅ <b>Detected:</b> S%dE%d • <code>%s</code> • <b>%s</b>", parsedMetadata.Season, parsedMetadata.Episode, parsedMetadata.Quality, parsedMetadata.Title))
			} else {
				m.Reply(fmt.Sprintf("✅ <b>Detected:</b> <b>%s</b> • <code>%s</code>", parsedMetadata.Title, parsedMetadata.Quality))
			}
		} else {
			url := strings.TrimSpace(fileMsg.Text())
			if !strings.HasPrefix(url, "https://t.me/") && !strings.HasPrefix(url, "http://t.me/") {
				m.Reply("❌ <b>Invalid Input</b>\n\nSend a file or valid URL: <code>https://t.me/username/messageID</code>\n\n<i>Or send <code>/done</code> to finish</i>")
				continue
			}
			isFile = false
		}

		if mediaType == "tv" {
			if parsedMetadata != nil && parsedMetadata.Season > 0 {
				state.Season = parsedMetadata.Season
				state.Episode = parsedMetadata.Episode
			} else {
				seasonEpResp, err := m.Ask("→ Enter season and episode\n\n<b>Example:</b> <code>1 5</code> for S01E05")
				if err != nil {
					m.Reply("❌ <b>Error</b>\n\n" + err.Error())
					continue
				}

				parts := strings.Fields(seasonEpResp.Text())
				if len(parts) < 2 {
					m.Reply("❌ <b>Invalid Format</b>\n\n<b>Expected:</b> <code>season episode</code>\n<b>Example:</b> <code>1 5</code>")
					continue
				}

				season, err := strconv.Atoi(parts[0])
				if err != nil {
					m.Reply("❌ <b>Invalid Season Number</b>")
					continue
				}
				episode, err := strconv.Atoi(parts[1])
				if err != nil {
					m.Reply("❌ <b>Invalid Episode Number</b>")
					continue
				}

				state.Season = season
				state.Episode = episode
			}
		}

		if parsedMetadata != nil && parsedMetadata.Quality != "" {
			state.Quality = parsedMetadata.Quality
		} else {
			qualityResp, err := m.Ask("<b>Video Quality</b>\n\n→ Enter quality: <code>1080p</code>, <code>720p</code>, <code>480p</code>, etc.")
			if err != nil {
				m.Reply("❌ <b>Error</b>\n\n" + err.Error())
				continue
			}
			state.Quality = qualityResp.Text()
		}

		var saveErr error
		if isFile {
			saveErr = saveMediaFromForwardedFile(fileMsg, state)
		} else {
			saveErr = saveMediaFromURL(fileMsg.Text(), state)
		}

		if saveErr != nil {
			m.Reply("❌ <b>Save Failed</b>\n\n" + saveErr.Error())
		} else {
			if state.MediaType == "tv" {
				m.Reply(fmt.Sprintf("✅ <b>Added</b>\n\n→ S%02dE%02d • <code>%s</code>", state.Season, state.Episode, state.Quality))
			} else {
				m.Reply(fmt.Sprintf("✅ <b>Added</b>\n\n→ <code>%s</code>", state.Quality))
			}
		}
	}

	return nil
}

func saveMediaFromURL(url string, state *MediaAddState) error {
	parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://"), "/")
	if len(parts) < 3 || parts[0] != "t.me" {
		return fmt.Errorf("invalid Telegram URL format")
	}

	username := parts[1]
	msgIDStr := parts[2]

	msgIDStr = strings.Split(msgIDStr, "?")[0]
	msgIDStr = strings.Split(msgIDStr, "#")[0]

	messageID, err := strconv.Atoi(msgIDStr)
	if err != nil {
		return fmt.Errorf("invalid message ID: %s", msgIDStr)
	}

	peer, err := bot.ResolvePeer(username)
	if err != nil {
		return fmt.Errorf("failed to resolve username '%s': %w", username, err)
	}

	var chatID int64
	switch p := peer.(type) {
	case *tg.InputPeerUser:
		chatID = p.UserID
	case *tg.InputPeerChat:
		chatID = p.ChatID
	case *tg.InputPeerChannel:
		chatID = p.ChannelID
	default:
		return fmt.Errorf("unsupported peer type")
	}

	fi, err := bot.GetMessageByID(chatID, int32(messageID))
	if err != nil {
		return fmt.Errorf("failed to get message by ID: %w", err)
	}

	err = db.AddMedia(
		state.TMDBID,
		state.MediaType,
		state.Title,
		fi.File.FileID,
		messageID,
		chatID,
		fi.File.Size,
		fi.File.Name,
		state.Season,
		state.Episode,
		state.Quality,
		state.CDNBotIndex,
	)

	return err
}

type TMDBSearchResult struct {
	ID         int
	Title      string
	Year       string
	Type       string
	PosterPath string
}

func searchTMDB(query string) ([]TMDBSearchResult, error) {
	url := fmt.Sprintf("https://api.themoviedb.org/3/search/multi?api_key=%s&query=%s",
		config.TMDBAPIKey, url.QueryEscape(query))

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Results []struct {
			ID           int    `json:"id"`
			Title        string `json:"title"`
			Name         string `json:"name"`
			MediaType    string `json:"media_type"`
			PosterPath   string `json:"poster_path"`
			ReleaseDate  string `json:"release_date"`
			FirstAirDate string `json:"first_air_date"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var results []TMDBSearchResult
	for _, r := range result.Results {
		if r.MediaType != "movie" && r.MediaType != "tv" {
			continue
		}

		title := r.Title
		if title == "" {
			title = r.Name
		}

		year := r.ReleaseDate
		if year == "" {
			year = r.FirstAirDate
		}
		if len(year) >= 4 {
			year = year[:4]
		}

		mediaType := r.MediaType
		if mediaType == "movie" {
			mediaType = "movie"
		} else {
			mediaType = "tv"
		}

		results = append(results, TMDBSearchResult{
			ID:         r.ID,
			Title:      title,
			Year:       year,
			Type:       mediaType,
			PosterPath: r.PosterPath,
		})
	}

	return results, nil
}

func saveMediaFromForwardedFile(msg *tg.NewMessage, state *MediaAddState) error {
	indexChannelID := config.IndexChannel
	useIndexChannel := indexChannelID != 0

	var targetChatID int64
	var targetMsgID int
	var fileID string
	var fileSize int64
	var fileName string

	if useIndexChannel {
		targetMsg, err := msg.ForwardTo(indexChannelID)
		if err != nil {
			return fmt.Errorf("failed to forward message to index channel: %w", err)
		}
		targetChatID = indexChannelID
		targetMsgID = int(targetMsg.ID)
	} else {
		targetChatID = msg.ChatID()
		targetMsgID = int(msg.ID)
	}

	fileID = msg.File.FileID
	fileSize = msg.File.Size
	fileName = msg.File.Name

	if fileName != "" && state.Season == 0 && state.MediaType == "tv" {
		parsed := ParseFilenameFunc(fileName)
		if parsed.Season > 0 {
			msg.Reply(fmt.Sprintf("💡 <b>Detected:</b> S%02dE%02d", parsed.Season, parsed.Episode))
		}
	}

	err := db.AddMedia(
		state.TMDBID,
		state.MediaType,
		fmt.Sprintf("%s [%s]", state.Title, state.Quality),
		fileID,
		targetMsgID,
		targetChatID,
		fileSize,
		fileName,
		state.Season,
		state.Episode,
		state.Quality,
		state.CDNBotIndex,
	)

	return err
}

func getTMDBFromIMDB(imdbID string) (int, string, string, string, error) {
	url := fmt.Sprintf("https://api.themoviedb.org/3/find/%s?api_key=%s&external_source=imdb_id",
		imdbID, config.TMDBAPIKey)

	resp, err := http.Get(url)
	if err != nil {
		return 0, "", "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", "", "", err
	}

	var result struct {
		MovieResults []struct {
			ID         int    `json:"id"`
			Title      string `json:"title"`
			PosterPath string `json:"poster_path"`
		} `json:"movie_results"`
		TVResults []struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			PosterPath string `json:"poster_path"`
		} `json:"tv_results"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, "", "", "", err
	}

	if len(result.MovieResults) > 0 {
		return result.MovieResults[0].ID, "movie", result.MovieResults[0].Title, result.MovieResults[0].PosterPath, nil
	}

	if len(result.TVResults) > 0 {
		return result.TVResults[0].ID, "tv", result.TVResults[0].Name, result.TVResults[0].PosterPath, nil
	}

	return 0, "", "", "", fmt.Errorf("no results found for IMDb ID: %s", imdbID)
}
