
package handlers

import (
	"encoding/csv"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

// ShotsToGoalStats represents stats for shots-to-goal conversion rate.
type ShotsToGoalStats struct {
	Team               string  `json:"team"`
	Shots              int     `json:"shots"`
	Goals              int     `json:"goals"`
	ConversionRate     float64 `json:"conversion_rate"`
}
func ProcessShotsToGoalHandler(c echo.Context) error {
	file, err := os.Open("data/shotsfeb1.csv")
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
	teamIdx := findColumnIndex(columns, "teamCode")

	if eventIdx == -1 || teamIdx == -1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSV does not contain required columns"})
	}

	stats := make(map[string]*ShotsToGoalStats)

	for _, record := range records[1:] {
		event := record[eventIdx]
		team := record[teamIdx]

		if _, ok := stats[team]; !ok {
			stats[team] = &ShotsToGoalStats{Team: team}
		}

		if strings.ToLower(event) == "goal" {
			stats[team].Goals++
		}

		if strings.ToLower(event) == "shot" || strings.ToLower(event) == "goal" {
			stats[team].Shots++
		}
	}

	for _, stat := range stats {
		if stat.Shots > 0 {
			stat.ConversionRate = float64(stat.Goals) / float64(stat.Shots)
		}
	}

	return c.JSON(http.StatusOK, stats)
}