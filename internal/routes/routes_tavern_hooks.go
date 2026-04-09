// setup:feature:demo

package routes

import (
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
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

	// Middleware: count publishes on each hooks topic. Tavern topic middleware
	// uses colon-segment matching, so these slash-delimited topics must be
	// registered explicitly.
	for _, topic := range []string{
		TopicTavernHooksSource,
		TopicTavernHooksDeriv,
		TopicTavernHooksLog,
		TopicTavernHooksStats,
	} {
		broker.UseTopics(topic, func(next tavern.PublishFunc) tavern.PublishFunc {
			return func(t, msg string) {
				lab.AddPublishStats(len(msg))
				next(t, msg)
			}
		})
	}

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
	flusher, err := startSSEResponse(c)
	if err != nil {
		return err
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
			// Preserve SSE comment/control frames such as keepalives.
			msg := tm.Data
			if !strings.HasPrefix(msg, ":") && !strings.HasPrefix(msg, "event: tavern-") {
				msg = tavern.NewSSEMessage(tm.Topic, tm.Data).String()
			}
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
	return renderToString("render hooks source", views.HooksSourceDisplay(source))
}

func renderHooksDerived(derived string) string {
	return renderToString("render hooks derived", views.HooksDerivedDisplay(derived))
}

func renderHooksLog(events []demo.HookEvent) string {
	return renderToString("render hooks log", views.HooksLogEntries(events))
}

func renderHooksStats(count, byteCount int64) string {
	return renderToString("render hooks stats", views.HooksStatsDisplay(count, byteCount))
}
