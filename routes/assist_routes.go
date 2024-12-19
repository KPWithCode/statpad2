package routes

import (
	"github.com/KPWithCode/statpad2/handlers"
	"github.com/labstack/echo/v4"
)

func AssistRoutes(e *echo.Echo) {
	// Define a route for processing assists
	e.GET("/process-assists", handlers.ProcessAssistsHandler)
}
