// Package handler provides HTTP request handlers and utility functions for rendering components.
package handler

import (
	"catgoose/dothog/internal/logger"
	"catgoose/dothog/internal/routes/hypermedia"
	"catgoose/dothog/internal/routes/middleware"
	"catgoose/dothog/internal/version"
	"catgoose/dothog/web/views"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	corecomponents "catgoose/dothog/web/components/core"

	"github.com/a-h/templ"
	"github.com/catgoose/dio"
	"github.com/labstack/echo/v4"
)

const pageLabel = "pageLabel"

// SetPageLabel sets a human-readable label for the current page, used as the
// terminal breadcrumb. Call before RenderBaseLayout.
func SetPageLabel(c echo.Context, label string) {
	c.Set(pageLabel, label)
}

// appNavComponent builds the NavBar with the active item set for the given path
func appNavComponent(path string) templ.Component {
	items := hypermedia.SetActiveNavItemPrefix([]hypermedia.NavItem{
		// setup:feature:demo:start
		{Label: "Home", Href: "/"},
		{Label: "Dashboard", Href: "/dashboard"},
		{
			Label: "Patterns",
			Children: []hypermedia.NavItem{
				{Label: "Controls", Href: "/hypermedia/controls"},
				{Label: "CRUD", Href: "/hypermedia/crud"},
				{Label: "Lists", Href: "/hypermedia/lists"},
				{Label: "Interactions", Href: "/hypermedia/interactions"},
				{Label: "State", Href: "/hypermedia/state"},
				{Label: "Errors", Href: "/hypermedia/errors"},
				{Label: "Components", Href: "/hypermedia/components"},
				{Label: "Components 2", Href: "/hypermedia/components2"},
				{Label: "Components 3", Href: "/hypermedia/components3"},
				// setup:feature:sse:start
				{Label: "Real-time", Href: "/hypermedia/realtime"},
				// setup:feature:sse:end
			},
		},
		{
			Label: "Demo",
			Children: []hypermedia.NavItem{
				{Label: "Logging", Href: "/demo/logging"},
				{Label: "Repository", Href: "/demo/repository"},
				{Label: "Inventory", Href: "/demo/inventory"},
				{Label: "Catalog", Href: "/demo/catalog"},
				{Label: "People", Href: "/demo/people"},
				{Label: "Kanban", Href: "/demo/kanban"},
				{Label: "Approvals", Href: "/demo/approvals"},
				{Label: "Vendors", Href: "/demo/vendors"},
				{Label: "Feed", Href: "/demo/feed"},
				{Label: "Settings", Href: "/demo/settings"},
				{Label: "Bulk", Href: "/demo/bulk"},
			},
		},
		// setup:feature:demo:end
		// setup:feature:session_settings:start
		{Label: "Preferences", Href: "/user/settings"},
		// setup:feature:session_settings:end
		{
			Label: "Admin",
			Children: []hypermedia.NavItem{
				// setup:feature:demo:start
				{Label: "SQLite", Href: "/admin"},
				// setup:feature:demo:end
				{Label: "Error Traces", Href: "/admin/error-traces"},
				{Label: "System", Href: "/admin/system"},
				{Label: "Config", Href: "/admin/config"},
			},
		},
	}, path)
	return corecomponents.NavBar(items)
}

// RenderBaseLayout wraps the component in a base layout and renders it.
// If the request carries a ?from= query parameter matching a registered origin,
// breadcrumbs are rendered between the navbar and the page content.
func RenderBaseLayout(c echo.Context, cmp templ.Component) error {
	nav := appNavComponent(c.Request().URL.Path)
	var csrfToken string
	// setup:feature:csrf:start
	if t, ok := c.Get("csrf_token").(string); ok {
		csrfToken = t
	}
	// setup:feature:csrf:end
	theme := "light"
	// setup:feature:session_settings:start
	theme = middleware.GetSessionSettings(c).Theme
	// setup:feature:session_settings:end

	var crumbs []hypermedia.Breadcrumb
	from := c.QueryParam("from")
	pathCrumbs := buildPathCrumbs(c.Request().URL.Path, from, getRoutes)

	if mask := hypermedia.ParseFromParam(from); mask != 0 {
		// Prepend registered origins when ?from= is present.
		crumbs = append(hypermedia.ResolveFromMask(mask), pathCrumbs...)
	} else if len(pathCrumbs) > 1 {
		// No ?from= but multiple path segments — show path-based breadcrumbs
		// so detail pages always have a way back to their parent.
		crumbs = append([]hypermedia.Breadcrumb{{Label: hypermedia.BreadcrumbLabelHome, Href: "/"}}, pathCrumbs...)
	}

	// Allow handlers to override the terminal crumb label via SetPageLabel.
	if label, ok := c.Get(pageLabel).(string); ok && label != "" && len(crumbs) > 0 {
		crumbs[len(crumbs)-1].Label = label
	}

	return RenderComponent(c, views.Index(cmp, nav, csrfToken, dio.Dev(), theme, crumbs, version.Version))
}

var getRoutes map[string]bool

// InitRouteSet builds the set of GET-routable paths from the Echo router.
// Call once after all routes are registered, before the server starts.
func InitRouteSet(e *echo.Echo) {
	getRoutes = make(map[string]bool)
	for _, r := range e.Routes() {
		if r.Method == http.MethodGet {
			getRoutes[r.Path] = true
		}
	}
}

// buildPathCrumbs derives breadcrumb segments from the URL path. Only
// intermediate segments that correspond to a registered GET route produce a
// linked breadcrumb; segments with no route are silently skipped. The terminal
// segment always appears (unlinked). The from param is forwarded on
// intermediate links.
func buildPathCrumbs(path, from string, routes map[string]bool) []hypermedia.Breadcrumb {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	segments := strings.Split(trimmed, "/")
	if len(segments) <= 1 {
		return nil
	}

	var crumbs []hypermedia.Breadcrumb
	// Intermediate segments: only include if the path is a real route.
	for i := 0; i < len(segments)-1; i++ {
		fullPath := "/" + strings.Join(segments[:i+1], "/")
		if !routes[fullPath] {
			continue
		}
		label := segments[i]
		if len(label) > 0 {
			label = strings.ToUpper(label[:1]) + label[1:]
		}
		crumbs = append(crumbs, hypermedia.Breadcrumb{
			Label: label,
			Href:  hypermedia.FromNav(fullPath, from),
		})
	}

	// Terminal segment: always present, never linked.
	terminal := segments[len(segments)-1]
	if len(terminal) > 0 {
		terminal = strings.ToUpper(terminal[:1]) + terminal[1:]
	}
	crumbs = append(crumbs, hypermedia.Breadcrumb{Label: terminal})
	return crumbs
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
	logger.WithContext(c.Request().Context()).Error("Request error", "error", err, "status_code", statusCode, "message", message)
	c.Response().Status = statusCode
	renderErr := RenderComponent(c, corecomponents.ErrorStatus(statusCode, message, err, c.Request().URL.Path, requestID, true))
	if renderErr != nil {
		return c.HTML(http.StatusInternalServerError, fmt.Sprintf(
			`<div class="bg-error text-error-content p-3 shadow-lg text-sm">
				<p class="mb-1"><strong>Message:</strong> Failed to render error view</p>
				<p class="mb-1"><strong>Render Error:</strong> %s</p>
				<p class="mb-1"><strong>Internal Error:</strong> %s</p>
			</div>`, renderErr.Error(), err.Error()))
	}
	return nil
}

// HandleHypermediaError is a convenience wrapper that builds an HTTPError
// from handler arguments and returns it for the middleware to render.
// When no controls are supplied, sensible defaults are provided based on the
// status code so that every error response is a navigable hypermedia state.
func HandleHypermediaError(c echo.Context, statusCode int, message string, err error, controls ...hypermedia.Control) error {
	if len(controls) == 0 {
		requestID := middleware.GetRequestID(c)
		controls = defaultControls(statusCode, requestID)
	}
	ec := middleware.HypermediaError(c, statusCode, message, err, controls...)
	return hypermedia.NewHTTPError(ec)
}

// defaultControls returns recovery controls appropriate for the given HTTP status code.
// Every error path includes a Report Issue button so users can easily report problems.
func defaultControls(statusCode int, requestID string) []hypermedia.Control {
	back := hypermedia.BackButton(hypermedia.LabelGoBack)
	home := hypermedia.GoHomeButton(hypermedia.LabelGoHome, "/", hypermedia.TargetBody)
	dismiss := hypermedia.DismissButton(hypermedia.LabelDismiss)
	report := hypermedia.ReportIssueButton(hypermedia.LabelReportIssue, requestID)

	switch {
	case statusCode == http.StatusBadRequest || statusCode == http.StatusUnprocessableEntity:
		return []hypermedia.Control{dismiss, report}
	case statusCode == http.StatusNotFound:
		return []hypermedia.Control{back, home, report}
	case statusCode == http.StatusUnauthorized:
		return []hypermedia.Control{
			hypermedia.RedirectLink(hypermedia.LabelLogIn, "/login"),
			home,
			report,
		}
	case statusCode == http.StatusForbidden:
		return []hypermedia.Control{back, home, report}
	case statusCode >= 500:
		return []hypermedia.Control{dismiss, home, report}
	default:
		return []hypermedia.Control{dismiss, report}
	}
}

// HandleNotFound renders a full-page 404 within the base layout for direct
// navigation. HTMX requests fall through to ErrorHandlerMiddleware for OOB
// banner delivery. Register with e.RouteNotFound.
func HandleNotFound(c echo.Context) error {
	if c.Request().Header.Get("HX-Request") == "true" {
		return echo.NewHTTPError(http.StatusNotFound, "Not Found")
	}
	c.Response().Status = http.StatusNotFound
	return RenderBaseLayout(c, views.NotFoundPage(c.Request().URL.Path))
}

// HandleComponent is a handler that renders a templ component
func HandleComponent(cmp templ.Component) echo.HandlerFunc {
	return func(c echo.Context) error {
		return RenderBaseLayout(c, cmp)
	}
}
