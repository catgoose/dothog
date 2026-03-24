// Package health provides a structured /health endpoint for ops monitoring.
package health

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Response is the public health metadata returned by /health.
type Response struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Version  string `json:"version"`
	Uptime   string `json:"uptime"`
	Database string `json:"database,omitempty"`
	Stats    any    `json:"stats,omitempty"`
}

// StatsFunc returns app-specific metrics for the health response.
// Derived apps implement this to add their own stats (counts, queue depths, etc.).
type StatsFunc func(ctx context.Context) any

// Pinger is satisfied by *sql.DB and *sqlx.DB.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Config holds the dependencies for building a health response.
type Config struct {
	Name      string
	Version   string
	StartTime time.Time
	DB        Pinger    // nil = skip DB check
	Stats     StatsFunc // nil = no stats
}

// Check builds a health response by pinging the database (if configured)
// and collecting app-specific stats.
func Check(ctx context.Context, cfg Config) Response {
	h := Response{
		Name:    cfg.Name,
		Version: cfg.Version,
		Uptime:  formatUptime(time.Since(cfg.StartTime)),
	}

	if cfg.DB != nil {
		if err := cfg.DB.PingContext(ctx); err != nil {
			h.Status = "degraded"
			h.Database = "disconnected"
			return h
		}
		h.Status = "healthy"
		h.Database = "connected"
	} else {
		h.Status = "healthy"
	}

	if cfg.Stats != nil {
		h.Stats = cfg.Stats(ctx)
	}

	return h
}

// NewPingerFromDB wraps a *sql.DB as a Pinger.
func NewPingerFromDB(db *sql.DB) Pinger {
	return db
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd%dh%dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
