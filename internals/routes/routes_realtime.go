// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"

	"catgoose/go-htmx-demo/internals/routes/handler"
	"catgoose/go-htmx-demo/internals/ssebroker"
	"catgoose/go-htmx-demo/web/views"

	"github.com/labstack/echo/v4"
)

func (ar *appRoutes) initRealtimeRoutes(broker *ssebroker.SSEBroker) {
	ar.e.GET("/hypermedia/realtime", ar.handleRealtimePage())
	ar.e.GET("/hypermedia/realtime/poll", handleRealtimePoll)
	ar.e.GET("/hypermedia/realtime/sse-connect", handleSSEConnect)
	ar.e.GET("/sse/system", handleSSESystem(broker))
	ar.e.GET("/sse/dashboard", handleSSEDashboard(broker))

	// Start background publishers.
	go ar.publishSystemStats(broker)
	go ar.publishMetrics(broker)
	go ar.publishServices(broker)
	go ar.publishEvents(broker)
}

func (ar *appRoutes) handleRealtimePage() echo.HandlerFunc {
	return func(c echo.Context) error {
		stats := ssebroker.CollectRuntimeStats(time.Now())
		snap := initialMetrics()
		services := initialServices()
		return handler.RenderBaseLayout(c, views.RealtimePage(stats, snap, services))
	}
}

func handleRealtimePoll(c echo.Context) error {
	n := incrementPollCount()
	return handler.RenderComponent(c, views.PollCountFragment(n))
}

func handleSSEConnect(c echo.Context) error {
	interval := 5
	if v := c.QueryParam("interval"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 60 {
			interval = n
		}
	}
	return handler.RenderComponent(c, views.SSEConnectBlock(interval))
}

func handleSSESystem(broker *ssebroker.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().WriteHeader(http.StatusOK)

		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return fmt.Errorf("streaming unsupported")
		}

		ch, unsub := broker.Subscribe(ssebroker.TopicSystemStats)
		defer unsub()

		ctx := c.Request().Context()
		for {
			select {
			case <-ctx.Done():
				return nil
			case msg, ok := <-ch:
				if !ok {
					return nil
				}
				fmt.Fprint(c.Response(), msg)
				flusher.Flush()
			}
		}
	}
}

// handleSSEDashboard multiplexes 4 SSE topics into one event stream.
// Accepts ?interval=N (1–60 seconds) to throttle per-topic updates.
func handleSSEDashboard(broker *ssebroker.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		interval := 5 * time.Second
		if v := c.QueryParam("interval"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 60 {
				interval = time.Duration(n) * time.Second
			}
		}

		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().WriteHeader(http.StatusOK)

		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return fmt.Errorf("streaming unsupported")
		}

		chStats, unsubStats := broker.Subscribe(ssebroker.TopicSystemStats)
		defer unsubStats()
		chMetrics, unsubMetrics := broker.Subscribe(ssebroker.TopicDashMetrics)
		defer unsubMetrics()
		chServices, unsubServices := broker.Subscribe(ssebroker.TopicDashServices)
		defer unsubServices()
		chEvents, unsubEvents := broker.Subscribe(ssebroker.TopicDashEvents)
		defer unsubEvents()

		lastSent := make(map[string]time.Time)
		forward := func(topic, msg string) {
			if time.Since(lastSent[topic]) >= interval {
				fmt.Fprint(c.Response(), msg)
				flusher.Flush()
				lastSent[topic] = time.Now()
			}
		}

		ctx := c.Request().Context()
		for {
			select {
			case <-ctx.Done():
				return nil
			case msg, ok := <-chStats:
				if !ok {
					return nil
				}
				forward("stats", msg)
			case msg, ok := <-chMetrics:
				if !ok {
					return nil
				}
				forward("metrics", msg)
			case msg, ok := <-chServices:
				if !ok {
					return nil
				}
				forward("services", msg)
			case msg, ok := <-chEvents:
				if !ok {
					return nil
				}
				forward("events", msg)
			}
		}
	}
}

var statsBufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func (ar *appRoutes) publishSystemStats(broker *ssebroker.SSEBroker) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	start := time.Now()
	for {
		select {
		case <-ar.ctx.Done():
			return
		case <-ticker.C:
			if !broker.HasSubscribers(ssebroker.TopicSystemStats) {
				continue
			}
			stats := ssebroker.CollectRuntimeStats(start)
			buf := statsBufPool.Get().(*bytes.Buffer)
			buf.Reset()
			if err := views.SystemStatsOOB(stats).Render(context.Background(), buf); err != nil {
				statsBufPool.Put(buf)
				continue
			}
			msg := ssebroker.NewSSEMessage("system-stats", buf.String()).String()
			statsBufPool.Put(buf)
			broker.Publish(ssebroker.TopicSystemStats, msg)
		}
	}
}

// --- Metrics publisher ---

func initialMetrics() views.MetricsSnapshot {
	return views.MetricsSnapshot{
		RPS:        1200,
		ErrorPct:   0.3,
		P99Ms:      42,
		CPUPercent: 35,
		MemPercent: 52,
		Network: []views.NetworkPoint{{
			InMBps:  12.5,
			OutMBps: 8.3,
		}},
		MaxNetwork: 120,
		ConnActive: 15,
		ConnIdle:   12,
		ConnWait:   8,
	}
}

func (ar *appRoutes) publishMetrics(broker *ssebroker.SSEBroker) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	snap := initialMetrics()

	// Simulation state
	rps := snap.RPS
	errPct := snap.ErrorPct
	p99 := snap.P99Ms
	cpu := snap.CPUPercent
	mem := snap.MemPercent
	netIn := 12.5
	netOut := 8.3
	connActive := snap.ConnActive
	connIdle := snap.ConnIdle
	connWait := snap.ConnWait

	for {
		select {
		case <-ar.ctx.Done():
			return
		case <-ticker.C:
			if !broker.HasSubscribers(ssebroker.TopicDashMetrics) {
				continue
			}

			// Random walk for RPS with occasional spikes
			rps += (rand.Float64() - 0.48) * 120
			if rand.Float64() < 0.05 {
				rps += 400 + rand.Float64()*300
				errPct += 1.5 + rand.Float64()*2
			}
			rps = math.Max(200, math.Min(3000, rps))

			// Error rate drifts back toward baseline
			errPct += (rand.Float64() - 0.55) * 0.4
			errPct = math.Max(0.1, math.Min(8.0, errPct))

			// P99 latency correlates loosely with RPS
			p99 += (rand.Float64() - 0.5) * 15
			if rps > 2000 {
				p99 += 10
			}
			p99 = math.Max(10, math.Min(300, p99))

			// CPU/Memory drift
			cpu += (rand.Float64() - 0.48) * 5
			cpu = math.Max(5, math.Min(98, cpu))
			mem += (rand.Float64() - 0.5) * 3
			mem = math.Max(15, math.Min(95, mem))

			// Network traffic drift
			netIn += (rand.Float64() - 0.48) * 8
			netIn = math.Max(1, math.Min(80, netIn))
			netOut += (rand.Float64() - 0.5) * 6
			netOut = math.Max(0.5, math.Min(60, netOut))

			pt := views.NetworkPoint{
				InMBps:  math.Round(netIn*10) / 10,
				OutMBps: math.Round(netOut*10) / 10,
			}
			snap.Network = append(snap.Network, pt)
			if len(snap.Network) > 15 {
				snap.Network = snap.Network[len(snap.Network)-15:]
			}

			// Recalculate max network for normalization
			maxNet := 0.0
			for _, p := range snap.Network {
				combined := p.InMBps + p.OutMBps
				if combined > maxNet {
					maxNet = combined
				}
			}
			snap.MaxNetwork = maxNet * 1.1

			// Connection pool redistribution (total ~35)
			total := connActive + connIdle + connWait
			shift := rand.IntN(5) - 2
			connActive += shift
			if connActive < 3 {
				connActive = 3
			}
			if connActive > total-4 {
				connActive = total - 4
			}
			remaining := total - connActive
			connIdle = remaining/2 + rand.IntN(3) - 1
			if connIdle < 1 {
				connIdle = 1
			}
			if connIdle > remaining-1 {
				connIdle = remaining - 1
			}
			connWait = remaining - connIdle

			snap.RPS = math.Round(rps)
			snap.ErrorPct = math.Round(errPct*10) / 10
			snap.P99Ms = math.Round(p99*10) / 10
			snap.CPUPercent = math.Round(cpu*10) / 10
			snap.MemPercent = math.Round(mem*10) / 10
			snap.ConnActive = connActive
			snap.ConnIdle = connIdle
			snap.ConnWait = connWait

			buf := statsBufPool.Get().(*bytes.Buffer)
			buf.Reset()
			if err := views.MetricsOOB(snap).Render(context.Background(), buf); err != nil {
				statsBufPool.Put(buf)
				continue
			}
			msg := ssebroker.NewSSEMessage("dashboard-metrics", buf.String()).String()
			statsBufPool.Put(buf)
			broker.Publish(ssebroker.TopicDashMetrics, msg)
		}
	}
}

// --- Services publisher ---

var serviceNames = []string{"api-gateway", "auth-svc", "user-svc", "order-svc", "payment-svc"}

func initialServices() []views.ServiceStatus {
	services := make([]views.ServiceStatus, len(serviceNames))
	for i, name := range serviceNames {
		load := 0.3 + rand.Float64()*0.4
		services[i] = views.ServiceStatus{
			Name:   name,
			Load:   math.Round(load*100) / 100,
			Status: statusFromLoad(load),
		}
	}
	return services
}

func statusFromLoad(load float64) string {
	switch {
	case load > 0.85:
		return "critical"
	case load > 0.70:
		return "degraded"
	default:
		return "healthy"
	}
}

func (ar *appRoutes) publishServices(broker *ssebroker.SSEBroker) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	services := initialServices()

	for {
		select {
		case <-ar.ctx.Done():
			return
		case <-ticker.C:
			if !broker.HasSubscribers(ssebroker.TopicDashServices) {
				continue
			}

			for i := range services {
				services[i].Load += (rand.Float64() - 0.48) * 0.12
				services[i].Load = math.Max(0.05, math.Min(1.0, services[i].Load))
				services[i].Load = math.Round(services[i].Load*100) / 100
				services[i].Status = statusFromLoad(services[i].Load)
			}

			buf := statsBufPool.Get().(*bytes.Buffer)
			buf.Reset()
			if err := views.ServicesOOB(services).Render(context.Background(), buf); err != nil {
				statsBufPool.Put(buf)
				continue
			}
			msg := ssebroker.NewSSEMessage("dashboard-services", buf.String()).String()
			statsBufPool.Put(buf)
			broker.Publish(ssebroker.TopicDashServices, msg)
		}
	}
}

// --- Events publisher ---

type eventTemplate struct {
	Kind     string
	Messages []string
}

var eventTemplates = []eventTemplate{
	{"deploy", []string{
		"Deployed user-svc v2.14.3 to production",
		"Rolling update: api-gateway v1.8.0 (3/5 pods)",
		"Canary deploy: payment-svc v3.1.0 at 10% traffic",
	}},
	{"alert", []string{
		"High error rate on order-svc (>5% 5xx for 2m)",
		"Memory usage above 85% on auth-svc-pod-7",
		"SSL certificate expiring in 7 days for api.example.com",
	}},
	{"scale", []string{
		"Auto-scaled user-svc: 3 -> 5 replicas (CPU > 70%)",
		"Scale-down: order-svc 8 -> 4 replicas (low traffic)",
		"HPA triggered for api-gateway: target CPU 60%",
	}},
	{"restart", []string{
		"Restarted payment-svc-pod-3 (OOMKilled)",
		"Liveness probe failed: auth-svc-pod-2, restarting",
		"CrashLoopBackOff resolved: order-svc-pod-5",
	}},
	{"rollback", []string{
		"Rolled back api-gateway v1.8.0 -> v1.7.9 (error spike)",
		"Auto-rollback triggered: payment-svc health check failed",
	}},
}

func (ar *appRoutes) publishEvents(broker *ssebroker.SSEBroker) {
	for {
		// Random interval 800ms–2s
		delay := 800 + rand.IntN(1200)
		timer := time.NewTimer(time.Duration(delay) * time.Millisecond)

		select {
		case <-ar.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if !broker.HasSubscribers(ssebroker.TopicDashEvents) {
				continue
			}

			tmpl := eventTemplates[rand.IntN(len(eventTemplates))]
			evt := views.DashboardEvent{
				Time:    time.Now(),
				Kind:    tmpl.Kind,
				Message: tmpl.Messages[rand.IntN(len(tmpl.Messages))],
			}

			buf := statsBufPool.Get().(*bytes.Buffer)
			buf.Reset()
			if err := views.EventOOB(evt).Render(context.Background(), buf); err != nil {
				statsBufPool.Put(buf)
				continue
			}
			msg := ssebroker.NewSSEMessage("dashboard-events", buf.String()).String()
			statsBufPool.Put(buf)
			broker.Publish(ssebroker.TopicDashEvents, msg)
		}
	}
}
