package routes

import (
	"time"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

func (ar *appRoutes) initLifelineRoutes(broker *tavern.SSEBroker) {
	ar.e.GET("/sse/app", handleLifelineSSE(broker))
}

func handleLifelineSSE(broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		msgs, unsub := broker.Subscribe(TopicAppLifeline)
		defer unsub()

		return tavern.StreamSSE(
			c.Request().Context(),
			c.Response(),
			msgs,
			func(s string) string { return s },
			tavern.WithStreamHeartbeat(30*time.Second),
		)
	}
}
