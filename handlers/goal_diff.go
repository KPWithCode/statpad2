package handlers

import (
	"encoding/csv"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
)

type GoalDifferentialStats struct {
	Team                        string             `json:"team"`
	GoalDifferentialPerGame     float64            `json:"goal_differential_per_game"`
	WinProbabilityByDifferential map[int]float64   `json:"win_probability_by_differential"`
	TotalGames                  int                `json:"total_games"`
	GoalDifferentialCounts      map[int]int        `json:"goal_differential_counts"`
	TotalWins                   int                `json:"total_wins"`
}

func ProcessGoalDifferentialHandler(c echo.Context) error {
	filePath := c.QueryParam("filePath")
	if filePath == "" {
		filePath = "data/shots_2024.csv"
	}

	file, err := os.Open(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to open CSV file"})
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse CSV"})
	}

	if len(records) < 2 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSV file is empty or invalid"})
	}

	// Column indices
	columns := records[0]
	teamIdx := findColumnIndex(columns, "teamCode")
	eventIdx := findColumnIndex(columns, "event")
	gameIdx := findColumnIndex(columns, "game_id")
	homeTeamIdx := findColumnIndex(columns, "homeTeamCode")
	awayTeamIdx := findColumnIndex(columns, "awayTeamCode")
	homeGoalsIdx := findColumnIndex(columns, "homeTeamGoals")
	awayGoalsIdx := findColumnIndex(columns, "awayTeamGoals")
	homeWinIdx := findColumnIndex(columns, "homeTeamWon")

	if teamIdx == -1 || eventIdx == -1 || gameIdx == -1 || homeTeamIdx == -1 || awayTeamIdx == -1 || homeGoalsIdx == -1 || awayGoalsIdx == -1 || homeWinIdx == -1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSV does not contain required columns"})
	}

	teamStats := make(map[string]*GoalDifferentialStats)

	for _, record := range records[1:] {
		homeTeam := record[homeTeamIdx]
		awayTeam := record[awayTeamIdx]
		homeGoals, err := strconv.Atoi(record[homeGoalsIdx])
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid homeTeamGoals value"})
		}
		awayGoals, err := strconv.Atoi(record[awayGoalsIdx])
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid awayTeamGoals value"})
		}
		homeWin, err := strconv.ParseBool(record[homeWinIdx])
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid homeTeamWon value"})
		}

		homeDifferential := homeGoals - awayGoals
		awayDifferential := awayGoals - homeGoals

		processDifferential(homeTeam, homeDifferential, homeWin, teamStats)
		processDifferential(awayTeam, awayDifferential, !homeWin, teamStats)
	}

	// Calculate final stats
	for _, stats := range teamStats {
		// Calculate goal differential per game
		totalDifferential := 0
		for differential, count := range stats.GoalDifferentialCounts {
			totalDifferential += differential * count
		}
		stats.GoalDifferentialPerGame = float64(totalDifferential) / float64(stats.TotalGames)

		// Calculate win probability by goal differential
		for differential, count := range stats.GoalDifferentialCounts {
			if count > 0 {
				winCount := 0
				if differential > 0 {
					winCount = stats.TotalWins // Wins are counted for positive differentials
				}
				stats.WinProbabilityByDifferential[differential] = float64(winCount) / float64(count) * 100.0
			}
		}
	}

	return c.JSON(http.StatusOK, teamStats)
}

func processDifferential(team string, differential int, won bool, teamStats map[string]*GoalDifferentialStats) {
	if _, ok := teamStats[team]; !ok {
		teamStats[team] = &GoalDifferentialStats{
			Team:                        team,
			WinProbabilityByDifferential: make(map[int]float64),
			GoalDifferentialCounts:      make(map[int]int),
		}
	}

	stats := teamStats[team]

	stats.TotalGames++
	stats.GoalDifferentialCounts[differential]++

	if won && differential > 0 {
		stats.TotalWins++
	}
}
