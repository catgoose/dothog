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

	components "catgoose/dothog/web/components/core"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

// ── Simulation state ────────────────────────────────────────────────────────

type numSim struct {
	txnSec    float64
	revenue   float64
	users     float64
	queue     float64
	cacheHit  float64
	errors    float64
	p99       float64
	cpu       float64
	mem       float64
	uptime    time.Time
	deploys   int
	sla       float64
	incidents int

	prevTxn   float64
	prevUsers float64
	prevQueue float64
	prevCache float64
	prevP99   float64
	prevCPU   float64
}

func newNumSim() *numSim {
	return &numSim{
		txnSec:   3200,
		revenue:  48500,
		users:    11200,
		queue:    12,
		cacheHit: 96.5,
		errors:   87,
		p99:      28,
		cpu:      42,
		mem:      9.8,
		uptime:   time.Now().Add(-14*24*time.Hour - 7*time.Hour - 23*time.Minute),
		deploys:  2,
		sla:      99.97,
	}
}

// tick advances the simulation by one 100ms step.
// Deltas are scaled for 10 Hz (1/10th of the 1 Hz magnitudes).
func (s *numSim) tick() {
	s.prevTxn = s.txnSec
	s.prevUsers = s.users
	s.prevQueue = s.queue
	s.prevCache = s.cacheHit
	s.prevP99 = s.p99
	s.prevCPU = s.cpu

	s.txnSec = clampF(s.txnSec+(rand.Float64()-0.48)*30, 800, 9000)
	s.revenue += s.txnSec * 0.0035 * (0.8 + rand.Float64()*0.4)
	s.users = clampF(s.users+(rand.Float64()-0.5)*15, 3000, 30000)
	s.queue = clampF(s.queue+(rand.Float64()-0.55)*0.4, 0, 200)
	s.cacheHit = clampF(s.cacheHit+(rand.Float64()-0.48)*0.08, 82, 99.9)
	s.p99 = clampF(s.p99+(rand.Float64()-0.48)*0.6, 5, 500)
	s.cpu = clampF(s.cpu+(rand.Float64()-0.5)*0.8, 5, 98)
	s.mem = clampF(s.mem+(rand.Float64()-0.5)*0.02, 4, 15.5)

	// Correlate p99 with cpu load
	if s.cpu > 80 {
		s.p99 += (s.cpu - 80) * 0.05
	}

	// Error accumulation with occasional spike
	if rand.Float64() < 0.003 {
		s.errors += float64(10 + rand.IntN(30))
		s.incidents++
	} else if rand.Float64() < 0.1 {
		s.errors++
	}

	// SLA derived from error rate
	errorRate := s.errors / math.Max(s.txnSec*86400, 1) * 100
	s.sla = clampF(100.0-errorRate*5, 95, 100)

	// Occasional deploy
	if rand.Float64() < 0.0001 {
		s.deploys++
	}
}

func (s *numSim) buildTiles() []views.NumTile {
	uptimeDur := time.Since(s.uptime)
	days := int(uptimeDur.Hours()) / 24
	hours := int(uptimeDur.Hours()) % 24
	mins := int(uptimeDur.Minutes()) % 60

	tile := func(id string, t views.NumTile) views.NumTile {
		t.ID = id
		t.IntervalMs = getTileInterval(id)
		t.Scale = getTileScale(id)
		return t
	}

	return []views.NumTile{
		tile("num-txn", views.NumTile{
			Title: "Transactions/sec", Color: "info",
			Value: fmtCommas(int(s.txnSec)), Delta: fmtDelta(s.txnSec, s.prevTxn),
			DeltaUp: s.txnSec >= s.prevTxn,
		}),
		tile("num-revenue", views.NumTile{
			Title: "Revenue Today", Color: "success", Subtitle: "accumulating",
			Value: fmt.Sprintf("$%s", fmtMoney(s.revenue)), Delta: fmt.Sprintf("$%.0f/min", s.txnSec*0.035*60),
			DeltaUp: true,
		}),
		tile("num-users", views.NumTile{
			Title: "Active Users", Color: "info",
			Value: fmtCommas(int(s.users)), Delta: fmtDelta(s.users, s.prevUsers),
			DeltaUp: s.users >= s.prevUsers,
		}),
		tile("num-queue", views.NumTile{
			Title: "Queue Depth", Color: queueColor(s.queue),
			Value: fmt.Sprintf("%d", int(s.queue)), Delta: fmtDelta(s.queue, s.prevQueue),
			DeltaUp: s.queue <= s.prevQueue,
		}),
		tile("num-cache", views.NumTile{
			Title: "Cache Hit Rate", Color: cacheColor(s.cacheHit),
			Value: fmt.Sprintf("%.1f%%", s.cacheHit), Delta: fmtDeltaPct(s.cacheHit, s.prevCache),
			DeltaUp: s.cacheHit >= s.prevCache,
		}),
		tile("num-errors", views.NumTile{
			Title: "Errors (24h)", Color: errorCountColor(s.errors),
			Value: fmtCommas(int(s.errors)), Subtitle: fmt.Sprintf("%d incidents", s.incidents),
		}),
		tile("num-p99", views.NumTile{
			Title: "P99 Latency", Color: latencyColor(s.p99),
			Value: fmt.Sprintf("%.0fms", s.p99), Delta: fmtDelta(s.p99, s.prevP99),
			DeltaUp: s.p99 <= s.prevP99,
		}),
		tile("num-cpu", views.NumTile{
			Title: "CPU Load", Color: cpuColor(s.cpu),
			Value: fmt.Sprintf("%.0f%%", s.cpu), Delta: fmtDeltaPct(s.cpu, s.prevCPU),
			DeltaUp: s.cpu <= s.prevCPU,
		}),
		tile("num-mem", views.NumTile{
			Title: "Memory", Color: memColor(s.mem),
			Value: fmt.Sprintf("%.1f GB", s.mem), Subtitle: fmt.Sprintf("of 16 GB (%.0f%%)", s.mem/16*100),
		}),
		tile("num-uptime", views.NumTile{
			Title: "Uptime", Color: "success", Neutral: true,
			Value: fmt.Sprintf("%dd %dh %dm", days, hours, mins),
		}),
		tile("num-deploys", views.NumTile{
			Title: "Deploys Today", Color: "info", Neutral: true,
			Value: fmt.Sprintf("%d", s.deploys),
		}),
		tile("num-sla", views.NumTile{
			Title: "SLA Compliance", Color: slaColor(s.sla),
			Value: fmt.Sprintf("%.2f%%", s.sla),
		}),
	}
}

// ── Color thresholds ────────────────────────────────────────────────────────

func queueColor(v float64) string {
	switch {
	case v > 100:
		return "error"
	case v > 50:
		return "warning"
	default:
		return "success"
	}
}

func cacheColor(v float64) string {
	switch {
	case v < 85:
		return "error"
	case v < 95:
		return "warning"
	default:
		return "success"
	}
}

func errorCountColor(v float64) string {
	switch {
	case v > 500:
		return "error"
	case v > 200:
		return "warning"
	default:
		return ""
	}
}

func latencyColor(v float64) string {
	switch {
	case v > 200:
		return "error"
	case v > 100:
		return "warning"
	default:
		return "success"
	}
}

func cpuColor(v float64) string {
	switch {
	case v > 85:
		return "error"
	case v > 70:
		return "warning"
	default:
		return "success"
	}
}

func memColor(v float64) string {
	switch {
	case v > 14:
		return "error"
	case v > 12:
		return "warning"
	default:
		return ""
	}
}

func slaColor(v float64) string {
	switch {
	case v < 99.0:
		return "error"
	case v < 99.9:
		return "warning"
	default:
		return "success"
	}
}

// ── Formatting helpers ──────────────────────────────────────────────────────

func fmtCommas(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b []byte
	rem := len(s) % 3
	if rem > 0 {
		b = append(b, s[:rem]...)
	}
	for i := rem; i < len(s); i += 3 {
		if len(b) > 0 {
			b = append(b, ',')
		}
		b = append(b, s[i:i+3]...)
	}
	return string(b)
}

func fmtMoney(v float64) string {
	whole := int(v)
	cents := int((v - float64(whole)) * 100)
	return fmt.Sprintf("%s.%02d", fmtCommas(whole), cents)
}

func fmtDelta(cur, prev float64) string {
	d := cur - prev
	if math.Abs(d) < 0.5 {
		return "—"
	}
	return fmt.Sprintf("%.0f", math.Abs(d))
}

func fmtDeltaPct(cur, prev float64) string {
	d := cur - prev
	if math.Abs(d) < 0.01 {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", math.Abs(d))
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ── Per-tile interval state ─────────────────────────────────────────────────

// Default intervals (milliseconds) — tuned per metric context.
var numDefaultIntervals = map[string]int{
	"num-txn":     500,    // volatile, sub-second
	"num-queue":   500,    // operational, sub-second
	"num-p99":     500,    // performance critical, sub-second
	"num-cpu":     3000,   // 3s — OS-level, moderate
	"num-users":   5000,   // 5s — moderate churn
	"num-errors":  5000,   // 5s — accumulating counter
	"num-revenue": 10000,  // 10s — accumulating slowly
	"num-cache":   10000,  // 10s — relatively stable
	"num-mem":     15000,  // 15s — changes slowly
	"num-sla":     15000,  // 15s — derived, slow-moving
	"num-uptime":  60000,  // 1min — minutes-level granularity
	"num-deploys": 300000, // 5min — rare events
}

var numTileIntervals struct {
	intervals map[string]int       // ms per tile
	lastSent  map[string]time.Time // last publish time per tile
	saved     map[string]int       // snapshot before master override
	mu        sync.RWMutex
}

func initTileIntervals() {
	numTileIntervals.intervals = make(map[string]int, len(numDefaultIntervals))
	numTileIntervals.lastSent = make(map[string]time.Time, len(numDefaultIntervals))
	for id, iv := range numDefaultIntervals {
		numTileIntervals.intervals[id] = iv
	}
}

func getTileInterval(id string) int {
	numTileIntervals.mu.RLock()
	defer numTileIntervals.mu.RUnlock()
	if iv, ok := numTileIntervals.intervals[id]; ok {
		return iv
	}
	return 1000
}

func getTileScale(id string) string {
	return components.AutoScale(getTileInterval(id))
}

// ── Routes ──────────────────────────────────────────────────────────────────

var (
	numBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}
	numBroker  *tavern.SSEBroker
)

func handleNumericalInterval(c echo.Context) error {
	tileID := c.FormValue("tile")
	ms, _ := strconv.Atoi(c.FormValue("interval_ms"))
	if ms < 100 {
		ms = 100
	} else if ms > 86400000 {
		ms = 86400000
	}

	numTileIntervals.mu.Lock()
	numTileIntervals.intervals[tileID] = ms
	numTileIntervals.mu.Unlock()

	// Broadcast OOB slider update to all clients
	broadcastTileSlider(tileID, ms)

	return c.NoContent(http.StatusNoContent)
}

// broadcastTileSlider renders a tile's IntervalSlider with OOB=true and publishes
// it so all connected clients see the updated slider state.
func broadcastTileSlider(tileID string, ms int) {
	if numBroker == nil || !numBroker.HasSubscribers(TopicNumericalDash) {
		return
	}
	cfg := components.IntervalSliderCfg{
		ID:          fmt.Sprintf("iv-%s", tileID),
		TargetKey:   "tile",
		TargetValue: tileID,
		IntervalMs:  ms,
		Scale:       components.AutoScale(ms),
		PostURL:     "/hypermedia/realtime/tile-interval",
		OOB:         true,
	}
	buf := numBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	if err := components.IntervalSlider(cfg).Render(context.Background(), buf); err != nil {
		numBufPool.Put(buf)
		return
	}
	msg := tavern.NewSSEMessage("numerical-dash", buf.String()).String()
	numBufPool.Put(buf)
	numBroker.Publish(TopicNumericalDash, msg)
}

func handleSSENumerical(broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().WriteHeader(200)
		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return fmt.Errorf("streaming not supported")
		}
		flusher.Flush()

		ch, unsub := broker.Subscribe(TopicNumericalDash)
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

// ── Publisher ────────────────────────────────────────────────────────────────

func (ar *appRoutes) publishNumerical(broker *tavern.SSEBroker) {
	// Fast tick: check tile intervals at 100ms resolution.
	// Simulation also advances each tick with scaled deltas.
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	sim := newNumSim()
	ctx := context.Background()

	for {
		select {
		case <-ar.ctx.Done():
			return
		case <-ticker.C:
			if !broker.HasSubscribers(TopicNumericalDash) {
				continue
			}

			now := time.Now()
			sim.tick()

			// Build all tiles, filter to those whose interval has elapsed
			allTiles := sim.buildTiles()
			var due []views.NumTile

			numTileIntervals.mu.Lock()
			for _, t := range allTiles {
				ms := numTileIntervals.intervals[t.ID]
				if ms < 100 {
					ms = 100
				}
				last := numTileIntervals.lastSent[t.ID]
				if now.Sub(last) >= time.Duration(ms)*time.Millisecond {
					due = append(due, t)
					numTileIntervals.lastSent[t.ID] = now
				}
			}
			numTileIntervals.mu.Unlock()

			if len(due) == 0 {
				continue
			}

			buf := numBufPool.Get().(*bytes.Buffer)
			buf.Reset()
			if err := views.NumericalOOB(due).Render(ctx, buf); err != nil {
				numBufPool.Put(buf)
				continue
			}

			msg := tavern.NewSSEMessage("numerical-dash", buf.String()).String()
			numBufPool.Put(buf)
			broker.Publish(TopicNumericalDash, msg)
		}
	}
}
