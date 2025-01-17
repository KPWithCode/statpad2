package nbahandler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type Matchup struct {
	HomeTeam string `json:"home_team"`
	AwayTeam string `json:"away_team"`
}

type Efficiency struct {
	Team                  string  `json:"team"`
	RelativeOffEfficiency float64 `json:"relative_off_efficiency"`
	RelativeDefEfficiency float64 `json:"relative_def_efficiency"`
}

type ShotQuery struct {
	Team       string  `json:"team"`
	ShotQuality float64 `json:"shot_quality"`
	Conversion  float64 `json:"conversion_rate"`
}

type MatchupInsight struct {
	HomeTeam        string  `json:"home_team"`
	AwayTeam        string  `json:"away_team"`
	HomeOffEff      float64 `json:"home_off_eff"`
	HomeDefEff      float64 `json:"home_def_eff"`
	AwayOffEff      float64 `json:"away_off_eff"`
	AwayDefEff      float64 `json:"away_def_eff"`
	HomeShotQuality float64 `json:"home_shot_quality"`
	AwayShotQuality float64 `json:"away_shot_quality"`
}

func fetchMatchups() ([]Matchup, error) {
	const url = "https://api.pbpstats.com/live/games/nba"
	response, err := http.Get(url)
	if err != nil || response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch matchups: %w", err)
	}
	defer response.Body.Close()

	var data struct {
		GameData []struct {
			Home string `json:"home"`
			Away string `json:"away"`
		} `json:"game_data"`
	}

	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse matchups: %w", err)
	}

	var matchups []Matchup
	for _, game := range data.GameData {
		// Directly append team codes as strings
		matchups = append(matchups, Matchup{
			HomeTeam: game.Home[:3], // Use just the first 3 letters as the team code
			AwayTeam: game.Away[:3], // Same here
		})
	}
	return matchups, nil
}

func fetchEfficiency(team string) (Efficiency, error) {
	url := fmt.Sprintf("https://api.pbpstats.com/get-relative-off-def-efficiency/nba?team=%s", team)
	var efficiency Efficiency

	response, err := http.Get(url)
	if err != nil || response.StatusCode != http.StatusOK {
		return efficiency, fmt.Errorf("failed to fetch efficiency for team %s: %w", team, err)
	}
	defer response.Body.Close()

	if err := json.NewDecoder(response.Body).Decode(&efficiency); err != nil {
		return efficiency, fmt.Errorf("failed to parse efficiency data for team %s: %w", team, err)
	}
	return efficiency, nil
}

func fetchShotData(team string) (ShotQuery, error) {
	url := fmt.Sprintf("https://api.pbpstats.com/get-shot-query-data/nba?team=%s", team)
	var shotData ShotQuery

	response, err := http.Get(url)
	if err != nil || response.StatusCode != http.StatusOK {
		return shotData, fmt.Errorf("failed to fetch shot data for team %s: %w", team, err)
	}
	defer response.Body.Close()

	if err := json.NewDecoder(response.Body).Decode(&shotData); err != nil {
		return shotData, fmt.Errorf("failed to parse shot data for team %s: %w", team, err)
	}
	return shotData, nil
}

func DailyMatchupEfficiencyHandler(c echo.Context) error {
	matchups, err := fetchMatchups()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch matchups"})
	}

	var insights []MatchupInsight
	for _, matchup := range matchups {
		homeEff, err := fetchEfficiency(matchup.HomeTeam)
		if err != nil {
			fmt.Printf("Error fetching efficiency for home team %s: %v\n", matchup.HomeTeam, err)
			continue
		}

		awayEff, err := fetchEfficiency(matchup.AwayTeam)
		if err != nil {
			fmt.Printf("Error fetching efficiency for away team %s: %v\n", matchup.AwayTeam, err)
			continue
		}

		homeShot, err := fetchShotData(matchup.HomeTeam)
		if err != nil {
			fmt.Printf("Error fetching shot data for home team %s: %v\n", matchup.HomeTeam, err)
			continue
		}

		awayShot, err := fetchShotData(matchup.AwayTeam)
		if err != nil {
			fmt.Printf("Error fetching shot data for away team %s: %v\n", matchup.AwayTeam, err)
			continue
		}

		insights = append(insights, MatchupInsight{
			HomeTeam:        matchup.HomeTeam,
			AwayTeam:        matchup.AwayTeam,
			HomeOffEff:      homeEff.RelativeOffEfficiency,
			HomeDefEff:      homeEff.RelativeDefEfficiency,
			AwayOffEff:      awayEff.RelativeOffEfficiency,
			AwayDefEff:      awayEff.RelativeDefEfficiency,
			HomeShotQuality: homeShot.ShotQuality,
			AwayShotQuality: awayShot.ShotQuality,
		})
	}

	if len(insights) == 0 {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "No data available for today's matchups"})
	}

	return c.JSON(http.StatusOK, insights)
}
