package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"catgoose/dothog/internal/logger"
	"github.com/catgoose/promolog"

	"github.com/labstack/echo/v4"
)

// generateRequestID generates a unique request ID
func generateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}

// RequestIDMiddleware generates a request ID for each request and adds it to the context
func RequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := generateRequestID()

			c.Response().Header().Set("X-Request-ID", requestID)

			ctx := context.WithValue(c.Request().Context(), promolog.RequestIDKey, requestID)
			ctx = promolog.NewBufferContext(ctx)
			c.SetRequest(c.Request().WithContext(ctx))

			start := time.Now()
			err := next(c)
			latency := time.Since(start)

			logger.WithContext(c.Request().Context()).Info("Request completed",
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", c.Response().Status,
				"latency_ms", latency.Milliseconds(),
			)

			return err
		}
	}
}

// GetRequestID retrieves the request ID from the Go context.
// The request ID is set by RequestIDMiddleware on every request.
func GetRequestID(c echo.Context) string {
	if id := c.Request().Context().Value(promolog.RequestIDKey); id != nil {
		if requestID, ok := id.(string); ok {
			return requestID
		}
	}
	return ""
}
