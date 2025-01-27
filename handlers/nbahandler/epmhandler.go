package nbahandler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)



func getEPMCheatSheet(teams []string) (map[string]map[string]float64, error) {
    apiKey := os.Getenv("MYSPORTSFEEDS_API_KEY")
    if apiKey == "" {
        return nil, fmt.Errorf("API key not found in environment variables")
    }

    teamsParam := strings.Join(teams, ",")
    endpoint := fmt.Sprintf("https://api.mysportsfeeds.com/v2.1/pull/nba/2024-2025-regular/player_stats_totals.json?team=%s", teamsParam)
    
    req, err := http.NewRequest("GET", endpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("error creating request: %v", err)
    }

    req.SetBasicAuth(apiKey, "MYSPORTSFEEDS")

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error making request: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API returned non-200 status code: %d, body: %s",
            resp.StatusCode, string(bodyBytes))
    }

    var response struct {
        PlayerStatsTotals []struct {
            Player struct {
                Position string `json:"primaryPosition"`
                CurrentTeam struct {
                    Abbreviation string `json:"abbreviation"`
                } `json:"currentTeam"`
            } `json:"player"`
            Stats struct {
                Miscellaneous struct {
                    PlusMinusPerGame float64 `json:"plusMinusPerGame"`
                    MinSecondsPerGame float64 `json:"minSecondsPerGame"`
                } `json:"miscellaneous"`
            } `json:"stats"`
        } `json:"playerStatsTotals"`
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("error reading response: %v", err)
    }

    if err := json.Unmarshal(body, &response); err != nil {
        return nil, fmt.Errorf("error parsing JSON: %v", err)
    }

    // Group players by team and position groups
    teamEPM := make(map[string]map[string]float64)
    positionGroups := map[string][]string{
        "Backcourt": {"PG", "SG", "SF"},
        "Frontcourt": {"PF", "C"},
    }

    for _, team := range teams {
        teamEPM[team] = make(map[string]float64)
        
        // Calculate EPM for each position group
        for groupName, positions := range positionGroups {
            var totalWeightedEPM float64
            var totalMinutes float64

            for _, player := range response.PlayerStatsTotals {
                if player.Player.CurrentTeam.Abbreviation == team && 
                   contains(positions, player.Player.Position) {
                    
                    // Weighted EPM by minutes played
                    totalWeightedEPM += player.Stats.Miscellaneous.PlusMinusPerGame * 
                                        (player.Stats.Miscellaneous.MinSecondsPerGame / 60)
                    totalMinutes += player.Stats.Miscellaneous.MinSecondsPerGame / 60
                }
            }

            // Calculate average weighted EPM
            avgEPM := totalWeightedEPM / totalMinutes

            // Categorize EPM
            var epmRating string
            switch {
            case avgEPM > 5:
                epmRating = "High"
            case avgEPM > -5:
                epmRating = "Average"
            default:
                epmRating = "Low"
            }

            teamEPM[team][groupName] = avgEPM
            teamEPM[team][groupName + "Rating"] = float64(len(epmRating))
        }
    }

    return teamEPM, nil
}

// Helper function to check if a value is in a slice
func contains(slice []string, val string) bool {
    for _, item := range slice {
        if item == val {
            return true
        }
    }
    return false
}

func EPMHandler(c echo.Context) error {
    teams, err := getTodaysGameSchedule()
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }

    epmCheatSheet, err := getEPMCheatSheet(teams)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }

    return c.JSON(http.StatusOK, epmCheatSheet)
}