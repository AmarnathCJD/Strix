package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/mux"
)

func (s *Server) handleIMDBRating(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	imdbID := vars["imdb_id"]

	if imdbID == "" {
		json.NewEncoder(w).Encode(IMDBRating{Error: "IMDB ID is required"})
		return
	}

	rating := scrapeIMDBRating(imdbID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rating)
}

func scrapeIMDBRating(imdbID string) IMDBRating {
	url := fmt.Sprintf("https://www.imdb.com/title/%s/", imdbID)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return IMDBRating{Error: "Failed to create request"}
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return IMDBRating{Error: "Failed to fetch IMDB page"}
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return IMDBRating{Error: "Failed to parse HTML"}
	}

	jsonMeta := doc.Find("script[type='application/ld+json']").First().Text()
	if jsonMeta == "" {
		return IMDBRating{Error: "JSON-LD data not found"}
	}

	var jsonObj map[string]any
	if err := json.Unmarshal([]byte(jsonMeta), &jsonObj); err != nil {
		return IMDBRating{Error: "Failed to parse JSON-LD data"}
	}

	var rating float64 = 0.0
	var votes float64 = 0.0

	if jsonObj["aggregateRating"] != nil {
		aggregateRating := jsonObj["aggregateRating"].(map[string]any)
		if aggregateRating["ratingValue"] != nil {
			rating = aggregateRating["ratingValue"].(float64)
		}
		if aggregateRating["ratingCount"] != nil {
			votes = aggregateRating["ratingCount"].(float64)
		}
	}

	if rating > 0 {
		ratingStr := fmt.Sprintf("%.1f", rating)
		votesStr := formatVotesFromFloat(votes)
		return IMDBRating{
			Rating: ratingStr,
			Votes:  votesStr,
		}
	}

	return IMDBRating{Error: "Rating not found"}
}

func formatVotesFromFloat(votes float64) string {
	if votes == 0 {
		return "N/A"
	}
	if votes >= 1000000 {
		return fmt.Sprintf("%.1fM", votes/1000000)
	} else if votes >= 1000 {
		return fmt.Sprintf("%.1fK", votes/1000)
	}
	return fmt.Sprintf("%.0f", votes)
}

func formatVotes(votes string) string {
	if len(votes) >= 7 {
		return votes[:len(votes)-6] + "." + votes[len(votes)-6:len(votes)-5] + "M"
	} else if len(votes) >= 4 {
		return votes[:len(votes)-3] + "." + votes[len(votes)-3:len(votes)-2] + "K"
	}
	return votes
}

func formatVotesWithCommas(votes string) string {
	n := len(votes)
	if n <= 3 {
		return votes
	}

	var result strings.Builder
	for i, digit := range votes {
		if i > 0 && (n-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(digit)
	}
	return result.String()
}
