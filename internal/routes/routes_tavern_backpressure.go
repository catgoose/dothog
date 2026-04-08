// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

type tavernBackpressRoutes struct {
	mainBroker  *tavern.SSEBroker
	demoBroker  *tavern.SSEBroker
	lab         *demo.BackpressureLab
	batchWindow atomic.Int64 // nanoseconds; 0 = raw (flush per message)
}

func (ar *appRoutes) initTavernBackpressRoutes(mainBroker *tavern.SSEBroker) {
	lab := demo.NewBackpressureLab()

	demoBroker := tavern.NewSSEBroker(
		tavern.WithBufferSize(3),
		tavern.WithMetrics(),
		tavern.WithAdaptiveBackpressure(tavern.AdaptiveBackpressure{
			ThrottleAt:   3,
			SimplifyAt:   6,
			DisconnectAt: 10,
		}),
		tavern.WithDropOldest(),
	)

	demoBroker.OnBackpressureTierChange(func(sub *tavern.SubscriberInfo, oldTier, newTier tavern.BackpressureTier) {
		lab.RecordTierChange(sub.Topic, sub.ID, int(oldTier), int(newTier))
	})

	for _, t := range []string{"bp-alpha", "bp-beta", "bp-gamma"} {
		demoBroker.SetSimplifiedRenderer(t, func(msg string) string {
			return "[simplified] " + msg
		})
	}

	bp := &tavernBackpressRoutes{
		mainBroker: mainBroker,
		demoBroker: demoBroker,
		lab:        lab,
	}

	mainBroker.RunPublisher(ar.ctx, bp.startTrafficGenerator)
	mainBroker.RunPublisher(ar.ctx, bp.startMetricsPublisher)

	ar.e.GET("/realtime/tavern/backpressure", bp.handlePage)
	ar.e.GET("/sse/tavern/backpressure", echo.WrapHandler(mainBroker.SSEHandler(TopicTavernBackpress)))
	ar.e.GET("/sse/tavern/backpressure/stream", bp.handleStreamSSE)
	ar.e.POST("/realtime/tavern/backpressure/preset", bp.handlePreset)
	ar.e.POST("/realtime/tavern/backpressure/batch", bp.handleBatch)
}

func (bp *tavernBackpressRoutes) handlePage(c echo.Context) error {
	data := bp.buildData()
	bw := time.Duration(bp.batchWindow.Load())
	return handler.RenderBaseLayout(c, views.TavernBackpressurePage(data, bw))
}

func (bp *tavernBackpressRoutes) handlePreset(c echo.Context) error {
	bp.lab.SetPreset(c.FormValue("preset"))
	return c.NoContent(http.StatusNoContent)
}

func (bp *tavernBackpressRoutes) handleBatch(c echo.Context) error {
	ms, err := strconv.Atoi(c.FormValue("ms"))
	if err != nil || ms < 0 || ms > 500 {
		return c.String(http.StatusBadRequest, "invalid batch window")
	}
	bp.batchWindow.Store(int64(time.Duration(ms) * time.Millisecond))
	return c.HTML(http.StatusOK, formatBatchLabel(ms))
}

func (bp *tavernBackpressRoutes) handleStreamSSE(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)
	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	msgs, unsub := bp.demoBroker.SubscribeMulti("bp-alpha", "bp-beta", "bp-gamma")
	defer unsub()

	ctx := c.Request().Context()
	w := c.Response()

	for {
		bw := time.Duration(bp.batchWindow.Load())
		if bw == 0 {
			// Raw mode: flush per message.
			select {
			case <-ctx.Done():
				return nil
			case tm, ok := <-msgs:
				if !ok {
					return nil
				}
				simplified := strings.HasPrefix(tm.Data, "[simplified] ")
				html := renderBPStreamEvent(tm.Topic, tm.Data, simplified)
				_, _ = fmt.Fprint(w, tavern.NewSSEMessage("bp-stream", html).String())
				flusher.Flush()
			}
		} else {
			// Batch mode: collect during window, emit one swap.
			timer := time.NewTimer(bw)
			var rows bytes.Buffer
			collecting := true
			for collecting {
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil
				case tm, ok := <-msgs:
					if !ok {
						timer.Stop()
						return nil
					}
					simplified := strings.HasPrefix(tm.Data, "[simplified] ")
					html := renderBPStreamEvent(tm.Topic, tm.Data, simplified)
					rows.WriteString(html)
				case <-timer.C:
					collecting = false
				}
			}
			if rows.Len() > 0 {
				_, _ = fmt.Fprint(w, tavern.NewSSEMessage("bp-stream-batch", rows.String()).String())
				flusher.Flush()
			}
		}
	}
}

func formatBatchLabel(ms int) string {
	if ms == 0 {
		return "raw"
	}
	return fmt.Sprintf("%dms", ms)
}

func (bp *tavernBackpressRoutes) buildData() views.TavernBackpressureData {
	return views.TavernBackpressureData{
		ActivePreset: bp.lab.ActivePreset(),
		Metrics:      bp.demoBroker.Metrics(),
		Topics:       []string{"bp-alpha", "bp-beta", "bp-gamma"},
		TierChanges:  bp.lab.TierChanges(),
	}
}

func (bp *tavernBackpressRoutes) startTrafficGenerator(ctx context.Context) {
	topics := []string{"bp-alpha", "bp-beta", "bp-gamma"}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			preset := bp.lab.ActivePreset()
			interval, ok := demo.BackpressurePresets[preset]
			if !ok {
				interval = 2 * time.Second
			}
			ticker.Reset(interval)

			topic := topics[rand.IntN(len(topics))]
			msg := fmt.Sprintf("event from %s at %s", topic, time.Now().Format("15:04:05.000"))
			bp.demoBroker.Publish(topic, msg)
		}
	}
}

func (bp *tavernBackpressRoutes) startMetricsPublisher(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var lastPreset string
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !bp.mainBroker.HasSubscribers(TopicTavernBackpress) {
				continue
			}
			data := bp.buildData()

			metricsHTML := renderBPMetrics(data)
			bp.mainBroker.Publish(TopicTavernBackpress, tavern.NewSSEMessage("bp-metrics", metricsHTML).String())

			tierLogHTML := renderBPTierLog(data)
			bp.mainBroker.Publish(TopicTavernBackpress, tavern.NewSSEMessage("bp-tier-log", tierLogHTML).String())

			tierName := bpTierNameFromInt(bp.lab.HighestTier())
			tierHTML := renderBPCurrentTier(tierName)
			bp.mainBroker.Publish(TopicTavernBackpress, tavern.NewSSEMessage("bp-current-tier", tierHTML).String())

			if data.ActivePreset != lastPreset {
				lastPreset = data.ActivePreset
				bp.mainBroker.Publish(TopicTavernBackpress, tavern.NewSSEMessage("bp-preset", data.ActivePreset).String())
			}
		}
	}
}

func bpTierNameFromInt(tier int) string {
	switch tier {
	case 0:
		return "normal"
	case 1:
		return "throttle"
	case 2:
		return "simplify"
	case 3:
		return "disconnect"
	default:
		return fmt.Sprintf("tier-%d", tier)
	}
}

func renderBPMetrics(data views.TavernBackpressureData) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render bp metrics")
	if err := views.TavernBackpressureMetrics(data).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}

func renderBPTierLog(data views.TavernBackpressureData) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render bp tier log")
	if err := views.TavernBackpressureTierLog(data).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}

func renderBPStreamEvent(topic, message string, simplified bool) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render bp stream event")
	if err := views.BackpressureStreamEvent(topic, message, simplified).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}

func renderBPCurrentTier(tierName string) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render bp current tier")
	if err := views.BackpressureCurrentTier(tierName).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}
