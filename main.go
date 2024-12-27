package main
import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/KPWithCode/statpad2/routes"
    nba "github.com/KPWithCode/statpad2/routes/nbaroutes"

)


func main() {
e := echo.New()
e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
	AllowOrigins:     []string{"http://www.localhost:3000/"},
	AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	AllowCredentials: true, // Set to true if you want to allow credentials (cookies, HTTP authentication) to be included in the CORS request
	MaxAge:           3600, // Max age of the CORS options preflight request in seconds
}))

	routes.EventRoutes(e)
	routes.AssistRoutes(e)
	routes.GoalRoutes(e)
	nba.NBARoutes(e)
	

	
e.Logger.Fatal(e.Start(":8000"))

}