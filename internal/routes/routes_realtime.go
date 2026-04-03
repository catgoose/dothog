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

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/internal/health"
	"github.com/catgoose/tavern"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

// ── Per-section interval state ──────────────────────────────────────────────

var rtCardDefaults = map[string]int{
	"network":     1000, // Network Traffic — 1s
	"latency":     2000, // Latency Histogram — 2s
	"error-spark": 2000, // Error Rate Trend — 2s
	"req-dist":    2000, // Request Distribution — 2s
	"throughput":  2000, // Throughput Split — 2s
	"gauges":      2000, // CPU & Memory — 2s
	"disk-io":     3000, // Disk I/O — 3s
	"conn-pool":   3000, // Connection Pool — 3s
	"sys-stats":   5000, // System Metrics — 5s
	"services":    3000, // Service Health — 3s
	"svc-latency": 3000, // Per-Service Latency — 3s
	"events":      1500, // Event Feed — 1.5s
}

var rtIntervals struct {
	intervals map[string]int
	lastSent  map[string]time.Time
	saved     map[string]int // snapshot before master override
	mu        sync.RWMutex
}

func initRTIntervals() {
	rtIntervals.intervals = make(map[string]int, len(rtCardDefaults))
	rtIntervals.lastSent = make(map[string]time.Time, len(rtCardDefaults))
	for id, iv := range rtCardDefaults {
		rtIntervals.intervals[id] = iv
	}
}

func isDue(cardID string, now time.Time) bool {
	ms := rtIntervals.intervals[cardID]
	if ms < 100 {
		ms = 100
	}
	if now.Sub(rtIntervals.lastSent[cardID]) >= time.Duration(ms)*time.Millisecond {
		rtIntervals.lastSent[cardID] = now
		return true
	}
	return false
}

func (ar *appRoutes) initRealtimeRoutes(broker *tavern.SSEBroker) {
	initRTIntervals()
	ar.e.GET("/hypermedia/realtime", ar.handleRealtimePage())
	ar.e.POST("/hypermedia/realtime/interval", handleRTInterval)
	ar.e.POST("/hypermedia/realtime/interval-all", handleRTIntervalAll)
	ar.e.POST("/hypermedia/realtime/interval-restore", handleRTIntervalRestore)
	ar.e.GET("/sse/system", handleSSESystem(broker))
	ar.e.GET("/sse/dashboard", handleSSEDashboard(broker))

	// Numerical tile publisher (shares the page, separate SSE stream)
	initTileIntervals()
	ar.e.POST("/hypermedia/realtime/tile-interval", handleNumericalInterval)
	ar.e.GET("/sse/numerical", handleSSENumerical(broker))

	go ar.publishSystemStats(broker)
	go ar.publishRealtimeDashboard(broker)
	go ar.publishNumerical(broker)
}

func handleRTIntervalAll(c echo.Context) error {
	ms, _ := strconv.Atoi(c.FormValue("interval_ms"))
	if ms < 100 {
		ms = 100
	} else if ms > 86400000 {
		ms = 86400000
	}

	// Save individual intervals before overriding
	rtIntervals.mu.Lock()
	if rtIntervals.saved == nil {
		rtIntervals.saved = make(map[string]int, len(rtIntervals.intervals))
		for id, iv := range rtIntervals.intervals {
			rtIntervals.saved[id] = iv
		}
	}
	for id := range rtIntervals.intervals {
		rtIntervals.intervals[id] = ms
	}
	rtIntervals.mu.Unlock()

	numTileIntervals.mu.Lock()
	if numTileIntervals.saved == nil {
		numTileIntervals.saved = make(map[string]int, len(numTileIntervals.intervals))
		for id, iv := range numTileIntervals.intervals {
			numTileIntervals.saved[id] = iv
		}
	}
	for id := range numTileIntervals.intervals {
		numTileIntervals.intervals[id] = ms
	}
	numTileIntervals.mu.Unlock()

	return c.NoContent(http.StatusNoContent)
}

func handleRTIntervalRestore(c echo.Context) error {
	rtIntervals.mu.Lock()
	if rtIntervals.saved != nil {
		for id, iv := range rtIntervals.saved {
			rtIntervals.intervals[id] = iv
		}
		rtIntervals.saved = nil
	}
	rtIntervals.mu.Unlock()

	numTileIntervals.mu.Lock()
	if numTileIntervals.saved != nil {
		for id, iv := range numTileIntervals.saved {
			numTileIntervals.intervals[id] = iv
		}
		numTileIntervals.saved = nil
	}
	numTileIntervals.mu.Unlock()

	return c.NoContent(http.StatusNoContent)
}

func handleRTInterval(c echo.Context) error {
	section := c.FormValue("section")
	ms, _ := strconv.Atoi(c.FormValue("interval_ms"))
	if ms < 100 {
		ms = 100
	} else if ms > 86400000 {
		ms = 86400000
	}
	rtIntervals.mu.Lock()
	rtIntervals.intervals[section] = ms
	rtIntervals.mu.Unlock()
	return c.NoContent(http.StatusNoContent)
}

func (ar *appRoutes) handleRealtimePage() echo.HandlerFunc {
	return func(c echo.Context) error {
		stats := health.CollectRuntimeStats(time.Now())
		snap := initialMetrics()
		services := initialServices()
		svcLatencies := initialServiceLatencies()
		tiles := newNumSim().buildTiles()
		return handler.RenderBaseLayout(c, views.RealtimePage(stats, snap, services, svcLatencies, tiles))
	}
}

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
				_, _ = fmt.Fprint(c.Response(), msg)
				flusher.Flush()
			}
		}
	}
}

// handleSSEDashboard streams dashboard card updates from the unified publisher.
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
				_, _ = fmt.Fprint(c.Response(), msg)
				flusher.Flush()
			}
		}
	}
}

var statsBufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func (ar *appRoutes) publishSystemStats(broker *tavern.SSEBroker) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	start := time.Now()
	for {
		select {
		case <-ar.ctx.Done():
			return
		case <-ticker.C:
			if !broker.HasSubscribers(TopicSystemStats) {
				continue
			}
			stats := health.CollectRuntimeStats(start)
			buf := statsBufPool.Get().(*bytes.Buffer)
			buf.Reset()
			if err := views.SystemStatsOOB(stats).Render(shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "publish system stats"), buf); err != nil {
				statsBufPool.Put(buf)
				continue
			}
			msg := tavern.NewSSEMessage("system-stats", buf.String()).String()
			statsBufPool.Put(buf)
			broker.Publish(TopicSystemStats, msg)
		}
	}
}

// --- Simulation data initializers ---

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

// --- Event templates ---

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

// --- Unified dashboard publisher ---

func (ar *appRoutes) publishRealtimeDashboard(broker *tavern.SSEBroker) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Initialize metrics simulation state
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

	// Initialize services simulation state
	services := initialServices()
	svcLatencies := initialServiceLatencies()

	ctx := context.Background()

	for {
		select {
		case <-ar.ctx.Done():
			return
		case <-ticker.C:
			if !broker.HasSubscribers(TopicDashMetrics) {
				continue
			}

			// --- Advance metrics simulation (scaled 0.5x for 500ms ticks) ---

			rps += (rand.Float64() - 0.48) * 60
			if rand.Float64() < 0.025 {
				rps += 200 + rand.Float64()*150
				errPct += 0.75 + rand.Float64()
			}
			rps = math.Max(200, math.Min(3000, rps))

			errPct += (rand.Float64() - 0.55) * 0.2
			errPct = math.Max(0.1, math.Min(8.0, errPct))

			p99 += (rand.Float64() - 0.5) * 7.5
			if rps > 2000 {
				p99 += 5
			}
			p99 = math.Max(10, math.Min(300, p99))

			cpu += (rand.Float64() - 0.48) * 2.5
			cpu = math.Max(5, math.Min(98, cpu))
			mem += (rand.Float64() - 0.5) * 1.5
			mem = math.Max(15, math.Min(95, mem))

			netIn += (rand.Float64() - 0.48) * 4
			netIn = math.Max(1, math.Min(80, netIn))
			netOut += (rand.Float64() - 0.5) * 3
			netOut = math.Max(0.5, math.Min(60, netOut))

			pt := views.NetworkPoint{
				InMBps:  math.Round(netIn*10) / 10,
				OutMBps: math.Round(netOut*10) / 10,
			}
			snap.Network = append(snap.Network, pt)
			if len(snap.Network) > 15 {
				snap.Network = snap.Network[len(snap.Network)-15:]
			}

			maxNet := 0.0
			for _, p := range snap.Network {
				combined := p.InMBps + p.OutMBps
				if combined > maxNet {
					maxNet = combined
				}
			}
			snap.MaxNetwork = maxNet * 1.1

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

			p50 += (rand.Float64() - 0.5) * 4
			p50 = math.Max(5, math.Min(p90-5, p50))
			p90 += (rand.Float64() - 0.5) * 6
			p90 = math.Max(p50+5, math.Min(p99-5, p90))

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

			snap.ErrorHistory = append(snap.ErrorHistory, views.ErrorRatePoint{Value: math.Round(errPct*10) / 10})
			if len(snap.ErrorHistory) > 30 {
				snap.ErrorHistory = snap.ErrorHistory[len(snap.ErrorHistory)-30:]
			}

			diskRead += (rand.Float64() - 0.48) * 6
			diskRead = math.Max(1, math.Min(200, diskRead))
			diskWrite += (rand.Float64() - 0.5) * 5
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

			// --- Advance services simulation (scaled 0.5x) ---

			maxMs := 0.0
			for i := range services {
				services[i].Load += (rand.Float64() - 0.48) * 0.06
				services[i].Load = math.Max(0.05, math.Min(1.0, services[i].Load))
				services[i].Load = math.Round(services[i].Load*100) / 100
				services[i].Status = statusFromLoad(services[i].Load)

				baseLat := 20 + services[i].Load*80
				lat := baseLat + (rand.Float64()-0.5)*10
				lat = math.Max(5, math.Min(300, lat))
				lat = math.Round(lat*10) / 10
				svcLatencies[i].History = append(svcLatencies[i].History, lat)
				if len(svcLatencies[i].History) > 20 {
					svcLatencies[i].History = svcLatencies[i].History[len(svcLatencies[i].History)-20:]
				}
				for _, v := range svcLatencies[i].History {
					if v > maxMs {
						maxMs = v
					}
				}
			}

			// --- Render due cards ---

			now := time.Now()
			stats := health.CollectRuntimeStats(ar.startTime)

			buf := statsBufPool.Get().(*bytes.Buffer)
			buf.Reset()
			needsPublish := false

			rtIntervals.mu.Lock()

			if isDue("network", now) {
				views.OOBNetworkChart(snap).Render(ctx, buf)
				needsPublish = true
			}
			if isDue("latency", now) {
				views.OOBLatencyHistChart(snap).Render(ctx, buf)
				needsPublish = true
			}
			if isDue("error-spark", now) {
				views.OOBErrorSparkline(snap).Render(ctx, buf)
				needsPublish = true
			}
			if isDue("req-dist", now) {
				views.OOBRequestDistChart(snap).Render(ctx, buf)
				needsPublish = true
			}
			if isDue("throughput", now) {
				views.OOBThroughputSplitChart(snap).Render(ctx, buf)
				needsPublish = true
			}
			if isDue("gauges", now) {
				views.OOBCpuMemGauges(snap).Render(ctx, buf)
				needsPublish = true
			}
			if isDue("disk-io", now) {
				views.OOBDiskIOChart(snap).Render(ctx, buf)
				needsPublish = true
			}
			if isDue("conn-pool", now) {
				views.OOBConnPool(snap).Render(ctx, buf)
				needsPublish = true
			}
			if isDue("sys-stats", now) {
				views.OOBDashboardStats(stats).Render(ctx, buf)
				needsPublish = true
			}
			if isDue("services", now) {
				views.OOBServicesChart(services).Render(ctx, buf)
				needsPublish = true
			}
			if isDue("svc-latency", now) {
				views.OOBServiceLatencyChart(svcLatencies, maxMs*1.1).Render(ctx, buf)
				needsPublish = true
			}
			if isDue("events", now) {
				tmpl := eventTemplates[rand.IntN(len(eventTemplates))]
				evt := views.DashboardEvent{
					Time:    now,
					Kind:    tmpl.Kind,
					Message: tmpl.Messages[rand.IntN(len(tmpl.Messages))],
				}
				views.OOBEventItem(evt).Render(ctx, buf)
				needsPublish = true
			}

			rtIntervals.mu.Unlock()

			if needsPublish {
				msg := tavern.NewSSEMessage("dashboard-metrics", buf.String()).String()
				statsBufPool.Put(buf)
				broker.Publish(TopicDashMetrics, msg)
			} else {
				statsBufPool.Put(buf)
			}
		}
	}
}
