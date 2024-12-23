package routes

import (
	"github.com/KPWithCode/statpad2/handlers"
	"github.com/labstack/echo/v4"
)

func GoalRoutes(e *echo.Echo) {
	// Define a route for processing goals
	e.GET("/process-goals", handlers.ProcessGoalsHandler)
	e.GET("process-goals-against", handlers.ProcessGoalsAgainstHandler)
}
