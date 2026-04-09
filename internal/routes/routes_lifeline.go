package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

func (ar *appRoutes) initLifelineRoutes(broker *tavern.SSEBroker) {
	ar.e.GET("/sse/app", handleLifelineSSE(broker))
}

func handleLifelineSSE(broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().WriteHeader(http.StatusOK)

		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return fmt.Errorf("streaming unsupported")
		}

		msgs, unsub := broker.Subscribe(TopicAppLifeline)
		defer unsub()

		heartbeat := time.NewTicker(30 * time.Second)
		defer heartbeat.Stop()

		ctx := c.Request().Context()
		for {
			select {
			case <-ctx.Done():
				return nil
			case msg, ok := <-msgs:
				if !ok {
					return nil
				}
				_, _ = fmt.Fprint(c.Response(), msg)
				flusher.Flush()
			case <-heartbeat.C:
				_, _ = fmt.Fprintf(c.Response(), ": heartbeat\n\n")
				flusher.Flush()
			}
		}
	}
}
