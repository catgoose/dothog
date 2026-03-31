package middleware

import (
	"net/http"

	hypermedia "github.com/catgoose/linkwell"

	"github.com/labstack/echo/v4"
)

// Errors middleware package provides helper methods for returning HTTP respones
// from echo contexts

// errorOpts returns the default ErrorControlOpts for convenience error helpers.
func errorOpts() hypermedia.ErrorControlOpts {
	return hypermedia.ErrorControlOpts{HomeURL: "/", LoginURL: "/login"}
}

// newError builds a hypermedia.HTTPError with controls dispatched from ErrorControlsForStatus.
// For 500+ errors, a ReportIssueButton is appended.
func newError(c echo.Context, statusCode int, message string) error {
	requestID := GetRequestID(c)
	controls := hypermedia.ErrorControlsForStatus(statusCode, errorOpts())
	if statusCode >= 500 {
		controls = append(controls, hypermedia.ReportIssueButton(hypermedia.LabelReportIssue, requestID))
	}
	ec := hypermedia.ErrorContext{
		StatusCode: statusCode,
		Message:    message,
		Route:      c.Request().URL.Path,
		RequestID:  requestID,
		Closable:   true,
		Controls:   controls,
	}
	return hypermedia.NewHTTPError(ec)
}

// BadRequest returns a 400 Bad Request error
// Use for invalid input, missing required parameters, or malformed requests
func BadRequest(c echo.Context, message string) error {
	return newError(c, http.StatusBadRequest, message)
}

// Unauthorized returns a 401 Unauthorized error
// Use when authentication is required but not provided or invalid
func Unauthorized(c echo.Context, message string) error {
	return newError(c, http.StatusUnauthorized, message)
}

// Forbidden returns a 403 Forbidden error
// Use when authentication is valid but the user lacks permission for the resource
func Forbidden(c echo.Context, message string) error {
	return newError(c, http.StatusForbidden, message)
}

// NotFound returns a 404 Not Found error
// Use when the requested resource doesn't exist
func NotFound(c echo.Context, message string) error {
	return newError(c, http.StatusNotFound, message)
}

// InternalServerError returns a 500 Internal Server Error
// Use for unexpected server errors, database failures, or unhandled exceptions
func InternalServerError(c echo.Context, message string) error {
	return newError(c, http.StatusInternalServerError, message)
}

// ServiceUnavailable returns a 503 Service Unavailable error
// Use when the service is temporarily unavailable or overloaded
func ServiceUnavailable(c echo.Context, message string) error {
	return newError(c, http.StatusServiceUnavailable, message)
}

// HypermediaError builds a hypermedia.ErrorContext populated with request metadata
// (route, requestID) from the echo context. Pass the result to hypermedia.NewHTTPError
// to return it from a handler, or to response.Builder.OOBErrorStatus to compose it
// alongside a primary component.
func HypermediaError(c echo.Context, statusCode int, message string, err error, controls ...hypermedia.Control) hypermedia.ErrorContext {
	return hypermedia.ErrorContext{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
		Route:      c.Request().URL.Path,
		RequestID:  GetRequestID(c),
		Closable:   true,
		Controls:   controls,
	}
}
