package routes

import (
	"github.com/KPWithCode/statpad2/handlers"
	"github.com/labstack/echo/v4"
)

func UpcomingEvents(e *echo.Echo) {
	e.GET("/upcoming-events", handlers.GetUpcomingSports)
	
}
