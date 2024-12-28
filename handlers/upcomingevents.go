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

// Market represents the structure of each market's data
type Market struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Odds  struct {
		Home  string `json:"home,omitempty"`
		Away  string `json:"away,omitempty"`
		Draw  string `json:"draw,omitempty"` // For markets like soccer
	} `json:"odds"`
}

// Event represents the structure of the event returned by the API
type SportsEvent struct {
	ID           string    `json:"id"`
	SportKey     string    `json:"sport_key"`
	SportTitle   string    `json:"sport_title"`
	CommenceTime string    `json:"commence_time"`
	HomeTeam     string    `json:"home_team"`
	AwayTeam     string    `json:"away_team"`
	Markets      []Market  `json:"markets"`
}

// GroupedEvents is the structure for grouping events by sport_key
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
	markets := "h2h,spreads"
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

	groupedEvents := make(GroupedEvents)
	for _, event := range events {
		groupedEvents[event.SportTitle] = append(groupedEvents[event.SportTitle], event)
	}
	// Return the events as JSON response
	return c.JSON(http.StatusOK, events)
}
