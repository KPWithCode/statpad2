package nbaroutes

import (
	"github.com/KPWithCode/statpad2/handlers/nbahandler"
	"github.com/labstack/echo/v4"
)

func NBARoutes(e *echo.Echo) {
	// Define a route for processing goals
	e.GET("/nba/upcoming", nbahandler.NBAGamesToday)
	e.GET("/nba/powerrankings", nbahandler.TeamPowerMetric)
	e.GET("/nba/teamscatterplot", nbahandler.TeamScatterEfficiencyHandler)
	e.GET("/nba/offdefofficiency",nbahandler.RelativeEfficiencyHandler)
	e.GET("/nba/dailymatchupefficiency", nbahandler.DailyMatchupEfficiencyHandler)
	e.GET("/nba/subunit", nbahandler.FetchSubunitStatsHandler)

}
