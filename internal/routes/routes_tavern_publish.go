// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

type tavernPublishRoutes struct {
	broker *tavern.SSEBroker
	lab    *demo.PublishLab
}

func (ar *appRoutes) initTavernPublishRoutes(broker *tavern.SSEBroker) {
	lab := demo.NewPublishLab()
	p := &tavernPublishRoutes{broker: broker, lab: lab}

	// Count deliveries via middleware on each pub topic.
	broker.UseTopics(TopicTavernPubRaw, func(next tavern.PublishFunc) tavern.PublishFunc {
		return func(t, msg string) {
			lab.RawCount.Add(1)
			next(t, msg)
		}
	})
	broker.UseTopics(TopicTavernPubDebounce, func(next tavern.PublishFunc) tavern.PublishFunc {
		return func(t, msg string) {
			lab.DebouncedCount.Add(1)
			next(t, msg)
		}
	})
	broker.UseTopics(TopicTavernPubThrottle, func(next tavern.PublishFunc) tavern.PublishFunc {
		return func(t, msg string) {
			lab.ThrottledCount.Add(1)
			next(t, msg)
		}
	})
	broker.UseTopics(TopicTavernPubChanged, func(next tavern.PublishFunc) tavern.PublishFunc {
		return func(t, msg string) {
			lab.IfChangedCount.Add(1)
			next(t, msg)
		}
	})

	broker.DefineGroup("tavern-pub-all", []string{
		TopicTavernPubRaw,
		TopicTavernPubDebounce,
		TopicTavernPubThrottle,
		TopicTavernPubChanged,
	})

	ar.e.GET("/realtime/tavern/publish", p.handlePage)
	ar.e.GET("/sse/tavern/publish", echo.WrapHandler(broker.GroupHandler("tavern-pub-all")))
	ar.e.POST("/realtime/tavern/publish/spam", p.handleSpam)
	ar.e.POST("/realtime/tavern/publish/reset", p.handleReset)

	_ = lab // suppress unused if needed
}

func (p *tavernPublishRoutes) handlePage(c echo.Context) error {
	data := views.TavernPublishData{
		RawCount:       p.lab.RawCount.Load(),
		DebouncedCount: p.lab.DebouncedCount.Load(),
		ThrottledCount: p.lab.ThrottledCount.Load(),
		IfChangedCount: p.lab.IfChangedCount.Load(),
	}
	return handler.RenderBaseLayout(c, views.TavernPublishPage(data))
}

func (p *tavernPublishRoutes) handleSpam(c echo.Context) error {
	mode := c.FormValue("mode")
	ts := time.Now().Format("15:04:05")

	for i := range 20 {
		var message string
		if mode == "varied" {
			message = fmt.Sprintf("varied-%d", i+1)
		} else {
			message = "identical-payload"
		}

		html := renderPubEvent(i+1, message, ts)

		p.broker.Publish(TopicTavernPubRaw, html)
		p.broker.PublishDebounced(TopicTavernPubDebounce, html, 500*time.Millisecond)
		p.broker.PublishThrottled(TopicTavernPubThrottle, html, 200*time.Millisecond)
		p.broker.PublishIfChanged(TopicTavernPubChanged, html)
	}

	return c.NoContent(http.StatusNoContent)
}

func (p *tavernPublishRoutes) handleReset(c echo.Context) error {
	p.lab.Reset()
	return c.NoContent(http.StatusNoContent)
}

func renderPubEvent(seq int, message, timestamp string) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render pub event")
	if err := views.PubEventEntry(seq, message, timestamp).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}
