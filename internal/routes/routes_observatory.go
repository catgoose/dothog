// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

// demoTopics are the synthetic topics created inside the demo broker.
var demoTopics = []string{"sensors", "orders", "telemetry", "alerts", "logs"}

type observatoryRoutes struct {
	mainBroker *tavern.SSEBroker // delivers SSE to the observatory page
	demoBroker *tavern.SSEBroker // the broker being observed
	state      *demo.ObservatoryState
}

func (ar *appRoutes) initObservatoryRoutes(mainBroker *tavern.SSEBroker) {
	obs := demo.NewObservatoryState()

	demoBroker := tavern.NewSSEBroker(
		tavern.WithBufferSize(5),
		tavern.WithMetrics(),
		tavern.WithObservability(tavern.ObservabilityConfig{
			PublishLatency:  true,
			TopicThroughput: true,
			SubscriberLag:   true,
		}),
		tavern.WithAdaptiveBackpressure(tavern.AdaptiveBackpressure{
			ThrottleAt:   3,
			SimplifyAt:   6,
			DisconnectAt: 10,
		}),
		tavern.WithAdmissionControl(func(_ string, currentCount int) bool {
			max := obs.MaxPerTopic()
			return max <= 0 || currentCount < max
		}),
		tavern.WithDropOldest(),
	)

	demoBroker.OnBackpressureTierChange(func(sub *tavern.SubscriberInfo, oldTier, newTier tavern.BackpressureTier) {
		obs.RecordTierChange(sub.Topic, sub.ID, int(newTier))
	})

	// Register simplified renderers for demo topics so the simplify tier is
	// visible in the tier-change log.
	for _, t := range demoTopics {
		demoBroker.SetSimplifiedRenderer(t, func(msg string) string {
			return "[simplified] " + msg
		})
	}

	o := &observatoryRoutes{
		mainBroker: mainBroker,
		demoBroker: demoBroker,
		state:      obs,
	}

	// Background: generate synthetic traffic on the demo broker.
	mainBroker.RunPublisher(ar.ctx, o.startTrafficGenerator)

	// Background: periodically collect metrics and publish to observatory topic.
	mainBroker.RunPublisher(ar.ctx, o.startMetricsPublisher)

	ar.e.GET("/realtime/observatory", o.handlePage)
	ar.e.GET("/sse/observatory", echo.WrapHandler(mainBroker.SSEHandler(TopicObservatory)))
	ar.e.POST("/realtime/observatory/stress", o.handleStressToggle)
	ar.e.POST("/realtime/observatory/max-per-topic", o.handleMaxPerTopic)
}

func (o *observatoryRoutes) handlePage(c echo.Context) error {
	metrics := o.demoBroker.Metrics()
	counts := o.demoBroker.TopicCounts()
	tierChanges := o.state.RecentTierChanges()

	var obsSnap *tavern.ObservabilitySnapshot
	if obs := o.demoBroker.Observability(); obs != nil {
		s := obs.Snapshot(o.demoBroker)
		obsSnap = &s
	}

	return handler.RenderBaseLayout(c, views.ObservatoryPage(views.ObservatoryData{
		Topics:       demoTopics,
		Metrics:      metrics,
		Counts:       counts,
		ObsSnap:      obsSnap,
		TierChanges:  tierChanges,
		StressActive: o.state.StressActive(),
		MaxPerTopic:  o.state.MaxPerTopic(),
	}))
}

// startTrafficGenerator publishes synthetic messages to the demo broker at
// varying rates to create observable traffic.
func (o *observatoryRoutes) startTrafficGenerator(ctx context.Context) {
	// Per-topic publish intervals (ms).
	intervals := map[string]int{
		"sensors":   100,
		"orders":    500,
		"telemetry": 200,
		"alerts":    2000,
		"logs":      300,
	}
	var wg sync.WaitGroup
	for _, topic := range demoTopics {
		wg.Add(1)
		go func(t string, ms int) {
			defer wg.Done()
			ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
			defer ticker.Stop()
			seq := 0
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					seq++
					msg := fmt.Sprintf("%s event #%d", t, seq)
					o.demoBroker.Publish(t, msg)
				}
			}
		}(topic, intervals[topic])
	}
	<-ctx.Done()
	wg.Wait()
}

// startMetricsPublisher collects demo broker metrics every second and publishes
// the rendered observatory panels to the main broker's observatory topic.
func (o *observatoryRoutes) startMetricsPublisher(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	buf := new(bytes.Buffer)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !o.mainBroker.HasSubscribers(TopicObservatory) {
				continue
			}
			buf.Reset()

			metrics := o.demoBroker.Metrics()
			counts := o.demoBroker.TopicCounts()
			tierChanges := o.state.RecentTierChanges()

			var obsSnap *tavern.ObservabilitySnapshot
			if obs := o.demoBroker.Observability(); obs != nil {
				s := obs.Snapshot(o.demoBroker)
				obsSnap = &s
			}

			data := views.ObservatoryData{
				Topics:       demoTopics,
				Metrics:      metrics,
				Counts:       counts,
				ObsSnap:      obsSnap,
				TierChanges:  tierChanges,
				StressActive: o.state.StressActive(),
				MaxPerTopic:  o.state.MaxPerTopic(),
			}

			if err := views.ObservatoryUpdate(data).Render(ctx, buf); err != nil {
				continue
			}

			sseMsg := tavern.NewSSEMessage("observatory-update", buf.String()).String()
			o.mainBroker.Publish(TopicObservatory, sseMsg)
		}
	}
}

// handleMaxPerTopic updates the per-topic subscriber cap on the demo broker.
// The new cap takes effect for new subscriptions on the next admission check.
func (o *observatoryRoutes) handleMaxPerTopic(c echo.Context) error {
	n, err := strconv.Atoi(c.FormValue("max"))
	if err != nil || n < 0 || n > 1000 {
		return c.String(http.StatusBadRequest, "invalid max")
	}
	o.state.SetMaxPerTopic(n)
	return c.HTML(http.StatusOK, fmt.Sprintf("%d", n))
}

// handleStressToggle starts or stops the stress test against the demo broker.
func (o *observatoryRoutes) handleStressToggle(c echo.Context) error {
	if o.state.StressActive() {
		o.state.CancelStress()
		return handler.RenderComponent(c, views.ObservatoryControls(o.controlsData()))
	}

	// Detach from request context — stress test outlives the HTTP request.
	bgCtx, bgCancel := context.WithCancel(context.Background())
	o.state.SetStress(true, bgCancel)

	// Spawn slow subscribers that deliberately read slowly to trigger
	// backpressure on the demo broker. The number per topic matches the
	// current admission cap so the configured Max Subscribers/Topic value
	// is visibly meaningful.
	perTopic := o.state.MaxPerTopic()
	if perTopic <= 0 {
		perTopic = 1
	}
	for _, topic := range demoTopics {
		for i := 0; i < perTopic; i++ {
			go func(t string) {
				msgs, unsub := o.demoBroker.Subscribe(t)
				if msgs == nil {
					return
				}
				defer unsub()
				for {
					select {
					case <-bgCtx.Done():
						return
					case _, ok := <-msgs:
						if !ok {
							return
						}
						// Deliberately slow: triggers buffer fill and backpressure.
						select {
						case <-bgCtx.Done():
							return
						case <-time.After(500*time.Millisecond + time.Duration(rand.IntN(200))*time.Millisecond):
						}
					}
				}
			}(topic)
		}
	}

	return handler.RenderComponent(c, views.ObservatoryControls(o.controlsData()))
}

// controlsData builds a minimal ObservatoryData for re-rendering the controls
// fragment after a stress toggle.
func (o *observatoryRoutes) controlsData() views.ObservatoryData {
	return views.ObservatoryData{
		StressActive: o.state.StressActive(),
		MaxPerTopic:  o.state.MaxPerTopic(),
	}
}

