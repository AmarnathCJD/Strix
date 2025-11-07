package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	TMDBAPIKey   string
	GeminiAPIKey string
	Port         string
	FilesDir     string

	AppID    int
	AppHash  string
	BotToken string

	CDNBots []string

	LogChannel   int64
	IndexChannel int64
	OwnerID      int64
	AuthUsers    []int64

	MongoURL string
	DBName   string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		TMDBAPIKey:   getEnv("TMDB_API_KEY", ""),
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		Port:         getEnv("PORT", "8080"),
		FilesDir:     getEnv("FILES_DIR", "./media"),
		AppHash:      getEnv("APP_HASH", ""),
		BotToken:     getEnv("BOT_TOKEN", ""),
		MongoURL:     getEnv("MONGO_URL", "mongodb://localhost:27017"),
		DBName:       getEnv("DB_NAME", "strix"),
	}

	if cfg.TMDBAPIKey == "" {
		log.Fatal("TMDB_API_KEY is required")
	}

	appID, _ := strconv.Atoi(getEnv("APP_ID", "0"))
	cfg.AppID = appID

	logChannel, _ := strconv.ParseInt(getEnv("LOG_CHANNEL", "0"), 10, 64)
	cfg.LogChannel = logChannel

	indexChannel, _ := strconv.ParseInt(getEnv("INDEX_CHANNEL", "0"), 10, 64)
	cfg.IndexChannel = indexChannel

	ownerID, _ := strconv.ParseInt(getEnv("OWNER_ID", "0"), 10, 64)
	cfg.OwnerID = ownerID

	authUsersStr := getEnv("AUTH_USERS_ID", "")
	if authUsersStr != "" {
		for _, idStr := range strings.Split(authUsersStr, ",") {
			id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
			if err == nil {
				cfg.AuthUsers = append(cfg.AuthUsers, id)
			}
		}
	}

	for i := 1; i <= 10; i++ {
		token := getEnv("CDN_BOT_"+strconv.Itoa(i), "")
		if token != "" {
			cfg.CDNBots = append(cfg.CDNBots, token)
		}
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
