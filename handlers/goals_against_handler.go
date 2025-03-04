package handlers

import (
	"encoding/csv"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
)

type GoalsAgainst struct {
	Team         string             `json:"team"`
	GoalsPerGame map[string]float64 `json:"goals_per_game"`
	TotalGoals   map[string]int     `json:"total_goals"`
	TotalGames   int                `json:"total_games"`
}

type TeamRanking struct {
	Team                string  `json:"team"`
	Rank                int     `json:"rank"`
	GoalsAgainstPerGame float64 `json:"goals_against_per_game"`
}

func ProcessGoalsAgainstHandler(c echo.Context) error {
	file, err := os.Open("data/march3.csv")
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
	seasonIdx := findColumnIndex(columns, "season")

	if positionIdx == -1 || teamIdx == -1 || eventIdx == -1 || gameIdx == -1 ||
		isHomeTeamIdx == -1 || homeTeamCodeIdx == -1 || awayTeamCodeIdx == -1 || seasonIdx == -1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSV does not contain required columns"})
	}

	// Define the current season
	currentSeason := "2024"

	teamStats := make(map[string]*GoalsAgainst)
	gameCount := make(map[string]map[string]bool)

	for _, record := range records[1:] {
		season := record[seasonIdx]
		if season != currentSeason {
			continue // Skip records that are not from the current season
		}

		position := record[positionIdx]
		team := record[teamIdx]
		event := record[eventIdx]
		gameID := record[gameIdx]
		isHomeTeam := record[isHomeTeamIdx] == "true"
		homeTeam := record[homeTeamCodeIdx]
		awayTeam := record[awayTeamCodeIdx]

		if _, ok := teamStats[team]; !ok {
			teamStats[team] = &GoalsAgainst{
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
				teamStats[defendingTeam] = &GoalsAgainst{
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

	// Create a list of teams and calculate their goals against per game (average)
	teamRankings := []TeamRanking{}
	for _, stats := range teamStats {
		totalGoalsAgainst := 0.0
		for _, goals := range stats.TotalGoals {
			totalGoalsAgainst += float64(goals)
		}
		goalsAgainstPerGame := totalGoalsAgainst / float64(stats.TotalGames)
		teamRankings = append(teamRankings, TeamRanking{
			Team:                stats.Team,
			GoalsAgainstPerGame: goalsAgainstPerGame,
		})
	}

	// Sort teams by goals against per game (from most to least)
	sort.Slice(teamRankings, func(i, j int) bool {
		return teamRankings[i].GoalsAgainstPerGame > teamRankings[j].GoalsAgainstPerGame
	})

	// Assign ranks based on sorted order
	for i := range teamRankings {
		teamRankings[i].Rank = i + 1
	}

	// Return the response
	return c.JSON(http.StatusOK, teamRankings)
}
