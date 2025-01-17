package nbahandler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"


	"github.com/labstack/echo/v4"
)

type RelativeEfficiency struct {
	Team                   string  `json:"team"`
	Season                 string  `json:"season"` // Assuming the API returns the season
	RelativeOffEfficiency  float64 `json:"relative_off_efficiency"`
	RelativeDefEfficiency  float64 `json:"relative_def_efficiency"`
}

func RelativeEfficiencyHandler(c echo.Context) error {
	league := "nba" // League fixed to "nba"
	season := "2024-25" // The season you want to filter by

	// Define all NBA teams
	teams := []string{
		"Lakers", "Celtics", "Nets", "Warriors", "Bucks", "Heat", "Suns", "76ers", 
		"Knicks", "Mavericks", "Timberwolves", "Nuggets", "Clippers", "Raptors", 
		"Kings", "Pelicans", "Hawks", "Bulls", "Magic", "Hornets", "Cavaliers", 
		"Pistons", "Pacers", "Thunder", "Wizards", "Spurs", "Jazz", "Rockets",
	}

	// Create a map to hold data for each team
	teamData := make(map[string]RelativeEfficiency)

	// Loop through each team and fetch relative efficiency data
	for _, team := range teams {
		url := fmt.Sprintf("https://api.pbpstats.com/get-relative-off-def-efficiency/%s?team=%s", league, team)
		response, err := http.Get(url)
		if err != nil || response.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(response.Body)
			fmt.Println("Error fetching data for team:", team, "Error:", string(body))
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to fetch data for team %s: %s", team, body)})
		}
		defer response.Body.Close()

		var efficiencyData RelativeEfficiency
		err = json.NewDecoder(response.Body).Decode(&efficiencyData)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to parse efficiency data for team %s", team)})
		}

		// Only add data for the 2024-2025 season
		if efficiencyData.Season == season {
			teamData[team] = efficiencyData
		}
	}

	// Return the map of team data for the 2024-2025 season
	return c.JSON(http.StatusOK, teamData)
}
