package telegram

import (
	"sync"

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
var ExtractCodecFunc func(string) string
var GeminiAPIKey string

// In-memory auth cache
var (
	authUsersCache    = make(map[int64]bool)
	authUsersMutex    sync.RWMutex
	publicAccessCache bool
	publicAccessMutex sync.RWMutex
)

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

	// Load auth users and settings into memory
	if err := loadAuthCache(); err != nil {
		return err
	}

	registerCommands()

	return nil
}

func loadAuthCache() error {
	// Load auth users
	authUsers, err := db.GetAllAuthUsers()
	if err != nil {
		return err
	}

	authUsersMutex.Lock()
	authUsersCache = make(map[int64]bool)
	for _, user := range authUsers {
		authUsersCache[user.UserID] = true
	}
	authUsersMutex.Unlock()

	// Load public access setting
	publicAccess, err := db.GetPublicAccess()
	if err != nil {
		return err
	}

	publicAccessMutex.Lock()
	publicAccessCache = publicAccess
	publicAccessMutex.Unlock()

	return nil
}

func registerCommands() {
	bot.On("command:add", HandleAddMedia)
	bot.On("command:addmulti", HandleAddMulti)
	bot.On("command:stats", HandleStats)
	bot.On("command:search", HandleSearch)
	bot.On("command:s", HandleSearchByTitle)
	bot.On("command:auth", HandleAddAuth)
	bot.On("command:removeauth", HandleRemoveAuth)
	bot.On("command:listauth", HandleListAuth)
	bot.On("command:setpublic", HandleSetPublic)
	bot.On(tg.OnCallbackQuery, HandleCallback)
	bot.On(tg.OnNewMessage, HandleNewMessage)
}

func isOwner(userID int64) bool {
	return userID == config.OwnerID
}

func isAuthUser(userID int64) bool {
	authUsersMutex.RLock()
	defer authUsersMutex.RUnlock()
	return authUsersCache[userID]
}

func isPublicAccess() bool {
	publicAccessMutex.RLock()
	defer publicAccessMutex.RUnlock()
	return publicAccessCache
}

func isAuthorized(userID int64) bool {
	if isOwner(userID) {
		return true
	}
	if isAuthUser(userID) {
		return true
	}
	return false
}

func canSearch(userID int64) bool {
	if isPublicAccess() {
		return true
	}
	return isAuthorized(userID)
}

func canAdd(userID int64) bool {
	return isAuthorized(userID)
}
