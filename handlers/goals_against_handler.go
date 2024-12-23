package handlers

import (
	"encoding/csv"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

func ProcessGoalsAgainstHandler(c echo.Context) error {
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

	// Column indices
	columns := records[0]
	positionIdx := findColumnIndex(columns, "playerPositionThatDidEvent")
	teamIdx := findColumnIndex(columns, "teamCode")
	eventIdx := findColumnIndex(columns, "event")
	gameIdx := findColumnIndex(columns, "game_id")
	isHomeTeamIdx := findColumnIndex(columns, "isHomeTeam")
	homeTeamCodeIdx := findColumnIndex(columns, "homeTeamCode")
	awayTeamCodeIdx := findColumnIndex(columns, "awayTeamCode")

	if positionIdx == -1 || teamIdx == -1 || eventIdx == -1 || gameIdx == -1 || 
	   isHomeTeamIdx == -1 || homeTeamCodeIdx == -1 || awayTeamCodeIdx == -1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSV does not contain required columns"})
	}

	teamStats := make(map[string]*GoalsStats)
	gameCount := make(map[string]map[string]bool)

	for _, record := range records[1:] {
		position := record[positionIdx]
		team := record[teamIdx]
		event := record[eventIdx]
		gameID := record[gameIdx]
		isHomeTeam := record[isHomeTeamIdx] == "true"
		homeTeam := record[homeTeamCodeIdx]
		awayTeam := record[awayTeamCodeIdx]

		if _, ok := teamStats[team]; !ok {
			teamStats[team] = &GoalsStats{
				Team:         team,
				GoalsPerGame: make(map[string]float64),
				TotalGoals:   make(map[string]int),
			}
			gameCount[team] = make(map[string]bool)
		}

		stats := teamStats[team]

		if !gameCount[team][gameID] {
			gameCount[team][gameID] = true
			stats.TotalGames++
		}

		if strings.ToLower(event) == "goal" {
			// Identify the defending team
			defendingTeam := awayTeam
			if isHomeTeam {
				defendingTeam = homeTeam
			}

			if _, ok := teamStats[defendingTeam]; !ok {
				teamStats[defendingTeam] = &GoalsStats{
					Team:         defendingTeam,
					GoalsPerGame: make(map[string]float64),
					TotalGoals:   make(map[string]int),
				}
				gameCount[defendingTeam] = make(map[string]bool)
			}

			// Increment goals against for the defending team
			defendingStats := teamStats[defendingTeam]
			defendingStats.TotalGoals[position]++
		}
	}

	// Calculate goals against per game for each position
	for _, stats := range teamStats {
		for pos, totalGoals := range stats.TotalGoals {
			stats.GoalsPerGame[pos] = float64(totalGoals) / float64(stats.TotalGames)
		}
	}

	return c.JSON(http.StatusOK, teamStats)
}
