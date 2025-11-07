package telegram

import (
	"slices"

	tg "github.com/amarnathcjd/gogram/telegram"

	cfg "strix/config"
	"strix/database"
)

type FileMetadata struct {
	Season  int
	Episode int
	Quality string
	Title   string
	Year    int
}

var bot *tg.Client
var db *database.DB
var config *cfg.Config
var ParseFilenameFunc func(string) *FileMetadata
var IsVideoFileFunc func(string) bool
var GeminiAPIKey string

func InitBot(c *cfg.Config, d *database.DB) error {
	config = c
	db = d

	client, err := tg.NewClient(tg.ClientConfig{
		AppID:    int32(config.AppID),
		AppHash:  config.AppHash,
		Cache:    tg.NewCache("strix.db"),
		LogLevel: tg.LogInfo,
	})

	if err != nil {
		return err
	}

	client.Conn()
	if err := client.LoginBot(config.BotToken); err != nil {
		return err
	}

	bot = client
	registerCommands()

	return nil
}

func registerCommands() {
	bot.On("command:add", HandleAddMedia)
	bot.On("command:addmulti", HandleAddMulti)
	bot.On("command:stats", HandleStats)
	bot.On("command:search", HandleSearch)
	bot.On(tg.OnNewMessage, HandleNewMessage)
}

func isAuthorized(userID int64) bool {
	if userID == config.OwnerID {
		return true
	}
	return slices.Contains(config.AuthUsers, userID)
}
