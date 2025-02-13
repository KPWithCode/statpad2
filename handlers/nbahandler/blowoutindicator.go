package nbahandler

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
)

type TeamBI struct {
	ID           int    `json:"id"`
	City         string `json:"city"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
}

type BITeamStats struct {
	Team  TeamBI `json:"team"`
	Stats struct {
		GamesPlayed int `json:"gamesPlayed"`
		Offense     struct {
			Pts        int     `json:"pts"`
			PtsPerGame float64 `json:"ptsPerGame"`
		} `json:"offense"`
		Defense struct {
			PtsAgainst        int     `json:"ptsAgainst"`
			PtsAgainstPerGame float64 `json:"ptsAgainstPerGame"`
		} `json:"defense"`
		Standings struct {
			Wins   int     `json:"wins"`
			Losses int     `json:"losses"`
			WinPct float64 `json:"winPct"`
		} `json:"standings"`
	} `json:"stats"`
}

type TeamStatsResponseBI struct {
	LastUpdatedOn    string       `json:"lastUpdatedOn"`
	TeamStatsTotals []BITeamStats `json:"teamStatsTotals"`
}

type BlowoutPrediction struct {
	HomeTeam           string  `json:"homeTeam"`
	AwayTeam           string  `json:"awayTeam"`
	FavoredTeam        string  `json:"favoredTeam"`
	PredictedMargin    float64 `json:"predictedMargin"`
	BlowoutProbability float64 `json:"blowoutProbability"`
	Factors            struct {
		NetRating     float64 `json:"netRating"`
		PythWinPct    float64 `json:"pythWinPct"`
		HomeAdvantage float64 `json:"homeAdvantage"`
	} `json:"factors"`
}

func fetchTeamStatsBI() (*TeamStatsResponseBI, error) {
	apiKey := os.Getenv("MYSPORTSFEEDS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API key not found")
	}

	endpoint := "https://api.mysportsfeeds.com/v2.1/pull/nba/2024-2025-regular/team_stats_totals.json"
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var response TeamStatsResponseBI
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return &response, nil
}

func calculatePythagoreanWinPctBI(pointsScored, pointsAllowed float64) float64 {
	exponent := 13.91 // NBA-specific Pythagorean exponent
	return math.Pow(pointsScored, exponent) / (math.Pow(pointsScored, exponent) + math.Pow(pointsAllowed, exponent))
}

func calculateBlowoutProbability(homeTeam, awayTeam BITeamStats) BlowoutPrediction {
	// Calculate net ratings
	homeNetRating := homeTeam.Stats.Offense.PtsPerGame - homeTeam.Stats.Defense.PtsAgainstPerGame
	awayNetRating := awayTeam.Stats.Offense.PtsPerGame - awayTeam.Stats.Defense.PtsAgainstPerGame
	netRatingDiff := homeNetRating - awayNetRating

	// Calculate Pythagorean win expectancy
	homePythWinPct := calculatePythagoreanWinPctBI(homeTeam.Stats.Offense.PtsPerGame, homeTeam.Stats.Defense.PtsAgainstPerGame)
	awayPythWinPct := calculatePythagoreanWinPctBI(awayTeam.Stats.Offense.PtsPerGame, awayTeam.Stats.Defense.PtsAgainstPerGame)
	pythWinPctDiff := homePythWinPct - awayPythWinPct

	// Home court advantage (approximately 3 points in NBA)
	homeAdvantage := 3.0

	// Combine factors to predict margin
	predictedMargin := (netRatingDiff * 0.4) + (pythWinPctDiff * 15.0) + homeAdvantage

	// Create the prediction object
	prediction := BlowoutPrediction{
		HomeTeam: fmt.Sprintf("%s %s", homeTeam.Team.City, homeTeam.Team.Name),
		AwayTeam: fmt.Sprintf("%s %s", awayTeam.Team.City, awayTeam.Team.Name),
	}

	// Determine favored team and ensure margin is positive
	if predictedMargin >= 0 {
		prediction.FavoredTeam = prediction.HomeTeam
		prediction.PredictedMargin = predictedMargin
	} else {
		prediction.FavoredTeam = prediction.AwayTeam
		prediction.PredictedMargin = -predictedMargin
	}

	// Calculate probability of blowout (14+ point margin)
	blowoutThreshold := 14.0
	prediction.BlowoutProbability = 1.0 / (1.0 + math.Exp(-0.2*(prediction.PredictedMargin-blowoutThreshold)))

	prediction.Factors.NetRating = netRatingDiff
	prediction.Factors.PythWinPct = pythWinPctDiff
	prediction.Factors.HomeAdvantage = homeAdvantage

	return prediction
}

func BlowoutPredictorHandler(c echo.Context) error {
	// Fetch today's schedule
	schedule, err := fetchTodaysScheduleII()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Error fetching schedule: %v", err)})
	}

	// Fetch team stats
	biTeamStats, err := fetchTeamStatsBI()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Error fetching team stats: %v", err)})
	}

	// Create a map for easy team lookup
	teamStatsMap := make(map[string]BITeamStats)
	for _, stats := range biTeamStats.TeamStatsTotals {
		teamStatsMap[stats.Team.Abbreviation] = stats
	}

	// Calculate blowout predictions for each game
	var predictions []BlowoutPrediction
	for i := 0; i < len(schedule); i += 2 {
		homeTeam := teamStatsMap[schedule[i]]
		awayTeam := teamStatsMap[schedule[i+1]]
		
		prediction := calculateBlowoutProbability(homeTeam, awayTeam)
		predictions = append(predictions, prediction)
	}

	// Sort predictions by blowout probability
	for i := range predictions {
		for j := i + 1; j < len(predictions); j++ {
			if predictions[j].BlowoutProbability > predictions[i].BlowoutProbability {
				predictions[i], predictions[j] = predictions[j], predictions[i]
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"date":        time.Now().Format("2006-01-02"),
		"predictions": predictions,
	})
}