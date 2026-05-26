package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/middleware"

	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupExamples builds an Echo wired with correlation, the production
// HTTPErrorHandler, and the LinkRelationsMiddleware so the layout chain
// (RenderBaseLayout → AppNavLayout → LocalContextBar) sees the registered
// link relations. Mirrors the production InitEcho shape closely enough that
// route tests prove real rendered behavior.
func setupExamples(t *testing.T) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.Use(middleware.LinkRelationsMiddleware())
	e.HTTPErrorHandler = handler.NewHTTPErrorHandler(nil)
	ar := &AppRoutes{
		e:   e,
		ctx: context.Background(),
	}
	ar.initScaffoldAdminRoutes()
	ar.initExamplesRoutes()
	return e
}

// TestExamples_Index_Renders pins the parent resource: GET /examples must
// return 200, identify itself as the Examples discovery page, link to every
// scaffold-owned teaching child, and emit the ResourceGrid markup.
func TestExamples_Index_Renders(t *testing.T) {
	e := setupExamples(t)

	req := httptest.NewRequest(http.MethodGet, "/examples", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	// Identity: the page declares itself as the Examples discovery surface.
	assert.Contains(t, body, "Examples",
		"GET /examples must render the Examples discovery title")
	assert.Contains(t, body, "Scaffold-owned reference implementations",
		"GET /examples must carry the discovery copy")

	// Child links: every teaching child must be reachable from the parent.
	assert.Contains(t, body, `href="/examples/error-scenarios"`,
		"GET /examples must link to the error-scenarios child")
	assert.Contains(t, body, "Error Scenarios",
		"GET /examples must label the error-scenarios card")
	assert.Contains(t, body, `href="/examples/forms"`,
		"GET /examples must link to the forms child")
	assert.Contains(t, body, "Forms",
		"GET /examples must label the forms card")

	// ResourceGrid output: the responsive grid wrapper and at least one
	// ResourceCard (anchor with the card classes) must be present.
	lower := strings.ToLower(body)
	assert.Contains(t, lower, "grid grid-cols-1",
		"GET /examples must render the ResourceGrid wrapper")
	assert.Contains(t, lower, "hover:border-primary/30",
		"GET /examples must render at least one ResourceCard tile")
}

// TestExamples_Index_ErrorScenariosChildReachable proves the parent/child
// wiring at the runtime level: GET /examples/error-scenarios must succeed
// after initExamplesRoutes (the parent init is responsible for chaining
// initErrorScenariosRoutes).
func TestExamples_Index_ErrorScenariosChildReachable(t *testing.T) {
	e := setupExamples(t)

	req := httptest.NewRequest(http.MethodGet, "/examples/error-scenarios", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code,
		"/examples/error-scenarios must register as part of initExamplesRoutes")
}

// TestExamples_Index_FormsChildReachable proves the forms child rides the
// same parent seam. setupExamples calls initExamplesRoutes — not
// initFormsRoutes directly — so reaching /examples/forms here exercises
// the chain initExamplesRoutes → initFormsRoutes, which the standalone
// forms tests (setupForms) cannot prove.
func TestExamples_Index_FormsChildReachable(t *testing.T) {
	e := setupExamples(t)

	req := httptest.NewRequest(http.MethodGet, "/examples/forms", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code,
		"/examples/forms must register as part of initExamplesRoutes")
	body := rec.Body.String()
	assert.Contains(t, body, "Forms",
		"forms page reached through the parent seam must render its title")
	assert.Contains(t, body, `id="forms-shell"`,
		"forms page reached through the parent seam must render the form shell")
}

// TestExamples_SiteMap_RendersOnIndex proves the parent page renders the
// footer site map from the linkwell hub tree. After initExamplesRoutes, the
// /examples hub is registered; GET /examples must include a site-map section
// that links to the hub center and its children.
func TestExamples_SiteMap_RendersOnIndex(t *testing.T) {
	e := setupExamples(t)

	req := httptest.NewRequest(http.MethodGet, "/examples", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	// The site-map block carries the bg-base-200/30 + grid wrappers; if the
	// component were stripped or gated off, these would disappear.
	assert.Contains(t, body, "bg-base-200/30",
		"site map must render its footer band on scaffold-facing pages")
	assert.Contains(t, body, "grid grid-cols-2",
		"site map must render its responsive grid")
	// Hub center + every spoke must surface in the rendered output.
	assert.Contains(t, body, "Examples",
		"site map must list the /examples hub title")
	assert.Contains(t, body, `href="/examples/error-scenarios"`,
		"site map must link each spoke of the /examples hub")
	assert.Contains(t, body, `href="/examples/forms"`,
		"site map must link the forms spoke of the /examples hub")
	assert.Contains(t, body, `href="/admin"`,
		"site map must also surface the scaffold-owned admin hub")
	assert.Contains(t, body, `href="/admin/health"`,
		"site map must include admin children from the same linkwell tree")
}

// TestExamples_Forms_InheritsChromeFromParentSeam pins that the forms child
// inherits the same scaffold chrome — hub-parent context bar + footer site
// map — that error-scenarios proves elsewhere. Going through setupExamples
// (which calls initExamplesRoutes) is what makes this proof load-bearing:
// the context bar and site map only carry the /examples hub because
// initExamplesRoutes registered the linkwell hub before initFormsRoutes ran.
func TestExamples_Forms_InheritsChromeFromParentSeam(t *testing.T) {
	e := setupExamples(t)

	req := httptest.NewRequest(http.MethodGet, "/examples/forms", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	assert.Contains(t, body, `id="local-context-bar"`,
		"forms must inherit the LocalContextBar anchor from AppNavLayout")
	assert.Contains(t, body, "↑ Examples",
		"forms must show the hub-parent affordance to /examples via linkwell")
	assert.Contains(t, body, "bg-base-200/30",
		"forms must inherit the SiteMap footer band from the scaffold layout")
	assert.Contains(t, body, `href="/examples"`,
		"forms site map must link the hub center back to /examples")
}

// TestExamples_SiteMap_RendersOnChild pins the same outcome on the child
// page so the site map is proven to be real runtime chrome, not just a
// landing-page treatment.
func TestExamples_SiteMap_RendersOnChild(t *testing.T) {
	e := setupExamples(t)

	req := httptest.NewRequest(http.MethodGet, "/examples/error-scenarios", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	assert.Contains(t, body, "bg-base-200/30",
		"site map must render on /examples/error-scenarios, not just on the landing page")
	assert.Contains(t, body, `href="/examples"`,
		"site map must link the hub center from a child page")
}

// TestExamples_ContextBar_RendersOnChild proves the child page renders the
// real context-bar path end-to-end: LinkRelationsMiddleware runs, the route
// handler renders through RenderBaseLayout → AppNavLayout → LocalContextBar,
// and the response body carries the hub-parent affordance ("↑ Examples")
// pointing at /examples.
func TestExamples_ContextBar_RendersOnChild(t *testing.T) {
	e := setupExamples(t)

	req := httptest.NewRequest(http.MethodGet, "/examples/error-scenarios", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	// The LocalContextBar component always emits its anchor div, but the
	// inner strip only appears when there are groups. Both must be present
	// on a child of /examples after the hub registration in
	// initExamplesRoutes wires through middleware + layout.
	assert.Contains(t, body, `id="local-context-bar"`,
		"AppNavLayout must include the LocalContextBar anchor on every page")
	assert.Contains(t, body, `↑ Examples`,
		"context bar must render the hub-parent affordance for /examples/error-scenarios")
	assert.Contains(t, body, `href="/examples"`,
		"context bar must link the hub-parent back to /examples")
}
