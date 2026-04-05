package middleware

import (
	"context"
	"errors"
	"net/http"

	"catgoose/dothog/internal/logger"
	// setup:feature:session_settings:start
	"catgoose/dothog/internal/session"
	// setup:feature:session_settings:end
	"github.com/catgoose/promolog"
	"github.com/catgoose/linkwell"
	corecomponents "catgoose/dothog/web/components/core"

	"github.com/labstack/echo/v4"
)

// handleError logs the error and returns an HTML error component.
// For HTMX requests the error banner is delivered as an OOB swap to #error-status.
func handleError(c echo.Context, statusCode int, message string, err error) error {
	if errors.Is(c.Request().Context().Err(), context.Canceled) {
		return nil
	}

	requestID := GetRequestID(c)
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

	if c.Request().Header.Get("HX-Request") == "true" {
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

	// Non-HTMX: render a full HATEOAS error page with navigation controls
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
	return corecomponents.ErrorPage(ec).Render(c.Request().Context(), c.Response())
}

// handleErrorWithContext renders a full hypermedia error response from an ErrorContext.
// For HTMX requests the error banner is always delivered as an OOB swap to #error-status.
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

	// Non-HTMX: render a full HATEOAS error page (full page can't be dismissed)
	if c.Request().Header.Get("HX-Request") != "true" {
		ec.Closable = false
		ec.Theme = errorPageTheme(c)
		c.Response().Status = ec.StatusCode
		return corecomponents.ErrorPage(ec).Render(c.Request().Context(), c.Response())
	}

	// HTMX: deliver error banner via OOB swap
	if ec.OOBTarget == "" {
		ec.OOBTarget = linkwell.DefaultErrorStatusTarget
	}
	if ec.OOBSwap == "" {
		ec.OOBSwap = "innerHTML"
	}
	c.Response().WriteHeader(ec.StatusCode)
	return corecomponents.ErrorStatusFromContext(ec).Render(c.Request().Context(), c.Response())
}

// NewHTTPErrorHandler returns an echo.HTTPErrorHandler that renders errors as
// hypermedia responses. Assign it to e.HTTPErrorHandler in place of the default.
// When reqLogStore is non-nil, the per-request log buffer is promoted to the
// shared store on error so it can be retrieved for issue reports.
func NewHTTPErrorHandler(reqLogStore promolog.Storer) func(err error, c echo.Context) {
	return func(err error, c echo.Context) {
		// Bypass finalized compression writer by unwrapping to the raw
		// http.ResponseWriter. The httpcompression middleware may have
		// closed its writer by the time the error handler runs, so we
		// write directly to the underlying writer.
		for {
			if u, ok := c.Response().Writer.(interface{ Unwrap() http.ResponseWriter }); ok {
				c.Response().Writer = u.Unwrap()
			} else {
				break
			}
		}
		// Determine status code from error type before promoting.
		statusCode := http.StatusInternalServerError
		var hhe *linkwell.HTTPError
		var he *echo.HTTPError
		if errors.As(err, &hhe) {
			statusCode = hhe.EC.StatusCode
		} else if errors.As(err, &he) {
			statusCode = he.Code
		}

		// Promote per-request log buffer to the shared store on error.
		if reqLogStore != nil {
			requestID := GetRequestID(c)
			if requestID != "" {
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

		// If the response is already committed, don't modify it
		if c.Response().Committed {
			return
		}

		// 1. HTTPError — rich error with hypermedia controls
		if hhe != nil {
			if renderErr := handleErrorWithContext(c, hhe.EC); renderErr != nil {
				logger.WithContext(c.Request().Context()).Error("Failed to render error", "error", renderErr)
			}
			return
		}

		// 2. echo.HTTPError — convert to HTML for HTMX requests
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

		// 3. Fallback — generic 500
		if renderErr := handleError(c, http.StatusInternalServerError, "operation failed", err); renderErr != nil {
			logger.WithContext(c.Request().Context()).Error("Failed to render error", "error", renderErr)
		}
	}
}

// errorPageTheme returns the DaisyUI theme for full-page error renders.
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
