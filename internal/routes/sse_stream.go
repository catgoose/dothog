package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// startSSEResponse sets the standard SSE response headers, writes the
// status line, and returns the flusher. On non-flusher writers it returns
// an error the caller can return directly.
//
// Demo SSE handlers across the realtime section repeated this seven-line
// boilerplate. Centralizing it lets each handler express only its own
// subscribe + read-loop logic.
func startSSEResponse(c echo.Context) (http.Flusher, error) {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)
	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming unsupported")
	}
	return flusher, nil
}
