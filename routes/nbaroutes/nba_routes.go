package nbaroutes

import (
	"github.com/KPWithCode/statpad2/handlers/nbahandler"
	"github.com/labstack/echo/v4"
)

func NBARoutes(e *echo.Echo) {
	// Define a route for processing goals
	e.GET("/nba/teamscatterplot", nbahandler.TeamScatterEfficiencyHandler)
	
}
