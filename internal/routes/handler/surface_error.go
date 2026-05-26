package handler

import (
	corecomponents "catgoose/dothog/web/components/core"

	"github.com/a-h/templ"
	"github.com/catgoose/linkwell"
	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
)

// SurfaceError is the handler-owned error value that carries chosen surface
// intent through the central HTTPErrorHandler. Return one from a route handler
// to let the central path render at the chosen surface (page, document,
// banner) while the late hook still owns trace promotion, request-ID stamping,
// and the raw-writer restore. Optional Body overrides RenderError(p) so a
// route can supply richer in-chrome content (e.g. NotFoundPage's resource
// navigation grid) without bypassing the central pipeline.
type SurfaceError struct {
	Body    templ.Component
	EC      linkwell.ErrorContext
	Surface corecomponents.ErrorSurface
	Detail  string
}

// Error reports the underlying status message so SurfaceError satisfies the
// error interface and slots into Echo's HTTPErrorHandler dispatch.
func (e *SurfaceError) Error() string {
	if e == nil {
		return ""
	}
	if e.EC.Err != nil {
		return e.EC.Err.Error()
	}
	return e.EC.Message
}

// Unwrap exposes the wrapped error so errors.Is/As keep working when the
// caller passed a wrapped cause.
func (e *SurfaceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.EC.Err
}

// NewSurfaceError stamps route/request-ID onto an ErrorContext and returns the
// handler-owned surface-carrying error ready to bubble to NewHTTPErrorHandler.
// Status defaults the controls via the standard ErrorControlsForStatus rules
// (500+ adds a Report Issue button); pass extra controls to extend that set.
func NewSurfaceError(
	c echo.Context,
	surface corecomponents.ErrorSurface,
	statusCode int,
	title, detail string,
	err error,
	controls ...linkwell.Control,
) *SurfaceError {
	if len(controls) == 0 {
		opts := linkwell.ErrorControlOpts{HomeURL: "/", LoginURL: "/login"}
		controls = linkwell.ErrorControlsForStatus(statusCode, opts)
		if statusCode >= 500 {
			requestID := promolog.GetRequestID(c.Request().Context())
			controls = append(controls, linkwell.ReportIssueButton(linkwell.LabelReportIssue, requestID))
		}
	}
	ec := HypermediaError(c, statusCode, title, err, controls...)
	return &SurfaceError{Surface: surface, EC: ec, Detail: detail}
}

// WithBody attaches a templ.Component the central renderer composes instead
// of the default RenderError(p) for the chosen surface. Page surface routes
// it through RenderBaseLayout; any other surface uses RenderComponent.
func (e *SurfaceError) WithBody(body templ.Component) *SurfaceError {
	e.Body = body
	return e
}
