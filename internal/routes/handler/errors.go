package handler

import (
	"net/http"

	"github.com/catgoose/linkwell"
	"github.com/catgoose/promolog"

	"github.com/labstack/echo/v4"
)

// errorOpts returns the default ErrorControlOpts for convenience error helpers.
func errorOpts() linkwell.ErrorControlOpts {
	return linkwell.ErrorControlOpts{HomeURL: "/", LoginURL: "/login"}
}

// newError builds a linkwell.HTTPError with controls dispatched from
// ErrorControlsForStatus. For 500+ errors a ReportIssueButton is appended.
func newError(c echo.Context, statusCode int, message string) error {
	requestID := promolog.GetRequestID(c.Request().Context())
	controls := linkwell.ErrorControlsForStatus(statusCode, errorOpts())
	if statusCode >= 500 {
		controls = append(controls, linkwell.ReportIssueButton(linkwell.LabelReportIssue, requestID))
	}
	ec := linkwell.ErrorContext{
		StatusCode: statusCode,
		Message:    message,
		Route:      c.Request().URL.Path,
		RequestID:  requestID,
		Closable:   true,
		Controls:   controls,
	}
	return linkwell.NewHTTPError(ec)
}

// BadRequest is a 400 for invalid input, missing required parameters, or malformed requests.
func BadRequest(c echo.Context, message string) error {
	return newError(c, http.StatusBadRequest, message)
}

// Unauthorized is a 401 when authentication is required but missing or invalid.
func Unauthorized(c echo.Context, message string) error {
	return newError(c, http.StatusUnauthorized, message)
}

// Forbidden is a 403 when authentication is valid but lacks permission for the resource.
func Forbidden(c echo.Context, message string) error {
	return newError(c, http.StatusForbidden, message)
}

// NotFound is a 404 when the requested resource doesn't exist.
func NotFound(c echo.Context, message string) error {
	return newError(c, http.StatusNotFound, message)
}

// InternalServerError is a 500 for unexpected server errors, database failures, or unhandled exceptions.
func InternalServerError(c echo.Context, message string) error {
	return newError(c, http.StatusInternalServerError, message)
}

// ServiceUnavailable is a 503 when the service is temporarily unavailable or overloaded.
func ServiceUnavailable(c echo.Context, message string) error {
	return newError(c, http.StatusServiceUnavailable, message)
}

// HypermediaError stamps route and requestID from c into a linkwell.ErrorContext;
// pass to linkwell.NewHTTPError for handler returns, or render alongside OOB swaps.
func HypermediaError(c echo.Context, statusCode int, message string, err error, controls ...linkwell.Control) linkwell.ErrorContext {
	return linkwell.ErrorContext{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
		Route:      c.Request().URL.Path,
		RequestID:  promolog.GetRequestID(c.Request().Context()),
		Closable:   true,
		Controls:   controls,
	}
}
