package nbahandler

import (
	"fmt"
	"math"
	"net/http"
	"sort"

	"github.com/labstack/echo/v4"
)

 const (
	pythagoreanExponent = 13.91 
    winPctMultiplier    = 100.0
)

type PythagoreanTeam struct {
    Team              string  `json:"team"`
    ExpectedWinPct    float64 `json:"expectedWinPct"`
    ActualWinPct      float64 `json:"actualWinPct"`
    WinPctDifferential float64 `json:"winPctDifferential"`
    PointsScoredPerGame float64 `json:"pointsScoredPerGame"`
    PointsAllowedPerGame float64 `json:"pointsAllowedPerGame"`
    ActualWins          float64 `json:"actualWins"`
    ExpectedWins        float64 `json:"expectedWins"`
}

func calculatePythagoreanWinPct(pointsScoredPerGame, pointsAllowedPerGame float64) float64 {
    // Guard against division by zero or negative numbers
    if pointsScoredPerGame <= 0 || pointsAllowedPerGame <= 0 {
        return 0.0
    }
    
    // The actual Pythagorean formula:
    // (Points Scored^exponent) / (Points Scored^exponent + Points Allowed^exponent)
    ptsForExp := math.Pow(pointsScoredPerGame, pythagoreanExponent)
    ptsAgainstExp := math.Pow(pointsAllowedPerGame, pythagoreanExponent)
    
    // Calculate win percentage
    expectedWinPct := (ptsForExp / (ptsForExp + ptsAgainstExp))
    
    return expectedWinPct * winPctMultiplier // Convert to percentage
}

func PythagoreanHandler(c echo.Context) error {
    teamStats, err := fetchTeamStats()
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status": "error",
            "error":  fmt.Sprintf("Failed to fetch team stats: %v", err),
        })
    }

    results := make([]PythagoreanTeam, 0, len(teamStats))
    
    for _, team := range teamStats {
        // Get points per game from the Stats structure
        pointsScoredPerGame := team.Stats.Offense.PtsPerGame
        pointsAllowedPerGame := team.Stats.Defense.PtsAgainstPerGame
        
        // Calculate expected win percentage using Pythagorean formula
        expectedWinPct := calculatePythagoreanWinPct(pointsScoredPerGame, pointsAllowedPerGame)
        
        // Get actual win percentage from standings
        actualWinPct := team.Stats.Standings.WinPct * winPctMultiplier
        
        // Calculate expected wins based on games played
        gamesPlayed := team.Stats.Standings.Wins + team.Stats.Standings.Losses
        expectedWins := (expectedWinPct / 100.0) * gamesPlayed
        
        results = append(results, PythagoreanTeam{
            Team:                fmt.Sprintf("%s %s", team.Team.City, team.Team.Name),
            ExpectedWinPct:      roundToTwoDecimals(expectedWinPct),
            ActualWinPct:        roundToTwoDecimals(actualWinPct),
            WinPctDifferential:  roundToTwoDecimals(actualWinPct - expectedWinPct),
            PointsScoredPerGame: roundToTwoDecimals(pointsScoredPerGame),
            PointsAllowedPerGame: roundToTwoDecimals(pointsAllowedPerGame),
            ActualWins:          team.Stats.Standings.Wins,
            ExpectedWins:        roundToTwoDecimals(expectedWins),
        })
    }

    // Sort by expected win percentage
    sort.Slice(results, func(i, j int) bool {
        return results[i].ExpectedWinPct > results[j].ExpectedWinPct
    })

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status": "success",
        "data": map[string]interface{}{
            "teams": results,
            "metadata": map[string]interface{}{
                "pythagoreanExponent": pythagoreanExponent,
                "formula": "Win% = (Points Per Game^13.91) / (Points Per Game^13.91 + Points Allowed Per Game^13.91)",
                "note": "Uses points per game to calculate expected winning percentage",
            },
        },
    })
}