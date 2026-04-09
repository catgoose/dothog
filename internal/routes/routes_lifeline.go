package routes

import (
	"fmt"
	"time"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

func (ar *appRoutes) initLifelineRoutes(broker *tavern.SSEBroker) {
	ar.e.GET("/sse/app", handleLifelineSSE(broker))
}

func handleLifelineSSE(broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		flusher, err := startSSEResponse(c)
		if err != nil {
			return err
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
