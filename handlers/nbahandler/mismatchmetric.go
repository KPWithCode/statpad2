package nbahandler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

type PlayerStats struct {
	Player struct {
		ID             int    `json:"id"`
		FirstName      string `json:"firstName"`
		LastName       string `json:"lastName"`
		PrimaryPosition string `json:"primaryPosition"`
		JerseyNumber   int    `json:"jerseyNumber"`
		CurrentTeam    struct {
			ID           int    `json:"id"`
			Abbreviation string `json:"abbreviation"`
		} `json:"currentTeam"`
	} `json:"player"`
	Stats struct {
		UsagePercent      float64 `json:"usgPct"`   // Usage Rate
		TrueShootingPct   float64 `json:"tsPct"`    // True Shooting Percentage
		EffectiveFgPct    float64 `json:"efgPct"`   // Effective Field Goal Percentage
		Points            float64 `json:"pts"`
		Assists           float64 `json:"ast"`
		Rebounds          float64 `json:"reb"`
		FieldGoals        struct {
			FG2PtMade   float64 `json:"fg2PtMade"`
			FG3PtMade   float64 `json:"fg3PtMade"`
		} `json:"fieldGoals"`
		Defense struct {
			Stl float64 `json:"stl"` // Steals per game (for defense)
		} `json:"defense"`
	} `json:"stats"`
}

type MismatchMetric struct {
	OffensiveRating  float64 `json:"offensive_rating"`
	DefensiveRating  float64 `json:"defensive_rating"`
	MismatchScore    float64 `json:"mismatch_score"`
}

func GetMismatchHandler(c echo.Context) error {
	player := c.QueryParam("player")
	team := c.QueryParam("team")
	season := "current" // Always using current season
	format := "json"    // Hardcoded format as JSON

	// Build the API URL
	apiKey := os.Getenv("MYSPORTSFEEDS_API_KEY")
	url := fmt.Sprintf("https://api.mysportsfeeds.com/v2.1/pull/nba/%s/player_stats_totals.%s?team=%s&player=%s", season, format, team, player)

	// Make the request to MySportsFeeds API
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error creating request"})
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error making request"})
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error reading response"})
	}

	var playerStatsResponse struct {
		PlayerStatsTotals []PlayerStats `json:"playerStatsTotals"`
	}
	if err := json.Unmarshal(body, &playerStatsResponse); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error unmarshaling response"})
	}

	// Assuming we only have one player in the response (or choosing the first one)
	if len(playerStatsResponse.PlayerStatsTotals) == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Player not found"})
	}

	playerStats := playerStatsResponse.PlayerStatsTotals[0]

	// Calculate mismatch score
	mismatchMetric := calculateMismatchMetric(playerStats)

	// Return mismatch metric
	return c.JSON(http.StatusOK, mismatchMetric)
}

func calculateMismatchMetric(playerStats PlayerStats) MismatchMetric {
	// Offensive Rating: Combine USG%, TS%, and eFG%
	offensiveRating := (playerStats.Stats.UsagePercent + playerStats.Stats.TrueShootingPct + playerStats.Stats.EffectiveFgPct) / 3.0

	// Defensive Rating (using steals per game)
	defensiveRating := playerStats.Stats.Defense.Stl

	// Calculate mismatch score (simple example: offensive rating minus defensive rating)
	mismatchScore := offensiveRating - defensiveRating

	// Return mismatch metric
	return MismatchMetric{
		OffensiveRating: offensiveRating,
		DefensiveRating: defensiveRating,
		MismatchScore:   mismatchScore,
	}
}