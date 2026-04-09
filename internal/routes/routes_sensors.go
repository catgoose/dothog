// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

type sensorRoutes struct {
	broker *tavern.SSEBroker
	grid   *demo.SensorGrid
}

func (ar *appRoutes) initSensorRoutes(broker *tavern.SSEBroker) {
	grid := demo.NewSensorGrid()
	s := &sensorRoutes{broker: broker, grid: grid}

	ar.e.GET("/realtime/sensors", s.handlePage)
	ar.e.GET("/sse/sensors", s.handleSSE)
	ar.e.POST("/realtime/sensors/flood", s.handleFloodToggle)

	broker.RunPublisher(ar.ctx, s.startPublisher)
}

func (s *sensorRoutes) handlePage(c echo.Context) error {
	readings := s.grid.AllReadings()
	return handler.RenderBaseLayout(c, views.SensorsPage(readings))
}

func (s *sensorRoutes) handleSSE(c echo.Context) error {
	pattern := c.QueryParam("pattern")
	if pattern == "" {
		pattern = "sensors/**"
	}

	flusher, err := startSSEResponse(c)
	if err != nil {
		return err
	}

	// Build snapshot function for initial state delivery
	snapshotFn := func() string {
		snap := s.grid.Snapshot(pattern)
		var buf bytes.Buffer
		for _, topic := range s.grid.AllTopics() {
			r, ok := snap[topic]
			if !ok {
				continue
			}
			html := s.renderSensorCard(r)
			if html == "" {
				continue
			}
			msg := tavern.NewSSEMessage("sensor-update", html).String()
			buf.WriteString(msg)
		}
		return buf.String()
	}

	msgs, unsub := s.broker.SubscribeGlobWith(pattern,
		tavern.SubWithSnapshot(snapshotFn),
	)
	defer unsub()

	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()

	ctx := c.Request().Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case tm, ok := <-msgs:
			if !ok {
				return nil
			}
			msg := tavern.NewSSEMessage("sensor-update", tm.Data).String()
			_, _ = fmt.Fprint(c.Response(), msg)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprintf(c.Response(), ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (s *sensorRoutes) handleFloodToggle(c echo.Context) error {
	s.grid.SetFloodMode(!s.grid.IsFlooding())
	return handler.RenderComponent(c, views.FloodButton(s.grid.IsFlooding()))
}

func (s *sensorRoutes) startPublisher(ctx context.Context) {
	normalInterval := 500 * time.Millisecond
	floodInterval := 20 * time.Millisecond
	throttle := 500 * time.Millisecond

	ticker := time.NewTicker(normalInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed := s.grid.Tick()
			flooding := s.grid.IsFlooding()

			for _, r := range changed {
				html := s.renderSensorCard(r)
				if html == "" {
					continue
				}
				if flooding {
					s.broker.Publish(r.Topic, html)
				} else {
					s.broker.PublishThrottled(r.Topic, html, throttle)
				}
			}

			// Adjust ticker rate based on flood mode
			if flooding {
				ticker.Reset(floodInterval)
			} else {
				ticker.Reset(normalInterval)
			}
		}
	}
}

func (s *sensorRoutes) renderSensorCard(r demo.SensorReading) string {
	return renderToString("render sensor card", views.SensorCard(r, s.grid.History(r.Topic)))
}
