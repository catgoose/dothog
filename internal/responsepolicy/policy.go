// Package responsepolicy owns the always-on web-standards response policy for
// the dothog scaffold: 103 Early Hints / Link preload, Server-Timing, dorman
// security headers, and Vary on HX-Request. Behavior is implicit baseline
// framework infrastructure — it has no setup feature toggle — and lives in
// one explicit owner instead of being scattered across routes.go and
// middleware helper files.
package responsepolicy

import (
	"github.com/catgoose/dorman"
	"github.com/labstack/echo/v4"
)

// Config exposes the deployment-level knobs the always-on response policy
// needs to honor. PreloadLinks lets the host app declare which assets the
// browser should fetch ahead of body bytes (sent via 103 Early Hints when the
// transport supports it, or via Link headers on the final response when not).
// BehindProxy disables 103 hints because intermediaries (the templ proxy in
// `mage watch`, certain reverse proxies) mangle 1xx framing.
type Config struct {
	Security     dorman.SecurityHeadersConfig
	PreloadLinks []string
	BehindProxy  bool
}

// Install registers the full web-standards middleware chain on e in the
// canonical order: Preload → Server-Timing → SecurityHeaders → Vary. Call
// once from InitEcho before any other application middleware. Recover,
// correlation, request logging, compression, auth/session/csrf, and the
// HTTPErrorHandler stay in InitEcho — they are not response-policy concerns.
func Install(e *echo.Echo, cfg Config) {
	if len(cfg.PreloadLinks) > 0 {
		e.Use(Preload(cfg.PreloadLinks, cfg.BehindProxy))
	}
	e.Use(ServerTiming())
	e.Use(echo.WrapMiddleware(dorman.SecurityHeaders(cfg.Security)))
	e.Use(Vary("HX-Request"))
}
