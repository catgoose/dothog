package handler

import (
	"strings"

	corecomponents "catgoose/dothog/web/components/core"

	"github.com/labstack/echo/v4"
)

// Capability header names. Wire side, kept neutral so the JS hook in
// web/assets/public/js/app/htmx.error-capabilities.js and the server-side
// parser stay in sync without leaking framework specifics.
const (
	HeaderErrorAcceptSurfaces  = "X-Error-Accept-Surfaces"
	HeaderErrorFallbackSurface = "X-Error-Fallback-Surface"
)

// ErrorCapabilities is the client's per-request declaration of which error
// surfaces the triggering DOM context can honestly accept and which one to
// degrade to when none of the accepted set matches. An empty Accept list
// means "no capability advertised" and callers should fall through to the
// server's defaults rather than guessing.
type ErrorCapabilities struct {
	Fallback corecomponents.ErrorSurface
	Accept   []corecomponents.ErrorSurface
}

// HasAdvertised reports whether the client sent any capability headers at
// all. Used to distinguish "no advertisement" from "advertisement with empty
// values" so old clients keep their pre-negotiation behavior.
func (cap ErrorCapabilities) HasAdvertised() bool {
	return len(cap.Accept) > 0 || cap.Fallback != ""
}

// accepts reports whether the client's Accept list includes s.
func (cap ErrorCapabilities) accepts(s corecomponents.ErrorSurface) bool {
	for _, a := range cap.Accept {
		if a == s {
			return true
		}
	}
	return false
}

// ParseErrorCapabilities reads the capability headers off c. Returns an empty
// value when no headers are present so callers can short-circuit to existing
// defaults.
func ParseErrorCapabilities(c echo.Context) ErrorCapabilities {
	header := c.Request().Header
	return ErrorCapabilities{
		Accept:   parseSurfaceList(header.Get(HeaderErrorAcceptSurfaces)),
		Fallback: corecomponents.ErrorSurface(strings.TrimSpace(header.Get(HeaderErrorFallbackSurface))),
	}
}

func parseSurfaceList(raw string) []corecomponents.ErrorSurface {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]corecomponents.ErrorSurface, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		out = append(out, corecomponents.ErrorSurface(s))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NegotiateSurface picks the actual render surface for an HTMX request that
// advertised capabilities. The server's chosen surface wins when the client
// accepts it; otherwise the client's Fallback applies; otherwise the safe
// banner default falls out. When the client advertised nothing the chosen
// surface passes through untouched (callers should not invoke this in that
// case — see HasAdvertised).
func NegotiateSurface(chosen corecomponents.ErrorSurface, caps ErrorCapabilities) corecomponents.ErrorSurface {
	if !caps.HasAdvertised() {
		return chosen
	}
	if caps.accepts(chosen) {
		return chosen
	}
	if caps.Fallback != "" {
		return caps.Fallback
	}
	return corecomponents.SurfaceBanner
}
