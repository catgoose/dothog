package routes

import (
	"errors"
	"net/http"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"
	corecomponents "catgoose/dothog/web/components/core"

	"github.com/labstack/echo/v4"
)

// initErrorScenariosRoutes wires the scaffold-facing /examples/error-scenarios
// teaching surface. Each scenario demonstrates a realistic combination of
// request mode, handler intent, chosen surface, and recovery controls — what
// derived-app authors will hit when they wire their own error responses
// against the central pipeline. Routes go through handler.NewSurfaceError so
// what users see here matches what real handlers produce.
//
// Always-on (not feature-gated): derived apps inherit the page after
// `mage setup` regardless of the demo feature so the teaching surface is
// available to scaffolds that never select demo. Registered as a child of
// the /examples hub by initExamplesRoutes; no separate Hub() call needed
// here.
func (ar *AppRoutes) initErrorScenariosRoutes() {
	es := ar.e.Group("/examples/error-scenarios")
	es.GET("", handler.HandleComponent(views.ErrorScenariosPage()))

	// Scenario 1 — route-level 404 in chrome.
	// Request: direct navigation (non-HTMX). Handler returns SurfacePage so
	// the central handler composes the error inside AppNavLayout: nav stays,
	// the error sits centered in the main content area as a real route view.
	es.GET("/route-404", func(c echo.Context) error {
		return handler.NewSurfaceError(c, corecomponents.SurfacePage,
			http.StatusNotFound,
			"Project not found",
			"We couldn't find project ID 8832 in your workspace.",
			errors.New("workspace lookup miss for id=8832"))
	})

	// Scenario 2 — standalone document/boundary failure.
	// Request: direct navigation that hits an auth/system boundary where the
	// host chrome is intentionally unavailable. SurfaceDocument owns its own
	// HTML shell. No Report Issue button: the boundary case isn't reportable
	// by the user; it's a "you're at a place we couldn't bring you fully into."
	es.GET("/auth-boundary", func(c echo.Context) error {
		return handler.NewSurfaceError(c, corecomponents.SurfaceDocument,
			http.StatusUnauthorized,
			"Sign in required",
			"This workspace requires authentication. Sign in to continue.",
			errors.New("session token missing or expired"))
	})

	// Scenario 3 — localized HTMX validation (inline).
	// Request: HTMX form submit advertises data-error-accept="inline,banner".
	// Server returns SurfaceInline; negotiation honors it. Renders into the
	// form's hx-target without touching the global banner.
	es.GET("/validation-inline", func(c echo.Context) error {
		return handler.NewSurfaceError(c, corecomponents.SurfaceInline,
			http.StatusUnprocessableEntity,
			"Email is required",
			"The form needs an email before we can save the contact.",
			errors.New("validation: email empty"))
	})

	// Scenario 4 — container-owning HTMX failure (inline-full).
	// Request: HTMX request inside a sized panel advertises
	// data-error-accept="inline-full,banner". The error fills the panel
	// instead of replacing only the row, so the user understands the whole
	// region failed (rate limit hit on a metrics tile, etc.).
	es.GET("/panel-inline-full", func(c echo.Context) error {
		return handler.NewSurfaceError(c, corecomponents.SurfaceInlineFull,
			http.StatusTooManyRequests,
			"Rate limit reached",
			"Metric API rejected the request. Try again in a few seconds.",
			errors.New("upstream 429 from metrics API"))
	})

	// Scenario 5 — HTMX banner fallback when no scoped surface is accepted.
	// Request: HTMX trigger with no data-error-accept advertisement, or with
	// a surface the route can't satisfy. Central handler degrades to the
	// global banner OOB swap; the user keeps their current page.
	es.GET("/banner-fallback", func(c echo.Context) error {
		return handler.NewSurfaceError(c, corecomponents.SurfaceBanner,
			http.StatusInternalServerError,
			"Background job failed",
			"A scheduled sync raised an unrecoverable error.",
			errors.New("worker pool: deadline exceeded"))
	})

	// Scenario 6 — reportable 500 with Request ID + Report Issue control.
	// Request: direct navigation. SurfaceDocument with the 500-default
	// control set (Report Issue gets appended automatically by NewSurfaceError
	// for 500+); user gets a copyable Request ID and a one-click report path
	// that ties into the trace store.
	es.GET("/reportable-500", func(c echo.Context) error {
		return handler.NewSurfaceError(c, corecomponents.SurfaceDocument,
			http.StatusInternalServerError,
			"Operation failed",
			"Something went wrong while saving your changes.",
			errors.New("pq: deadlock detected on orders table"))
	})
}
