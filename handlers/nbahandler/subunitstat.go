package nbahandler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)



type SubunitStats struct {
	Players      []string `json:"players"`
	OffEfficiency float64 `json:"off_efficiency"`
	DefEfficiency float64 `json:"def_efficiency"`
}

func FetchSubunitStatsHandler(c echo.Context) error {
	league := c.QueryParam("league")
	players := c.QueryParam("players")
	response, err := http.Get(fmt.Sprintf("https://api.pbpstats.com/get-lineup-subunit-stats/%s?players=%s", league, players))
	if err != nil || response.StatusCode != http.StatusOK {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch subunit stats"})
	}
	defer response.Body.Close()

	var subunitData []SubunitStats
	err = json.NewDecoder(response.Body).Decode(&subunitData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse subunit stats"})
	}

	return c.JSON(http.StatusOK, subunitData)
}