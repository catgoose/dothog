// Package admininfo provides types for admin dashboard pages.
package admininfo

// SystemInfo holds runtime stats for the admin system page.
type SystemInfo struct {
	GoVersion  string
	OS         string
	Arch       string
	NumCPU     int
	Goroutines int
	NumThread  int
	Uptime     string

	HeapAllocMB  string
	HeapSysMB    string
	StackInUseMB string
	SysMB        string
	TotalAllocMB string

	GCCycles        uint32
	LastPauseMicros uint64
	NextGCMB        string
	HeapObjects     uint64
	LiveObjects     uint64
}

// ConfigEntry is a single key/value pair for display, with optional masking.
type ConfigEntry struct {
	Key   string
	Value string
}
