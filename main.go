package main

import (
	"log"
	"net/http"
	"os"

	"strix/config"
	"strix/database"
	"strix/telegram"

	"github.com/gorilla/mux"
)

const (
	tmdbBaseURL  = "https://api.themoviedb.org/3"
	tmdbImageURL = "https://image.tmdb.org/t/p"
)

type Server struct {
	config *config.Config
	db     *database.DB
	router *mux.Router
}

func main() {
	cfg := config.Load()

	db, err := database.Init(cfg.MongoURL, cfg.DBName)

	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	telegram.ParseFilenameFunc = parseFilename
	telegram.IsVideoFileFunc = isVideoFile
	telegram.GeminiAPIKey = cfg.GeminiAPIKey

	if err := telegram.InitBot(cfg, db); err != nil {
		log.Fatal("Failed to initialize Telegram bot:", err)
	}

	if err := os.MkdirAll(cfg.FilesDir, 0755); err != nil {
		log.Fatal("Failed to create files directory:", err)
	}

	server := &Server{
		config: cfg,
		db:     db,
		router: mux.NewRouter(),
	}

	server.setupRoutes()

	log.Printf("→ Server starting on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, server.router))
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(enableCORS)

	fs := http.FileServer(http.Dir("assets"))
	s.router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", fs))

	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/search", s.handleSearch).Methods("GET")
	api.HandleFunc("/tv/{id:[0-9]+}", s.handleTVDetails).Methods("GET")
	api.HandleFunc("/tv/{id:[0-9]+}/season/{season:[0-9]+}", s.handleSeasonDetails).Methods("GET")
	api.HandleFunc("/movie/{id:[0-9]+}", s.handleMovieDetails).Methods("GET")
	api.HandleFunc("/trending", s.handleTrending).Methods("GET")
	api.HandleFunc("/files", s.handleListFiles).Methods("GET")
	api.HandleFunc("/files/search", s.handleSearchFiles).Methods("GET")
	api.HandleFunc("/imdb/{imdb_id}", s.handleIMDBRating).Methods("GET")
	api.HandleFunc("/media/movie/{tmdb_id:[0-9]+}", s.handleGetMovieFiles).Methods("GET")
	api.HandleFunc("/media/tv/{tmdb_id:[0-9]+}/season/{season:[0-9]+}", s.handleGetSeasonFiles).Methods("GET")
	api.HandleFunc("/media/tv/{tmdb_id:[0-9]+}/season/{season:[0-9]+}/episode/{episode:[0-9]+}", s.handleGetEpisodeFile).Methods("GET")

	s.router.HandleFunc("/search", s.handleSearchFiles).Methods("GET")

	s.router.HandleFunc("/stream/{token}", s.handleStream).Methods("GET")
	s.router.HandleFunc("/play", s.handleStreamPage).Methods("GET")
	s.router.HandleFunc("/", s.handleHome).Methods("GET")
	s.router.HandleFunc("/tv/{id:[0-9]+}", s.handleTVPage).Methods("GET")
	s.router.HandleFunc("/movie/{id:[0-9]+}", s.handleMoviePage).Methods("GET")
	s.router.HandleFunc("/search", s.handleSearchPage).Methods("GET")
}
