package handlers

import (
	"encoding/csv"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
    "math"

	"github.com/labstack/echo/v4"
)

type DangerZoneStats struct {
	Team                   string  `json:"team"`
	TotalShotsAllowed      int     `json:"total_shots_allowed"`
	DangerZoneShotsAllowed int     `json:"danger_zone_shots_allowed"`
	DangerZonePercentage   float64 `json:"danger_zone_percentage"`
    Rank                   int     `json:"rank"`
}

func ProcessDangerZone(c echo.Context) error {
	file, err := os.Open("data/feb8.csv")
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
	teamIdx := findColumnIndex(columns, "teamCode") // Updated from "defendingTeam"
	distanceIdx := findColumnIndex(columns, "arenaAdjustedShotDistance")
	angleIdx := findColumnIndex(columns, "shotAngleAdjusted")
    shotTypeIdx := findColumnIndex(columns, "shotType")

	if teamIdx == -1 || distanceIdx == -1 || angleIdx == -1 || shotTypeIdx == -1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSV does not contain required columns"})
	}

	stats := make(map[string]*DangerZoneStats)

	for _, record := range records[1:] {
		if len(record) <= max(teamIdx, distanceIdx, angleIdx, shotTypeIdx) { // Prevent index out of range
			continue
		}

		team := record[teamIdx]
		if team == "" {
			continue
		}

		shotDistance, err1 := strconv.ParseFloat(record[distanceIdx], 64)
		shotAngle, err2 := strconv.ParseFloat(record[angleIdx], 64)
		if err1 != nil || err2 != nil {
			continue // Skip invalid rows
		}

        shotType := strings.ToLower(record[shotTypeIdx]) // Normalize for case insensitivity

		// Consider only deflections, tips, rebounds, and regular shots
		if !(shotType == "shot" || shotType == "deflection" || shotType == "tip" || shotType == "rebound" || shotType == "wrist" || shotType == "snap" || shotType == "slap" || shotType == "back") {
			continue
		}

        
		if _, exists := stats[team]; !exists {
			stats[team] = &DangerZoneStats{Team: team}
		}

		stats[team].TotalShotsAllowed++

		// Danger zone: distance <= 10 or angle between 30-60
		// if (shotDistance >= 10 && shotDistance <= 20) || math.Abs(shotAngle) <=22.5 {
		// 	stats[team].DangerZoneShotsAllowed++
		// }
        isDangerZone := false
		
		switch {
		case shotDistance <= 20:
			// Inner slot area - highest danger
			isDangerZone = true
		case shotDistance <= 30 && math.Abs(shotAngle) <= 35:
			// Moderate distance but good angle
			isDangerZone = true
		case (shotType == "tip" || shotType == "deflection") && shotDistance <= 25:
			// Tips and deflections are dangerous from slightly further out
			isDangerZone = true
		}

		if isDangerZone {
			stats[team].DangerZoneShotsAllowed++
		}
	}

    var statsList []*DangerZoneStats
	// Compute percentages
	for _, stat := range stats {
		if stat.TotalShotsAllowed > 0 {
			stat.DangerZonePercentage = (float64(stat.DangerZoneShotsAllowed) / float64(stat.TotalShotsAllowed)) * 100
		}
        statsList = append(statsList, stat)
	}
    // Sort by DangerZonePercentage in descending order (higher % ranks higher)
	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].DangerZonePercentage > statsList[j].DangerZonePercentage
	})
    for i, stat := range statsList {
		stat.Rank = i + 1
	}


	return c.JSON(http.StatusOK, stats)
}

// max returns the maximum value of given integers
func max(nums ...int) int {
	maxNum := nums[0]
	for _, num := range nums {
		if num > maxNum {
			maxNum = num
		}
	}
	return maxNum
}


// According to data most goals(34.3 %) occur within 10 to 20 feet of 
// the net. The tip-in and backhand are the next most effective shots in 
// that same area with 15.1% and 13.5% success rates respectively.