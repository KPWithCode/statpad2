package nbahandler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

type Game struct {
	ID          string     `json:"id"`
	Schedule    Schedule   `json:"schedule"`
	LastUpdated string     `json:"lastUpdatedOn"`
}

// Schedule represents the schedule details of the game
type Schedule struct {
	ID          string  `json:"id"`
	StartTime   string  `json:"startTime"`
	AwayTeam    TeamRef `json:"awayTeam"`
	HomeTeam    TeamRef `json:"homeTeam"`
	Venue       string  `json:"venue"`
	PlayedStatus string  `json:"playedStatus"`
}

// TeamRef represents a reference to a team
type TeamRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func GetNBADailyGames(c echo.Context) error {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load environment variables"})
	}

	// Get the API key from the environment
	apiKey := os.Getenv("MYSPORTSFEEDS_API_KEY")
	if apiKey == "" {
		log.Println("MYSPORTSFEEDS_API_KEY is not set in the environment")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "API key is missing"})
	}

	// Get date parameter from the request
	date := c.Param("date")

	// Construct the request URL for unplayed games
	url := fmt.Sprintf(
		"https://api.mysportsfeeds.com/v2.1/pull/nba/2024-2025-regular/date/%s/games.json",
		date,
	)

	// Make the API request to fetch games
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error making API request: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch games"})
	}
	defer resp.Body.Close()

	// Parse the response
	var response struct {
		Games []Game `json:"games"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Error parsing API response: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse games"})
	}

	// Filter unplayed games
	var unplayedGames []Game
	for _, game := range response.Games {
		if game.Schedule.PlayedStatus == "UNPLAYED" {
			unplayedGames = append(unplayedGames, game)
		}
	}

	// Return the unplayed games as JSON response
	return c.JSON(http.StatusOK, unplayedGames)
}
