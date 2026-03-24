package middleware

import (
	"context"
	"errors"
	"net/http"

	"catgoose/dothog/internal/logger"
	"github.com/catgoose/promolog"
	"catgoose/dothog/internal/routes/hypermedia"
	"catgoose/dothog/internal/routes/response"
	corecomponents "catgoose/dothog/web/components/core"

	"github.com/a-h/templ"
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

	if c.Request().Header.Get("HX-Request") == "true" {
		ec := hypermedia.ErrorContext{
			StatusCode: statusCode,
			Message:    message,
			Err:        err,
			Route:      c.Request().URL.Path,
			RequestID:  requestID,
			Closable:   true,
			Controls: []hypermedia.Control{
				hypermedia.ReportIssueButton(hypermedia.LabelReportIssue, requestID),
			},
		}
		return response.New(c).
			Status(statusCode).
			Component(templ.NopComponent).
			OOBErrorStatus(ec).
			Send()
	}

	// Non-HTMX: render a full HATEOAS error page with default controls
	ec := hypermedia.ErrorContext{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
		Route:      c.Request().URL.Path,
		RequestID:  requestID,
		Controls: []hypermedia.Control{
			hypermedia.BackButton("Go Back"),
			hypermedia.GoHomeButton("Go Home", "/", "body"),
			hypermedia.ReportIssueButton(hypermedia.LabelReportIssue, requestID),
		},
	}
	c.Response().Status = statusCode
	return corecomponents.ErrorPage(ec).Render(c.Request().Context(), c.Response())
}

// handleErrorWithContext renders a full hypermedia error response from an ErrorContext.
// For HTMX requests the error banner is always delivered as an OOB swap to #error-status.
func handleErrorWithContext(c echo.Context, ec hypermedia.ErrorContext) error {
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
		c.Response().Status = ec.StatusCode
		return corecomponents.ErrorPage(ec).Render(c.Request().Context(), c.Response())
	}

	// HTMX: deliver error banner via OOB swap
	return response.New(c).
		Status(ec.StatusCode).
		Component(templ.NopComponent).
		OOBErrorStatus(ec).
		Send()
}

// ErrorHandlerMiddleware automatically wraps errors returned by handlers in HandleError.
// When a reqLogStore is provided, the per-request log buffer is promoted to
// the shared store on error so it can be retrieved for issue reports.
func ErrorHandlerMiddleware(reqLogStore *promolog.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err == nil {
				return nil
			}

			// Determine status code from error type before promoting.
			statusCode := http.StatusInternalServerError
			var hhe *hypermedia.HTTPError
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
						entries = buf.Entries
					}
					var userID string
				// setup:feature:auth:start
				userID, _ = c.Get("azureId").(string)
				if userID == "" {
					logger.WithContext(c.Request().Context()).Warn("Error trace missing UserID: azureId not set on echo context")
				}
				// setup:feature:auth:end
					reqLogStore.Promote(c.Request().Context(), promolog.ErrorTrace{
						RequestID:  requestID,
						ErrorChain: err.Error(),
						StatusCode: statusCode,
						Route:      c.Request().URL.Path,
						Method:     c.Request().Method,
						UserAgent:  c.Request().UserAgent(),
						RemoteIP:   c.RealIP(),
						UserID:     userID,
						Entries:    entries,
					})
				}
			}

			// If the response is already committed, don't modify it
			if c.Response().Committed {
				return nil
			}

			// 1. HTTPError — rich error with hypermedia controls
			if hhe != nil {
				return handleErrorWithContext(c, hhe.EC)
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
				return handleError(c, he.Code, message, err)
			}

			// 3. Fallback — generic 500
			return handleError(c, http.StatusInternalServerError, "operation failed", err)
		}
	}
}
