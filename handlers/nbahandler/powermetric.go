package nbahandler


import (
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sort"

	"github.com/labstack/echo/v4"
)

// Team represents a single team's data
type Team struct {
	Name        string
	Conference  string 
	A4F, oEFF, dEFF, eDIFF, pDIFF, rSOS, CONS float64
	PowerMetric float64
}

type TeamResponse struct {
	Name        string  `json:"name"`
	Conference  string  `json:"conference"`
	PowerMetric float64 `json:"power_metric"`
}

// CalculatePowerMetric computes the Power Metric for a team
func CalculatePowerMetric(team *Team) {
	team.PowerMetric = (0.3 * team.A4F) + (0.2 * team.oEFF) - (0.2 * team.dEFF) +
		(0.2 * team.eDIFF) + (0.1 * team.pDIFF) - (0.1 * team.rSOS) - (0.1 * team.CONS)
}

// ReadCSV reads the CSV file and parses the data into Team structs
func ReadCSV(filePath string) ([]Team, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var teams []Team
	for i, record := range records {
		if i == 0 {
			// Skip the header
			continue
		}
		a4f, _ := strconv.ParseFloat(record[1], 64)  // Adjust index for 'a4f'
		oEFF, _ := strconv.ParseFloat(record[2], 64) // Adjust index for 'oEFF'
		dEFF, _ := strconv.ParseFloat(record[3], 64) // Adjust index for 'dEFF'
		eDIFF, _ := strconv.ParseFloat(record[4], 64) // Adjust index for 'eDIFF'
		pDIFF, _ := strconv.ParseFloat(record[5], 64) // Adjust index for 'pDIFF'
		rSOS, _ := strconv.ParseFloat(record[6], 64)  // Adjust index for 'rSOS'
		CONS, _ := strconv.ParseFloat(record[7], 64)  // Adjust index for 'CONS'
		conference := record[2] 

		team := Team{
			Name: record[1],
			Conference: conference,
			A4F: a4f, oEFF: oEFF, dEFF: dEFF, eDIFF: eDIFF,
			pDIFF: pDIFF, rSOS: rSOS, CONS: CONS,
		}
		CalculatePowerMetric(&team)
		teams = append(teams, team)
	}

	return teams, nil
}

// Handler for calculating and displaying team strengths
func TeamPowerMetric(c echo.Context) error {
	teams, err := ReadCSV("./data/powermetric.csv") // Update path if needed
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error reading CSV: %v", err))
	}

	// Sort teams by PowerMetric in descending order
	sort.Slice(teams, func(i, j int) bool {
		return teams[i].PowerMetric > teams[j].PowerMetric
	})

	var response []TeamResponse
	for _, team := range teams {
		response = append(response, TeamResponse{
			Name:        team.Name,
			Conference:  team.Conference,
			PowerMetric: team.PowerMetric,
		})
	}

// 	// Group teams by conference
// var eastTeams, westTeams []Team
// for _, team := range teams {
//     if team.Conference == "Eastern" {
//         eastTeams = append(eastTeams, team)
//     } else {
//         westTeams = append(westTeams, team)
//     }
// }

// // Sort each group by PowerMetric
// sort.Slice(eastTeams, func(i, j int) bool {
//     return eastTeams[i].PowerMetric > eastTeams[j].PowerMetric
// })

// sort.Slice(westTeams, func(i, j int) bool {
//     return westTeams[i].PowerMetric > westTeams[j].PowerMetric
// })

// // Combine the sorted teams into one response
// response := append(eastTeams, westTeams...)


	// Generate output
	return c.JSON(http.StatusOK, response)
}