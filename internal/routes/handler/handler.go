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
		{Label: "Hypermedia", Href: "/hypermedia"},
		{Label: "Demo", Href: "/demo"},
		// setup:feature:demo:end
		// setup:feature:session_settings:start
		{Label: "Preferences", Href: "/user/settings"},
		// setup:feature:session_settings:end
		{Label: "Settings", Href: "/settings"},
		{Label: "Admin", Href: "/admin"},
	}, path)
	return corecomponents.NavBar(items)
}

// LayoutFunc renders a page component into a full HTML page.
// The default layout uses the dothog Index template with nav, breadcrumbs, and theme.
// Derived apps can override this via SetLayout to use their own page wrapper.
type LayoutFunc func(c echo.Context, cmp templ.Component) error

var customLayout LayoutFunc

// SetLayout overrides the default page layout. Call once at startup before
// serving requests. Pass nil to restore the default dothog layout.
func SetLayout(fn LayoutFunc) {
	customLayout = fn
}

// RenderBaseLayout wraps the component in a page layout and renders it.
// If a custom layout was set via SetLayout, it is used instead of the default.
func RenderBaseLayout(c echo.Context, cmp templ.Component) error {
	if customLayout != nil {
		return customLayout(c, cmp)
	}
	return renderDefaultLayout(c, cmp)
}

// layoutCtx holds common layout state extracted from the request context.
type layoutCtx struct {
	csrfToken string
	theme     string
	crumbs    []hypermedia.Breadcrumb
}

// getLayoutCtx extracts CSRF token, theme, and breadcrumbs from the request.
func getLayoutCtx(c echo.Context) layoutCtx {
	var csrfToken string
	// setup:feature:csrf:start
	if t, ok := c.Get("csrf_token").(string); ok {
		csrfToken = t
	}
	// setup:feature:csrf:end
	theme := "dark"
	if t, ok := c.Get("theme").(string); ok && t != "" {
		theme = t
	}
	// setup:feature:session_settings:start
	theme = middleware.GetSessionSettings(c).Theme
	// setup:feature:session_settings:end

	var crumbs []hypermedia.Breadcrumb
	from := c.QueryParam("from")
	pathCrumbs := buildPathCrumbs(c.Request().URL.Path, from, getRoutes)

	if mask := hypermedia.ParseFromParam(from); mask != 0 {
		crumbs = append(hypermedia.ResolveFromMask(mask), pathCrumbs...)
	} else if len(pathCrumbs) > 1 {
		crumbs = append([]hypermedia.Breadcrumb{{Label: hypermedia.BreadcrumbLabelHome, Href: "/"}}, pathCrumbs...)
	}

	if label, ok := c.Get(pageLabel).(string); ok && label != "" && len(crumbs) > 0 {
		crumbs[len(crumbs)-1].Label = label
	}

	return layoutCtx{csrfToken: csrfToken, theme: theme, crumbs: crumbs}
}

// renderDefaultLayout is the standard dothog layout with nav, breadcrumbs, and theme.
func renderDefaultLayout(c echo.Context, cmp templ.Component) error {
	nav := appNavComponent(c.Request().URL.Path)
	lc := getLayoutCtx(c)
	return RenderComponent(c, views.Index(cmp, nav, lc.csrfToken, dio.Dev(), lc.theme, lc.crumbs, version.Display(), appName))
}

// AppNavLayoutFunc returns a LayoutFunc that uses the responsive app-nav layout.
// items defines the navigation structure. promoted (may be nil) gets the raised
// mobile button treatment. maxVisible controls how many items appear in the
// main bar; the rest go to the overflow menu (0 = show all).
func AppNavLayoutFunc(items []hypermedia.NavItem, promoted *hypermedia.NavItem, maxVisible int) LayoutFunc {
	if maxVisible <= 0 {
		maxVisible = len(items)
	}
	return func(c echo.Context, cmp templ.Component) error {
		path := c.Request().URL.Path
		activeItems := hypermedia.SetActiveNavItemPrefix(items, path)
		lc := getLayoutCtx(c)
		return RenderComponent(c, views.AppNavLayout(
			cmp, activeItems, promoted, maxVisible,
			lc.csrfToken, dio.Dev(), lc.theme,
			lc.crumbs, version.Display(), appName,
		))
	}
}

var (
	getRoutes map[string]bool
	appName   string
)

// InitRouteSet builds the set of GET-routable paths from the Echo router and
// stores the app name for use in page titles. Call once after all routes are
// registered, before the server starts.
func InitRouteSet(e *echo.Echo, name string) {
	appName = name
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
// navigation. HTMX requests return a hypermedia error with back/home/report
// controls, rendered as an OOB banner by ErrorHandlerMiddleware.
// Register with e.RouteNotFound.
func HandleNotFound(c echo.Context) error {
	if c.Request().Header.Get("HX-Request") == "true" {
		return HandleHypermediaError(c, http.StatusNotFound, "Not Found", nil)
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
