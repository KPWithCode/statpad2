package nbahandler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

type TeamTS struct {
	ID      int    `json:"id"`
	City    string `json:"city"`
	Name    string `json:"name"`
	Abbr    string `json:"abbreviation"`
	LogoURL string `json:"officialLogoImageSrc"`
}

type StatsTS struct {
	Team         TeamTS        `json:"team"`
	Stats        FilteredStats `json:"stats"` // Added this wrapper
	TSPercentage float64       `json:"tsPercentage"`
}

type FilteredStats struct {
	FTPct    float64 `json:"ftPct"`
	FG2PtPct float64 `json:"fg2PtPct"`
	FG3PtPct float64 `json:"fg3PtPct"`
}

type internalStatsData struct {
	FieldGoals FieldGoals `json:"fieldGoals"`
	FreeThrows FreeThrows `json:"freeThrows"`
}

type internalStatsTS struct {
	Team  TeamTS            `json:"team"`
	Stats internalStatsData `json:"stats"`
}

type internalResponse struct {
	TeamStats []internalStatsTS `json:"teamStatsTotals"`
}
type Response struct {
	TeamStats []StatsTS `json:"teamStatsTotals"`
}

func calculateTSPercentage(fieldGoals FieldGoals, freeThrows FreeThrows) float64 {
	pointsScored := fieldGoals.FG2PtMade*2 + fieldGoals.FG3PtMade*3 + freeThrows.FTMade
	totalAttempts := fieldGoals.FG2PtAtt + fieldGoals.FG3PtAtt + 0.44*freeThrows.FTAtt
	// safety check for division by 0
	if totalAttempts == 0 {
		return 0
	}
	tsPercentage := (pointsScored / (2 * totalAttempts)) * 100
	return tsPercentage
}

func TrueShootingHandler(c echo.Context) error {
	if err := loadEnvVars(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Error loading environment variables: %v", err),
		})
	}

	apiKey := os.Getenv("MYSPORTSFEEDS_API_KEY")
	if apiKey == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "API key not found in environment variables",
		})
	}

	authString := fmt.Sprintf("%s:%s", apiKey, "MYSPORTSFEEDS")
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(authString))

	url := "https://api.mysportsfeeds.com/v2.1/pull/nba/2024-2025-regular/team_stats_totals.json"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Error creating request: %v", err),
		})
	}

	req.Header.Set("Authorization", "Basic "+encodedAuth)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Error making request: %v", err),
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("API request failed with status code: %d", resp.StatusCode),
		})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Error reading response body: %v", err),
		})
	}

	var internalResp internalResponse
	if err := json.Unmarshal(body, &internalResp); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Error unmarshalling response: %v", err),
		})
	}

	response := Response{
		TeamStats: make([]StatsTS, len(internalResp.TeamStats)),
	}

	for i, team := range internalResp.TeamStats {
		tsPercentage := calculateTSPercentage(
			team.Stats.FieldGoals,
			team.Stats.FreeThrows,
		)

		response.TeamStats[i] = StatsTS{
			Team: team.Team,
			Stats: FilteredStats{
				FTPct:    team.Stats.FreeThrows.FTPct,
				FG2PtPct: team.Stats.FieldGoals.FG2PtPct,
				FG3PtPct: team.Stats.FieldGoals.FG3PtPct,
			},
			TSPercentage: roundToTwoDecimals(tsPercentage),
		}
	}

	return c.JSON(http.StatusOK, response)
}
