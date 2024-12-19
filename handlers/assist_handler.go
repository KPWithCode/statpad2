package handlers

import (
	"encoding/csv"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

type AssistsStats struct {
	Team           string             `json:"team"`
	AssistsPerGame map[string]float64 `json:"assists_per_game"` // Position -> Avg Assists Allowed
	TotalGames     int                `json:"total_games"`
	TotalAssists   map[string]int     `json:"total_assists"`    // Position -> Total Assists
	PlayerAssists  map[string]int     `json:"player_assists"`   // Player -> Total Assists
}

func ProcessAssistsHandler(c echo.Context) error {
	// Open the predefined local CSV file
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
	teamIdx := findColumnIndex(columns, "team")
	eventIdx := findColumnIndex(columns, "event")
	gameIdx := findColumnIndex(columns, "game_id")
	goalIdx := findColumnIndex(columns, "goal")
	lastEventCategoryIdx := findColumnIndex(columns, "lastEventCategory")
	lastEventPlayerIdx := findColumnIndex(columns, "playerNumThatDidLastEvent")

	if positionIdx == -1 || teamIdx == -1 || eventIdx == -1 || gameIdx == -1 || goalIdx == -1 || lastEventCategoryIdx == -1 || lastEventPlayerIdx == -1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSV does not contain required columns"})
	}

	teamStats := make(map[string]*AssistsStats)
	gameCount := make(map[string]map[string]bool)

	for _, record := range records[1:] {
		goal := strings.ToLower(record[goalIdx]) == "1" // Check if the current event is a goal
		lastEventCategory := strings.ToLower(record[lastEventCategoryIdx])
		lastEventPlayer := record[lastEventPlayerIdx]
		position := record[positionIdx]
		team := record[teamIdx]
		gameID := record[gameIdx]

		if _, ok := teamStats[team]; !ok {
			teamStats[team] = &AssistsStats{
				Team:           team,
				AssistsPerGame: make(map[string]float64),
				TotalAssists:   make(map[string]int),
				PlayerAssists:  make(map[string]int),
			}
			gameCount[team] = make(map[string]bool)
		}

		stats := teamStats[team]

		if !gameCount[team][gameID] {
			gameCount[team][gameID] = true
			stats.TotalGames++
		}

		// If the event is a goal and the last event was a "pass", record an assist
		if goal && lastEventCategory == "pass" {
			stats.TotalAssists[position]++
			stats.PlayerAssists[lastEventPlayer]++
		}
	}

	// Calculate assists per game for each position
	for _, stats := range teamStats {
		for pos, totalAssists := range stats.TotalAssists {
			stats.AssistsPerGame[pos] = float64(totalAssists) / float64(stats.TotalGames)
		}
	}

	return c.JSON(http.StatusOK, teamStats)
}
// func ProcessAssistsHandler(c echo.Context) error {
// 	// Open the predefined local CSV file
// 	file, err := os.Open("data/shots_2024.csv")
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to open CSV file"})
// 	}
// 	defer file.Close()

// 	reader := csv.NewReader(file)
// 	records, err := reader.ReadAll()
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse CSV"})
// 	}

// 	if len(records) < 2 {
// 		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSV file is empty or invalid"})
// 	}

// 	// Column indices
// 	columns := records[0]
// 	positionIdx := findColumnIndex(columns, "ShooterPosition")
// 	teamAgainstIdx := findColumnIndex(columns, "DefendingTeam")
// 	outcomeIdx := findColumnIndex(columns, "Event")
// 	gameIdx := findColumnIndex(columns, "GameId")

// 	if positionIdx == -1 || teamAgainstIdx == -1 || outcomeIdx == -1 || gameIdx == -1 {
// 		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSV does not contain required columns"})
// 	}

// 	teamStats := make(map[string]*AssistsStats)
// 	gameCount := make(map[string]map[string]bool)

// 	for _, record := range records[1:] {
// 		position := record[positionIdx]
// 		teamAgainst := record[teamAgainstIdx]
// 		outcome := record[outcomeIdx]
// 		gameID := record[gameIdx]

// 		if _, ok := teamStats[teamAgainst]; !ok {
// 			teamStats[teamAgainst] = &AssistsStats{
// 				Team:           teamAgainst,
// 				AssistsPerGame: make(map[string]float64),
// 				TotalAssists:   make(map[string]int),
// 			}
// 			gameCount[teamAgainst] = make(map[string]bool)
// 		}

// 		stats := teamStats[teamAgainst]

// 		if !gameCount[teamAgainst][gameID] {
// 			gameCount[teamAgainst][gameID] = true
// 			stats.TotalGames++
// 		}

// 		if strings.ToLower(outcome) == "assist" {
// 			stats.TotalAssists[position]++
// 		}
// 	}

// 	// Calculate assists per game for each position
// 	for _, stats := range teamStats {
// 		for pos, totalAssists := range stats.TotalAssists {
// 			stats.AssistsPerGame[pos] = float64(totalAssists) / float64(stats.TotalGames)
// 		}
// 	}

// 	return c.JSON(http.StatusOK, teamStats)
// }

// // // Helper to find column index
// // func findColumnIndex(columns []string, name string) int {
// // 	for i, col := range columns {
// // 		if col == name {
// // 			return i
// // 		}
// // 	}
// // 	return -1
// // }
