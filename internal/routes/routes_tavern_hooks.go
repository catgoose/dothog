// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

type tavernHooksRoutes struct {
	broker *tavern.SSEBroker
	lab    *demo.HooksLab
}

func (ar *appRoutes) initTavernHooksRoutes(broker *tavern.SSEBroker) {
	lab := demo.NewHooksLab()
	h := &tavernHooksRoutes{broker: broker, lab: lab}

	// Middleware: count all publishes on tavern/hooks/* topics.
	broker.UseTopics("tavern/hooks/*", func(next tavern.PublishFunc) tavern.PublishFunc {
		return func(t, msg string) {
			lab.AddPublishStats(len(msg))
			next(t, msg)
		}
	})

	// After hook: derive stats from source and publish to derived topic.
	broker.After(TopicTavernHooksSource, func() {
		lab.RecordHook("after", "computing derived value")
		source := lab.Source()
		derived := computeDerived(source)
		html := renderHooksDerived(derived)
		broker.Publish(TopicTavernHooksDeriv, html)

		// Also publish updated log and stats.
		h.publishLogAndStats()
	})

	// OnMutate: edit POSTs trigger source publish.
	broker.OnMutate("tavern-hooks", func(evt tavern.MutationEvent) {
		content := evt.Data.(string)
		lab.Update(content)
		lab.RecordHook("on-mutate", fmt.Sprintf("source updated (%d chars)", len(content)))
		html := renderHooksSource(content)
		broker.Publish(TopicTavernHooksSource, html)
	})

	// Group for the SSE endpoint with snapshot support.
	broker.DefineGroup("tavern-hooks-all", []string{TopicTavernHooksSource, TopicTavernHooksDeriv})

	ar.e.GET("/realtime/tavern/hooks", h.handlePage)
	ar.e.GET("/sse/tavern/hooks", h.handleSSE)
	ar.e.POST("/realtime/tavern/hooks/mutate", h.handleMutate)
}

func (h *tavernHooksRoutes) handlePage(c echo.Context) error {
	source := h.lab.Source()
	data := views.TavernHooksData{
		Source:  source,
		Derived: computeDerived(source),
		HookLog: h.lab.HookLog(),
	}
	data.PubCount, data.PubBytes = h.lab.PublishStats()
	return handler.RenderBaseLayout(c, views.TavernHooksPage(data))
}

func (h *tavernHooksRoutes) handleMutate(c echo.Context) error {
	content := c.FormValue("content")
	h.broker.NotifyMutate("tavern-hooks", tavern.MutationEvent{
		ID:   "tavern-hooks",
		Data: content,
	})
	return c.NoContent(http.StatusNoContent)
}

func (h *tavernHooksRoutes) handleSSE(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	// Deliver snapshot on subscribe.
	snapshot := h.buildSnapshot()
	_, _ = fmt.Fprint(c.Response(), snapshot)
	flusher.Flush()

	ch, unsub := h.broker.SubscribeMulti(TopicTavernHooksSource, TopicTavernHooksDeriv, TopicTavernHooksLog, TopicTavernHooksStats)
	defer unsub()

	ctx := c.Request().Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case tm, ok := <-ch:
			if !ok {
				return nil
			}
			// wrapForGroup-style: use topic as event name.
			msg := tavern.NewSSEMessage(tm.Topic, tm.Data).String()
			_, _ = fmt.Fprint(c.Response(), msg)
			flusher.Flush()
		}
	}
}

func (h *tavernHooksRoutes) buildSnapshot() string {
	source := h.lab.Source()
	derived := computeDerived(source)

	var buf strings.Builder
	buf.WriteString(tavern.NewSSEMessage(TopicTavernHooksSource, renderHooksSource(source)).String())
	buf.WriteString(tavern.NewSSEMessage(TopicTavernHooksDeriv, renderHooksDerived(derived)).String())
	buf.WriteString(tavern.NewSSEMessage(TopicTavernHooksLog, renderHooksLog(h.lab.HookLog())).String())
	count, bytes := h.lab.PublishStats()
	buf.WriteString(tavern.NewSSEMessage(TopicTavernHooksStats, renderHooksStats(count, bytes)).String())
	return buf.String()
}

func (h *tavernHooksRoutes) publishLogAndStats() {
	logHTML := renderHooksLog(h.lab.HookLog())
	h.broker.Publish(TopicTavernHooksLog, logHTML)

	count, bytes := h.lab.PublishStats()
	statsHTML := renderHooksStats(count, bytes)
	h.broker.Publish(TopicTavernHooksStats, statsHTML)
}

func computeDerived(source string) string {
	words := len(strings.Fields(source))
	chars := utf8.RuneCountInString(source)
	lines := strings.Count(source, "\n") + 1
	return fmt.Sprintf("Words: %d | Characters: %d | Lines: %d", words, chars, lines)
}

func renderHooksSource(source string) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render hooks source")
	if err := views.HooksSourceDisplay(source).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}

func renderHooksDerived(derived string) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render hooks derived")
	if err := views.HooksDerivedDisplay(derived).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}

func renderHooksLog(events []demo.HookEvent) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render hooks log")
	if err := views.HooksLogEntries(events).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}

func renderHooksStats(count, byteCount int64) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render hooks stats")
	if err := views.HooksStatsDisplay(count, byteCount).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}
