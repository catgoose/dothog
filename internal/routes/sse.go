package routes

import (
	"time"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

// defaultSSEHeartbeat is the keepalive cadence streamSSE applies when a caller
// does not pass its own tavern.WithStreamHeartbeat. It sits under the common
// 60s proxy/browser idle cutoff so quiet connections are not dropped.
const defaultSSEHeartbeat = 30 * time.Second

// streamSSE is the route-owned wrapper around tavern.StreamSSE for handlers
// that stream a subscription channel directly rather than through
// broker.SSEHandler. It marks the response unbuffered for reverse proxies
// (X-Accel-Buffering: no) and applies defaultSSEHeartbeat, then defers to
// tavern for the SSE write mechanics.
//
// Callers still pass tavern.StreamSSEOption values for snapshots, resume
// frames, a custom frame writer, or a different heartbeat; a caller-supplied
// WithStreamHeartbeat overrides the default. The wrapper deliberately owns
// neither subscription nor filter nor replay choices — those stay at the call
// site, so a handler with fan-in or replay logic keeps it visible instead of
// hidden behind the helper.
func streamSSE[T any](
	c echo.Context,
	ch <-chan T,
	encode func(T) string,
	opts ...tavern.StreamSSEOption,
) error {
	c.Response().Header().Set("X-Accel-Buffering", "no")
	merged := append([]tavern.StreamSSEOption{tavern.WithStreamHeartbeat(defaultSSEHeartbeat)}, opts...)
	return tavern.StreamSSE(c.Request().Context(), c.Response(), ch, encode, merged...)
}
