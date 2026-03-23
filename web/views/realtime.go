// setup:feature:demo

package views

import (
	"fmt"
	"math"
	"time"

	"catgoose/dothog/internal/ssebroker"
)

// --- Dashboard data types ---

// NetworkPoint is a single data point for network traffic.
type NetworkPoint struct {
	InMBps  float64
	OutMBps float64
}

// LatencyBucket holds P50/P90/P99 for a single time tick.
type LatencyBucket struct {
	P50 float64
	P90 float64
	P99 float64
}

// ErrorRatePoint is a single point in the error rate sparkline history.
type ErrorRatePoint struct {
	Value float64
}

// DiskIOPoint holds read/write throughput at one tick.
type DiskIOPoint struct {
	ReadMBps  float64
	WriteMBps float64
}

// StatusDistribution holds counts per HTTP status class.
type StatusDistribution struct {
	S2xx int
	S3xx int
	S4xx int
	S5xx int
}

// ServiceLatency holds per-service P99 latency history for multi-line chart.
type ServiceLatency struct {
	Name    string
	History []float64
}

// MetricsSnapshot holds the current dashboard metrics state.
type MetricsSnapshot struct {
	RPS        float64
	ErrorPct   float64
	P99Ms      float64
	CPUPercent float64
	MemPercent float64
	Network    []NetworkPoint
	MaxNetwork float64 // max combined MB/s for chart normalization
	ConnActive int
	ConnIdle   int
	ConnWait   int
	// New chart data
	LatencyHist  []LatencyBucket
	ErrorHistory []ErrorRatePoint
	DiskIO       []DiskIOPoint
	StatusDist   StatusDistribution
	MaxLatency   float64
	MaxDiskIO    float64
}

// networkAreaStyles returns CSS style strings for a Charts.css area chart.
// Each entry has --start (previous normalized size) and --size (current).
func networkAreaStyles(snap MetricsSnapshot) []string {
	pts := snap.Network
	max := maxOrOne(snap.MaxNetwork)
	styles := make([]string, len(pts))
	for i, pt := range pts {
		size := (pt.InMBps + pt.OutMBps) / max
		start := 0.0
		if i > 0 {
			prev := pts[i-1]
			start = (prev.InMBps + prev.OutMBps) / max
		}
		styles[i] = fmt.Sprintf("--start: %s; --size: %s; --color: %s",
			fmtSize(start), fmtSize(size), networkBarColor(pt))
	}
	return styles
}

// networkBarColor returns a hex color based on combined in+out MB/s.
func networkBarColor(pt NetworkPoint) string {
	combined := pt.InMBps + pt.OutMBps
	switch {
	case combined > 80:
		return "#f87171" // red-400
	case combined > 40:
		return "#fbbf24" // amber-400
	default:
		return "#34d399" // emerald-400
	}
}

// fmtMBps formats megabytes per second.
func fmtMBps(v float64) string {
	return fmt.Sprintf("%.1f MB/s", v)
}

// gaugeColor returns a DaisyUI color class based on percentage thresholds.
func gaugeColor(pct float64) string {
	switch {
	case pct > 85:
		return "text-error"
	case pct > 65:
		return "text-warning"
	default:
		return "text-success"
	}
}

// latestNetworkIn returns the inbound MB/s of the last network point.
func latestNetworkIn(snap MetricsSnapshot) float64 {
	if len(snap.Network) == 0 {
		return 0
	}
	return snap.Network[len(snap.Network)-1].InMBps
}

// latestNetworkOut returns the outbound MB/s of the last network point.
func latestNetworkOut(snap MetricsSnapshot) float64 {
	if len(snap.Network) == 0 {
		return 0
	}
	return snap.Network[len(snap.Network)-1].OutMBps
}

// maxOrOne returns v if positive, otherwise 1 (avoids division by zero).
func maxOrOne(v float64) float64 {
	if v > 0 {
		return v
	}
	return 1
}

// fmtSize formats a float to 3 decimal places for CSS custom property values.
func fmtSize(v float64) string {
	return fmt.Sprintf("%.3f", v)
}

// ServiceStatus represents the health and load of a single service.
type ServiceStatus struct {
	Name   string
	Load   float64 // 0.0–1.0
	Status string  // "healthy", "degraded", "critical"
}

// serviceBarColor returns a hex color based on service status.
func serviceBarColor(s ServiceStatus) string {
	switch s.Status {
	case "critical":
		return "#f87171" // red-400
	case "degraded":
		return "#fbbf24" // amber-400
	default:
		return "#34d399" // emerald-400
	}
}

// DashboardEvent represents a single entry in the live event feed.
type DashboardEvent struct {
	Time    time.Time
	Kind    string // "deploy", "alert", "scale", "restart", "rollback"
	Message string
}

// eventBadgeClass returns a DaisyUI badge class for the event kind.
func eventBadgeClass(kind string) string {
	switch kind {
	case "deploy":
		return "badge-info"
	case "alert":
		return "badge-error"
	case "scale":
		return "badge-warning"
	case "restart":
		return "badge-accent"
	case "rollback":
		return "badge-secondary"
	default:
		return "badge-ghost"
	}
}

// eventIcon returns a small indicator for each event kind.
func eventIcon(kind string) string {
	switch kind {
	case "deploy":
		return "D"
	case "alert":
		return "!"
	case "scale":
		return "S"
	case "restart":
		return "R"
	case "rollback":
		return "B"
	default:
		return "?"
	}
}

// fmtPct formats a percentage with one decimal place.
func fmtPct(v float64) string {
	return fmt.Sprintf("%.1f%%", v)
}

// fmtMs formats milliseconds with one decimal place.
func fmtMs(v float64) string {
	return fmt.Sprintf("%.1fms", v)
}

// fmtRPS formats requests per second with zero decimal places.
func fmtRPS(v float64) string {
	return fmt.Sprintf("%.0f", math.Round(v))
}

type statsEntry struct {
	ID      string
	Label   string
	Value   string
	Section string
}

func statsEntries(s ssebroker.SystemStats) []statsEntry {
	return []statsEntry{
		// Runtime
		{"stat-uptime", "Uptime", s.Uptime, "Runtime"},
		{"stat-goversion", "Go Version", s.GoVersion, "Runtime"},
		{"stat-osarch", "OS / Arch", s.OS + "/" + s.Arch, "Runtime"},
		{"stat-numcpu", "CPUs", fmt.Sprintf("%d", s.NumCPU), "Runtime"},
		{"stat-goroutines", "Goroutines", fmt.Sprintf("%d", s.Goroutines), "Runtime"},
		{"stat-numthread", "OS Threads", fmt.Sprintf("%d", s.NumThread), "Runtime"},
		{"stat-updated", "Updated", s.Timestamp, "Runtime"},

		// Memory
		{"stat-heapalloc", "Heap Alloc", fmt.Sprintf("%.2f MB", s.HeapAllocMB), "Memory"},
		{"stat-heapsys", "Heap Sys", fmt.Sprintf("%.2f MB", s.HeapSysMB), "Memory"},
		{"stat-heapidle", "Heap Idle", fmt.Sprintf("%.2f MB", s.HeapIdleMB), "Memory"},
		{"stat-heapreleased", "Heap Released", fmt.Sprintf("%.2f MB", s.HeapReleasedMB), "Memory"},
		{"stat-stackinuse", "Stack In Use", fmt.Sprintf("%.2f MB", s.StackInUseMB), "Memory"},
		{"stat-sysmb", "Sys Total", fmt.Sprintf("%.2f MB", s.SysMB), "Memory"},
		{"stat-totalalloc", "Total Alloc", fmt.Sprintf("%.2f MB", s.TotalAllocMB), "Memory"},

		// GC
		{"stat-gccycles", "GC Cycles", fmt.Sprintf("%d", s.GCCycles), "GC"},
		{"stat-lastpause", "Last GC Pause", fmt.Sprintf("%d µs", s.LastPauseMicros), "GC"},
		{"stat-nextgc", "Next GC Target", fmt.Sprintf("%.2f MB", s.NextGCMB), "GC"},

		// Allocator
		{"stat-heapobjects", "Heap Objects", fmt.Sprintf("%d", s.HeapObjects), "Allocator"},
		{"stat-mallocs", "Mallocs", fmt.Sprintf("%d", s.Mallocs), "Allocator"},
		{"stat-frees", "Frees", fmt.Sprintf("%d", s.Frees), "Allocator"},
		{"stat-liveobjects", "Live Objects", fmt.Sprintf("%d", s.LiveObjects), "Allocator"},
	}
}

// --- New chart style helpers ---

// latencyHistStyle holds CSS styles for one bucket of the latency histogram (3 series).
type latencyHistStyle struct {
	P50Style string
	P90Style string
	P99Style string
}

func latencyHistStyles(snap MetricsSnapshot) []latencyHistStyle {
	max := maxOrOne(snap.MaxLatency)
	styles := make([]latencyHistStyle, len(snap.LatencyHist))
	for i, b := range snap.LatencyHist {
		styles[i] = latencyHistStyle{
			P50Style: fmt.Sprintf("--size: %s", fmtSize(b.P50/max)),
			P90Style: fmt.Sprintf("--size: %s", fmtSize(b.P90/max)),
			P99Style: fmt.Sprintf("--size: %s", fmtSize(b.P99/max)),
		}
	}
	return styles
}

func errorSparklineStyles(snap MetricsSnapshot) []string {
	if len(snap.ErrorHistory) == 0 {
		return nil
	}
	// Use a fixed ceiling of 10% so small fluctuations don't look chaotic.
	// If the actual max exceeds 10%, scale to that instead.
	max := 10.0
	for _, pt := range snap.ErrorHistory {
		if pt.Value > max {
			max = pt.Value
		}
	}
	max *= 1.1
	styles := make([]string, len(snap.ErrorHistory))
	for i, pt := range snap.ErrorHistory {
		size := pt.Value / max
		start := 0.0
		if i > 0 {
			start = snap.ErrorHistory[i-1].Value / max
		}
		styles[i] = fmt.Sprintf("--start: %s; --size: %s", fmtSize(start), fmtSize(size))
	}
	return styles
}

func throughputInStyles(snap MetricsSnapshot) []string {
	max := maxOrOne(snap.MaxNetwork)
	styles := make([]string, len(snap.Network))
	for i, pt := range snap.Network {
		size := pt.InMBps / max
		start := 0.0
		if i > 0 {
			start = snap.Network[i-1].InMBps / max
		}
		styles[i] = fmt.Sprintf("--start: %s; --size: %s", fmtSize(start), fmtSize(size))
	}
	return styles
}

func throughputOutStyles(snap MetricsSnapshot) []string {
	max := maxOrOne(snap.MaxNetwork)
	styles := make([]string, len(snap.Network))
	for i, pt := range snap.Network {
		size := pt.OutMBps / max
		start := 0.0
		if i > 0 {
			start = snap.Network[i-1].OutMBps / max
		}
		styles[i] = fmt.Sprintf("--start: %s; --size: %s", fmtSize(start), fmtSize(size))
	}
	return styles
}

// diskIOStyle holds per-point styles for Read and Write series.
type diskIOStyle struct {
	ReadStyle  string
	WriteStyle string
}

func diskIOStyles(snap MetricsSnapshot) []diskIOStyle {
	max := maxOrOne(snap.MaxDiskIO)
	styles := make([]diskIOStyle, len(snap.DiskIO))
	for i, pt := range snap.DiskIO {
		styles[i] = diskIOStyle{
			ReadStyle:  fmt.Sprintf("--size: %s", fmtSize(pt.ReadMBps/max)),
			WriteStyle: fmt.Sprintf("--size: %s", fmtSize(pt.WriteMBps/max)),
		}
	}
	return styles
}

// statusDistStyle holds the style and label for one status class bar.
type statusDistStyle struct {
	Label string
	Style string
	Count int
}

func statusDistStyles(snap MetricsSnapshot) []statusDistStyle {
	d := snap.StatusDist
	max := d.S2xx
	if d.S3xx > max {
		max = d.S3xx
	}
	if d.S4xx > max {
		max = d.S4xx
	}
	if d.S5xx > max {
		max = d.S5xx
	}
	m := float64(maxOrOne(float64(max)))
	return []statusDistStyle{
		{"2xx", fmt.Sprintf("--size: %s; --color: #34d399", fmtSize(float64(d.S2xx)/m)), d.S2xx},
		{"3xx", fmt.Sprintf("--size: %s; --color: #38bdf8", fmtSize(float64(d.S3xx)/m)), d.S3xx},
		{"4xx", fmt.Sprintf("--size: %s; --color: #fbbf24", fmtSize(float64(d.S4xx)/m)), d.S4xx},
		{"5xx", fmt.Sprintf("--size: %s; --color: #f87171", fmtSize(float64(d.S5xx)/m)), d.S5xx},
	}
}

func svcLatestLatency(svc ServiceLatency) float64 {
	if len(svc.History) == 0 {
		return 0
	}
	return svc.History[len(svc.History)-1]
}

func svcLatencyBarStyle(svc ServiceLatency, maxMs float64) string {
	lat := svcLatestLatency(svc)
	max := maxOrOne(maxMs)
	size := lat / max
	color := "#34d399" // green
	if lat > 150 {
		color = "#f87171" // red
	} else if lat > 80 {
		color = "#fbbf24" // amber
	}
	return fmt.Sprintf("--size: %s; --color: %s", fmtSize(size), color)
}

