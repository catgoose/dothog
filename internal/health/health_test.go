package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// stubPinger satisfies Pinger with a configurable error so tests can drive
// both healthy and degraded paths without a real database.
type stubPinger struct {
	err error
}

func (s stubPinger) PingContext(_ context.Context) error {
	return s.err
}

func TestCheck_HealthyWithDB(t *testing.T) {
	cfg := Config{
		StartTime: time.Now().Add(-2 * time.Hour),
		DB:        stubPinger{},
		Name:      "dothog",
		Version:   "v0",
	}

	h := Check(context.Background(), cfg)
	assert.Equal(t, "healthy", h.Status)
	assert.Equal(t, "connected", h.Database)
	assert.Equal(t, "dothog", h.Name)
	assert.Nil(t, h.Stats)
}

func TestCheck_DegradedWhenDBPingFails(t *testing.T) {
	cfg := Config{
		StartTime: time.Now(),
		DB:        stubPinger{err: errors.New("ping failed")},
		Name:      "dothog",
	}

	h := Check(context.Background(), cfg)
	assert.Equal(t, "degraded", h.Status)
	assert.Equal(t, "disconnected", h.Database)
}

func TestCheck_StatsIncludedOnDegradedResponse(t *testing.T) {
	statsCalled := false
	cfg := Config{
		StartTime: time.Now(),
		DB:        stubPinger{err: errors.New("ping failed")},
		Stats: func(_ context.Context) any {
			statsCalled = true
			return map[string]any{"goroutines": 7}
		},
		Name: "dothog",
	}

	h := Check(context.Background(), cfg)
	assert.Equal(t, "degraded", h.Status)
	assert.True(t, statsCalled, "stats hook must run even when the DB ping fails")
	assert.NotNil(t, h.Stats, "Stats must populate on degraded responses so dashboards can still render runtime data")
}

func TestCheck_NoDBPathIsHealthy(t *testing.T) {
	cfg := Config{
		StartTime: time.Now(),
		Name:      "dothog",
	}

	h := Check(context.Background(), cfg)
	assert.Equal(t, "healthy", h.Status)
	assert.Empty(t, h.Database, "Database field must stay empty when no Pinger is configured")
}

func TestCheck_StatsIncludedOnHealthyResponse(t *testing.T) {
	cfg := Config{
		StartTime: time.Now(),
		DB:        stubPinger{},
		Stats: func(_ context.Context) any {
			return "ok"
		},
		Name: "dothog",
	}

	h := Check(context.Background(), cfg)
	assert.Equal(t, "healthy", h.Status)
	assert.Equal(t, "ok", h.Stats)
}

func TestCheck_NoStatsKeepsResponseFieldNil(t *testing.T) {
	cfg := Config{StartTime: time.Now()}

	h := Check(context.Background(), cfg)
	assert.Nil(t, h.Stats)
}
