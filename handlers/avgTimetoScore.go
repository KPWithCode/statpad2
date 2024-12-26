package handlers

import (
	"encoding/csv"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

type TimeToScoreStats struct {
	Team                   string  `json:"team"`
	TotalGoalTime          int     `json:"total_goal_time"`
	Goals                  int     `json:"goals"`
	AverageTimeToScore     float64 `json:"average_time_to_score"`
	TotalFirstGoalTime     int     `json:"total_first_goal_time"`
	FirstGoals             int     `json:"first_goals"`
	AverageTimeToFirstGoal float64 `json:"average_time_to_first_goal"`
}

// current season data
func ProcessTimeToScoreHandler(c echo.Context) error {
	file, err := os.Open("data/shots_2024.csv")
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
	eventIdx := findColumnIndex(columns, "event")
	timeIdx := findColumnIndex(columns, "time")
	teamIdx := findColumnIndex(columns, "teamCode")
	seasonIdx := findColumnIndex(columns, "season")
	gameIDIdx := findColumnIndex(columns, "game_id")

	if eventIdx == -1 || timeIdx == -1 || teamIdx == -1 || seasonIdx == -1 || gameIDIdx == -1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSV does not contain required columns"})
	}

	// Assume current season is "2024". This could be dynamically retrieved if needed.
	currentSeason := "2024"

	stats := make(map[string]*TimeToScoreStats)
	firstGoalTracker := make(map[string]bool) // Tracks if a first goal has been recorded for a game

	for _, record := range records[1:] {
		event := record[eventIdx]
		time, err := strconv.Atoi(record[timeIdx])
		if err != nil {
			continue
		}
		timeInMinutes := time / 60
		team := record[teamIdx]
		season := record[seasonIdx]
		gameID := record[gameIDIdx]

		// Filter by current season and only consider "goal" events
		if strings.ToLower(season) != currentSeason || strings.ToLower(event) != "goal" {
			continue
		}

		if _, ok := stats[team]; !ok {
			stats[team] = &TimeToScoreStats{Team: team}
		}

		// Track total goal time and goals
		stats[team].TotalGoalTime += timeInMinutes
		stats[team].Goals++

		// Check for the first goal in the game
		if !firstGoalTracker[gameID] {
			firstGoalTracker[gameID] = true
			stats[team].TotalFirstGoalTime += timeInMinutes
			stats[team].FirstGoals++
		}
	}

	// Calculate averages
	for _, stat := range stats {
		if stat.Goals > 0 {
			stat.AverageTimeToScore = float64(stat.TotalGoalTime) / float64(stat.Goals)
		}
		if stat.FirstGoals > 0 {
			stat.AverageTimeToFirstGoal = float64(stat.TotalFirstGoalTime) / float64(stat.FirstGoals)
		}
	}

	return c.JSON(http.StatusOK, stats)
}

