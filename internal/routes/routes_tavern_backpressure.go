// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

type tavernBackpressRoutes struct {
	mainBroker *tavern.SSEBroker
	demoBroker *tavern.SSEBroker
	lab        *demo.BackpressureLab
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
	ar.e.POST("/realtime/tavern/backpressure/preset", bp.handlePreset)
}

func (bp *tavernBackpressRoutes) handlePage(c echo.Context) error {
	data := bp.buildData()
	return handler.RenderBaseLayout(c, views.TavernBackpressurePage(data))
}

func (bp *tavernBackpressRoutes) handlePreset(c echo.Context) error {
	bp.lab.SetPreset(c.FormValue("preset"))
	return c.NoContent(http.StatusNoContent)
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

			if data.ActivePreset != lastPreset {
				lastPreset = data.ActivePreset
				bp.mainBroker.Publish(TopicTavernBackpress, tavern.NewSSEMessage("bp-preset", data.ActivePreset).String())
			}
		}
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
