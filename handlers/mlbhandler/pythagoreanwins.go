package mlbhandler

import (
	"encoding/json"
	"net/http"
	"io/ioutil"

	"github.com/labstack/echo/v4"
)

type TeamStatsEntry struct {
	Team struct {
		Name string `json:"name"`
	} `json:"team"`
	Stats struct {
		RunsScored   float64 `json:"runsScored"`
		RunsAllowed  float64 `json:"runsAllowed"`
	} `json:"stats"`
}

type TeamStatsResponse struct {
	TeamStatsTotals []TeamStatsEntry `json:"teamStatsTotals"`
}

func fetchTeamStats() ([]TeamStatsEntry, error) {
	// Fetching data from external MLB API
	resp, err := http.Get("https://api.mysportsfeeds.com/v2.1/pull/mlb/current/team_stats_totals.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var teamStatsResponse TeamStatsResponse
	err = json.Unmarshal(body, &teamStatsResponse)
	if err != nil {
		return nil, err
	}

	return teamStatsResponse.TeamStatsTotals, nil
}

type PythagoreanTeam struct {
	Team           string  `json:"team"`
	ExpectedWinPct float64 `json:"expectedWinPct"`
}

func PythagoreanHandler(c echo.Context) error {
	teamStats, err := fetchTeamStats()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch team stats"})
	}

	var results []PythagoreanTeam
	for _, entry := range teamStats {
		// Applying the Pythagorean win expectancy formula
		denominator := (entry.Stats.RunsScored * entry.Stats.RunsScored) + (entry.Stats.RunsAllowed * entry.Stats.RunsAllowed)
		expectedWinPct := entry.Stats.RunsScored * entry.Stats.RunsScored / denominator

		results = append(results, PythagoreanTeam{
			Team:           entry.Team.Name,
			ExpectedWinPct: expectedWinPct,
		})
	}

	return c.JSON(http.StatusOK, results)
}
