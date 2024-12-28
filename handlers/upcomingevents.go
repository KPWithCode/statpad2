package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

type Outcome struct {
	Name  string   `json:"name"`
	Price float64  `json:"price"`
	Point *float64 `json:"point,omitempty"`
}

type Market struct {
	Key      string    `json:"key"`
	Title    string    `json:"title"`
	Outcomes []Outcome `json:"outcomes"`
}

type Bookmaker struct {
	Key        string    `json:"key"`
	Title      string    `json:"title"`
	LastUpdate string    `json:"last_update"`
	Markets    []Market  `json:"markets"`
}

// SportsEvent represents the structure of an event, including teams and associated bookmakers
type SportsEvent struct {
	ID           string      `json:"id"`
	SportKey     string      `json:"sport_key"`
	SportTitle   string      `json:"sport_title"`
	CommenceTime string      `json:"commence_time"`
	HomeTeam     string      `json:"home_team"`
	AwayTeam     string      `json:"away_team"`
	Bookmakers   []Bookmaker `json:"bookmakers"`
}

type GroupedEvents map[string][]SportsEvent

// GetUpcomingSports fetches events for upcoming sports, including odds and markets
func GetUpcomingSports(c echo.Context) error {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load environment variables"})
	}

	// Get the API key from the environment
	apiKey := os.Getenv("ODDS_API_KEY")
	if apiKey == "" {
		log.Println("ODDS_API_KEY is not set in the environment")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "API key is missing"})
	}

	// Construct the request URL
	baseURL := "https://api.the-odds-api.com/v4/sports/upcoming/odds"
	regions := "us"
	markets := "h2h,spreads,totals"
	oddsFormat := "american"
	url := fmt.Sprintf(
		"%s/?apiKey=%s&regions=%s&markets=%s&oddsFormat=%s",
		baseURL, apiKey, regions, markets, oddsFormat,
	)

	// Make the API request
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error making API request: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch events"})
	}
	defer resp.Body.Close()

	// Check for a non-200 HTTP status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("API responded with status code: %d", resp.StatusCode)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch events"})
	}

	// Parse the response
	var events []SportsEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		log.Printf("Error parsing API response: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse events"})
	}

	// Limit the events to the next 10 upcoming games
	if len(events) > 10 {
		events = events[:10]
	}

	// Filter out only the desired bookmakers
	desiredBookmakers := map[string]bool{
		"fanduel":   true,
		"betmgm": true,
		"draftkings": true,
		"betrivers": true,
		"bovada":    true,
	}

	// Filter events to only include relevant bookmakers and structure the data
	var filteredEvents []SportsEvent
	for _, event := range events {
		var filteredBookmakers []Bookmaker
		for _, bookmaker := range event.Bookmakers {
			if desiredBookmakers[bookmaker.Key] {
				filteredBookmakers = append(filteredBookmakers, bookmaker)
			}
		}
		// Add event with filtered bookmakers
		if len(filteredBookmakers) > 0 {
			event.Bookmakers = filteredBookmakers
			filteredEvents = append(filteredEvents, event)
		}
	}

	// Group the events by sport title
	groupedEvents := make(GroupedEvents)
	for _, event := range filteredEvents {
		groupedEvents[event.SportTitle] = append(groupedEvents[event.SportTitle], event)
	}

	return c.JSON(http.StatusOK, groupedEvents)
}
