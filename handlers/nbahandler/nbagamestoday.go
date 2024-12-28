package nbahandler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

type Outcome struct {
    Name  string   `json:"name"`
    Price float64  `json:"price"`
    Point *float64 `json:"point,omitempty"` // Optional field (for spreads)
}

type Market struct {
    Key        string    `json:"key"`
    LastUpdate string    `json:"last_update"`
    Outcomes   []Outcome `json:"outcomes"`
}

type Bookmaker struct {
    Key        string    `json:"key"`
    Title      string    `json:"title"`
    LastUpdate string    `json:"last_update"`
    Markets    []Market  `json:"markets"`
}

type NBAEvents struct {
    Key         string      `json:"key"`
    LastUpdate  string      `json:"last_update"`
    Bookmakers  []Bookmaker `json:"bookmakers"`
}


// GetUpcomingSports fetches events for upcoming sports, including odds and markets
// GetUpcomingSports fetches events for upcoming sports, including odds and markets
func NBAGamesToday(c echo.Context) error {
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
	baseURL := "https://api.the-odds-api.com/v4/sports/basketball_nba/odds"
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

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read response body"})
	}

	// Print the raw API response for debugging
	fmt.Printf("API Response: %s\n", string(body))

	// Now parse the response from the body
	var events []NBAEvents
	if err := json.Unmarshal(body, &events); err != nil {
		log.Printf("Error parsing API response: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse events"})
	}

	// Define the list of desired bookmakers
	desiredBookmakers := []string{
		"fanduel", "betmgm", "draftkings", "betrivers", "bovada", "ceasers",
	}

	// Filter events to only include the desired bookmakers
	var filteredEvents []NBAEvents
	for _, event := range events {
		var filteredBookmakers []Bookmaker
		for _, bookmaker := range event.Bookmakers {
			// Only include bookmakers that are in the desired list
			if contains(desiredBookmakers, bookmaker.Key) {
				filteredBookmakers = append(filteredBookmakers, bookmaker)
			}
		}

		// If the event has any valid bookmakers, add it to the filtered events
		if len(filteredBookmakers) > 0 {
			event.Bookmakers = filteredBookmakers
			filteredEvents = append(filteredEvents, event)
		}
	}

	return c.JSON(http.StatusOK, filteredEvents)
}

// Helper function to check if a slice contains a given value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
