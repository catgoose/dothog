// Package handler provides HTTP request handlers and utility functions for rendering components.
package handler

import (
	// setup:feature:session_settings:start
	"catgoose/dothog/internal/domain"
	// setup:feature:session_settings:end
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
	path      string
	crumbs    []hypermedia.Breadcrumb
	links     []hypermedia.LinkRelation
	hubs      []hypermedia.HubEntry
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
	path := c.Request().URL.Path
	from := c.QueryParam("from")

	// Priority 1: explicit navigation context via ?from= bitmask
	if mask := hypermedia.ParseFromParam(from); mask != 0 {
		pathCrumbs := buildPathCrumbs(path, from, getRoutes)
		crumbs = append(hypermedia.ResolveFromMask(mask), pathCrumbs...)
	}

	// Priority 2: declared document hierarchy via rel="up" chain
	// setup:feature:demo:start
	if len(crumbs) == 0 {
		crumbs = hypermedia.BreadcrumbsFromLinks(path)
	}
	// setup:feature:demo:end

	// Priority 3: derive from URL path segments (last resort)
	if len(crumbs) == 0 {
		pathCrumbs := buildPathCrumbs(path, from, getRoutes)
		if len(pathCrumbs) > 1 {
			crumbs = append([]hypermedia.Breadcrumb{{Label: hypermedia.BreadcrumbLabelHome, Href: "/"}}, pathCrumbs...)
		}
	}

	if label, ok := c.Get(pageLabel).(string); ok && label != "" && len(crumbs) > 0 {
		crumbs[len(crumbs)-1].Label = label
	}

	// setup:feature:demo:start
	links := middleware.GetLinkRelations(c)
	hubs := hypermedia.Hubs()
	// setup:feature:demo:end

	return layoutCtx{
		csrfToken: csrfToken,
		theme:     theme,
		path:      c.Request().URL.Path,
		crumbs:    crumbs,
		// setup:feature:demo:start
		links: links,
		hubs:  hubs,
		// setup:feature:demo:end
	}
}

// appNavNavConfig returns a NavConfig with icons for use with the AppNavLayout.
func appNavNavConfig() hypermedia.NavConfig {
	return hypermedia.NavConfig{
		AppName: appName,
		Items: []hypermedia.NavItem{
			{Label: "Home", Href: "/", Icon: "M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"},
			{Label: "Dashboard", Href: "/dashboard", Icon: "M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zm10 0a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zm10 0a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"},
			{Label: "Hypermedia", Href: "/hypermedia", Icon: "M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"},
			{Label: "Demo", Href: "/demo", Icon: "M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"},
			{Label: "Settings", Href: "/settings", Icon: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"},
			{Label: "Admin", Href: "/admin", Icon: "M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"},
		},
		MaxVisible: 6,
	}
}

// renderDefaultLayout is the standard dothog layout with nav, breadcrumbs, and theme.
func renderDefaultLayout(c echo.Context, cmp templ.Component) error {
	lc := getLayoutCtx(c)
	// setup:feature:session_settings:start
	settings := middleware.GetSessionSettings(c)
	if settings.Layout == domain.LayoutApp {
		cfg := appNavNavConfig()
		cfg.Items = hypermedia.SetActiveNavItemPrefix(cfg.Items, lc.path)
		return RenderComponent(c, views.AppNavLayout(
			cmp, cfg,
			lc.csrfToken, dio.Dev(), lc.theme,
			lc.crumbs, lc.links, lc.path, version.Display(), lc.hubs,
		))
	}
	// setup:feature:session_settings:end
	nav := appNavComponent(c.Request().URL.Path)
	return RenderComponent(c, views.Index(cmp, nav, lc.csrfToken, dio.Dev(), lc.theme, lc.crumbs, lc.links, lc.path, version.Display(), appName, lc.hubs))
}

// AppNavLayoutFunc returns a LayoutFunc that uses the responsive app-nav layout.
// The NavConfig defines navigation structure and optional custom slots.
func AppNavLayoutFunc(cfg hypermedia.NavConfig) LayoutFunc {
	return func(c echo.Context, cmp templ.Component) error {
		path := c.Request().URL.Path
		navCfg := cfg
		navCfg.Items = hypermedia.SetActiveNavItemPrefix(cfg.Items, path)
		if navCfg.MaxVisible <= 0 {
			navCfg.MaxVisible = len(navCfg.Items)
		}
		if navCfg.AppName == "" {
			navCfg.AppName = appName
		}
		lc := getLayoutCtx(c)
		return RenderComponent(c, views.AppNavLayout(
			cmp, navCfg,
			lc.csrfToken, dio.Dev(), lc.theme,
			lc.crumbs, lc.links, lc.path, version.Display(), lc.hubs,
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
	renderErr := corecomponents.ErrorStatus(statusCode, message, err, c.Request().URL.Path, requestID, true).Render(c.Request().Context(), c.Response())
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
		opts := hypermedia.ErrorControlOpts{HomeURL: "/", LoginURL: "/login"}
		controls = hypermedia.ErrorControlsForStatus(statusCode, opts)
		if statusCode >= 500 {
			requestID := middleware.GetRequestID(c)
			controls = append(controls, hypermedia.ReportIssueButton(hypermedia.LabelReportIssue, requestID))
		}
	}
	ec := middleware.HypermediaError(c, statusCode, message, err, controls...)
	return hypermedia.NewHTTPError(ec)
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
