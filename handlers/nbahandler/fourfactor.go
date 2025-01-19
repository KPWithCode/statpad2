package nbahandler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

type TeamMyFeedsStatsResponse struct {
	TeamStatsTotals []TeamMyFeedStatsEntry `json:"teamStatsTotals"`
}

// TeamMyFeedStatsEntry represents each team's entry in the API response
type TeamMyFeedStatsEntry struct {
	Team  TeamInfo `json:"team"`
	Stats Stats    `json:"stats"`
}

// TeamInfo represents basic team information
type TeamInfo struct {
	ID           int           `json:"id"`
	City         string        `json:"city"`
	Name         string        `json:"name"`
	Abbreviation string        `json:"abbreviation"`
	HomeVenue    Venue         `json:"homeVenue"`
	SocialMedia  []SocialMedia `json:"socialMediaAccounts"`
}

type Venue struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type SocialMedia struct {
	MediaType string `json:"mediaType"`
	Value     string `json:"value"`
}

// Stats represents all statistical categories
type Stats struct {
	Standings     Standings     `json:"standings"`
	FieldGoals    FieldGoals    `json:"fieldGoals"`
	FreeThrows    FreeThrows    `json:"freeThrows"`
	Rebounds      Rebounds      `json:"rebounds"`
	Offense       Offense       `json:"offense"`
	Defense       Defense       `json:"defense"`
	Miscellaneous Miscellaneous `json:"miscellaneous"`
}

type FieldGoals struct {
	FGMade           float64 `json:"fgMade"`
	FGAtt            float64 `json:"fgAtt"`
	FGPct            float64 `json:"fgPct"`
	FG2PtMade        float64 `json:"fg2PtMade"`
	FG2PtAtt         float64 `json:"fg2PtAtt"`
	FG2PtPct         float64 `json:"fg2PtPct"`
	FG3PtMade        float64 `json:"fg3PtMade"`
	FG3PtAtt         float64 `json:"fg3PtAtt"`
	FG3PtPct         float64 `json:"fg3PtPct"`
	FGMadePerGame    float64 `json:"fgMadePerGame"`
	FGAttPerGame     float64 `json:"fgAttPerGame"`
	FG2PtMadePerGame float64 `json:"fg2PtMadePerGame"`
	FG2PtAttPerGame  float64 `json:"fg2PtAttPerGame"`
	FG3PtMadePerGame float64 `json:"fg3PtMadePerGame"`
	FG3PtAttPerGame  float64 `json:"fg3PtAttPerGame"`
}

type FreeThrows struct {
	FTMade        float64 `json:"ftMade"`
	FTAtt         float64 `json:"ftAtt"`
	FTPct         float64 `json:"ftPct"`
	FTMadePerGame float64 `json:"ftMadePerGame"`
	FTAttPerGame  float64 `json:"ftAttPerGame"`
}

type Rebounds struct {
	OffReb        float64 `json:"offReb"`
	DefReb        float64 `json:"defReb"`
	Reb           float64 `json:"reb"`
	OffRebPerGame float64 `json:"offRebPerGame"`
	DefRebPerGame float64 `json:"defRebPerGame"`
	RebPerGame    float64 `json:"rebPerGame"`
}

type Offense struct {
	Pts        float64 `json:"pts"`
	PtsPerGame float64 `json:"ptsPerGame"`
	Ast        float64 `json:"ast"`
	AstPerGame float64 `json:"astPerGame"`
}

type Defense struct {
	TOV               float64 `json:"tov"`
	TOVPerGame        float64 `json:"tovPerGame"`
	STL               float64 `json:"stl"`
	STLPerGame        float64 `json:"stlPerGame"`
	BLK               float64 `json:"blk"`
	BLKPerGame        float64 `json:"blkPerGame"`
	BLKAgainst        float64 `json:"blkAgainst"`
	BLKAgainstPerGame float64 `json:"blkAgainstPerGame"`
	PtsAgainst        float64 `json:"ptsAgainst"`
	PtsAgainstPerGame float64 `json:"ptsAgainstPerGame"`
}

type Miscellaneous struct {
	Fouls             float64 `json:"fouls"`
	FoulsPerGame      float64 `json:"foulsPerGame"`
	FoulsDrawn        float64 `json:"foulsDrawn"`
	FoulsDrawnPerGame float64 `json:"foulsDrawnPerGame"`
	FoulPers          float64 `json:"foulPers"`
	FoulPersPerGame   float64 `json:"foulPersPerGame"`
	FoulTech          float64 `json:"foulTech"`
	FoulTechPerGame   float64 `json:"foulTechPerGame"`
	PlusMinus         float64 `json:"plusMinus"`
	PlusMinusPerGame  float64 `json:"plusMinusPerGame"`
}

type Standings struct {
	Wins      float64 `json:"wins"`
	Losses    float64 `json:"losses"`
	WinPct    float64 `json:"winPct"`
	GamesBack float64 `json:"gamesBack"`
}

// FourFactorsTeam represents the four factors analysis for a team
type FourFactorsTeam struct {
	Team          string  `json:"team"`
	EFGPercentage float64 `json:"eFGPercentage"`
	TORate        float64 `json:"turnoverRate"`
	ORBRate       float64 `json:"offensiveReboundRate"`
	FTRate        float64 `json:"freeThrowRate"`
	OverallRate   float64 `json:"overallRate"`
}

func loadEnvVars() error {
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("Error loading .env file")
	}
	return nil
}

func fetchTeamStats() ([]TeamMyFeedStatsEntry, error) {
	// Load environment variables
	if err := loadEnvVars(); err != nil {
		return nil, fmt.Errorf("error loading environment variables: %v", err)
	}

	// Get the API key from environment variable
	apiKey := os.Getenv("MYSPORTSFEEDS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API key not found in environment variables")
	}

	// Encode the API key with the password "MYSPORTSFEEDS" in base64
	authString := fmt.Sprintf("%s:%s", apiKey, "MYSPORTSFEEDS")
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(authString))

	// Make the API request with the encoded API key in the Authorization header
	url := "https://api.mysportsfeeds.com/v2.1/pull/nba/2024-2025-regular/team_stats_totals.json"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set the Authorization header with the Base64-encoded credentials
	req.Header.Set("Authorization", "Basic "+encodedAuth)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	// Read the body of the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Unmarshal the response into our struct
	var response TeamMyFeedsStatsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %v", err)
	}

	return response.TeamStatsTotals, nil
}

func FourFactorsHandler(c echo.Context) error {
	teamStats, err := fetchTeamStats()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to fetch team stats: %v", err),
		})
	}

	var results []FourFactorsTeam
	for _, entry := range teamStats {
		// Compute Effective Field Goal Percentage (eFG%)
		eFGPercentage := 0.0
		if entry.Stats.FieldGoals.FGAtt > 0 {
			eFGPercentage = ((entry.Stats.FieldGoals.FGMade + 0.5*entry.Stats.FieldGoals.FG3PtMade) / 
				entry.Stats.FieldGoals.FGAtt) * 100
		}

		// Compute Turnover Rate (TOV%)
		// Formula: TOV / (FGA + 0.44 * FTA + TOV)
		TORate := 0.0
		possessions := entry.Stats.FieldGoals.FGAtt + 
			0.44*entry.Stats.FreeThrows.FTAtt + 
			entry.Stats.Defense.TOV
		if possessions > 0 {
			TORate = (entry.Stats.Defense.TOV / possessions) * 100
		}

		// Compute Offensive Rebound Rate (ORB%)
		// Formula: ORB / (ORB + Opposition DRB)
		ORBRate := 0.0
		totalRebounds := entry.Stats.Rebounds.OffReb + entry.Stats.Rebounds.DefReb
		if totalRebounds > 0 {
			ORBRate = (entry.Stats.Rebounds.OffReb / totalRebounds) * 100
		}

		// Compute Free Throw Rate (FT Rate)
		// Formula: FTA/FGA
		FTRate := 0.0
		if entry.Stats.FieldGoals.FGAtt > 0 {
			FTRate = (entry.Stats.FreeThrows.FTAtt / entry.Stats.FieldGoals.FGAtt) * 100
		}

		overallScore := (eFGPercentage + (100 - TORate) + ORBRate + FTRate) / 4.0

		results = append(results, FourFactorsTeam{
			Team:          fmt.Sprintf("%s %s", entry.Team.City, entry.Team.Name),
			EFGPercentage: roundToTwoDecimals(eFGPercentage),
			TORate:        roundToTwoDecimals(TORate),
			ORBRate:       roundToTwoDecimals(ORBRate),
			FTRate:        roundToTwoDecimals(FTRate),
			OverallRate:   roundToTwoDecimals(overallScore),
		})
	}

	// Sort results by eFG% descending (optional)
	sort.Slice(results, func(i, j int) bool {
		return results[i].EFGPercentage > results[j].EFGPercentage
	})

	return c.JSON(http.StatusOK, results)
}

// Helper function to round float64 to two decimal places
func roundToTwoDecimals(num float64) float64 {
	return math.Round(num*100) / 100
}