// Package admininfo provides types for admin dashboard pages.
package admininfo

// SystemInfo holds runtime stats for the admin system page.
type SystemInfo struct {
	Version    string
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

// UserPreferences holds per-session user preferences.
// These are stored in-memory keyed by session cookie; applications should
// persist them to their own database when needed.
type UserPreferences struct {
	PageSize             int
	DateFormat           string
	CompactTables        bool
	EmailOnError         bool
	DesktopNotifications bool
	ReduceMotion         bool
	HighContrast         bool
}

// DefaultUserPreferences returns sensible defaults.
func DefaultUserPreferences() UserPreferences {
	return UserPreferences{
		PageSize:   20,
		DateFormat: "relative",
	}
}
