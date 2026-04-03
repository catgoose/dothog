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

	"catgoose/dothog/internal/health"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

// statsBufPool is a shared buffer pool used by various SSE publishers in the
// routes package.
var statsBufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// ── ScheduledPublishers ────────────────────────────────────────────────────

var (
	dashPub     *tavern.ScheduledPublisher
	dashTracker *intervalTracker
	sysPub      *tavern.ScheduledPublisher
)

var rtCardDefaults = map[string]time.Duration{
	"metrics":  1000 * time.Millisecond,
	"services": 3000 * time.Millisecond,
	"events":   1500 * time.Millisecond,
}

// ── Routes ──────────────────────────────────────────────────────────────────

func (ar *appRoutes) initRealtimeRoutes(broker *tavern.SSEBroker) {
	ar.e.GET("/hypermedia/realtime", ar.handleRealtimePage())
	ar.e.POST("/hypermedia/realtime/interval", handleRTInterval)
	ar.e.POST("/hypermedia/realtime/interval-all", handleRTIntervalAll)
	ar.e.POST("/hypermedia/realtime/interval-restore", handleRTIntervalRestore)
	ar.e.GET("/sse/system", handleSSESystem(broker))
	ar.e.GET("/sse/dashboard", handleSSEDashboard(broker))

	ar.initDashboardPublisher(broker)
	ar.initSystemStatsPublisher(broker)
}

func handleRTInterval(c echo.Context) error {
	section := c.FormValue("section")
	ms, _ := strconv.Atoi(c.FormValue("interval_ms"))
	if ms < 100 {
		ms = 100
	} else if ms > 86400000 {
		ms = 86400000
	}
	dashTracker.set(dashPub, section, time.Duration(ms)*time.Millisecond)
	return c.NoContent(http.StatusNoContent)
}

func handleRTIntervalAll(c echo.Context) error {
	ms, _ := strconv.Atoi(c.FormValue("interval_ms"))
	if ms < 100 {
		ms = 100
	} else if ms > 86400000 {
		ms = 86400000
	}
	d := time.Duration(ms) * time.Millisecond
	dashTracker.saveAndOverride(dashPub, d)
	numTracker.saveAndOverride(numPub, d)
	return c.NoContent(http.StatusNoContent)
}

func handleRTIntervalRestore(c echo.Context) error {
	dashTracker.restore(dashPub)
	numTracker.restore(numPub)
	return c.NoContent(http.StatusNoContent)
}

func (ar *appRoutes) handleRealtimePage() echo.HandlerFunc {
	return func(c echo.Context) error {
		stats := health.CollectRuntimeStats(time.Now())
		snap := initialMetrics()
		services := initialServices()
		svcLatencies := initialServiceLatencies()
		return handler.RenderBaseLayout(c, views.RealtimePage(stats, snap, services, svcLatencies))
	}
}

// ── SSE handlers ───────────────────────────────────────────────────────────

func handleSSESystem(broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().WriteHeader(http.StatusOK)

		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return fmt.Errorf("streaming unsupported")
		}

		ch, unsub := broker.Subscribe(TopicSystemStats)
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
				_, _ = fmt.Fprint(c.Response(), tavern.NewSSEMessage("system-stats", msg).String())
				flusher.Flush()
			}
		}
	}
}

func handleSSEDashboard(broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().WriteHeader(http.StatusOK)

		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return fmt.Errorf("streaming unsupported")
		}

		ch, unsub := broker.Subscribe(TopicDashMetrics)
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
				_, _ = fmt.Fprint(c.Response(), tavern.NewSSEMessage("dashboard-metrics", msg).String())
				flusher.Flush()
			}
		}
	}
}

// ── System stats publisher ─────────────────────────────────────────────────

func (ar *appRoutes) initSystemStatsPublisher(broker *tavern.SSEBroker) {
	sysPub = broker.NewScheduledPublisher(TopicSystemStats)
	sysPub.Register("stats", 2*time.Second, func(ctx context.Context, buf *bytes.Buffer) error {
		stats := health.CollectRuntimeStats(ar.startTime)
		return views.SystemStatsOOB(stats).Render(shared.WithContextIDAndDescription(ctx, shared.GenerateContextID(), "publish system stats"), buf)
	})
	broker.RunPublisher(ar.ctx, sysPub.Start)
}

// ── Dashboard publisher ────────────────────────────────────────────────────

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
		LatencyHist:  []views.LatencyBucket{{P50: 15, P90: 35, P99: 42}},
		ErrorHistory: []views.ErrorRatePoint{{Value: 0.3}},
		DiskIO:       []views.DiskIOPoint{{ReadMBps: 50, WriteMBps: 30}},
		StatusDist:   views.StatusDistribution{S2xx: 1100, S3xx: 36, S4xx: 36, S5xx: 4},
		MaxLatency:   50,
		MaxDiskIO:    100,
	}
}

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

func initialServiceLatencies() []views.ServiceLatency {
	svcLats := make([]views.ServiceLatency, len(serviceNames))
	for i, name := range serviceNames {
		svcLats[i] = views.ServiceLatency{
			Name:    name,
			History: []float64{20 + rand.Float64()*30},
		}
	}
	return svcLats
}

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

func (ar *appRoutes) initDashboardPublisher(broker *tavern.SSEBroker) {
	dashTracker = newIntervalTracker(rtCardDefaults)
	dashPub = broker.NewScheduledPublisher(TopicDashMetrics, tavern.WithBaseTick(500*time.Millisecond))

	// Shared simulation state
	snap := initialMetrics()
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
	p50 := 15.0
	p90 := 35.0
	diskRead := 50.0
	diskWrite := 30.0

	services := initialServices()
	svcLatencies := initialServiceLatencies()

	// _sim: advance simulation state every tick, writes nothing
	dashPub.Register("_sim", 500*time.Millisecond, func(_ context.Context, _ *bytes.Buffer) error {
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

		// P50/P90 random walk (enforce ordering)
		p50 += (rand.Float64() - 0.5) * 8
		p50 = math.Max(5, math.Min(p90-5, p50))
		p90 += (rand.Float64() - 0.5) * 12
		p90 = math.Max(p50+5, math.Min(p99-5, p90))

		// Latency histogram (rolling 10)
		snap.LatencyHist = append(snap.LatencyHist, views.LatencyBucket{
			P50: math.Round(p50*10) / 10,
			P90: math.Round(p90*10) / 10,
			P99: math.Round(p99*10) / 10,
		})
		if len(snap.LatencyHist) > 10 {
			snap.LatencyHist = snap.LatencyHist[len(snap.LatencyHist)-10:]
		}
		maxLat := 0.0
		for _, b := range snap.LatencyHist {
			if b.P99 > maxLat {
				maxLat = b.P99
			}
		}
		snap.MaxLatency = maxLat * 1.1

		// Error history (rolling 30)
		snap.ErrorHistory = append(snap.ErrorHistory, views.ErrorRatePoint{Value: math.Round(errPct*10) / 10})
		if len(snap.ErrorHistory) > 30 {
			snap.ErrorHistory = snap.ErrorHistory[len(snap.ErrorHistory)-30:]
		}

		// Disk I/O random walk (rolling 15)
		diskRead += (rand.Float64() - 0.48) * 12
		diskRead = math.Max(1, math.Min(200, diskRead))
		diskWrite += (rand.Float64() - 0.5) * 10
		diskWrite = math.Max(1, math.Min(150, diskWrite))
		snap.DiskIO = append(snap.DiskIO, views.DiskIOPoint{
			ReadMBps:  math.Round(diskRead*10) / 10,
			WriteMBps: math.Round(diskWrite*10) / 10,
		})
		if len(snap.DiskIO) > 15 {
			snap.DiskIO = snap.DiskIO[len(snap.DiskIO)-15:]
		}
		maxDisk := 0.0
		for _, d := range snap.DiskIO {
			combined := d.ReadMBps + d.WriteMBps
			if combined > maxDisk {
				maxDisk = combined
			}
		}
		snap.MaxDiskIO = maxDisk * 1.1

		// Status distribution (derived from RPS)
		reqTotal := int(math.Round(rps))
		s5xx := int(math.Round(errPct / 100 * float64(reqTotal)))
		s4xx := int(float64(reqTotal) * (0.02 + rand.Float64()*0.02))
		s3xx := int(float64(reqTotal) * (0.02 + rand.Float64()*0.02))
		s2xx := reqTotal - s3xx - s4xx - s5xx
		if s2xx < 0 {
			s2xx = 0
		}
		snap.StatusDist = views.StatusDistribution{S2xx: s2xx, S3xx: s3xx, S4xx: s4xx, S5xx: s5xx}

		snap.RPS = math.Round(rps)
		snap.ErrorPct = math.Round(errPct*10) / 10
		snap.P99Ms = math.Round(p99*10) / 10
		snap.CPUPercent = math.Round(cpu*10) / 10
		snap.MemPercent = math.Round(mem*10) / 10
		snap.ConnActive = connActive
		snap.ConnIdle = connIdle
		snap.ConnWait = connWait

		// Advance services simulation
		for i := range services {
			services[i].Load += (rand.Float64() - 0.48) * 0.12
			services[i].Load = math.Max(0.05, math.Min(1.0, services[i].Load))
			services[i].Load = math.Round(services[i].Load*100) / 100
			services[i].Status = statusFromLoad(services[i].Load)

			baseLat := 20 + services[i].Load*80
			lat := baseLat + (rand.Float64()-0.5)*20
			lat = math.Max(5, math.Min(300, lat))
			lat = math.Round(lat*10) / 10
			svcLatencies[i].History = append(svcLatencies[i].History, lat)
			if len(svcLatencies[i].History) > 20 {
				svcLatencies[i].History = svcLatencies[i].History[len(svcLatencies[i].History)-20:]
			}
		}

		return nil
	})

	// Metrics section — KPIs, charts, gauges, system stats
	dashPub.Register("metrics", rtCardDefaults["metrics"], func(ctx context.Context, buf *bytes.Buffer) error {
		stats := health.CollectRuntimeStats(ar.startTime)
		return views.MetricsOOB(snap, stats).Render(shared.WithContextIDAndDescription(ctx, shared.GenerateContextID(), "publish metrics"), buf)
	})

	// Services section — service health bars + per-service latency
	dashPub.Register("services", rtCardDefaults["services"], func(ctx context.Context, buf *bytes.Buffer) error {
		maxMs := 0.0
		for _, sl := range svcLatencies {
			for _, v := range sl.History {
				if v > maxMs {
					maxMs = v
				}
			}
		}
		return views.ServicesOOB(services, svcLatencies, maxMs*1.1).Render(shared.WithContextIDAndDescription(ctx, shared.GenerateContextID(), "publish services"), buf)
	})

	// Events section — live event feed
	dashPub.Register("events", rtCardDefaults["events"], func(ctx context.Context, buf *bytes.Buffer) error {
		tmpl := eventTemplates[rand.IntN(len(eventTemplates))]
		evt := views.DashboardEvent{
			Time:    time.Now(),
			Kind:    tmpl.Kind,
			Message: tmpl.Messages[rand.IntN(len(tmpl.Messages))],
		}
		return views.EventOOB(evt).Render(shared.WithContextIDAndDescription(ctx, shared.GenerateContextID(), "publish events"), buf)
	})

	broker.RunPublisher(ar.ctx, dashPub.Start)
}
