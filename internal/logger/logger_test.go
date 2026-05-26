package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"catgoose/dothog/internal/shared"

	"github.com/catgoose/promolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetLogger clears the package singletons so the next Init/Get fully rebuilds.
func resetLogger() {
	mu.Lock()
	logger = nil
	once = sync.Once{}
	handlerWrapper = nil
	mu.Unlock()
}

// captureLogger installs a JSON-backed test logger that writes into buf and
// restores the package-level logger on test cleanup. Use this to assert on
// log output without depending on the file/stdout sinks Init wires.
func captureLogger(t *testing.T, buf *bytes.Buffer) {
	t.Helper()
	handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})

	mu.Lock()
	prev := logger
	logger = slog.New(handler)
	mu.Unlock()

	t.Cleanup(func() {
		mu.Lock()
		logger = prev
		mu.Unlock()
	})
}

func parseRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	return rec
}

func TestInit_IsIdempotent(t *testing.T) {
	resetLogger()
	Init()
	first := Get()
	Init()
	second := Get()
	assert.Same(t, first, second, "Init guarded by sync.Once must produce a single logger across repeated calls")
}

func TestGet_ReturnsSameInstance(t *testing.T) {
	resetLogger()
	Init()
	a := Get()
	b := Get()
	assert.Same(t, a, b, "Get must return the same logger every call")
}

func TestInit_DoesNotCreateFileSinkUnderGoTest(t *testing.T) {
	resetLogger()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	tempDir := t.TempDir()
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(cwd))
	})

	Init()

	_, err = os.Stat(filepath.Join(tempDir, "log", appLogFile))
	require.True(t, os.IsNotExist(err), "go test should not create package-local log/dothog.log files")
}

func TestSetHandlerWrapper_AppliedDuringInit(t *testing.T) {
	resetLogger()

	called := false
	SetHandlerWrapper(func(h slog.Handler) slog.Handler {
		called = true
		return h
	})
	t.Cleanup(func() {
		mu.Lock()
		handlerWrapper = nil
		mu.Unlock()
	})

	Init()
	assert.True(t, called, "SetHandlerWrapper-supplied wrapper must be invoked during Init")
}

func TestLogLevels_DoNotPanic(t *testing.T) {
	resetLogger()
	Init()
	assert.NotPanics(t, func() {
		Debug("debug message")
		Info("info message")
		Warn("warn message")
		Error("error message")
	})
}

func TestWithContext_NilContextSafe(t *testing.T) {
	resetLogger()
	Init()
	// WithContext explicitly tolerates a nil context (see logger.go); the test
	// pins that contract.
	//nolint:staticcheck // SA1012: intentional nil-context test for documented fallback.
	assert.NotNil(t, WithContext(nil), "nil context should fall back to the base logger without panicking")
}

func TestWithContext_EmptyContextReturnsBaseLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	captureLogger(t, buf)

	WithContext(context.Background()).Info("hello")

	rec := parseRecord(t, buf)
	assert.Equal(t, "hello", rec["msg"])
	_, hasRequestID := rec["request_id"]
	_, hasContextID := rec["context_id"]
	_, hasContextDescription := rec["context_description"]
	assert.False(t, hasRequestID, "empty context must not produce request_id")
	assert.False(t, hasContextID, "empty context must not produce context_id")
	assert.False(t, hasContextDescription, "empty context must not produce context_description")
}

func TestWithContext_AttachesRequestID(t *testing.T) {
	buf := &bytes.Buffer{}
	captureLogger(t, buf)

	ctx := context.WithValue(context.Background(), promolog.RequestIDKey, "req-42")
	WithContext(ctx).Info("hello")

	rec := parseRecord(t, buf)
	assert.Equal(t, "req-42", rec["request_id"])
}

func TestWithContext_AttachesContextID(t *testing.T) {
	buf := &bytes.Buffer{}
	captureLogger(t, buf)

	ctx := context.WithValue(context.Background(), shared.ContextIDKeyValue, "ctx-99")
	WithContext(ctx).Info("hello")

	rec := parseRecord(t, buf)
	assert.Equal(t, "ctx-99", rec["context_id"])
}

func TestWithContext_AttachesContextDescription(t *testing.T) {
	buf := &bytes.Buffer{}
	captureLogger(t, buf)

	ctx := context.WithValue(context.Background(), shared.ContextDescriptionKeyValue, "user cache sync")
	WithContext(ctx).Info("hello")

	rec := parseRecord(t, buf)
	assert.Equal(t, "user cache sync", rec["context_description"])
}

func TestWithContext_AttachesAllFields(t *testing.T) {
	buf := &bytes.Buffer{}
	captureLogger(t, buf)

	ctx := context.Background()
	ctx = context.WithValue(ctx, promolog.RequestIDKey, "req-1")
	ctx = context.WithValue(ctx, shared.ContextIDKeyValue, "ctx-1")
	ctx = context.WithValue(ctx, shared.ContextDescriptionKeyValue, "boot")
	WithContext(ctx).Info("hello")

	rec := parseRecord(t, buf)
	assert.Equal(t, "req-1", rec["request_id"])
	assert.Equal(t, "ctx-1", rec["context_id"])
	assert.Equal(t, "boot", rec["context_description"])
}

func TestWith_AddsAttribute(t *testing.T) {
	buf := &bytes.Buffer{}
	captureLogger(t, buf)

	With("key", "value").Info("hello")

	rec := parseRecord(t, buf)
	assert.Equal(t, "value", rec["key"])
}

func TestWithGroup_NestsAttributes(t *testing.T) {
	buf := &bytes.Buffer{}
	captureLogger(t, buf)

	WithGroup("section").Info("hello", "field", "value")

	rec := parseRecord(t, buf)
	section, ok := rec["section"].(map[string]any)
	require.True(t, ok, "WithGroup must nest subsequent attributes under the named group: %v", rec)
	assert.Equal(t, "value", section["field"])
}

func TestThreadSafety_ConcurrentGet(t *testing.T) {
	resetLogger()
	Init()

	const workers = 10
	done := make(chan struct{}, workers)
	for range workers {
		go func() {
			defer func() { done <- struct{}{} }()
			assert.NotNil(t, Get())
			Info("concurrent log message")
		}()
	}
	for range workers {
		<-done
	}
}
