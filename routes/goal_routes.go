package routes

import (
	"github.com/KPWithCode/statpad2/handlers"
	"github.com/labstack/echo/v4"
)

func GoalRoutes(e *echo.Echo) {
	// Define a route for processing goals
	e.GET("/process-goals", handlers.ProcessGoalsHandler)
	e.GET("/process-goals-against", handlers.ProcessGoalsAgainstHandler)
	e.GET("/process-goal-diff", handlers.ProcessGoalDifferentialHandler)
	e.GET("/process-avgscoretime", handlers.ProcessTimeToScoreHandler)
	e.GET("/process-shotstogoals", handlers.ProcessShotsToGoalHandler)
}
