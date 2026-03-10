// Package handler provides HTTP request handlers and utility functions for rendering components.
package handler

import (
	log "catgoose/harmony/internals/logger"
	"catgoose/harmony/internals/routes/hypermedia"
	"catgoose/harmony/internals/routes/middleware"
	"catgoose/harmony/web/views"
	"context"
	"errors"
	"fmt"
	"net/http"

	corecomponents "catgoose/harmony/web/components/core"

	"github.com/a-h/templ"
	"github.com/catgoose/dio"
	"github.com/labstack/echo/v4"
)

// appNavComponent builds the NavBar with the active item set for the given path
func appNavComponent(path string) templ.Component {
	items := hypermedia.SetActiveNavItemPrefix([]hypermedia.NavItem{
		{Label: "Home", Href: "/"},
		// setup:feature:demo:start
		{Label: "Dashboard", Href: "/dashboard"},
		{
			Label: "Hypermedia Controls",
			Children: []hypermedia.NavItem{
				{Label: "Controls", Href: "/hypermedia/controls"},
				{Label: "CRUD", Href: "/hypermedia/crud"},
				{Label: "Lists", Href: "/hypermedia/lists"},
				{Label: "Interactions", Href: "/hypermedia/interactions"},
				{Label: "State", Href: "/hypermedia/state"},
				{Label: "Components", Href: "/hypermedia/components"},
				{Label: "Components 2", Href: "/hypermedia/components2"},
				{Label: "Components 3", Href: "/hypermedia/components3"},
				// setup:feature:sse:start
				{Label: "Real-time", Href: "/hypermedia/realtime"},
				// setup:feature:sse:end
			},
		},
		{Label: "Repository", Href: "/demo/repository"},
		{Label: "Admin", Href: "/admin"},
		// setup:feature:demo:end
	}, path)
	return corecomponents.NavBar(items)
}

// RenderBaseLayout wraps the component in a base layout and renders it
func RenderBaseLayout(c echo.Context, cmp templ.Component) error {
	nav := appNavComponent(c.Request().URL.Path)
	var csrfToken string
	// setup:feature:csrf:start
	if t, ok := c.Get("csrf_token").(string); ok {
		csrfToken = t
	}
	// setup:feature:csrf:end
	return RenderComponent(c, views.Index(cmp, nav, csrfToken, dio.Dev()))
}

// RenderComponent renders a templ component to the response
func RenderComponent(c echo.Context, cmp templ.Component) error {
	if err := cmp.Render(c.Request().Context(), c.Response()); err != nil {
		return HandleError(c, http.StatusInternalServerError, "Failed to render component", err)
	}
	return nil
}

// HandleError logs the error and return Hypermedia response
func HandleError(c echo.Context, statusCode int, message string, err error) error {
	// Check if the request context is canceled
	if errors.Is(c.Request().Context().Err(), context.Canceled) {
		return nil
	}
	requestID := middleware.GetRequestID(c)
	log.WithContext(c.Request().Context()).Error("Request error", "error", err, "status_code", statusCode, "message", message)
	c.Response().Status = statusCode
	renderErr := RenderComponent(c, corecomponents.ErrorStatus(statusCode, message, err, c.Request().URL.Path, requestID, true))
	if renderErr != nil {
		return c.HTML(http.StatusInternalServerError, fmt.Sprintf(
			`
			<div class="bg-rose-100 border-b border-b-rose-400 text-rose-800 p-2 shadow-md text-sm ">
				<p class="mb-1">
					<strong>Message:</strong> Failed to render error view
				</p>
				<p class="mb-1">
					<strong>Render Error:</strong>: %s
				</p>
				<p class="mb-1">
					<strong>Internal Error:</strong>: %s
				</p>
			</div>
	`, renderErr.Error(), err.Error()))
	}
	return nil
}

// HandleHypermediaError is a convenience wrapper that builds an HTTPError
// from handler arguments and returns it for the middleware to render.
// When no controls are supplied, sensible defaults are provided based on the
// status code so that every error response is a navigable hypermedia state.
func HandleHypermediaError(c echo.Context, statusCode int, message string, err error, controls ...hypermedia.Control) error {
	if len(controls) == 0 {
		controls = defaultControls(statusCode)
	}
	ec := middleware.HypermediaError(c, statusCode, message, err, controls...)
	return hypermedia.NewHTTPError(ec)
}

// defaultControls returns recovery controls appropriate for the given HTTP status code.
func defaultControls(statusCode int) []hypermedia.Control {
	back := hypermedia.BackButton(hypermedia.LabelGoBack)
	home := hypermedia.GoHomeButton(hypermedia.LabelGoHome, "/", hypermedia.TargetBody)
	dismiss := hypermedia.DismissButton(hypermedia.LabelDismiss)

	switch {
	case statusCode == http.StatusBadRequest || statusCode == http.StatusUnprocessableEntity:
		return []hypermedia.Control{dismiss}
	case statusCode == http.StatusNotFound:
		return []hypermedia.Control{back, home}
	case statusCode == http.StatusUnauthorized:
		return []hypermedia.Control{
			hypermedia.RedirectLink(hypermedia.LabelLogIn, "/login"),
			home,
		}
	case statusCode == http.StatusForbidden:
		return []hypermedia.Control{back, home}
	case statusCode >= 500:
		return []hypermedia.Control{dismiss, home}
	default:
		return []hypermedia.Control{dismiss}
	}
}

// HandleComponent is a handler that renders a templ component
func HandleComponent(cmp templ.Component) echo.HandlerFunc {
	return func(c echo.Context) error {
		return RenderBaseLayout(c, cmp)
	}
}
