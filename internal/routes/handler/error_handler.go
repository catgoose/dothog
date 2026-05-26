package handler

import (
	"context"
	"errors"
	"net/http"

	"catgoose/dothog/internal/htmxutil"
	"catgoose/dothog/internal/logger"
	"catgoose/dothog/internal/routes/middleware"
	// setup:feature:session_settings:start
	"catgoose/dothog/internal/session"
	// setup:feature:session_settings:end
	corecomponents "catgoose/dothog/web/components/core"
	"github.com/catgoose/linkwell"
	"github.com/catgoose/promolog"

	"github.com/labstack/echo/v4"
)

// handleError logs the error and renders the appropriate surface. HTMX
// requests get a banner OOB swap to #error-status; non-HTMX requests get the
// standalone document surface (auth/system boundary shell).
func handleError(c echo.Context, statusCode int, message string, err error) error {
	if errors.Is(c.Request().Context().Err(), context.Canceled) {
		return nil
	}

	requestID := promolog.GetRequestID(c.Request().Context())
	log := logger.WithContext(c.Request().Context()).With(
		"status_code", statusCode,
		"message", message,
		"route", c.Request().URL.Path,
		"method", c.Request().Method,
	)
	log.Error("Request error", "error", err)

	opts := linkwell.ErrorControlOpts{HomeURL: "/", LoginURL: "/login"}
	controls := linkwell.ErrorControlsForStatus(statusCode, opts)
	if statusCode >= 500 {
		controls = append(controls, linkwell.ReportIssueButton(linkwell.LabelReportIssue, requestID))
	}

	if htmxutil.IsHTMX(c.Request()) {
		ec := linkwell.ErrorContext{
			StatusCode: statusCode,
			Message:    message,
			Err:        err,
			Route:      c.Request().URL.Path,
			RequestID:  requestID,
			Closable:   true,
			Controls:   controls,
		}
		if ec.OOBTarget == "" {
			ec.OOBTarget = linkwell.DefaultErrorStatusTarget
		}
		if ec.OOBSwap == "" {
			ec.OOBSwap = "innerHTML"
		}
		c.Response().WriteHeader(statusCode)
		return corecomponents.ErrorStatusFromContext(ec).Render(c.Request().Context(), c.Response())
	}

	ec := linkwell.ErrorContext{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
		Route:      c.Request().URL.Path,
		RequestID:  requestID,
		Theme:      errorPageTheme(c),
		Controls:   controls,
	}
	c.Response().Status = statusCode
	return corecomponents.RenderError(documentPresentation(ec)).Render(c.Request().Context(), c.Response())
}

// handleErrorWithContext renders a full hypermedia error response from an
// ErrorContext. HTMX requests get a banner OOB swap; non-HTMX requests get the
// document surface and strip the dismiss control (a standalone page can't be
// closed).
func handleErrorWithContext(c echo.Context, ec linkwell.ErrorContext) error {
	if errors.Is(c.Request().Context().Err(), context.Canceled) {
		return nil
	}

	log := logger.WithContext(c.Request().Context()).With(
		"status_code", ec.StatusCode,
		"message", ec.Message,
		"route", c.Request().URL.Path,
		"method", c.Request().Method,
	)
	log.Error("Hypermedia request error", "error", ec.Err)

	if !htmxutil.IsHTMX(c.Request()) {
		ec.Closable = false
		ec.Theme = errorPageTheme(c)
		c.Response().Status = ec.StatusCode
		return corecomponents.RenderError(documentPresentation(ec)).Render(c.Request().Context(), c.Response())
	}

	if ec.OOBTarget == "" {
		ec.OOBTarget = linkwell.DefaultErrorStatusTarget
	}
	if ec.OOBSwap == "" {
		ec.OOBSwap = "innerHTML"
	}
	c.Response().WriteHeader(ec.StatusCode)
	return corecomponents.ErrorStatusFromContext(ec).Render(c.Request().Context(), c.Response())
}

// documentPresentation builds the standalone-shell ErrorPresentation from a
// linkwell.ErrorContext for non-HTMX renders.
func documentPresentation(ec linkwell.ErrorContext) corecomponents.ErrorPresentation {
	p := corecomponents.ErrorPresentation{
		Surface:   corecomponents.SurfaceDocument,
		Status:    ec.StatusCode,
		Title:     ec.Message,
		Route:     ec.Route,
		RequestID: ec.RequestID,
		Theme:     ec.Theme,
		Controls:  ec.Controls,
	}
	if ec.Err != nil {
		p.Detail = ec.Err.Error()
	}
	p.Normalize()
	return p
}

// surfacePresentation builds an ErrorPresentation for the SurfaceError path,
// preserving the caller-chosen surface and the optional detail string.
func surfacePresentation(se *SurfaceError) corecomponents.ErrorPresentation {
	p := corecomponents.ErrorPresentation{
		Surface:   se.Surface,
		Status:    se.EC.StatusCode,
		Title:     se.EC.Message,
		Detail:    se.Detail,
		Route:     se.EC.Route,
		RequestID: se.EC.RequestID,
		Theme:     se.EC.Theme,
		Controls:  se.EC.Controls,
		Closable:  se.EC.Closable,
		OOBTarget: se.EC.OOBTarget,
		OOBSwap:   se.EC.OOBSwap,
	}
	if se.Detail == "" && se.EC.Err != nil {
		p.Detail = se.EC.Err.Error()
	}
	p.Normalize()
	return p
}

// renderSurfaceError dispatches a handler-owned SurfaceError to the right
// render path. Non-HTMX requests honor the chosen surface directly: Page
// composes into the host AppNavLayout, Document writes a standalone shell,
// everything else falls through to RenderComponent. HTMX requests negotiate
// against the client's advertised capabilities (see error_capabilities.go) —
// without advertisement the central path keeps its prior banner-OOB behavior;
// with advertisement it honors the chosen surface when accepted, else degrades
// to the client's declared fallback (or banner if none). Page/Document are
// non-HTMX surfaces, so an HTMX render targeting them collapses back to the
// banner OOB swap. Optional Body overrides the default RenderError(p) on the
// non-HTMX path; HTMX renders always use the contract.
func renderSurfaceError(c echo.Context, se *SurfaceError) error {
	if errors.Is(c.Request().Context().Err(), context.Canceled) {
		return nil
	}

	log := logger.WithContext(c.Request().Context()).With(
		"status_code", se.EC.StatusCode,
		"message", se.EC.Message,
		"route", c.Request().URL.Path,
		"method", c.Request().Method,
		"surface", string(se.Surface),
	)
	log.Error("Surface error", "error", se.EC.Err)

	if htmxutil.IsHTMX(c.Request()) {
		return renderHTMXSurfaceError(c, se)
	}

	if se.EC.Theme == "" {
		se.EC.Theme = errorPageTheme(c)
	}
	c.Response().Status = se.EC.StatusCode
	if se.Body != nil {
		if se.Surface == corecomponents.SurfacePage {
			return RenderBaseLayout(c, se.Body)
		}
		return se.Body.Render(c.Request().Context(), c.Response())
	}
	p := surfacePresentation(se)
	if se.Surface == corecomponents.SurfacePage {
		return RenderBaseLayout(c, corecomponents.RenderError(p))
	}
	return corecomponents.RenderError(p).Render(c.Request().Context(), c.Response())
}

// renderHTMXSurfaceError negotiates the chosen surface against the client's
// declared capabilities and renders the result as an HTMX-safe fragment.
// Falls through to the existing banner OOB path when nothing was advertised
// or when negotiation lands on Banner / a non-HTMX surface.
func renderHTMXSurfaceError(c echo.Context, se *SurfaceError) error {
	caps := ParseErrorCapabilities(c)
	if !caps.HasAdvertised() {
		return handleErrorWithContext(c, se.EC)
	}
	chosen := NegotiateSurface(se.Surface, caps)
	switch chosen {
	case corecomponents.SurfaceInline, corecomponents.SurfaceInlineFull:
		p := surfacePresentation(se)
		p.Surface = chosen
		p.Closable = true
		c.Response().WriteHeader(se.EC.StatusCode)
		return corecomponents.RenderError(p).Render(c.Request().Context(), c.Response())
	default:
		// Banner / Page / Document on HTMX → safe banner OOB swap. Page and
		// Document are non-HTMX surfaces by design; collapsing them here is
		// the documented degrade rule, not a silent error.
		return handleErrorWithContext(c, se.EC)
	}
}

// errPromotedKey marks a request whose trace has already been promoted so
// re-entry into the central error handler does not duplicate that side effect.
const errPromotedKey = "_promolog_promoted"

// NewHTTPErrorHandler is the e.HTTPErrorHandler replacement that renders
// errors through the route-owned surface contract; non-nil reqLogStore
// promotes the per-request log buffer to the shared store so issue reports
// can retrieve it. Trace promotion is idempotent per request.
func NewHTTPErrorHandler(reqLogStore promolog.Storer) func(err error, c echo.Context) {
	return func(err error, c echo.Context) {
		// The httpcompression writer is finalized (closed) by the time the
		// error handler runs; restore the writer saved by RawWriterMiddleware
		// so the render below doesn't write through a closed compressor.
		middleware.RestoreRawWriter(c)

		statusCode := http.StatusInternalServerError
		var se *SurfaceError
		var hhe *linkwell.HTTPError
		var he *echo.HTTPError
		switch {
		case errors.As(err, &se):
			statusCode = se.EC.StatusCode
		case errors.As(err, &hhe):
			statusCode = hhe.EC.StatusCode
		case errors.As(err, &he):
			statusCode = he.Code
		}

		alreadyPromoted, _ := c.Get(errPromotedKey).(bool)
		if reqLogStore != nil && !alreadyPromoted {
			requestID := promolog.GetRequestID(c.Request().Context())
			if requestID != "" {
				c.Set(errPromotedKey, true)
				var entries []promolog.Entry
				if buf := promolog.GetBuffer(c.Request().Context()); buf != nil {
					entries = buf.Entries()
				}
				var userID string
				// setup:feature:auth:start
				userID, _ = c.Get("azureId").(string)
				if userID == "" {
					logger.WithContext(c.Request().Context()).Warn("Error trace missing UserID: azureId not set on echo context")
				}
				// setup:feature:auth:end
				if promoteErr := reqLogStore.Promote(c.Request().Context(), promolog.Trace{
					RequestID:  requestID,
					ErrorChain: err.Error(),
					StatusCode: statusCode,
					Route:      c.Request().URL.Path,
					Method:     c.Request().Method,
					UserAgent:  c.Request().UserAgent(),
					RemoteIP:   c.RealIP(),
					UserID:     userID,
					Entries:    entries,
				}); promoteErr != nil {
					logger.WithContext(c.Request().Context()).Error("Failed to promote error trace",
						"error", promoteErr)
				}
			}
		}

		if c.Response().Committed {
			return
		}

		if se != nil {
			if renderErr := renderSurfaceError(c, se); renderErr != nil {
				logger.WithContext(c.Request().Context()).Error("Failed to render error", "error", renderErr)
			}
			return
		}

		if hhe != nil {
			if renderErr := handleErrorWithContext(c, hhe.EC); renderErr != nil {
				logger.WithContext(c.Request().Context()).Error("Failed to render error", "error", renderErr)
			}
			return
		}

		if he != nil {
			message := ""
			if he.Message != nil {
				if msg, ok := he.Message.(string); ok {
					message = msg
				} else {
					message = "Unknown error"
				}
			}
			if renderErr := handleError(c, he.Code, message, err); renderErr != nil {
				logger.WithContext(c.Request().Context()).Error("Failed to render error", "error", renderErr)
			}
			return
		}

		if renderErr := handleError(c, http.StatusInternalServerError, "operation failed", err); renderErr != nil {
			logger.WithContext(c.Request().Context()).Error("Failed to render error", "error", renderErr)
		}
	}
}

// errorPageTheme returns the DaisyUI theme for document-surface error renders.
// Falls back to "dark" if session settings are unavailable.
func errorPageTheme(c echo.Context) string {
	// setup:feature:session_settings:start
	if s := session.GetSettings(c.Request()); s != nil && s.Theme != "" {
		return s.Theme
	}
	// setup:feature:session_settings:end
	if t, ok := c.Get("theme").(string); ok && t != "" {
		return t
	}
	return "dark"
}
