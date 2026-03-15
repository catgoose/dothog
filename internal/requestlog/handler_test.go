package requestlog

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"catgoose/dothog/internal/shared"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// discardHandler is a slog.Handler that records whether Handle was called.
type discardHandler struct {
	handled bool
}

func (d *discardHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (d *discardHandler) Handle(_ context.Context, _ slog.Record) error {
	d.handled = true
	return nil
}
func (d *discardHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return d }
func (d *discardHandler) WithGroup(name string) slog.Handler       { return d }

func TestHandler_CapturesEntries_WhenRequestIDInContext(t *testing.T) {
	inner := &discardHandler{}
	h := NewHandler(inner)

	ctx := context.WithValue(context.Background(), shared.RequestIDKeyValue, "req-123")
	ctx = NewBufferContext(ctx)

	rec := slog.NewRecord(time.Now(), slog.LevelError, "something broke", 0)
	rec.AddAttrs(slog.String("component", "auth"))

	err := h.Handle(ctx, rec)
	require.NoError(t, err)

	buf := GetBuffer(ctx)
	require.NotNil(t, buf)
	require.Len(t, buf.Entries, 1)
	assert.Equal(t, "something broke", buf.Entries[0].Message)
	assert.Equal(t, "ERROR", buf.Entries[0].Level)
	assert.Contains(t, buf.Entries[0].Attrs, "component=auth")
}

func TestHandler_DoesNotCapture_WhenNoRequestID(t *testing.T) {
	inner := &discardHandler{}
	h := NewHandler(inner)

	ctx := NewBufferContext(context.Background())

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "no request id", 0)
	err := h.Handle(ctx, rec)
	require.NoError(t, err)

	buf := GetBuffer(ctx)
	require.NotNil(t, buf)
	assert.Empty(t, buf.Entries)
}

func TestHandler_DelegatesToInnerHandler(t *testing.T) {
	inner := &discardHandler{}
	h := NewHandler(inner)

	ctx := context.Background()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)

	err := h.Handle(ctx, rec)
	require.NoError(t, err)
	assert.True(t, inner.handled)
}

func TestHandler_WithAttrs_SetsRequestID(t *testing.T) {
	inner := &discardHandler{}
	h := NewHandler(inner)

	// Attach request_id via WithAttrs — handler should capture without context key.
	h2 := h.WithAttrs([]slog.Attr{slog.String("request_id", "req-via-attrs")}).(*Handler)

	ctx := NewBufferContext(context.Background())
	rec := slog.NewRecord(time.Now(), slog.LevelWarn, "warning msg", 0)

	err := h2.Handle(ctx, rec)
	require.NoError(t, err)

	buf := GetBuffer(ctx)
	require.NotNil(t, buf)
	require.Len(t, buf.Entries, 1)
	assert.Equal(t, "warning msg", buf.Entries[0].Message)
	assert.Equal(t, "WARN", buf.Entries[0].Level)
}
