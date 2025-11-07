package main

type TMDBResponse struct {
	Results []MediaItem `json:"results"`
}

type MediaItem struct {
	ID           int     `json:"id"`
	Title        string  `json:"title,omitempty"`
	Name         string  `json:"name,omitempty"`
	Overview     string  `json:"overview"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
	VoteAverage  float64 `json:"vote_average"`
	ReleaseDate  string  `json:"release_date,omitempty"`
	FirstAirDate string  `json:"first_air_date,omitempty"`
	MediaType    string  `json:"media_type,omitempty"`
}

type TVDetails struct {
	ID                  int                 `json:"id"`
	Name                string              `json:"name"`
	Overview            string              `json:"overview"`
	PosterPath          string              `json:"poster_path"`
	BackdropPath        string              `json:"backdrop_path"`
	VoteAverage         float64             `json:"vote_average"`
	FirstAirDate        string              `json:"first_air_date"`
	Genres              []Genre             `json:"genres"`
	NumberOfSeasons     int                 `json:"number_of_seasons"`
	Seasons             []Season            `json:"seasons"`
	Status              string              `json:"status"`
	Tagline             string              `json:"tagline"`
	Networks            []Network           `json:"networks"`
	ProductionCountries []ProductionCountry `json:"production_countries"`
	ExternalIDs         ExternalIDs         `json:"external_ids"`
	Recommendations     struct {
		Results []MediaItem `json:"results"`
	} `json:"recommendations"`
}

type MovieDetails struct {
	ID                  int                 `json:"id"`
	Title               string              `json:"title"`
	Overview            string              `json:"overview"`
	PosterPath          string              `json:"poster_path"`
	BackdropPath        string              `json:"backdrop_path"`
	VoteAverage         float64             `json:"vote_average"`
	ReleaseDate         string              `json:"release_date"`
	Genres              []Genre             `json:"genres"`
	Runtime             int                 `json:"runtime"`
	Tagline             string              `json:"tagline"`
	Status              string              `json:"status"`
	Budget              int64               `json:"budget"`
	Revenue             int64               `json:"revenue"`
	ProductionCountries []ProductionCountry `json:"production_countries"`
	ExternalIDs         ExternalIDs         `json:"external_ids"`
	Recommendations     struct {
		Results []MediaItem `json:"results"`
	} `json:"recommendations"`
}

type ExternalIDs struct {
	IMDbID      string `json:"imdb_id"`
	FacebookID  string `json:"facebook_id"`
	InstagramID string `json:"instagram_id"`
	TwitterID   string `json:"twitter_id"`
}

type IMDBRating struct {
	Rating string `json:"rating"`
	Votes  string `json:"votes"`
	Error  string `json:"error,omitempty"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Season struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	PosterPath   string `json:"poster_path"`
	SeasonNumber int    `json:"season_number"`
	EpisodeCount int    `json:"episode_count"`
	AirDate      string `json:"air_date"`
}

type SeasonDetails struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Overview     string    `json:"overview"`
	PosterPath   string    `json:"poster_path"`
	SeasonNumber int       `json:"season_number"`
	Episodes     []Episode `json:"episodes"`
	AirDate      string    `json:"air_date"`
}

type Episode struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	StillPath     string  `json:"still_path"`
	EpisodeNumber int     `json:"episode_number"`
	SeasonNumber  int     `json:"season_number"`
	VoteAverage   float64 `json:"vote_average"`
	AirDate       string  `json:"air_date"`
	Runtime       int     `json:"runtime"`
}

type Network struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ProductionCountry struct {
	ISO  string `json:"iso_3166_1"`
	Name string `json:"name"`
}

type FileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Quality string `json:"quality"`
	Format  string `json:"format"`
	Path    string `json:"path"`
}

type SearchResponse struct {
	Query   string      `json:"query"`
	Results []MediaItem `json:"results"`
}
