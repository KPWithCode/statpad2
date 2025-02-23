package nbaroutes

import (
	"github.com/KPWithCode/statpad2/handlers/nbahandler"
	"github.com/labstack/echo/v4"
)

func NBARoutes(e *echo.Echo) {
	e.GET("/nba/fourfactor", nbahandler.FourFactorsHandler)
	e.GET("/nba/pythagorean", nbahandler.PythagoreanHandler)
	e.GET("/nba/trueshooting", nbahandler.TrueShootingHandler)
	e.GET("/nba/epm", nbahandler.EPMHandler)
	e.GET("/nba/blowoutindicator", nbahandler.BlowoutPredictorHandler)
	e.GET("/nba/trendlens", nbahandler.TrendLensHandler)

	// maybe
	e.GET("/nba/bayesian", nbahandler.BayesianMatchupHandler)
	e.GET("/nba/positionaldef", nbahandler.PositionalDefenseHandler)


}
