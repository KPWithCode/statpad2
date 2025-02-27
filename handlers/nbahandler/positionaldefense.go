package nbahandler

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

type PlayerResponse struct {
    LastUpdatedOn     string              `json:"lastUpdatedOn"`
    PlayerStatsTotals []PlayerStatsData   `json:"playerStatsTotals"`
}

type PlayerStatsData struct {
    Player Player     `json:"player"`
    Team   TeamInfo   `json:"team"`
    Stats  PlayerStatsPD `json:"stats"`
}

type Player struct {
    ID              int     `json:"id"`
    FirstName       string  `json:"firstName"`
    LastName        string  `json:"lastName"`
    PrimaryPosition string  `json:"primaryPosition"`
    CurrentTeam     TeamInfo `json:"currentTeam"`
}

type PlayerStatsPD struct {
    GamesPlayed    int            `json:"gamesPlayed"`
    Defense        PlayerDefense  `json:"defense"`
    MiscellaneousPD  MiscellaneousPD  `json:"miscellaneous"`
}

type PlayerDefense struct {
    Stl                int     `json:"stl"`
    StlPerGame        float64 `json:"stlPerGame"`
    Blk                int     `json:"blk"`
    BlkPerGame        float64 `json:"blkPerGame"`
    BlkAgainst         int     `json:"blkAgainst"`
    BlkAgainstPerGame float64 `json:"blkAgainstPerGame"`
	Tov				  int		`json:"tov"`
	TovPerGame		  float64	`json:"tovePerGame"`
}

type MiscellaneousPD struct {
    MinSeconds         int     `json:"minSeconds"`
    MinSecondsPerGame float64 `json:"minSecondsPerGame"`
    PlusMinusPerGame  float64 `json:"plusMinusPerGame"`
}

type PositionDefenseStats struct {
    Team            string  `json:"team"`
    Position        string  `json:"position"`
    StealsPerGame     float64 `json:"stealsPerGame"`
    BlocksPerGame     float64 `json:"blocksPerGame"`
	TurnoverPerGame  float64 `json:"tovPerGame"`
    DefensiveRating float64 `json:"defensiveRating"`
    DefensiveRank   int     `json:"defensiveRank"`
}

type MatchupDefenseStats struct {
    TeamOne      string                  `json:"teamOne"`
    TeamTwo      string                  `json:"teamTwo"`
    PositionStats []PositionDefenseStats `json:"positionStats"`
}

func loadPDEnvVars() error {
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("Error loading .env file")
	}
	return nil
}

func fetchTodaysSchedule() ([]string, error) {
    if err := loadPDEnvVars(); err != nil {
        return nil, fmt.Errorf("error loading environment variables: %v", err)
    }

    apiKey := os.Getenv("MYSPORTSFEEDS_API_KEY")
    if apiKey == "" {
        return nil, fmt.Errorf("API key not found in environment variables")
    }

    
	today := time.Now().AddDate(0, 0, 1).Format("20060102")
    fmt.Print(today)
    // today := time.Now().UTC().Format("20060102")
    // today := time.Now().Format("20060102")
	endpoint := fmt.Sprintf("https://api.mysportsfeeds.com/v2.1/pull/nba/current/date/%s/games.json", today)

    req, err := http.NewRequest("GET", endpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("error creating request: %v", err)
    }

    req.SetBasicAuth(apiKey, "MYSPORTSFEEDS")

	q := req.URL.Query()
    q.Add("status", "in-progress,unplayed,final") // Include all relevant game statuses
    q.Add("sort", "game.starttime.A")             // Sort by start time ascending
    q.Add("force", "true")
    req.URL.RawQuery = q.Encode()


    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error making request: %v", err)
    }
    defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, fmt.Errorf("no games scheduled for today")
	}

	if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API returned non-200 status code: %d, body: %s", 
            resp.StatusCode, string(bodyBytes))
    }
	body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("error reading response: %v", err)
    }

	var response struct {
        LastUpdatedOn string `json:"lastUpdatedOn"`
        Games []struct {
            Schedule struct {
                AwayTeam struct {
                    // ID           int    `json:"id"`
                    Abbreviation string `json:"abbreviation"`
                } `json:"awayTeam"`
                HomeTeam struct {
                    // ID           int    `json:"id"`
                    Abbreviation string `json:"abbreviation"`
                } `json:"homeTeam"`
                StartTime string `json:"startTime"`
                // PlayedStatus string `json:"playedStatus"`
            } `json:"schedule"`
            // Score struct {
            //     AwayScoreTotal int `json:"awayScoreTotal"`
            //     HomeScoreTotal int `json:"homeScoreTotal"`
            // } `json:"score"`
        } `json:"games"`
    }

	if err := json.Unmarshal(body, &response); err != nil {
        // Print first 100 characters of body for context
        preview := string(body)
        if len(preview) > 100 {
            preview = preview[:100] + "..."
        }
        return nil, fmt.Errorf("error parsing JSON: %v, body preview: %s", err, preview)
    }
	
    var teams []string
    for _, game := range response.Games {
        teams = append(teams, game.Schedule.AwayTeam.Abbreviation, game.Schedule.HomeTeam.Abbreviation)
    }
    return teams, nil
}



func fetchPlayerPositionalStats(playingTeams []string) ([]PlayerStatsData, error) {
	if err := loadEnvVars(); err != nil {
		return nil, fmt.Errorf("error loading environment variables: %v", err)
	}

	apiKey := os.Getenv("MYSPORTSFEEDS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API key not found in environment variables")
	}

    baseURL := "https://api.mysportsfeeds.com/v2.1/pull/nba"
    endpoint := fmt.Sprintf("%s/current/player_stats_totals.json", baseURL)

    req, err := http.NewRequest("GET", endpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("error creating request: %v", err)
    }

    req.SetBasicAuth(apiKey, "MYSPORTSFEEDS")

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error making request: %v", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("error reading response: %v", err)
    }

    var playerResponse PlayerResponse
    if err := json.Unmarshal(body, &playerResponse); err != nil {
        return nil, fmt.Errorf("error parsing JSON: %v", err)
    }
	var filteredPlayerStats []PlayerStatsData
	for _, player := range playerResponse.PlayerStatsTotals {
		for _, team := range playingTeams {
			if player.Team.Abbreviation == team {
				filteredPlayerStats = append(filteredPlayerStats,player)
				break
			}
		}
	}

    return filteredPlayerStats, nil
}

func PositionalDefenseHandler(c echo.Context) error {
    playingTeams, err := fetchTodaysScheduleII()
	if err != nil {
        if err.Error() == "no games scheduled for today" {
            return c.JSON(http.StatusOK, map[string]interface{}{
                "status": "success",
                "data": map[string]interface{}{
                    "message": "No NBA games scheduled for today",
                },
            })
        }
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status": "error",
            "error":  fmt.Sprintf("Failed to fetch today's schedule: %v", err),
        })
    }
	
	playerStats, err := fetchPlayerPositionalStats(playingTeams)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status": "error",
            "error":  fmt.Sprintf("Failed to fetch player stats: %v", err),
        })
	}
	positionDefenseStats := make(map[string][]PositionDefenseStats)

    for _, player := range playerStats {
        team := player.Team.Abbreviation
        pos := player.Player.PrimaryPosition

        // Calculate defensive stats for each team for each position
        var spg, bpg, tovPG float64
        spg = float64(player.Stats.Defense.StlPerGame)
        bpg = float64(player.Stats.Defense.BlkPerGame)
        tovPG = float64(player.Stats.Defense.TovPerGame)

        defRating := 0.3*spg + 0.3*bpg + 0.2*(1/(tovPG+0.1)) // Simple defensive rating formula

        positionDefenseStats[pos] = append(positionDefenseStats[pos], PositionDefenseStats{
            Team:             team,
            Position:         pos,
            StealsPerGame:    spg,
            BlocksPerGame:    bpg,
            TurnoverPerGame: tovPG,
            DefensiveRating:  defRating,
        })
    }

    // Create a map for the worst 10 defensive teams for each position
    worstDefensiveTeams := make(map[string][]PositionDefenseStats)

    // For each position, find the 10 worst defending teams
    for pos, stats := range positionDefenseStats {
        // Sort teams by defensive rating (descending for worst defense)
        sort.Slice(stats, func(i, j int) bool {
            return stats[i].DefensiveRating < stats[j].DefensiveRating
        })

        // Get the top 10 worst defending teams for the current position
        worstDefensiveTeams[pos] = stats[:min(10, len(stats))] // If there are less than 10 teams, take all of them
    }

    // Generate the response with stats for the worst 10 defending teams for each position
    return c.JSON(http.StatusOK, map[string]interface{}{
        "status": "success",
        "data": worstDefensiveTeams,
    })
	// teamPositionStats := make(map[string]map[string][]PlayerStatsData)
    // for _, player := range playerStats {
    //     team := player.Team.Abbreviation
    //     pos := player.Player.PrimaryPosition

    //     if teamPositionStats[team] == nil {
    //         teamPositionStats[team] = make(map[string][]PlayerStatsData)
    //     }
    //     teamPositionStats[team][pos] = append(teamPositionStats[team][pos], player)
    // }

    // // Calculate position-based defensive stats for each team
    // teamStats := make(map[string][]PositionDefenseStats)
    // for team, positions := range teamPositionStats {
    //     for pos, players := range positions {
    //         var totalMinutes, spg, bpg float64
    //         var plusMinusPG float64
	// 		var tovPG float64

    //         for _, player := range players {
    //             minutes := float64(player.Stats.MiscellaneousPD.MinSeconds) / 60
    //             totalMinutes += minutes
    //             spg += float64(player.Stats.Defense.StlPerGame)
    //             bpg += float64(player.Stats.Defense.BlkPerGame)
    //             plusMinusPG += player.Stats.MiscellaneousPD.PlusMinusPerGame
	// 			tovPG += float64(player.Stats.Defense.TovPerGame)
    //         }
	// 		if totalMinutes == 0 {
    //             continue // Skip positions with no recorded minutes
    //         }

    //         // stealsPer36 := (totalSteals / totalMinutes) * 36
    //         // blocksPer36 := (totalBlocks / totalMinutes) * 36
	// 		defRating := 0.3 * spg + 0.3 * bpg + 0.2 * plusMinusPG + 0.2 * (1 / (tovPG + 0.1))

	// 		// defRating := 1 * ((spg * 1) + (bpg * 1) + (plusMinusPG / float64(len(players))) * 2 + (tovPG / float64(len(players))) * 2)

        

    //         teamStats[team] = append(teamStats[team], PositionDefenseStats{
    //             Team:            team,
    //             Position:        pos,
    //             StealsPerGame:     roundToOneDecimal(spg),
    //             BlocksPerGame:     roundToOneDecimal(bpg),
    //             DefensiveRating: roundToOneDecimal(defRating),
    //         })
    //     }
    // }

    // // Create matchups
    // var matchups []MatchupDefenseStats
    // for i := 0; i < len(playingTeams); i += 2 {
    //     if i+1 >= len(playingTeams) {
    //         break
    //     }
        
    //     teamOne := playingTeams[i]
    //     teamTwo := playingTeams[i+1]
        
    //     matchupStats := MatchupDefenseStats{
    //         TeamOne: teamOne,
    //         TeamTwo: teamTwo,
    //         PositionStats: append(teamStats[teamOne], teamStats[teamTwo]...),
    //     }

    //     // Sort position stats by defensive rating
    //     sort.Slice(matchupStats.PositionStats, func(i, j int) bool {
    //         return matchupStats.PositionStats[i].DefensiveRating > matchupStats.PositionStats[j].DefensiveRating
    //     })

    //     // Assign ranks within the matchup
    //     for i := range matchupStats.PositionStats {
    //         matchupStats.PositionStats[i].DefensiveRank = i + 1
    //     }

    //     matchups = append(matchups, matchupStats)
    // }
	// return c.JSON(http.StatusOK, map[string]interface{}{
    //     "status": "success",
    //     "matchups": matchups,
    // })

}

func roundToOneDecimal(num float64) float64 {
    return math.Round(num*10) / 10
}
