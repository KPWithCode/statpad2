package nbahandler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/labstack/echo/v4"
)

type TLPlayerStats struct {
    Player struct {
        ID               int    `json:"id"`
        FirstName        string `json:"firstName"`
        LastName         string `json:"lastName"`
        PrimaryPosition  string `json:"primaryPosition"`
		OfficialImageSrc string `json:"officialImageSrc"`
        CurrentTeam      struct {
            ID           int    `json:"id"`
            Abbreviation string `json:"abbreviation"`
        } `json:"currentTeam"`
    } `json:"player"`
    Stats struct {
        GamesPlayed    int     `json:"gamesPlayed"`
        Offense struct {
            Pts         int     `json:"pts"`
            PtsPerGame  float64 `json:"ptsPerGame"`
            Ast         int     `json:"ast"`
            AstPerGame  float64 `json:"astPerGame"`
        } `json:"offense"`
        Rebounds struct {
            Reb         int     `json:"reb"`
            RebPerGame  float64 `json:"rebPerGame"`
        } `json:"rebounds"`
        Defense struct {
            Blk         int     `json:"blk"`
            BlkPerGame  float64 `json:"blkPerGame"`
            Stl         int     `json:"stl"`
            StlPerGame  float64 `json:"stlPerGame"`
            Tov         int     `json:"tov"`
            TovPerGame  float64 `json:"tovPerGame"`
        } `json:"defense"`
        FieldGoals struct {
            Fg3PtPct    float64 `json:"fg3PtPct"`
        } `json:"fieldGoals"`
        Miscellaneous struct {
            PlusMinus   int     `json:"plusMinus"`
        } `json:"miscellaneous"`
    } `json:"stats"`
}

type TLPlayerStatsResponse struct {
	PlayerStatsTotals []TLPlayerStats `json:"playerStatsTotals"`
}

type RateLimitedClient struct {
    apiKey          string
    lastRequestTime map[string]time.Time
    requestCount    int
    minuteStart     time.Time
    mu             sync.Mutex
}

func NewRateLimitedClient(apiKey string) *RateLimitedClient {
    return &RateLimitedClient{
        apiKey:          apiKey,
        lastRequestTime: make(map[string]time.Time),
        minuteStart:     time.Now(),
    }
}

func (c *RateLimitedClient) DoRequest(endpoint string, backoffSeconds int) ([]byte, error) {
    c.mu.Lock()
    
    // Check if we need to reset the minute counter
    now := time.Now()
    if now.Sub(c.minuteStart) >= time.Minute {
        c.requestCount = 0
        c.minuteStart = now
    }
    
    // Calculate new request cost (1 + backoff seconds)
    newCost := 1 + backoffSeconds
    
    // Check if this would exceed our per-minute limit
    if c.requestCount + newCost > 100 {
        c.mu.Unlock()
        return nil, fmt.Errorf("rate limit would be exceeded (current count: %d, new cost: %d)", 
            c.requestCount, newCost)
    }
    
    // Check if we need to wait for backoff
    if backoffSeconds > 0 {
        if lastRequest, exists := c.lastRequestTime[endpoint]; exists {
            timeSinceLastRequest := now.Sub(lastRequest)
            if timeSinceLastRequest < time.Duration(backoffSeconds)*time.Second {
                waitTime := time.Duration(backoffSeconds)*time.Second - timeSinceLastRequest
                c.mu.Unlock()
                time.Sleep(waitTime)
                c.mu.Lock()
            }
        }
    }
    
    // Update counters and timestamps
    c.requestCount += newCost
    c.lastRequestTime[endpoint] = time.Now()
    c.mu.Unlock()

    // Make the actual request
    req, err := http.NewRequest("GET", endpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("error creating request: %v", err)
    }

    req.SetBasicAuth(c.apiKey, "MYSPORTSFEEDS")

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error making request: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API returned non-200 status code: %d, body: %s",
            resp.StatusCode, string(bodyBytes))
    }

    return io.ReadAll(resp.Body)
}

func fetchTodaysScheduleIII() ([]string, error) {
    apiKey := os.Getenv("MYSPORTSFEEDS_API_KEY")
    if apiKey == "" {
        return nil, fmt.Errorf("API key not found in environment variables")
    }

    client := NewRateLimitedClient(apiKey)
    
    yesterday := time.Now().AddDate(0, 0, -1).Format("20060102")
    endpoint := fmt.Sprintf("https://api.mysportsfeeds.com/v2.1/pull/nba/2024-2025-regular/games.json?date=%s", yesterday)

    // Games endpoint doesn't require backoff
    data, err := client.DoRequest(endpoint, 0)
    if err != nil {
        return nil, err
    }

    var response struct {
        Games []struct {
            Schedule struct {
                AwayTeam struct {
                    Abbreviation string `json:"abbreviation"`
                } `json:"awayTeam"`
                HomeTeam struct {
                    Abbreviation string `json:"abbreviation"`
                } `json:"homeTeam"`
            } `json:"schedule"`
        } `json:"games"`
    }

    if err := json.Unmarshal(data, &response); err != nil {
        return nil, fmt.Errorf("error parsing JSON: %v", err)
    }

    var teams []string
    for _, game := range response.Games {
        teams = append(teams, game.Schedule.AwayTeam.Abbreviation, game.Schedule.HomeTeam.Abbreviation)
    }
    
    return teams, nil
}

func TrendLensHandler(c echo.Context) error {
	// Get today's games
	schedule, err := fetchTodaysScheduleIII()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Error fetching schedule: %v", err),
		})
	}

	// Create unique team list and join with commas
	teamsMap := make(map[string]bool)
	for _, team := range schedule {
		teamsMap[strings.ToLower(team)] = true
	}
	
	var teams []string
	for team := range teamsMap {
		teams = append(teams, team)
	}
	teamsList := strings.Join(teams, ",")

	// Calculate date range
	lastMonth := time.Now().AddDate(0, -1, 0).Format("20060102")
	today := time.Now().Format("20060102")

	// Fetch player stats from MySportsFeeds
	playerStats, err := fetchTLStats(lastMonth, today, teamsList)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Error fetching player stats: %v", err),
		})
	}

	// Initialize Algolia client
	algoliaAppID := os.Getenv("ALGOLIA_APP_ID")
	algoliaAPIKey := os.Getenv("ALGOLIA_API_KEY")
	algoliaIndexName := os.Getenv("ALGOLIA_INDEX_NAME")

	if algoliaAppID == "" || algoliaAPIKey == "" || algoliaIndexName == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Missing Algolia credentials",
		})
	}

	client,err := search.NewClient(algoliaAppID, algoliaAPIKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to initialize Algolia client: %v", err),
		})
	}
	// index, err := client.SearchSingleIndex(
	// 	client.NewApiSearchSingleIndexRequest(algoliaIndexName),
	// )
	// if err != nil {
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{
	// 		"error": fmt.Sprintf("Failed to initialize Algolia index: %v", err),
	// 	})
	// }
    var batchRequests []search.BatchRequest
    for _, stats := range playerStats.PlayerStatsTotals {
        batchRequests = append(batchRequests, *search.NewEmptyBatchRequest().
            // SetAction("updateObject")
			SetAction(search.Action("updateObject")).
            SetBody(map[string]interface{}{
                "objectID":      fmt.Sprintf("player_%d", stats.Player.ID),
                "playerID":      stats.Player.ID,
                "firstName":     stats.Player.FirstName,
                "lastName":      stats.Player.LastName,
                "fullName":      fmt.Sprintf("%s %s", stats.Player.FirstName, stats.Player.LastName),
                "position":      stats.Player.PrimaryPosition,
                "teamID":        stats.Player.CurrentTeam.ID,
                "teamAbbrev":    stats.Player.CurrentTeam.Abbreviation,
				"officialImageSrc": stats.Player.OfficialImageSrc,
                "gamesPlayed":   stats.Stats.GamesPlayed,
                "points":        stats.Stats.Offense.Pts,
                "rebounds":      stats.Stats.Rebounds,
                "assists":       stats.Stats.Offense.Ast,
				"pointsPerGame": stats.Stats.Offense.PtsPerGame,
				"rebPerGame":    stats.Stats.Rebounds.RebPerGame,
				"astPerGame":    stats.Stats.Offense.AstPerGame,
				"blkPerGame":    stats.Stats.Defense.BlkPerGame,
				"stlPerGame":    stats.Stats.Defense.StlPerGame,
				"tovPerGame":    stats.Stats.Defense.TovPerGame,
				"fg3ptPct":      stats.Stats.FieldGoals.Fg3PtPct,
				"plusMinus":     stats.Stats.Miscellaneous.PlusMinus,
                "lastUpdated":   time.Now().Format(time.RFC3339),
            }))
    }

    // Perform batch update
    response, err := client.Batch(client.NewApiBatchRequest(
        algoliaIndexName,
        search.NewEmptyBatchWriteParams().SetRequests(batchRequests),
    ))
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": fmt.Sprintf("Failed to save to Algolia: %v", err),
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
		"status":       "success",
		"recordsSaved": len(batchRequests),
		"taskID": response.TaskID,
		"date":         time.Now().Format(time.RFC3339),
	})
}

func fetchTLStats(lastMonth, today, teamsList string) (*TLPlayerStatsResponse, error) {
	apiKey := os.Getenv("MYSPORTSFEEDS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("MySportsFeeds API key not found")
	}

	endpoint := fmt.Sprintf(
		"https://api.mysportsfeeds.com/v2.1/pull/nba/2024-2025-regular/player_stats_totals.json?date=%s-%s&team=%s",
		lastMonth,
		today,
		teamsList,
	)

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

	var response TLPlayerStatsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return &response, nil
}