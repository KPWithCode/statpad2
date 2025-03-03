package handlers

import (
	"encoding/csv"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/labstack/echo/v4"
)

// ShotData represents individual shot data from the CSV
type ShotData struct {
	ShotID              string    `json:"shotID"`
	GameID              string    `json:"game_id"`
	ShooterName         string    `json:"shooterName"`
	ShooterPlayerID     string    `json:"shooterPlayerId"`
	GoalieNameForShot   string    `json:"goalieNameForShot"`
	GoalieIDForShot     string    `json:"goalieIdForShot"`
	TeamCode            string    `json:"teamCode"`
	HomeTeamCode        string    `json:"homeTeamCode"`
	AwayTeamCode        string    `json:"awayTeamCode"`
	ShotDistance        float64   `json:"shotDistance"`
	ShotAngle           float64   `json:"shotAngle"`
	ShotType            string    `json:"shotType"`
	Goal                bool      `json:"goal"`
	XGoal               float64   `json:"xGoal"`
	ShotRush            bool      `json:"shotRush"`
	ShotWasOnGoal       bool      `json:"shotWasOnGoal"`
	ShotOnEmptyNet      bool      `json:"shotOnEmptyNet"`
	Period              int       `json:"period"`
	TimeLeft            float64   `json:"timeLeft"`
	ShotGoalProbability float64   `json:"shotGoalProbability"`
	HomeSkatersOnIce    int       `json:"homeSkatersOnIce"`
	AwaySkatersOnIce    int       `json:"awaySkatersOnIce"`
	PlayerPosition      string    `json:"playerPositionThatDidEvent"`
	ShooterTimeOnIce    float64   `json:"shooterTimeOnIce"`
	Time                string    `json:"time"` // Time field for date filtering
	Date                time.Time // Parsed time for filtering
}

// PlayerStats aggregates shot data for a player
type PlayerStats struct {
	PlayerID        string  `json:"playerId"`
	PlayerName      string  `json:"playerName"`
	Position        string  `json:"position"`
	TeamCode        string  `json:"teamCode"`
	GamesPlayed     int     `json:"gamesPlayed"`
	ShotsAttempted  int     `json:"shotsAttempted"`
	ShotsOnGoal     int     `json:"shotsOnGoal"`
	Goals           int     `json:"goals"`
	RushShots       int     `json:"rushShots"`
	EmptyNetGoals   int     `json:"emptyNetGoals"`
	AverageDistance float64 `json:"averageDistance"`
	AverageAngle    float64 `json:"averageAngle"`
	TotalXGoals     float64 `json:"totalXGoals"`
	TimeOnIce       float64 `json:"timeOnIce"`
	PowerPlayShots  int     `json:"powerPlayShots"`
	PowerPlayGoals  int     `json:"powerPlayGoals"`
	HighDangerShots int     `json:"highDangerShots"`
	HighDangerGoals int     `json:"highDangerGoals"`
}

// Helper function to check if a shot is from a high-danger area
//
//	func isShotHighDanger(shotDistance float64, shotAngle float64) bool {
//		// Define high danger shots based on distance and angle
//		// Typically shots from slot area (less than 25 feet, angles less than 45 degrees)
//		return shotDistance < 25 && math.Abs(shotAngle) < 45
//	}
func isShotHighDanger(shotDistance float64, shotAngle float64, shotType string) bool {
	shotType = strings.ToLower(shotType)

	switch {
	case shotDistance <= 20 && math.Abs(shotAngle) <= 45:
		// Inner slot area - highest danger
		return true
	case shotDistance <= 30 && math.Abs(shotAngle) <= 35:
		// Moderate distance but good angle
		return true
	case (shotType == "tip" || shotType == "deflection") && shotDistance <= 25:
		// Tips and deflections are dangerous from slightly further out
		return true
	default:
		return false
	}
}

// Helper function to check if a shot was during power play
func isPowerPlayShot(homeSkatersOnIce, awaySkatersOnIce int, isHomeTeam bool) bool {
	if isHomeTeam {
		return homeSkatersOnIce > awaySkatersOnIce
	}
	return awaySkatersOnIce > homeSkatersOnIce
}

// Parse string to float with default value if error
func parseFloat(s string, defaultVal float64) float64 {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

// Parse string to int with default value if error
func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

// Parse string to bool with default value if error
func parseBool(s string, defaultVal bool) bool {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(s)
	if err != nil {
		// Try numeric conversion (1=true, 0=false)
		numVal, numErr := strconv.ParseInt(s, 10, 64)
		if numErr != nil {
			return defaultVal
		}
		return numVal != 0
	}
	return val
}

// Calculate advanced metrics for a player
func calculateAdvancedMetrics(playerStats PlayerStats) map[string]float64 {
	metrics := make(map[string]float64)

	// Round helper function
	roundToOneDecimal := func(val float64) float64 {
		return math.Round(val*10) / 10
	}

	// Shooting percentage
	if playerStats.ShotsOnGoal > 0 {
		shootingPct := float64(playerStats.Goals) / float64(playerStats.ShotsOnGoal) * 100
		metrics["shootingPct"] = roundToOneDecimal(shootingPct)
	} else {
		metrics["shootingPct"] = 0.0
	}

	if playerStats.GamesPlayed > 0 {
		shotsPerGame := float64(playerStats.ShotsAttempted) / float64(playerStats.GamesPlayed)
		metrics["shotsPerGame"] = roundToOneDecimal(shotsPerGame)
	} else {
		metrics["shotsPerGame"] = 0.0
	}

	// Shots on goal per game
	if playerStats.GamesPlayed > 0 {
		sogPerGame := float64(playerStats.ShotsOnGoal) / float64(playerStats.GamesPlayed)
		metrics["shotsOnGoalPerGame"] = roundToOneDecimal(sogPerGame)
	} else {
		metrics["shotsOnGoalPerGame"] = 0.0
	}
	// Shot on goal percentage
	// if playerStats.ShotsAttempted > 0 {
	// 	sogPct := float64(playerStats.ShotsOnGoal) / float64(playerStats.ShotsAttempted) * 100
	// 	metrics["shotOnGoalPct"] = roundToOneDecimal(sogPct)
	// } else {
	// 	metrics["shotOnGoalPct"] = 0.0
	// }

	// Expected goals per shot
	if playerStats.ShotsAttempted > 0 {
		xgPerShot := playerStats.TotalXGoals / float64(playerStats.ShotsAttempted)
		metrics["xGoalsPerShot"] = roundToOneDecimal(xgPerShot)
	} else {
		metrics["xGoalsPerShot"] = 0.0
	}

	// Goals per game
	if playerStats.GamesPlayed > 0 {
		goalsPerGame := float64(playerStats.Goals) / float64(playerStats.GamesPlayed)
		metrics["goalsPerGame"] = roundToOneDecimal(goalsPerGame)
	} else {
		metrics["goalsPerGame"] = 0.0
	}

	// xGoals per game
	if playerStats.GamesPlayed > 0 {
		xGoalsPerGame := playerStats.TotalXGoals / float64(playerStats.GamesPlayed)
		metrics["xGoalsPerGame"] = roundToOneDecimal(xGoalsPerGame)
	} else {
		metrics["xGoalsPerGame"] = 0.0
	}

	// Rush shooting percentage
	if playerStats.RushShots > 0 {
		rushShotPct := float64(playerStats.Goals) / float64(playerStats.RushShots) * 100
		metrics["rushShootingPct"] = roundToOneDecimal(rushShotPct)
	} else {
		metrics["rushShootingPct"] = 0.0
	}

	// Goals minus expected goals (finishing ability)
	metrics["goalsAboveExpected"] = roundToOneDecimal(float64(playerStats.Goals) - playerStats.TotalXGoals)

	// High danger shooting percentage
	if playerStats.HighDangerShots > 0 {
		hdShotPct := float64(playerStats.HighDangerShots) / float64(playerStats.ShotsAttempted) * 100
		metrics["highDangerShotPct"] = roundToOneDecimal(hdShotPct)
	} else {
		metrics["highDangerShotPct"] = 0.0
	}

	// High danger goal percentage
	if playerStats.HighDangerGoals > 0 {
		hdGoalPct := float64(playerStats.HighDangerGoals) / float64(playerStats.HighDangerShots) * 100
		metrics["highDangerGoalPct"] = roundToOneDecimal(hdGoalPct)
	} else {
		metrics["highDangerGoalPct"] = 0.0
	}

	// Power play shooting percentage
	if playerStats.PowerPlayShots > 0 {
		ppShootingPct := float64(playerStats.PowerPlayGoals) / float64(playerStats.PowerPlayShots) * 100
		metrics["powerPlayShootingPct"] = roundToOneDecimal(ppShootingPct)
	} else {
		metrics["powerPlayShootingPct"] = 0.0
	}

	// Hockey Card Rating (similar to simplified PER for NBA)
	// This creates a single number that represents shooting effectiveness
	if playerStats.ShotsAttempted > 0 {
		// Basic formula that weighs goals, high-danger efficiency, and shooting above expected
		hockeyCardRating := float64(playerStats.Goals)*1.0 +
			float64(playerStats.HighDangerGoals)*0.5 +
			(float64(playerStats.Goals)-playerStats.TotalXGoals)*2.0

		metrics["hockeyCardRating"] = roundToOneDecimal(hockeyCardRating)

		if playerStats.GamesPlayed > 0 {
			metrics["hockeyCardRatingPerGame"] = roundToOneDecimal(hockeyCardRating / float64(playerStats.GamesPlayed))
		} else {
			metrics["hockeyCardRatingPerGame"] = 0.0
		}
	} else {
		metrics["hockeyCardRating"] = 0.0
		metrics["hockeyCardRatingPerGame"] = 0.0
	}

	return metrics
}
func processCSVToShotData(csvData [][]string, headers []string) ([]ShotData, error) {
	var shotData []ShotData

	// Create map of header names to indices
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[header] = i
	}

	// Process each row in the CSV
	for i := 1; i < len(csvData); i++ {
		row := csvData[i]
		if len(row) != len(headers) {
			continue // Skip malformed rows
		}

		// Helper function to get value at a specific column
		getValue := func(header string) string {
			if idx, ok := headerMap[header]; ok && idx < len(row) {
				return row[idx]
			}
			return ""
		}

		// Check if this is truly a player shot (has player ID, name, etc.)
		playerID := getValue("shooterPlayerId")
		playerName := getValue("shooterName")
		if playerID == "" || playerName == "" {
			// Skip rows that don't have clear player attribution
			continue
		}

		// Determine if it's a shot on goal (must be explicitly SHOT event)
		eventType := getValue("event")
		isShotOnGoal := eventType == "SHOT"
		isGoal := parseBool(getValue("goal"), false)

		// Get time for date filtering
		timeStr := getValue("time")

		// Construct shot data from CSV row
		shot := ShotData{
			ShotID:              getValue("shotID"),
			GameID:              getValue("game_id"),
			ShooterName:         playerName,
			ShooterPlayerID:     playerID,
			GoalieNameForShot:   getValue("goalieNameForShot"),
			GoalieIDForShot:     getValue("goalieIdForShot"),
			TeamCode:            getValue("teamCode"),
			HomeTeamCode:        getValue("homeTeamCode"),
			AwayTeamCode:        getValue("awayTeamCode"),
			ShotDistance:        parseFloat(getValue("shotDistance"), 0),
			ShotAngle:           parseFloat(getValue("shotAngle"), 0),
			ShotType:            getValue("shotType"),
			Goal:                isGoal,
			XGoal:               parseFloat(getValue("xGoal"), 0),
			ShotRush:            parseBool(getValue("shotRush"), false),
			ShotWasOnGoal:       isShotOnGoal,
			ShotOnEmptyNet:      parseBool(getValue("shotOnEmptyNet"), false),
			Period:              parseInt(getValue("period"), 0),
			TimeLeft:            parseFloat(getValue("timeLeft"), 0),
			ShotGoalProbability: parseFloat(getValue("shotGoalProbability"), 0),
			HomeSkatersOnIce:    parseInt(getValue("homeSkatersOnIce"), 5),
			AwaySkatersOnIce:    parseInt(getValue("awaySkatersOnIce"), 5),
			PlayerPosition:      getValue("playerPositionThatDidEvent"),
			ShooterTimeOnIce:    parseFloat(getValue("shooterTimeOnIce"), 0),
			Time:                timeStr,
		}

		// Try to parse the time if available
		if timeStr != "" {
			// Attempt to parse time in various formats
			formats := []string{
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05",
				"01/02/2006 15:04:05",
				"01/02/2006",
				"2006-01-02",
			}

			for _, format := range formats {
				if t, err := time.Parse(format, timeStr); err == nil {
					shot.Date = t
					break
				}
			}
		}

		shotData = append(shotData, shot)
	}

	return shotData, nil
}

// }
// Update the aggregatePlayerStats function to only count player shots
func debugShotData(shotData []ShotData) {
	playerShots := 0
	teamShots := 0
	sotsOnGoal := 0

	// Count different types of shots
	for _, shot := range shotData {
		if shot.ShooterPlayerID != "" {
			playerShots++
			if shot.ShotWasOnGoal {
				sotsOnGoal++
			}
		} else {
			teamShots++
		}
	}

	fmt.Printf("Data Summary:\n")
	fmt.Printf("Total Shots: %d\n", len(shotData))
	fmt.Printf("Player Shots: %d\n", playerShots)
	fmt.Printf("Team Shots: %d\n", teamShots)
	fmt.Printf("Shots On Goal: %d\n", sotsOnGoal)

	// Sample some players to check their stats
	playerShotCounts := make(map[string]int)
	playerSOG := make(map[string]int)
	playerNames := make(map[string]string)

	for _, shot := range shotData {
		if shot.ShooterPlayerID != "" {
			playerShotCounts[shot.ShooterPlayerID]++
			playerNames[shot.ShooterPlayerID] = shot.ShooterName
			if shot.ShotWasOnGoal {
				playerSOG[shot.ShooterPlayerID]++
			}
		}
	}

	fmt.Printf("\nPlayer Shot Samples:\n")
	count := 0
	for id, name := range playerNames {
		if count >= 5 {
			break
		}
		fmt.Printf("%s (%s): Total Shots: %d, SOG: %d\n",
			name, id, playerShotCounts[id], playerSOG[id])
		count++
	}
}
func aggregatePlayerStats(shotData []ShotData) map[string]PlayerStats {
	// Map to store player stats
	playerStats := make(map[string]PlayerStats)

	// Map to track games played by each player
	playerGames := make(map[string]map[string]bool)

	// Process each shot
	for _, shot := range shotData {
		playerId := shot.ShooterPlayerID

		// Skip if player ID is missing
		if playerId == "" {
			continue
		}

		// Initialize player maps if needed
		if _, exists := playerStats[playerId]; !exists {
			playerStats[playerId] = PlayerStats{
				PlayerID:   playerId,
				PlayerName: shot.ShooterName,
				Position:   shot.PlayerPosition,
				TeamCode:   shot.TeamCode,
			}
		}

		// Initialize games set if needed
		if _, exists := playerGames[playerId]; !exists {
			playerGames[playerId] = make(map[string]bool)
		}

		// Add game to player's games played
		playerGames[playerId][shot.GameID] = true

		// Update player stats
		stats := playerStats[playerId]

		stats.ShotsAttempted++

		// Count as shot on goal ONLY if it's explicitly marked as a shot on goal
		if shot.ShotWasOnGoal {
			stats.ShotsOnGoal++
		}

		if shot.ShotOnEmptyNet && shot.ShotWasOnGoal {
			// If a shot is on an empty net and on goal, it should be a goal
			stats.Goals++
			stats.EmptyNetGoals++
			// We already counted it as a shot on goal above
		} else if shot.Goal {
			// For regular goals
			stats.Goals++

			// Ensure all goals are counted as shots on goal
			if !shot.ShotWasOnGoal {
				stats.ShotsOnGoal++
			}

			// Check again for empty net goals that might not have been caught in the first condition
			if shot.ShotOnEmptyNet {
				stats.EmptyNetGoals++
			}
		}

		// Rest of function remains the same
		if shot.ShotRush {
			stats.RushShots++
		}

		// Check for high danger shot
		if isShotHighDanger(shot.ShotDistance, shot.ShotAngle, shot.ShotType) {
			stats.HighDangerShots++
			if shot.Goal {
				stats.HighDangerGoals++
			}
		}

		// Check for power play
		isHomeTeam := shot.TeamCode == shot.HomeTeamCode
		if isPowerPlayShot(shot.HomeSkatersOnIce, shot.AwaySkatersOnIce, isHomeTeam) {
			stats.PowerPlayShots++
			if shot.Goal {
				stats.PowerPlayGoals++
			}
		}

		// Update shot details
		stats.AverageDistance = ((stats.AverageDistance * float64(stats.ShotsAttempted-1)) + shot.ShotDistance) / float64(stats.ShotsAttempted)
		stats.AverageAngle = ((stats.AverageAngle * float64(stats.ShotsAttempted-1)) + math.Abs(shot.ShotAngle)) / float64(stats.ShotsAttempted)
		stats.TotalXGoals += shot.XGoal
		stats.TimeOnIce += shot.ShooterTimeOnIce

		// Update the player stats in the map
		playerStats[playerId] = stats
	}

	// Update games played for each player and debug output
	fmt.Println("\nPlayer stats before division by games played:")
	for playerId, games := range playerGames {
		if stats, exists := playerStats[playerId]; exists {
			stats.GamesPlayed = len(games)

			if stats.PlayerName == "Jakob Chychrun" {
				fmt.Printf("Jakob Chychrun (before): Games: %d, SOG: %d, SOG/Game: %f\n",
					stats.GamesPlayed, stats.ShotsOnGoal,
					float64(stats.ShotsOnGoal)/float64(stats.GamesPlayed))
			}

			playerStats[playerId] = stats
		}
	}

	return playerStats
}

// Aggregate shots by game
func aggregateGameStats(shotData []ShotData) map[string]map[string]interface{} {
	gameStats := make(map[string]map[string]interface{})

	// Group shots by game
	gameShots := make(map[string][]ShotData)
	for _, shot := range shotData {
		gameID := shot.GameID
		gameShots[gameID] = append(gameShots[gameID], shot)
	}

	// Process each game
	for gameID, shots := range gameShots {
		// Initialize game stats
		stats := make(map[string]interface{})

		// Basic game info
		if len(shots) > 0 {
			stats["homeTeam"] = shots[0].HomeTeamCode
			stats["awayTeam"] = shots[0].AwayTeamCode
		}

		// Count shots and goals
		totalShots := len(shots)
		totalGoals := 0
		totalXGoals := 0.0

		homeShots := 0
		homeGoals := 0
		homeXGoals := 0.0

		awayShots := 0
		awayGoals := 0
		awayXGoals := 0.0

		for _, shot := range shots {
			// Add to totals
			if shot.Goal {
				totalGoals++
			}
			totalXGoals += shot.XGoal

			// Add to team-specific counts
			isHomeTeam := shot.TeamCode == shot.HomeTeamCode
			if isHomeTeam {
				homeShots++
				if shot.Goal {
					homeGoals++
				}
				homeXGoals += shot.XGoal
			} else {
				awayShots++
				if shot.Goal {
					awayGoals++
				}
				awayXGoals += shot.XGoal
			}
		}

		// Store statistics
		stats["totalShots"] = totalShots
		stats["totalGoals"] = totalGoals
		stats["totalXGoals"] = math.Round(totalXGoals*100) / 100

		stats["homeShots"] = homeShots
		stats["homeGoals"] = homeGoals
		stats["homeXGoals"] = math.Round(homeXGoals*100) / 100

		stats["awayShots"] = awayShots
		stats["awayGoals"] = awayGoals
		stats["awayXGoals"] = math.Round(awayXGoals*100) / 100

		// Calculate game pace (shots per minute)
		stats["gamePace"] = math.Round((float64(totalShots)/60.0)*10) / 10

		// Calculate xG share
		if homeXGoals+awayXGoals > 0 {
			homeXGShare := homeXGoals / (homeXGoals + awayXGoals)
			stats["homeXGShare"] = math.Round(homeXGShare*1000) / 1000
		}

		gameStats[gameID] = stats
	}

	return gameStats
}

// Main NHL Trend Lens handler for local CSV data
func NHLTrendLensHandler(c echo.Context) error {
	// Open and read CSV file
	file, err := os.Open("data/march3.csv") // Using specific file as requested
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

	// Get headers from the first row
	headers := records[0]

	// Check required columns
	eventIdx := findColumnIndex(headers, "event")
	timeIdx := findColumnIndex(headers, "time")
	teamIdx := findColumnIndex(headers, "teamCode")
	seasonIdx := findColumnIndex(headers, "season")
	gameIDIdx := findColumnIndex(headers, "game_id")

	if eventIdx == -1 || timeIdx == -1 || teamIdx == -1 || seasonIdx == -1 || gameIDIdx == -1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSV does not contain required columns"})
	}

	// Process CSV data into shot data
	shotData, err := processCSVToShotData(records, headers)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Error processing shot data: %v", err),
		})
	}
	debugShotData(shotData)
	// Calculate date thresholds for recent data
	monthAgo := time.Now().AddDate(0, 0, -30) // Filter for last two weeks

	// Filter shots for recent period (last 2 weeks)
	var recentShotData []ShotData

	// Two approaches to filtering:
	// 1. If we have valid dates in the shotData, use those
	// 2. If not, we can filter based on a percentage of the data (e.g., most recent 25%)

	hasValidDates := false
	for _, shot := range shotData {
		if !shot.Date.IsZero() {
			hasValidDates = true
			break
		}
	}

	if hasValidDates {
		// Filter by actual dates
		for _, shot := range shotData {
			if !shot.Date.IsZero() && shot.Date.After(monthAgo) {
				recentShotData = append(recentShotData, shot)
			}
		}
		fmt.Printf("Filtered %d shots as recent (last 2 weeks) out of %d total\n",
			len(recentShotData), len(shotData))
	} else {
		// If date parsing didn't work, use the most recent 1 month of data by factoring in the length of season and dividing
		// Sort shots by some indicator of recency if available
		// Use approximately 1 month of data (16.7% of season)
		// Dividing by 6 instead of 4 will give you roughly one month's worth. 6.2 is 16.7 exactly
		recentCount := int(float64(len(shotData)) / 6.2)
		if recentCount > 0 {
			startIdx := len(shotData) - recentCount
			recentShotData = shotData[startIdx:]
		} else {
			recentShotData = shotData // Use all data if very small dataset
		}
		fmt.Printf("Using last %d shots as recent period (16%%) since date filtering unavailable\n",
			len(recentShotData))
	}

	// Aggregate player stats
	playerStats := aggregatePlayerStats(shotData)
	recentPlayerStats := aggregatePlayerStats(recentShotData)

	// Aggregate game stats
	gameStats := aggregateGameStats(shotData)

	// Initialize Algolia client
	algoliaAppID := os.Getenv("ALGOLIA_APP_ID")
	algoliaAPIKey := os.Getenv("ALGOLIA_API_KEY")
	algoliaIndexName := os.Getenv("ALGOLIA_NHL_INDEX_NAME")

	if algoliaAppID == "" || algoliaAPIKey == "" || algoliaIndexName == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Missing Algolia credentials",
		})
	}

	client, err := search.NewClient(algoliaAppID, algoliaAPIKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to initialize Algolia client: %v", err),
		})
	}

	// Create batch requests for players
	var batchRequests []search.BatchRequest

	// Helper function to round to one decimal
	roundToOneDecimal := func(val float64) float64 {
		return math.Round(val*10) / 10
	}

	// Process all players
	for playerID, stats := range playerStats {
		// Skip players with minimal data
		if stats.ShotsAttempted < 5 {
			continue
		}

		// Calculate advanced metrics
		metrics := calculateAdvancedMetrics(stats)

		// Basic player info and stats
		batchBody := map[string]interface{}{
			"objectID":    fmt.Sprintf("nhl_player_%s", playerID),
			"playerID":    playerID,
			"playerName":  stats.PlayerName,
			"position":    stats.Position,
			"teamCode":    stats.TeamCode,
			"gamesPlayed": stats.GamesPlayed,

			// Shooting stats
			"shotsAttempted":     stats.ShotsAttempted,
			"shotsOnGoal":        stats.ShotsOnGoal,
			"goals":              stats.Goals,
			"emptyNetGoals":      stats.EmptyNetGoals,
			"shotsPerGame":       roundToOneDecimal(float64(stats.ShotsAttempted) / float64(stats.GamesPlayed)),
			"shotsOnGoalPerGame": roundToOneDecimal(float64(stats.ShotsOnGoal) / float64(stats.GamesPlayed)),
			"goalsPerGame":       roundToOneDecimal(float64(stats.Goals) / float64(stats.GamesPlayed)),

			// Shot quality metrics
			"averageDistance": roundToOneDecimal(stats.AverageDistance),
			"averageAngle":    roundToOneDecimal(stats.AverageAngle),
			"totalXGoals":     roundToOneDecimal(stats.TotalXGoals),
			"xGoalsPerGame":   roundToOneDecimal(stats.TotalXGoals / float64(stats.GamesPlayed)),

			// Shooting percentages
			"shootingPct":   metrics["shootingPct"],
			"shotOnGoalPct": metrics["shotOnGoalPct"],
			"xGoalsPerShot": metrics["xGoalsPerShot"],

			// Special situations
			"rushShots":         stats.RushShots,
			"rushShotPct":       roundToOneDecimal(float64(stats.RushShots) / float64(stats.ShotsAttempted) * 100),
			"highDangerShots":   stats.HighDangerShots,
			"highDangerGoals":   stats.HighDangerGoals,
			"highDangerShotPct": metrics["highDangerShotPct"],
			"highDangerGoalPct": metrics["highDangerGoalPct"],
			"powerPlayShots":    stats.PowerPlayShots,
			"powerPlayGoals":    stats.PowerPlayGoals,
			"powerPlayShotPct":  metrics["powerPlayShootingPct"],

			// Advanced metrics
			"goalsAboveExpected":      metrics["goalsAboveExpected"],
			"hockeyCardRating":        metrics["hockeyCardRating"],
			"hockeyCardRatingPerGame": metrics["hockeyCardRatingPerGame"],

			// Metadata
			"lastUpdated": time.Now().Format(time.RFC3339),
		}

		// Add recent period stats if they exist
		if recentStats, exists := recentPlayerStats[playerID]; exists && recentStats.GamesPlayed > 0 {
			recentMetrics := calculateAdvancedMetrics(recentStats)

			recentStatsMap := map[string]interface{}{
				"recentGamesPlayed":        recentStats.GamesPlayed,
				"recentShotsAttempted":     recentStats.ShotsAttempted,
				"recentShotsOnGoal":        recentStats.ShotsOnGoal,
				"recentGoals":              recentStats.Goals,
				"recentShotsPerGame":       roundToOneDecimal(float64(recentStats.ShotsAttempted) / float64(recentStats.GamesPlayed)),
				"recentShotsOnGoalPerGame": roundToOneDecimal(float64(recentStats.ShotsOnGoal) / float64(recentStats.GamesPlayed)),
				"recentGoalsPerGame":       roundToOneDecimal(float64(recentStats.Goals) / float64(recentStats.GamesPlayed)),
				"recentShootingPct":        recentMetrics["shootingPct"],
				"recentXGoals":             roundToOneDecimal(recentStats.TotalXGoals),
				"recentXGoalsPerGame":      roundToOneDecimal(recentStats.TotalXGoals / float64(recentStats.GamesPlayed)),
				"recentGoalsAboveExpected": recentMetrics["goalsAboveExpected"],
				"recentHighDangerGoals":    recentStats.HighDangerGoals,
				"recentHighDangerShots":    recentStats.HighDangerShots,
				"recentPowerPlayGoals":     recentStats.PowerPlayGoals,
				"recentHockeyCardRating":   recentMetrics["hockeyCardRating"],
				"recentHighDangerShotPct":  recentMetrics["highDangerShotPct"],
				"recentHighDangerGoalPct":  recentMetrics["highDangerGoalPct"],
			}

			// Add trend indicators (comparing recent to overall performance)
			if stats.GamesPlayed > recentStats.GamesPlayed {
				// Calculate trending indicators
				goalsScoringTrend := (float64(recentStats.Goals) / float64(recentStats.GamesPlayed)) -
					(float64(stats.Goals-recentStats.Goals) / float64(stats.GamesPlayed-recentStats.GamesPlayed))

				// Only calculate shooting percentage trend if there are shots on goal
				var shootingPctTrend float64
				if recentStats.ShotsOnGoal > 0 && (stats.ShotsOnGoal-recentStats.ShotsOnGoal) > 0 {
					shootingPctTrend = recentMetrics["shootingPct"] -
						((float64(stats.Goals-recentStats.Goals) / float64(stats.ShotsOnGoal-recentStats.ShotsOnGoal)) * 100)
				}

				xGoalsTrend := (recentStats.TotalXGoals / float64(recentStats.GamesPlayed)) -
					((stats.TotalXGoals - recentStats.TotalXGoals) / float64(stats.GamesPlayed-recentStats.GamesPlayed))

				recentStatsMap["goalsScoringTrend"] = roundToOneDecimal(goalsScoringTrend)
				recentStatsMap["shootingPctTrend"] = roundToOneDecimal(shootingPctTrend)
				recentStatsMap["xGoalsTrend"] = roundToOneDecimal(xGoalsTrend)

				// Add high danger shooting trends
				if recentStats.HighDangerShots > 0 && (stats.HighDangerShots-recentStats.HighDangerShots) > 0 {
					highDangerScoringTrend := (float64(recentStats.HighDangerGoals) / float64(recentStats.HighDangerShots)) -
						(float64(stats.HighDangerGoals-recentStats.HighDangerGoals) /
							float64(stats.HighDangerShots-recentStats.HighDangerShots))
					recentStatsMap["recentHighDangerScoringTrend"] = roundToOneDecimal(highDangerScoringTrend * 100)
				}

				// Add rush shot trends
				if recentStats.RushShots > 0 && stats.RushShots > recentStats.RushShots {
					rushShotPctTrend := (float64(recentStats.RushShots) / float64(recentStats.ShotsAttempted)) -
						(float64(stats.RushShots-recentStats.RushShots) /
							float64(stats.ShotsAttempted-recentStats.ShotsAttempted))
					recentStatsMap["rushShotPctTrend"] = roundToOneDecimal(rushShotPctTrend * 100)
				}

				// Add power play trend
				if recentStats.PowerPlayShots > 0 && stats.PowerPlayShots > recentStats.PowerPlayShots {
					ppGoalsTrend := (float64(recentStats.PowerPlayGoals) / float64(recentStats.PowerPlayShots)) -
						(float64(stats.PowerPlayGoals-recentStats.PowerPlayGoals) /
							float64(stats.PowerPlayShots-recentStats.PowerPlayShots))
					recentStatsMap["powerPlayScoringTrend"] = roundToOneDecimal(ppGoalsTrend * 100)
				}

				// Add hockey card rating trend
				hockeyCardTrend := recentMetrics["hockeyCardRatingPerGame"] -
					((metrics["hockeyCardRating"] - recentMetrics["hockeyCardRating"]) /
						float64(stats.GamesPlayed-recentStats.GamesPlayed))
				recentStatsMap["hockeyCardTrend"] = roundToOneDecimal(hockeyCardTrend)
			}

			// Add recent stats to batch body
			for k, v := range recentStatsMap {
				batchBody[k] = v
			}
		}

		batchRequests = append(batchRequests, *search.NewEmptyBatchRequest().
			SetAction(search.Action("updateObject")).
			SetBody(batchBody))
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
		"playerCount":  len(playerStats),
		"gameCount":    len(gameStats),
		"recordsSaved": len(batchRequests),
		"taskID":       response.TaskID,
		"date":         time.Now().Format(time.RFC3339),
	})
}
