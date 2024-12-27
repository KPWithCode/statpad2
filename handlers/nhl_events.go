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

// Event represents the structure of the event returned by the API
type Event struct {
	Id         string `json:"id"`
	SportKey   string `json:"sport_key"`
	SportTitle string `json:"sport_title"`
	CommenceTime string `json:"commence_time"`
	HomeTeam   string `json:"home_team"`
	AwayTeam   string `json:"away_team"`
}

// APIKey and constants
const (
	BaseURL = "https://api.the-odds-api.com/v4/sports/icehockey_nhl/events"
)

// GetHockeyEvents fetches events for NHL
func GetHockeyEvents(c echo.Context) error {
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

	// Define query parameters for the request
	dateFormat := "iso" // Optional, could be 'iso' or 'unix'

	// Construct the request URL for event list
	url := fmt.Sprintf(
		"https://api.the-odds-api.com/v4/sports/icehockey_nhl/events?apiKey=%s&dateFormat=%s",
		apiKey, dateFormat,
	)

	// Make the API request to fetch events
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error making API request: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch events"})
	}
	defer resp.Body.Close()

	// Parse the response
	var events []Event
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		log.Printf("Error parsing API response: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse events"})
	}

	// Return the events as JSON response
	return c.JSON(http.StatusOK, events)
}
