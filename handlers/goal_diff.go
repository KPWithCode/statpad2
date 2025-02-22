package handlers

import (
	"encoding/csv"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
)

type GoalDifferentialStats struct {
	Team                         string          `json:"team"`
	GoalDifferentialPerGame      float64         `json:"goal_differential_per_game"`
	WinProbabilityByDifferential map[int]float64 `json:"win_probability_by_differential"`
	TotalGames                   int             `json:"total_games"`
	GoalDifferentialCounts       map[int]int     `json:"goal_differential_counts"`
	TotalWins                    int             `json:"total_wins"`
	WinsByDifferential           map[int]int     `json:"wins_by_differential"`
}
type gameState struct {
	homeTeam  string
	awayTeam  string
	homeGoals int
	awayGoals int
	homeWin   bool
	period    int
}


func ProcessGoalDifferentialHandler(c echo.Context) error {
	filePath := c.QueryParam("filePath")
	if filePath == "" {
		filePath = "data/feb21shots.csv"
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

	columns := records[0]
	seasonIdx := findColumnIndex(columns, "season")
	teamIdx := findColumnIndex(columns, "teamCode")
	eventIdx := findColumnIndex(columns, "event")
	gameIdx := findColumnIndex(columns, "game_id")
	homeTeamIdx := findColumnIndex(columns, "homeTeamCode")
	awayTeamIdx := findColumnIndex(columns, "awayTeamCode")
	homeGoalsIdx := findColumnIndex(columns, "homeTeamGoals")
	awayGoalsIdx := findColumnIndex(columns, "awayTeamGoals")
	homeWinIdx := findColumnIndex(columns, "homeTeamWon")
	periodIdx := findColumnIndex(columns, "period")

	if seasonIdx == -1 || teamIdx == -1 || eventIdx == -1 || gameIdx == -1 || homeTeamIdx == -1 || awayTeamIdx == -1 ||
		homeGoalsIdx == -1 || awayGoalsIdx == -1 || homeWinIdx == -1 || periodIdx == -1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSV does not contain required columns"})
	}

	teamStats := make(map[string]*GoalDifferentialStats)
	gameStates := make(map[string][]gameState) // gameID -> []gameState

	for _, record := range records[1:] {
		if record[seasonIdx] != "2024" {
			continue
		}
		gameID := record[gameIdx]
		homeTeam := record[homeTeamIdx]
		awayTeam := record[awayTeamIdx]
		homeGoals, err := strconv.Atoi(record[homeGoalsIdx])
		if err != nil {
			continue
		}
		awayGoals, err := strconv.Atoi(record[awayGoalsIdx])
		if err != nil {
			continue
		}

		homeWin, err := strconv.ParseBool(record[homeWinIdx])
		if err != nil {
			continue
		}
		period, err := strconv.Atoi(record[periodIdx])
		if err != nil {
			continue
		}

		if _, exists := gameStates[gameID]; !exists {
			gameStates[gameID] = []gameState{}
		}

		gameStates[gameID] = append(gameStates[gameID], gameState{
			homeTeam:  homeTeam,
			awayTeam:  awayTeam,
			homeGoals: homeGoals,
			awayGoals: awayGoals,
			homeWin:   homeWin,
			period:    period,
		})

		if _, exists := teamStats[homeTeam]; !exists {
			teamStats[homeTeam] = newGoalDifferentialStats()
		}
		if _, exists := teamStats[awayTeam]; !exists {
			teamStats[awayTeam] = newGoalDifferentialStats()
		}

	}

	for _, game := range gameStates {
		var secondPeriodState gameState
		for _, state := range game {
			if state.period == 2 {
				secondPeriodState = state
				break
			}
		}

		if secondPeriodState.homeTeam == "" { // if there is no 2nd period recorded, skip the game
			continue
		}

		homeDiff := clamp(secondPeriodState.homeGoals-secondPeriodState.awayGoals, -4, 4)
		awayDiff := clamp(secondPeriodState.awayGoals-secondPeriodState.homeGoals, -4, 4)

		homeStats := teamStats[secondPeriodState.homeTeam]
		awayStats := teamStats[secondPeriodState.awayTeam]

		homeStats.GoalDifferentialCounts[homeDiff]++
		awayStats.GoalDifferentialCounts[awayDiff]++

		if secondPeriodState.homeWin {
			homeStats.WinsByDifferential[homeDiff]++
		} else {
			awayStats.WinsByDifferential[awayDiff]++
		}
	}

	for _, stats := range teamStats {
		stats.TotalGames = 0
		for _, count := range stats.GoalDifferentialCounts {
			stats.TotalGames += count
		}

		stats.TotalWins = 0
		for _, wins := range stats.WinsByDifferential {
			stats.TotalWins += wins
		}

		totalDifferential := 0
		for differential, count := range stats.GoalDifferentialCounts {
			totalDifferential += differential * count
		}
		if stats.TotalGames > 0 {
			stats.GoalDifferentialPerGame = float64(totalDifferential) / float64(stats.TotalGames)
		}

		for diff := -4; diff <= 4; diff++ {
			count := stats.GoalDifferentialCounts[diff]
			wins := stats.WinsByDifferential[diff]
			if count > 0 {
				stats.WinProbabilityByDifferential[diff] = float64(wins) / float64(count) * 100.0
			} else {
				stats.WinProbabilityByDifferential[diff] = -1000
			}
		}
	}

	type GoalDifferentialStatsWithNA struct { // New struct for JSON output
		Team                         string              `json:"team"`
		GoalDifferentialPerGame      float64             `json:"goal_differential_per_game"`
		WinProbabilityByDifferential map[int]interface{} `json:"win_probability_by_differential"` // Changed to interface{}
		TotalGames                   int                 `json:"total_games"`
		GoalDifferentialCounts       map[int]int         `json:"goal_differential_counts"`
		TotalWins                    int                 `json:"total_wins"`
		WinsByDifferential           map[int]int         `json:"wins_by_differential"`
	}

	teamStatsWithNA := make(map[string]*GoalDifferentialStatsWithNA)
	for team, stats := range teamStats {
		winProbs := make(map[int]interface{})
		for diff := -4; diff <= 4; diff++ {
			if stats.WinProbabilityByDifferential[diff] == -1000 {
				winProbs[diff] = "N/A" // Set "N/A" string for no data
			} else {
				winProbs[diff] = stats.WinProbabilityByDifferential[diff]
			}
		}

		teamStatsWithNA[team] = &GoalDifferentialStatsWithNA{
			Team:                         stats.Team,
			GoalDifferentialPerGame:      stats.GoalDifferentialPerGame,
			WinProbabilityByDifferential: winProbs,
			TotalGames:                   stats.TotalGames,
			GoalDifferentialCounts:       stats.GoalDifferentialCounts,
			TotalWins:                    stats.TotalWins,
			WinsByDifferential:           stats.WinsByDifferential,
		}
	}

	return c.JSON(http.StatusOK, teamStatsWithNA)
}

func newGoalDifferentialStats() *GoalDifferentialStats {
	stats := &GoalDifferentialStats{
		WinProbabilityByDifferential: make(map[int]float64),
		GoalDifferentialCounts:       make(map[int]int),
		WinsByDifferential:           make(map[int]int),
	}

	for diff := -4; diff <= 4; diff++ {
		stats.GoalDifferentialCounts[diff] = 0
		stats.WinsByDifferential[diff] = 0
		stats.WinProbabilityByDifferential[diff] = 0.0
	}
	return stats
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
