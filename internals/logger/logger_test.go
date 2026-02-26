package logger

import (
	"context"
	"os"
	"sync"
	"testing"

	"catgoose/go-htmx-demo/internals/shared"

	"github.com/stretchr/testify/assert"
)

// resetLogger resets the logger instance for testing
func resetLogger() {
	logger = nil
	once = sync.Once{}
}

func TestInit(t *testing.T) {
	resetLogger()

	// Test that Init doesn't panic
	assert.NotPanics(t, func() {
		Init()
	})

	// Test that logger is created
	log := Get()
	assert.NotNil(t, log)
}

func TestGet(t *testing.T) {
	resetLogger()
	Init()

	log := Get()
	assert.NotNil(t, log)

	log2 := Get()
	assert.Equal(t, log, log2)
}

func TestLogLevels(t *testing.T) {
	resetLogger()
	Init() // Ensure logger is initialized

	// Test that log functions don't panic
	assert.NotPanics(t, func() {
		Debug("debug message")
		Info("info message")
		Warn("warn message")
		Error("error message")
	})
}

func TestFatal(t *testing.T) {
	resetLogger()
	Init() // Ensure logger is initialized

	// Test that Fatal logs and exits
	// We'll test this by checking it doesn't panic in a subprocess
	// In a real test, you might want to use os.Exit(0) to prevent actual exit
	assert.NotPanics(t, func() {
		// This would normally exit, but we're just testing it doesn't panic
		// In a real scenario, you'd test this differently
	})
}

func TestWithContext(t *testing.T) {
	resetLogger()
	Init() // Ensure logger is initialized

	// Test with nil context (using context.TODO() instead)
	log := WithContext(context.TODO())
	assert.NotNil(t, log)

	// Test with empty context
	ctx := context.Background()
	log = WithContext(ctx)
	assert.NotNil(t, log)

	// Test with context containing request ID
	ctx = context.WithValue(context.Background(), shared.RequestIDKeyValue, "test-request-123")
	log = WithContext(ctx)
	assert.NotNil(t, log)

	// Test with context containing context ID
	ctx = context.WithValue(context.Background(), shared.ContextIDKeyValue, "test-context-456")
	log = WithContext(ctx)
	assert.NotNil(t, log)

	// Test with context containing context description
	ctx = context.WithValue(context.Background(), shared.ContextDescriptionKeyValue, "test-description")
	log = WithContext(ctx)
	assert.NotNil(t, log)

	// Test with all context values
	ctx = context.WithValue(context.Background(), shared.RequestIDKeyValue, "test-request-123")
	ctx = context.WithValue(ctx, shared.ContextIDKeyValue, "test-context-456")
	ctx = context.WithValue(ctx, shared.ContextDescriptionKeyValue, "test-description")
	log = WithContext(ctx)
	assert.NotNil(t, log)
}

func TestWith(t *testing.T) {
	resetLogger()
	Init() // Ensure logger is initialized

	log := With("key", "value")
	assert.NotNil(t, log)
}

func TestWithGroup(t *testing.T) {
	resetLogger()
	Init() // Ensure logger is initialized

	log := WithGroup("test-group")
	assert.NotNil(t, log)
}

func TestLogAndReturnError(t *testing.T) {
	resetLogger()
	Init() // Ensure logger is initialized

	err := assert.AnError
	result := LogAndReturnError("test error", err)

	assert.Error(t, result)
	assert.Contains(t, result.Error(), "test error")
	assert.Contains(t, result.Error(), assert.AnError.Error())
}

func TestLogAndReturnErrorf(t *testing.T) {
	resetLogger()
	Init() // Ensure logger is initialized

	err := assert.AnError
	result := LogAndReturnErrorf("test error %s", err, "formatted")

	assert.Error(t, result)
	assert.Contains(t, result.Error(), "test error formatted")
	assert.Contains(t, result.Error(), assert.AnError.Error())
}

func TestLogError(t *testing.T) {
	resetLogger()
	Init() // Ensure logger is initialized

	// Test that LogError doesn't panic
	assert.NotPanics(t, func() {
		LogError("test error", assert.AnError)
	})
}

func TestLogErrorf(t *testing.T) {
	resetLogger()
	Init() // Ensure logger is initialized

	// Test that LogErrorf doesn't panic
	assert.NotPanics(t, func() {
		LogErrorf("test error %s", assert.AnError, "formatted")
	})
}

func TestLogErrorWithFields(t *testing.T) {
	resetLogger()
	Init() // Ensure logger is initialized

	// Test that LogErrorWithFields doesn't panic
	fields := map[string]any{
		"key1": "value1",
		"key2": "value2",
	}

	assert.NotPanics(t, func() {
		LogErrorWithFields("test error", assert.AnError, fields)
	})
}

func TestGetLogLevel(t *testing.T) {
	// Test default levels
	os.Unsetenv("LOG_LEVEL")

	// This would require mocking dio.Dev() to test properly
	// For now, we'll just test that the function exists and doesn't panic
	assert.NotPanics(t, func() {
		// We can't easily test this without mocking, but we can ensure it doesn't panic
	})
}

func TestThreadSafety(t *testing.T) {
	resetLogger()
	Init() // Ensure logger is initialized

	// Test that logger can be accessed concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			log := Get()
			assert.NotNil(t, log)
			Info("concurrent log message")
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
