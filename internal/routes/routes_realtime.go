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
	components "catgoose/dothog/web/components/core"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
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
	units     map[string]string // client-chosen unit per section
	saved     map[string]int    // snapshot before master override
	pinned    map[string]bool   // pinned sections are excluded from master override
	mu        sync.RWMutex
}

var rtMaster struct {
	mu         sync.RWMutex
	enabled    bool
	intervalMs int
}

var (
	rtBroker  *tavern.SSEBroker
	rtDashPub *tavern.ScheduledPublisher
)

func initRTIntervals() {
	rtIntervals.intervals = make(map[string]int, len(rtCardDefaults))
	rtIntervals.units = make(map[string]string, len(rtCardDefaults))
	rtIntervals.pinned = make(map[string]bool, len(rtCardDefaults))
	for id, iv := range rtCardDefaults {
		rtIntervals.intervals[id] = iv
		rtIntervals.units[id] = components.AutoScale(iv)
	}
}

func (ar *appRoutes) initRealtimeRoutes(broker *tavern.SSEBroker) {
	rtBroker = broker
	numBroker = broker
	initRTIntervals()
	ar.e.GET("/realtime/dashboard", ar.handleRealtimePage())
	ar.e.POST("/realtime/dashboard/interval", handleRTInterval)
	ar.e.POST("/realtime/dashboard/interval-all", handleRTIntervalAll)
	ar.e.POST("/realtime/dashboard/interval-restore", handleRTIntervalRestore)
	ar.e.POST("/realtime/dashboard/pin", handleRTPin)
	ar.e.GET("/sse/system", echo.WrapHandler(broker.SSEHandler(TopicSystemStats)))
	ar.e.GET("/sse/dashboard", echo.WrapHandler(broker.SSEHandler(TopicDashMetrics)))

	// Numerical tile publisher (shares the page, separate SSE stream)
	initTileIntervals()
	ar.e.POST("/realtime/dashboard/tile-interval", handleNumericalInterval)
	ar.e.GET("/sse/numerical", echo.WrapHandler(broker.SSEHandler(TopicNumericalDash)))

	sysPub := ar.newSystemStatsPublisher(broker)
	broker.RunPublisher(ar.ctx, sysPub.Start)

	rtDashPub = ar.newDashboardPublisher(broker)
	broker.RunPublisher(ar.ctx, rtDashPub.Start)

	numPub := ar.newNumericalPublisher(broker)
	broker.RunPublisher(ar.ctx, numPub.Start)
}

func handleRTIntervalAll(c echo.Context) error {
	ms, _ := strconv.Atoi(c.FormValue("interval_ms"))
	if ms < 100 {
		ms = 100
	} else if ms > 86400000 {
		ms = 86400000
	}

	dur := time.Duration(ms) * time.Millisecond

	// Save individual intervals before overriding (skip pinned)
	rtIntervals.mu.Lock()
	if rtIntervals.saved == nil {
		rtIntervals.saved = make(map[string]int, len(rtIntervals.intervals))
		for id, iv := range rtIntervals.intervals {
			rtIntervals.saved[id] = iv
		}
	}
	for id := range rtIntervals.intervals {
		if !rtIntervals.pinned[id] {
			rtIntervals.intervals[id] = ms
			if rtDashPub != nil {
				rtDashPub.SetInterval(id, dur)
			}
		}
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
		if !numTileIntervals.pinned[id] {
			numTileIntervals.intervals[id] = ms
			if numDashPub != nil {
				numDashPub.SetInterval(id, dur)
			}
		}
	}
	numTileIntervals.mu.Unlock()

	// Track master state
	rtMaster.mu.Lock()
	rtMaster.enabled = true
	rtMaster.intervalMs = ms
	rtMaster.mu.Unlock()

	// Broadcast master toggle + slider OOB to all dashboard clients
	broadcastMasterState(true, ms)

	return c.NoContent(http.StatusNoContent)
}

func handleRTIntervalRestore(c echo.Context) error {
	rtIntervals.mu.Lock()
	if rtIntervals.saved != nil {
		for id, iv := range rtIntervals.saved {
			if !rtIntervals.pinned[id] {
				rtIntervals.intervals[id] = iv
				if rtDashPub != nil {
					rtDashPub.SetInterval(id, time.Duration(iv)*time.Millisecond)
				}
			}
		}
		rtIntervals.saved = nil
	}
	rtIntervals.mu.Unlock()

	numTileIntervals.mu.Lock()
	if numTileIntervals.saved != nil {
		for id, iv := range numTileIntervals.saved {
			if !numTileIntervals.pinned[id] {
				numTileIntervals.intervals[id] = iv
				if numDashPub != nil {
					numDashPub.SetInterval(id, time.Duration(iv)*time.Millisecond)
				}
			}
		}
		numTileIntervals.saved = nil
	}
	numTileIntervals.mu.Unlock()

	// Clear master state
	rtMaster.mu.Lock()
	rtMaster.enabled = false
	rtMaster.mu.Unlock()

	// Broadcast master toggle OOB (unchecked) to all dashboard clients
	broadcastMasterState(false, 0)

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
	unit := c.FormValue("unit")
	if unit == "" {
		unit = components.AutoScale(ms)
	}
	rtIntervals.mu.Lock()
	rtIntervals.intervals[section] = ms
	rtIntervals.units[section] = unit
	rtIntervals.mu.Unlock()

	if rtDashPub != nil {
		rtDashPub.SetInterval(section, time.Duration(ms)*time.Millisecond)
	}

	// Broadcast OOB slider update to all dashboard clients
	broadcastCardSlider(section, ms, unit)

	return c.NoContent(http.StatusNoContent)
}

func handleRTPin(c echo.Context) error {
	section := c.FormValue("section")
	if section == "" {
		section = c.FormValue("tile")
	}
	if section == "" {
		return c.NoContent(http.StatusBadRequest)
	}

	// Toggle for chart sections
	rtIntervals.mu.Lock()
	if _, ok := rtIntervals.intervals[section]; ok {
		rtIntervals.pinned[section] = !rtIntervals.pinned[section]
		pinned := rtIntervals.pinned[section]
		rtIntervals.mu.Unlock()
		broadcastPinButton(section, pinned)
		return c.NoContent(http.StatusNoContent)
	}
	rtIntervals.mu.Unlock()

	// Toggle for numerical tiles
	numTileIntervals.mu.Lock()
	if _, ok := numTileIntervals.intervals[section]; ok {
		numTileIntervals.pinned[section] = !numTileIntervals.pinned[section]
		pinned := numTileIntervals.pinned[section]
		numTileIntervals.mu.Unlock()
		broadcastPinButton(section, pinned)
		return c.NoContent(http.StatusNoContent)
	}
	numTileIntervals.mu.Unlock()

	return c.NoContent(http.StatusBadRequest)
}

func broadcastPinButton(section string, pinned bool) {
	if rtBroker == nil || !rtBroker.HasSubscribers(TopicDashMetrics) {
		return
	}
	buf := statsBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	if err := components.PinButton(section, "/realtime/dashboard/pin", pinned, true).Render(context.Background(), buf); err != nil {
		statsBufPool.Put(buf)
		return
	}
	html := buf.String()
	statsBufPool.Put(buf)
	rtBroker.Publish(TopicDashMetrics, html)
}

func (ar *appRoutes) handleRealtimePage() echo.HandlerFunc {
	return func(c echo.Context) error {
		stats := health.CollectRuntimeStats(time.Now())
		snap := initialMetrics()
		services := initialServices()
		svcLatencies := initialServiceLatencies()
		tiles := newNumSim().buildTiles()

		rtMaster.mu.RLock()
		masterEnabled := rtMaster.enabled
		masterMs := rtMaster.intervalMs
		rtMaster.mu.RUnlock()
		if masterMs == 0 {
			masterMs = 2000 // default
		}

		return handler.RenderBaseLayout(c, views.RealtimePage(stats, snap, services, svcLatencies, tiles, masterEnabled, masterMs))
	}
}


// broadcastCardSlider renders a card's IntervalSlider with OOB=true and publishes
// it so all connected dashboard clients see the updated slider state.
func broadcastCardSlider(section string, ms int, unit string) {
	if rtBroker == nil || !rtBroker.HasSubscribers(TopicDashMetrics) {
		return
	}
	cfg := components.IntervalSliderCfg{
		ID:          fmt.Sprintf("iv-%s", section),
		TargetKey:   "section",
		TargetValue: section,
		IntervalMs:  ms,
		Scale:       unit,
		PostURL:     "/realtime/dashboard/interval",
		OOB:         true,
	}
	buf := statsBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	if err := components.IntervalSlider(cfg).Render(context.Background(), buf); err != nil {
		statsBufPool.Put(buf)
		return
	}
	html := buf.String()
	statsBufPool.Put(buf)
	rtBroker.Publish(TopicDashMetrics, html)
}

// broadcastMasterState renders the master toggle OOB and (when enabled) the
// master slider OOB, then publishes to all dashboard clients.
func broadcastMasterState(enabled bool, ms int) {
	if rtBroker == nil || !rtBroker.HasSubscribers(TopicDashMetrics) {
		return
	}
	buf := statsBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	if err := views.OOBMasterToggle(enabled).Render(context.Background(), buf); err != nil {
		statsBufPool.Put(buf)
		return
	}
	if enabled && ms > 0 {
		cfg := components.IntervalSliderCfg{
			ID:          "iv-master",
			TargetKey:   "scope",
			TargetValue: "all",
			IntervalMs:  ms,
			Scale:       components.AutoScale(ms),
			PostURL:     "/realtime/dashboard/interval-all",
			OOB:         true,
		}
		if err := components.IntervalSlider(cfg).Render(context.Background(), buf); err != nil {
			statsBufPool.Put(buf)
			return
		}
	}
	html := buf.String()
	statsBufPool.Put(buf)
	rtBroker.Publish(TopicDashMetrics, html)
}

var statsBufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func (ar *appRoutes) newSystemStatsPublisher(broker *tavern.SSEBroker) *tavern.ScheduledPublisher {
	pub := broker.NewScheduledPublisher(TopicSystemStats, tavern.WithBaseTick(2*time.Second))
	start := time.Now()
	pub.Register("system-stats", 2*time.Second, func(ctx context.Context, buf *bytes.Buffer) error {
		stats := health.CollectRuntimeStats(start)
		return views.SystemStatsOOB(stats).Render(ctx, buf)
	})
	return pub
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

// ── Dashboard simulation state ─────────────────────────────────────────────

type dashSim struct {
	snap         *views.MetricsSnapshot
	services     []views.ServiceStatus
	svcLatencies []views.ServiceLatency
	rps          float64
	errPct       float64
	p99          float64
	cpu          float64
	mem          float64
	netIn        float64
	netOut       float64
	p50          float64
	p90          float64
	diskRead     float64
	diskWrite    float64
	maxSvcMs     float64
	connActive   int
	connIdle     int
	connWait     int
}

func newDashSim() *dashSim {
	snap := initialMetrics()
	return &dashSim{
		snap:         &snap,
		rps:          snap.RPS,
		errPct:       snap.ErrorPct,
		p99:          snap.P99Ms,
		cpu:          snap.CPUPercent,
		mem:          snap.MemPercent,
		netIn:        12.5,
		netOut:       8.3,
		connActive:   snap.ConnActive,
		connIdle:     snap.ConnIdle,
		connWait:     snap.ConnWait,
		p50:          15.0,
		p90:          35.0,
		diskRead:     50.0,
		diskWrite:    30.0,
		services:     initialServices(),
		svcLatencies: initialServiceLatencies(),
	}
}

func (s *dashSim) advance() {
	s.rps += (rand.Float64() - 0.48) * 60
	if rand.Float64() < 0.025 {
		s.rps += 200 + rand.Float64()*150
		s.errPct += 0.75 + rand.Float64()
	}
	s.rps = math.Max(200, math.Min(3000, s.rps))

	s.errPct += (rand.Float64() - 0.55) * 0.2
	s.errPct = math.Max(0.1, math.Min(8.0, s.errPct))

	s.p99 += (rand.Float64() - 0.5) * 7.5
	if s.rps > 2000 {
		s.p99 += 5
	}
	s.p99 = math.Max(10, math.Min(300, s.p99))

	s.cpu += (rand.Float64() - 0.48) * 2.5
	s.cpu = math.Max(5, math.Min(98, s.cpu))
	s.mem += (rand.Float64() - 0.5) * 1.5
	s.mem = math.Max(15, math.Min(95, s.mem))

	s.netIn += (rand.Float64() - 0.48) * 4
	s.netIn = math.Max(1, math.Min(80, s.netIn))
	s.netOut += (rand.Float64() - 0.5) * 3
	s.netOut = math.Max(0.5, math.Min(60, s.netOut))

	pt := views.NetworkPoint{
		InMBps:  math.Round(s.netIn*10) / 10,
		OutMBps: math.Round(s.netOut*10) / 10,
	}
	s.snap.Network = append(s.snap.Network, pt)
	if len(s.snap.Network) > 15 {
		s.snap.Network = s.snap.Network[len(s.snap.Network)-15:]
	}
	maxNet := 0.0
	for _, p := range s.snap.Network {
		combined := p.InMBps + p.OutMBps
		if combined > maxNet {
			maxNet = combined
		}
	}
	s.snap.MaxNetwork = maxNet * 1.1

	total := s.connActive + s.connIdle + s.connWait
	shift := rand.IntN(5) - 2
	s.connActive += shift
	if s.connActive < 3 {
		s.connActive = 3
	}
	if s.connActive > total-4 {
		s.connActive = total - 4
	}
	remaining := total - s.connActive
	s.connIdle = remaining/2 + rand.IntN(3) - 1
	if s.connIdle < 1 {
		s.connIdle = 1
	}
	if s.connIdle > remaining-1 {
		s.connIdle = remaining - 1
	}
	s.connWait = remaining - s.connIdle

	s.p50 += (rand.Float64() - 0.5) * 4
	s.p50 = math.Max(5, math.Min(s.p90-5, s.p50))
	s.p90 += (rand.Float64() - 0.5) * 6
	s.p90 = math.Max(s.p50+5, math.Min(s.p99-5, s.p90))

	s.snap.LatencyHist = append(s.snap.LatencyHist, views.LatencyBucket{
		P50: math.Round(s.p50*10) / 10,
		P90: math.Round(s.p90*10) / 10,
		P99: math.Round(s.p99*10) / 10,
	})
	if len(s.snap.LatencyHist) > 10 {
		s.snap.LatencyHist = s.snap.LatencyHist[len(s.snap.LatencyHist)-10:]
	}
	maxLat := 0.0
	for _, b := range s.snap.LatencyHist {
		if b.P99 > maxLat {
			maxLat = b.P99
		}
	}
	s.snap.MaxLatency = maxLat * 1.1

	s.snap.ErrorHistory = append(s.snap.ErrorHistory, views.ErrorRatePoint{Value: math.Round(s.errPct*10) / 10})
	if len(s.snap.ErrorHistory) > 30 {
		s.snap.ErrorHistory = s.snap.ErrorHistory[len(s.snap.ErrorHistory)-30:]
	}

	s.diskRead += (rand.Float64() - 0.48) * 6
	s.diskRead = math.Max(1, math.Min(200, s.diskRead))
	s.diskWrite += (rand.Float64() - 0.5) * 5
	s.diskWrite = math.Max(1, math.Min(150, s.diskWrite))
	s.snap.DiskIO = append(s.snap.DiskIO, views.DiskIOPoint{
		ReadMBps:  math.Round(s.diskRead*10) / 10,
		WriteMBps: math.Round(s.diskWrite*10) / 10,
	})
	if len(s.snap.DiskIO) > 15 {
		s.snap.DiskIO = s.snap.DiskIO[len(s.snap.DiskIO)-15:]
	}
	maxDisk := 0.0
	for _, d := range s.snap.DiskIO {
		combined := d.ReadMBps + d.WriteMBps
		if combined > maxDisk {
			maxDisk = combined
		}
	}
	s.snap.MaxDiskIO = maxDisk * 1.1

	reqTotal := int(math.Round(s.rps))
	s5xx := int(math.Round(s.errPct / 100 * float64(reqTotal)))
	s4xx := int(float64(reqTotal) * (0.02 + rand.Float64()*0.02))
	s3xx := int(float64(reqTotal) * (0.02 + rand.Float64()*0.02))
	s2xx := reqTotal - s3xx - s4xx - s5xx
	if s2xx < 0 {
		s2xx = 0
	}
	s.snap.StatusDist = views.StatusDistribution{S2xx: s2xx, S3xx: s3xx, S4xx: s4xx, S5xx: s5xx}

	s.snap.RPS = math.Round(s.rps)
	s.snap.ErrorPct = math.Round(s.errPct*10) / 10
	s.snap.P99Ms = math.Round(s.p99*10) / 10
	s.snap.CPUPercent = math.Round(s.cpu*10) / 10
	s.snap.MemPercent = math.Round(s.mem*10) / 10
	s.snap.ConnActive = s.connActive
	s.snap.ConnIdle = s.connIdle
	s.snap.ConnWait = s.connWait

	// Advance services simulation
	s.maxSvcMs = 0.0
	for i := range s.services {
		s.services[i].Load += (rand.Float64() - 0.48) * 0.06
		s.services[i].Load = math.Max(0.05, math.Min(1.0, s.services[i].Load))
		s.services[i].Load = math.Round(s.services[i].Load*100) / 100
		s.services[i].Status = statusFromLoad(s.services[i].Load)

		baseLat := 20 + s.services[i].Load*80
		lat := baseLat + (rand.Float64()-0.5)*10
		lat = math.Max(5, math.Min(300, lat))
		lat = math.Round(lat*10) / 10
		s.svcLatencies[i].History = append(s.svcLatencies[i].History, lat)
		if len(s.svcLatencies[i].History) > 20 {
			s.svcLatencies[i].History = s.svcLatencies[i].History[len(s.svcLatencies[i].History)-20:]
		}
		for _, v := range s.svcLatencies[i].History {
			if v > s.maxSvcMs {
				s.maxSvcMs = v
			}
		}
	}
}

// ── ScheduledPublisher for dashboard ───────────────────────────────────────

// getInterval returns the current interval in ms for a section, reading from
// rtIntervals under its RWMutex.
func getInterval(section string) int {
	rtIntervals.mu.RLock()
	iv := rtIntervals.intervals[section]
	rtIntervals.mu.RUnlock()
	if iv < 100 {
		iv = 100
	}
	return iv
}

func (ar *appRoutes) newDashboardPublisher(broker *tavern.SSEBroker) *tavern.ScheduledPublisher {
	pub := broker.NewScheduledPublisher(TopicDashMetrics, tavern.WithBaseTick(500*time.Millisecond))

	sim := newDashSim()

	// Simulation tick: advances state every 500ms, writes nothing to the buffer.
	// Registered first so it runs before any chart section on every tick.
	pub.Register("sim-tick", 500*time.Millisecond, func(_ context.Context, _ *bytes.Buffer) error {
		sim.advance()
		return nil
	})

	// Each section renders its OOB component + progress bar.
	// The ScheduledPublisher handles the per-section interval check internally.
	renderWithProgress := func(section string, render func(ctx context.Context, buf *bytes.Buffer) error) tavern.RenderFunc {
		return func(ctx context.Context, buf *bytes.Buffer) error {
			if err := render(ctx, buf); err != nil {
				return err
			}
			iv := getInterval(section)
			return components.IntervalProgress(components.ProgressID(section), iv).Render(ctx, buf)
		}
	}

	pub.Register("network", time.Duration(rtCardDefaults["network"])*time.Millisecond,
		renderWithProgress("network", func(ctx context.Context, buf *bytes.Buffer) error {
			return views.OOBNetworkChart(*sim.snap).Render(ctx, buf)
		}))

	pub.Register("latency", time.Duration(rtCardDefaults["latency"])*time.Millisecond,
		renderWithProgress("latency", func(ctx context.Context, buf *bytes.Buffer) error {
			return views.OOBLatencyHistChart(*sim.snap).Render(ctx, buf)
		}))

	pub.Register("error-spark", time.Duration(rtCardDefaults["error-spark"])*time.Millisecond,
		renderWithProgress("error-spark", func(ctx context.Context, buf *bytes.Buffer) error {
			return views.OOBErrorSparkline(*sim.snap).Render(ctx, buf)
		}))

	pub.Register("req-dist", time.Duration(rtCardDefaults["req-dist"])*time.Millisecond,
		renderWithProgress("req-dist", func(ctx context.Context, buf *bytes.Buffer) error {
			return views.OOBRequestDistChart(*sim.snap).Render(ctx, buf)
		}))

	pub.Register("throughput", time.Duration(rtCardDefaults["throughput"])*time.Millisecond,
		renderWithProgress("throughput", func(ctx context.Context, buf *bytes.Buffer) error {
			return views.OOBThroughputSplitChart(*sim.snap).Render(ctx, buf)
		}))

	pub.Register("gauges", time.Duration(rtCardDefaults["gauges"])*time.Millisecond,
		renderWithProgress("gauges", func(ctx context.Context, buf *bytes.Buffer) error {
			return views.OOBCpuMemGauges(*sim.snap).Render(ctx, buf)
		}))

	pub.Register("disk-io", time.Duration(rtCardDefaults["disk-io"])*time.Millisecond,
		renderWithProgress("disk-io", func(ctx context.Context, buf *bytes.Buffer) error {
			return views.OOBDiskIOChart(*sim.snap).Render(ctx, buf)
		}))

	pub.Register("conn-pool", time.Duration(rtCardDefaults["conn-pool"])*time.Millisecond,
		renderWithProgress("conn-pool", func(ctx context.Context, buf *bytes.Buffer) error {
			return views.OOBConnPool(*sim.snap).Render(ctx, buf)
		}))

	pub.Register("sys-stats", time.Duration(rtCardDefaults["sys-stats"])*time.Millisecond,
		renderWithProgress("sys-stats", func(ctx context.Context, buf *bytes.Buffer) error {
			stats := health.CollectRuntimeStats(ar.startTime)
			return views.OOBDashboardStats(stats).Render(ctx, buf)
		}))

	pub.Register("services", time.Duration(rtCardDefaults["services"])*time.Millisecond,
		renderWithProgress("services", func(ctx context.Context, buf *bytes.Buffer) error {
			return views.OOBServicesChart(sim.services).Render(ctx, buf)
		}))

	pub.Register("svc-latency", time.Duration(rtCardDefaults["svc-latency"])*time.Millisecond,
		renderWithProgress("svc-latency", func(ctx context.Context, buf *bytes.Buffer) error {
			return views.OOBServiceLatencyChart(sim.svcLatencies, sim.maxSvcMs*1.1).Render(ctx, buf)
		}))

	pub.Register("events", time.Duration(rtCardDefaults["events"])*time.Millisecond,
		renderWithProgress("events", func(ctx context.Context, buf *bytes.Buffer) error {
			tmpl := eventTemplates[rand.IntN(len(eventTemplates))]
			evt := views.DashboardEvent{
				Time:    time.Now(),
				Kind:    tmpl.Kind,
				Message: tmpl.Messages[rand.IntN(len(tmpl.Messages))],
			}
			return views.OOBEventItem(evt).Render(ctx, buf)
		}))

	return pub
}
