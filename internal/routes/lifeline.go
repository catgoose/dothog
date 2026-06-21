package routes

import (
	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

func (ar *AppRoutes) initLifelineRoutes(broker *tavern.SSEBroker) {
	ar.e.GET("/sse/app", handleLifelineSSE(broker))
}

func handleLifelineSSE(broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		msgs, unsub := broker.Subscribe(TopicAppLifeline)
		defer unsub()

		return streamSSE(c, msgs, func(s string) string { return s })
	}
}
