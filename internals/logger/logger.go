// Package logger provides structured logging utilities
// It uses Go's slog package to provide environment-aware logging with JSON format.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"catgoose/harmony/internals/shared"

	"github.com/catgoose/dio"
	"gopkg.in/natefinch/lumberjack.v2"
)

// HandlerWrapper is a function that wraps a slog.Handler (e.g. to add capturing).
type HandlerWrapper func(slog.Handler) slog.Handler

var (
	logger         *slog.Logger
	mu             sync.RWMutex
	once           sync.Once
	handlerWrapper HandlerWrapper
)

// SetHandlerWrapper registers a function that will wrap the slog.Handler
// during Init. Must be called before Init (or Get).
func SetHandlerWrapper(w HandlerWrapper) {
	mu.Lock()
	defer mu.Unlock()
	handlerWrapper = w
}

const appLogFile = "harmony.log"

// Init initializes the global logger with appropriate configuration
func Init() {
	once.Do(func() {
		// Ensure log directory exists
		logDir := "log"
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		}

		// Get log level from environment or use default
		logLevel := getLogLevel()

		// Create lumberjack rotator for log file with rotation
		logPath := filepath.Join(logDir, appLogFile)
		rotator := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    0,    // No size-based rotation (use time-based only)
			MaxBackups: 12,   // Keep 12 compressed backups
			MaxAge:     30,   // Rotate monthly (30 days)
			Compress:   true, // Compress rotated files
		}

		// Create multi-writer for both file and console output
		var logWriter io.Writer
		if dio.Dev() {
			logWriter = io.MultiWriter(os.Stdout, rotator)
		} else {
			logWriter = io.MultiWriter(os.Stderr, rotator)
		}

		// Create handler with multi-writer
		opts := &slog.HandlerOptions{
			Level:     logLevel,
			AddSource: dio.Dev(),
		}
		var handler slog.Handler = slog.NewJSONHandler(logWriter, opts)
		if handlerWrapper != nil {
			handler = handlerWrapper(handler)
		}

		mu.Lock()
		logger = slog.New(handler).With("runtime_id", shared.RuntimeID)
		mu.Unlock()
		slog.SetDefault(logger)
	})
}

// getLogLevel returns the log level based on environment variable
func getLogLevel() slog.Level {
	levelStr, err := dio.Env("LOG_LEVEL")
	if err != nil {
		if dio.Dev() {
			return slog.LevelDebug
		}
		return slog.LevelInfo
	}

	switch levelStr {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		// Log invalid level and use default
		if dio.Dev() {
			return slog.LevelDebug
		}
		return slog.LevelInfo
	}
}

// Get returns the global logger instance
func Get() *slog.Logger {
	mu.RLock()
	if logger != nil {
		defer mu.RUnlock()
		return logger
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()
	if logger == nil {
		Init()
	}
	return logger
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

// Fatal logs a fatal message and exits the application
func Fatal(msg string, args ...any) {
	Get().Error(msg, args...)
	os.Exit(1)
}

// WithContext returns a logger with context values
func WithContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return Get()
	}

	// Extract common context values
	args := make([]any, 0)

	// Add request ID if available
	if requestID := ctx.Value(shared.RequestIDKeyValue); requestID != nil {
		args = append(args, "request_id", requestID)
	}

	// Add context ID if available
	if contextID := ctx.Value(shared.ContextIDKeyValue); contextID != nil {
		args = append(args, "context_id", contextID)
	}

	// Add context description if available
	if contextDescription := ctx.Value(shared.ContextDescriptionKeyValue); contextDescription != nil {
		args = append(args, "context_description", contextDescription)
	}

	if len(args) > 0 {
		return Get().With(args...)
	}

	return Get()
}

// With returns a logger with the given attributes
func With(args ...any) *slog.Logger {
	return Get().With(args...)
}

// WithGroup returns a logger with the given group
func WithGroup(name string) *slog.Logger {
	return Get().WithGroup(name)
}

// LogAndReturnError logs an error and returns a formatted error
// This is useful for functions that need to log errors and return them
func LogAndReturnError(message string, err error) error {
	Get().Error(message, "error", err)
	return fmt.Errorf("%s: %w", message, err)
}

// LogAndReturnErrorf logs an error and returns a formatted error with additional context
func LogAndReturnErrorf(message string, err error, args ...any) error {
	Get().Error(fmt.Sprintf(message, args...), "error", err)
	return fmt.Errorf("%s: %w", fmt.Sprintf(message, args...), err)
}

// LogError logs an error without returning it
// This is useful for functions that handle errors internally
func LogError(message string, err error) {
	Get().Error(message, "error", err)
}

// LogErrorf logs an error with formatted message without returning it
func LogErrorf(message string, err error, args ...any) {
	Get().Error(fmt.Sprintf(message, args...), "error", err)
}

// LogErrorWithFields logs an error with additional structured fields
func LogErrorWithFields(message string, err error, fields map[string]any) {
	args := make([]any, 0, len(fields)*2+2)
	args = append(args, "error", err)
	for k, v := range fields {
		args = append(args, k, v)
	}
	Get().Error(message, args...)
}
