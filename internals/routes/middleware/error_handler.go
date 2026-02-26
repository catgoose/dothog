package middleware

import (
	"context"
	"errors"
	"net/http"

	"catgoose/go-htmx-demo/internals/logger"
	"catgoose/go-htmx-demo/internals/routes/hypermedia"
	"catgoose/go-htmx-demo/internals/routes/response"
	corecomponents "catgoose/go-htmx-demo/web/components/core"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

// handleError logs the error and returns an HTML error component
func handleError(c echo.Context, statusCode int, message string, err error) error {
	if errors.Is(c.Request().Context().Err(), context.Canceled) {
		return nil
	}

	requestID := GetRequestID(c)
	log := logger.WithContext(c.Request().Context()).With(
		"statusCode", statusCode,
		"message", message,
		"route", c.Request().URL.Path,
		"method", c.Request().Method,
	)
	log.Error("Request error", "error", err)

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Status = statusCode
		return corecomponents.ErrorStatus(statusCode, message, err, c.Request().URL.Path, requestID, true).Render(c.Request().Context(), c.Response())
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
		},
	}
	c.Response().Status = statusCode
	return corecomponents.ErrorPage(ec).Render(c.Request().Context(), c.Response())
}

// handleErrorWithContext renders a full hypermedia error response from an ErrorContext.
// When ec.OOBTarget is set the error panel is delivered as an OOB swap alongside
// a no-op primary component; otherwise it is rendered inline.
func handleErrorWithContext(c echo.Context, ec hypermedia.ErrorContext) error {
	if errors.Is(c.Request().Context().Err(), context.Canceled) {
		return nil
	}

	log := logger.WithContext(c.Request().Context()).With(
		"statusCode", ec.StatusCode,
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

	if ec.OOBTarget != "" {
		return response.New(c).
			Status(ec.StatusCode).
			Component(templ.NopComponent).
			OOBErrorStatus(ec).
			Send()
	}

	c.Response().Status = ec.StatusCode
	return corecomponents.ErrorStatusFromContext(ec).Render(c.Request().Context(), c.Response())
}

// ErrorHandlerMiddleware automatically wraps errors returned by handlers in HandleError
func ErrorHandlerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err == nil {
				return nil
			}

			// If the response is already committed, don't modify it
			if c.Response().Committed {
				return nil
			}

			// 1. HTTPError — rich error with hypermedia controls
			var hhe *hypermedia.HTTPError
			if errors.As(err, &hhe) {
				return handleErrorWithContext(c, hhe.EC)
			}

			// 2. echo.HTTPError — convert to HTML for HTMX requests
			var he *echo.HTTPError
			if errors.As(err, &he) {
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
