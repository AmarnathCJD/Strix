package telegram

import (
	"fmt"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func HandleStats(m *tg.NewMessage) error {
	if !isAuthorized(m.Sender.ID) {
		m.Reply("⚠️ <b>Access Denied</b>\n\nYou are not authorized to use this command.")
		return nil
	}

	stats, err := db.GetStats()
	if err != nil {
		m.Reply(fmt.Sprintf("❌ <b>Error</b>\n\nFailed to get stats: %v", err))
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
	if !isAuthorized(m.Sender.ID) {
		m.Reply("⚠️ <b>Access Denied</b>\n\nYou are not authorized to use this command.")
		return nil
	}

	query := strings.TrimSpace(strings.TrimPrefix(m.Text(), "/search"))
	if query == "" {
		m.Reply("<b>Search Files</b>\n\n<b>Usage:</b> <code>/search query</code>\n\n<b>Example:</b>\n<code>/search avengers</code>")
		return nil
	}

	results, err := db.SearchMedia(query)
	if err != nil {
		m.Reply(fmt.Sprintf("❌ <b>Error</b>\n\nFailed to search: %v", err))
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
		response.WriteString(fmt.Sprintf("   → Token: <code>%s</code>\n\n", GenerateStreamToken(media.ChatID, media.MessageID)))
	}

	m.Reply(response.String())
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
