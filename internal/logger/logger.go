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

	"catgoose/dothog/internal/shared"

	"github.com/catgoose/promolog"

	"catgoose/dothog/internal/env"
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

// SetHandlerWrapper installs a wrapper applied to the slog.Handler during Init;
// must be called before Init (or Get).
func SetHandlerWrapper(w HandlerWrapper) {
	mu.Lock()
	defer mu.Unlock()
	handlerWrapper = w
}

const appLogFile = "dothog.log"

// Init wires a JSON slog logger to log/dothog.log (monthly lumberjack rotation) plus stdout/stderr; idempotent via sync.Once.
func Init() {
	once.Do(func() {
		logDir := "log"
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		}

		logLevel := getLogLevel()

		logPath := filepath.Join(logDir, appLogFile)
		rotator := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    0,    // No size-based rotation (use time-based only)
			MaxBackups: 12,   // Keep 12 compressed backups
			MaxAge:     30,   // Rotate monthly (30 days)
			Compress:   true, // Compress rotated files
		}

		var logWriter io.Writer
		if env.Dev() {
			logWriter = io.MultiWriter(os.Stdout, rotator)
		} else {
			logWriter = io.MultiWriter(os.Stderr, rotator)
		}

		opts := &slog.HandlerOptions{
			Level:     logLevel,
			AddSource: env.Dev(),
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

// getLogLevel reads LOG_LEVEL; falls back to Debug in dev and Info otherwise.
func getLogLevel() slog.Level {
	levelStr, err := env.Get("LOG_LEVEL")
	if err != nil {
		if env.Dev() {
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
		if env.Dev() {
			return slog.LevelDebug
		}
		return slog.LevelInfo
	}
}

// Get is the global logger, triggering Init on first use.
func Get() *slog.Logger {
	mu.RLock()
	if logger != nil {
		defer mu.RUnlock()
		return logger
	}
	mu.RUnlock()

	Init()

	mu.RLock()
	defer mu.RUnlock()
	return logger
}

// Debug logs at Debug level on the package logger.
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

// Info logs at Info level on the package logger.
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Warn logs at Warn level on the package logger.
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error logs at Error level on the package logger.
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

// Fatal logs at Error level and then calls os.Exit(1).
func Fatal(msg string, args ...any) {
	Get().Error(msg, args...)
	os.Exit(1)
}

// WithContext attaches request_id, context_id, and context_description from ctx when present.
func WithContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return Get()
	}

	args := make([]any, 0)

	if requestID := ctx.Value(promolog.RequestIDKey); requestID != nil {
		args = append(args, "request_id", requestID)
	}

	if contextID := ctx.Value(shared.ContextIDKeyValue); contextID != nil {
		args = append(args, "context_id", contextID)
	}

	if contextDescription := ctx.Value(shared.ContextDescriptionKeyValue); contextDescription != nil {
		args = append(args, "context_description", contextDescription)
	}

	if len(args) > 0 {
		return Get().With(args...)
	}

	return Get()
}

// With derives a logger carrying args as default attributes.
func With(args ...any) *slog.Logger {
	return Get().With(args...)
}

// WithGroup nests all subsequent attributes under a group of the given name.
func WithGroup(name string) *slog.Logger {
	return Get().WithGroup(name)
}

// LogAndReturnError logs at Error and returns fmt.Errorf("%s: %w", message, err).
func LogAndReturnError(message string, err error) error {
	Get().Error(message, "error", err)
	return fmt.Errorf("%s: %w", message, err)
}

// LogAndReturnErrorf is LogAndReturnError with Printf-style formatting of message.
func LogAndReturnErrorf(message string, err error, args ...any) error {
	Get().Error(fmt.Sprintf(message, args...), "error", err)
	return fmt.Errorf("%s: %w", fmt.Sprintf(message, args...), err)
}

// LogError records err at Error level with message; use when the caller does
// not want to wrap and rethrow.
func LogError(message string, err error) {
	Get().Error(message, "error", err)
}

// LogErrorf is LogError with Printf-style formatting of message.
func LogErrorf(message string, err error, args ...any) {
	Get().Error(fmt.Sprintf(message, args...), "error", err)
}

// LogErrorWithFields logs at Error and flattens fields into key/value attribute pairs.
func LogErrorWithFields(message string, err error, fields map[string]any) {
	args := make([]any, 0, len(fields)*2+2)
	args = append(args, "error", err)
	for k, v := range fields {
		args = append(args, k, v)
	}
	Get().Error(message, args...)
}
