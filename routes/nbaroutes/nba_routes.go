package nbaroutes

import (
	"github.com/KPWithCode/statpad2/handlers/nbahandler"
	"github.com/labstack/echo/v4"
)

func NBARoutes(e *echo.Echo) {
	e.GET("/nba/fourfactor", nbahandler.FourFactorsHandler)
	e.GET("/nba/pythagorean", nbahandler.PythagoreanHandler)
	e.GET("/nba/trueshooting", nbahandler.TrueShootingHandler)
	e.GET("/nba/bayesian", nbahandler.BayesianMatchupHandler)
	e.GET("/nba/epm", nbahandler.EPMHandler)
	// e.GET("/nba/playertrends", nbahandler.PlayerTrendsHandler)
	
	
	e.GET("/nba/matchupcheatsheet", nbahandler.PlayerMatchupHandler)
	
	// e.GET("/nba/upcoming", nbahandler.NBAGamesToday)




	e.GET("/nba/playerdstats", nbahandler.FetchAndSavePlayerStats)
	e.GET("/nba/positionaldef", nbahandler.PositionalDefenseHandler)

	e.GET("/nba/mismatch", nbahandler.GetMismatchHandler)
	e.GET("/nba/powerrankings", nbahandler.TeamPowerMetric)
	e.GET("/nba/teamscatterplot", nbahandler.TeamScatterEfficiencyHandler)
	e.GET("/nba/offdefofficiency",nbahandler.RelativeEfficiencyHandler)
	e.GET("/nba/dailymatchupefficiency", nbahandler.DailyMatchupEfficiencyHandler)
	e.GET("/nba/subunit", nbahandler.FetchSubunitStatsHandler)

}
