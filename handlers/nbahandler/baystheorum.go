package nbahandler

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	// "sync"

	// "sync"
	"time"

	"github.com/labstack/echo/v4"
)

// TeamStatsResponse structures the JSON response from MySportsFeeds API
type TeamStatsResponse struct {
    TeamStatsTotals []struct {
        Team struct {
            Name string `json:"name"`
        } `json:"team"`
        Stats struct {
            Offense struct {
                PtsPerGame float64 `json:"ptsPerGame"` 
            } `json:"offense"`
            Defense struct {
                PtsAgainstPerGame float64 `json:"ptsAgainstPerGame"` 
            } `json:"defense"`
        } `json:"stats"`
    } `json:"teamStatsTotals"`
}

func fetchTeamInfo(teamAbbr string) (*TeamStatsResponse, error) {
    apiKey := os.Getenv("MYSPORTSFEEDS_API_KEY")
    if apiKey == "" {
        return nil, fmt.Errorf("API key not found in environment variables")
    }

    url := fmt.Sprintf("https://api.mysportsfeeds.com/v2.1/pull/nba/2024-2025-regular/team_stats_totals.json?team=%s", teamAbbr)
    
    maxRetries := 3
    for attempt := 0; attempt < maxRetries; attempt++ {
        req, err := http.NewRequest("GET", url, nil)
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

        // Handle 429 (Too Many Requests) with exponential backoff
        if resp.StatusCode == http.StatusTooManyRequests {
            waitTime := time.Duration(math.Pow(2, float64(attempt))) * time.Second
            time.Sleep(waitTime)
            continue
        }

        if resp.StatusCode != http.StatusOK {
            bodyBytes, _ := io.ReadAll(resp.Body)
            return nil, fmt.Errorf("API returned non-200 status code: %d, body: %s", 
                resp.StatusCode, string(bodyBytes))
        }

        body, err := io.ReadAll(resp.Body)
        if err != nil {
            return nil, fmt.Errorf("error reading response body: %v", err)
        }

        var teamStats TeamStatsResponse
        if err := json.Unmarshal(body, &teamStats); err != nil {
            preview := string(body)
            if len(preview) > 100 {
                preview = preview[:100] + "..."
            }
            return nil, fmt.Errorf("error parsing JSON: %v, body preview: %s", err, preview)
        }

        return &teamStats, nil
    }

    return nil, fmt.Errorf("failed to fetch team info after %d attempts", maxRetries)
}

func calculateWinProbability(offensePts, defensePts float64) float64 {
    // Bayesian theorem implementation for win probability
    
    // Prior probability of winning (baseline 50%)
    priorWinProbability := 0.5 
    
    // Probability of scoring more points than the opponent
    // Based on team's offensive performance vs opponent's defensive performance
    probScoringMore := offensePts / (offensePts + defensePts)
    
    // Total likelihood normalization factor
    totalLikelihood := 1.0 

    // Calculate final win probability
    winProbability := (probScoringMore * priorWinProbability) / totalLikelihood
    
    // Ensure probability is between 0 and 1
    if winProbability < 0 {
        return 0
    }
    if winProbability > 1 {
        return 1
    }
    
    return winProbability
}

// func BayesianMatchupHandler(c echo.Context) error {
//     teams, err := fetchTodaysScheduleII()
//     if err != nil {
//         return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not fetch today's schedule: " + err.Error()})
//     }

//     if len(teams) < 2 {
//         return c.JSON(http.StatusBadRequest, map[string]string{"error": "Not enough teams for a matchup"})
//     }

//     // Limit concurrent requests to avoid rate limiting
//     sem := make(chan struct{}, 3) 
//     var mu sync.Mutex
//     var allMatchups []map[string]interface{}
//     var wg sync.WaitGroup

//     for i := 0; i < len(teams); i += 2 {
//         if i+1 >= len(teams) {
//             break
//         }

//         wg.Add(1)
//         go func(i int) {
//             defer wg.Done()
            
//             sem <- struct{}{} // Acquire semaphore
//             defer func() { <-sem }() // Release semaphore
//             time.Sleep(500 * time.Millisecond)

//             teamStatsResponses := make([]*TeamStatsResponse, 2)
//             for j := 0; j < 2; j++ {
//                 stats, err := fetchTeamInfo(teams[i+j])
//                 if err != nil {
//                     fmt.Printf("Error fetching stats for %s: %v\n", teams[i+j], err)
//                     return
//                 }
//                 teamStatsResponses[j] = stats
//             }

//             matchupResults := make([]map[string]interface{}, 2)
//             for j, teamStats := range teamStatsResponses {
//                 if len(teamStats.TeamStatsTotals) == 0 {
//                     continue
//                 }

//                 stats := teamStats.TeamStatsTotals[0].Stats
//                 teamName := teamStats.TeamStatsTotals[0].Team.Name

//                 opponentDefensePts := teamStatsResponses[1-j].TeamStatsTotals[0].Stats.Defense.PtsAgainstPerGame
//                 winProbability := calculateWinProbability(stats.Offense.PtsPerGame, opponentDefensePts)

//                 matchupResults[j] = map[string]interface{}{
//                     "team": teamName,
//                     "abbreviation": teams[i+j],
//                     "winProbability": winProbability,
//                 }
//             }

//             mu.Lock()
//             allMatchups = append(allMatchups, map[string]interface{}{
//                 "matchup": matchupResults,
//             })
//             mu.Unlock()
//         }(i)
//     }

//     wg.Wait()

//     return c.JSON(http.StatusOK, map[string]interface{}{
//         "games": allMatchups,
//     })
// }

// ONLY 2 WORKING
func BayesianMatchupHandler(c echo.Context) error {
    teams, err := fetchTodaysScheduleII()
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not fetch today's schedule: " + err.Error()})
    }

    if len(teams) < 2 {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "Not enough teams for a matchup"})
    }

    teamStatsResponses := make([]*TeamStatsResponse, 2)
    for i := 0; i < 2; i++ {
        stats, err := fetchTeamInfo(teams[i])
        if err != nil {
            return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not fetch stats for team " + teams[i] + ": " + err.Error()})
        }
        teamStatsResponses[i] = stats
    }

    results := make([]map[string]interface{}, 2)
    for i, teamStats := range teamStatsResponses {
        if len(teamStats.TeamStatsTotals) == 0 {
            return c.JSON(http.StatusNotFound, map[string]string{"error": "No stats found for team " + teams[i]})
        }

        stats := teamStats.TeamStatsTotals[0].Stats
        teamName := teamStats.TeamStatsTotals[0].Team.Name

        var opponentDefensePts float64
        if i == 0 {
            opponentDefensePts = teamStatsResponses[1].TeamStatsTotals[0].Stats.Defense.PtsAgainstPerGame
        } else {
            opponentDefensePts = teamStatsResponses[0].TeamStatsTotals[0].Stats.Defense.PtsAgainstPerGame
        }

        winProbability := calculateWinProbability(stats.Offense.PtsPerGame, opponentDefensePts)

        results[i] = map[string]interface{}{
            "team": teamName,
            "abbreviation": teams[i],
            "winProbability": winProbability,
        }
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "matchup": results,
    })
}
