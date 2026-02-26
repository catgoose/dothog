// setup:feature:demo

package views

import (
	"fmt"
	"math"
	"time"

	"catgoose/go-htmx-template/internals/ssebroker"
)

// --- Dashboard data types ---

// NetworkPoint is a single data point for network traffic.
type NetworkPoint struct {
	InMBps  float64
	OutMBps float64
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
}

const networkSize = 15

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

// serviceStatusBadge returns a DaisyUI badge class for the service status.
func serviceStatusBadge(s ServiceStatus) string {
	switch s.Status {
	case "critical":
		return "badge-error"
	case "degraded":
		return "badge-warning"
	default:
		return "badge-success"
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

// statsSections returns the ordered section names.
func statsSections() []string {
	return []string{"Runtime", "Memory", "GC", "Allocator"}
}

// statsEntriesBySection groups entries by their section.
func statsEntriesBySection(entries []statsEntry) map[string][]statsEntry {
	m := make(map[string][]statsEntry)
	for _, e := range entries {
		m[e.Section] = append(m[e.Section], e)
	}
	return m
}
