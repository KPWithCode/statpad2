package nbahandler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// TeamScatterEfficiency represents team efficiency metrics for scatter plots.
type TeamScatterEfficiency struct {
	Team   string  `json:"team"`
	Metric float64 `json:"metric"`
}

// TeamScatterEfficiencyHandler fetches scatter plot efficiency metrics for the NBA 2024 season.
func TeamScatterEfficiencyHandler(c echo.Context) error {
	league := "nba"                           // Fixed to NBA
	season := "2023-24"                       // 2024 season
	seasonType := "Regular Season"            // Default to Regular Season
	xAxis := c.QueryParam("xAxis")            // Allow custom X-axis metric
	if xAxis == "" {
		xAxis = "PtsPer100Poss"               // Default X-axis metric
	}
	yAxis := c.QueryParam("yAxis")            // Allow custom Y-axis metric
	if yAxis == "" {
		yAxis = "SecondsPerPoss"             // Default Y-axis metric
	}
	xAxisType := c.QueryParam("xAxisType")    // Allow custom X-axis type
	if xAxisType == "" {
		xAxisType = "Team"                   // Default X-axis type
	}
	yAxisType := c.QueryParam("yAxisType")    // Allow custom Y-axis type
	if yAxisType == "" {
		yAxisType = "Team"                   // Default Y-axis type
	}

	// Construct the API URL
	url := fmt.Sprintf(
		"https://api.pbpstats.com/get-scatter-plots/%s?Season=%s&SeasonType=%s&Xaxis=%s&Yaxis=%s&XaxisType=%s&YaxisType=%s",
		league, season, seasonType, xAxis, yAxis, xAxisType, yAxisType,
	)

	response, err := http.Get(url)
	if err != nil || response.StatusCode != http.StatusOK {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch scatter plot data"})
	}
	defer response.Body.Close()

	var scatterData []TeamScatterEfficiency
	err = json.NewDecoder(response.Body).Decode(&scatterData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse scatter plot data"})
	}

	return c.JSON(http.StatusOK, scatterData)
}
