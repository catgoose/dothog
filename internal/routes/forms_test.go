package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/middleware"

	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupForms builds an Echo wired with correlation, the production
// HTTPErrorHandler, and LinkRelationsMiddleware so the /examples/forms
// route renders through the real layout chain. Used by every test in this
// file; mirrors the production InitEcho shape closely enough that
// rendered-body assertions are honest.
func setupForms(t *testing.T) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.Use(middleware.LinkRelationsMiddleware())
	e.HTTPErrorHandler = handler.NewHTTPErrorHandler(nil)
	ar := &AppRoutes{e: e, ctx: context.Background()}
	ar.initFormsRoutes()
	return e
}

func postForm(t *testing.T, e *echo.Echo, path string, fields url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(fields.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestForms_Index_Renders(t *testing.T) {
	e := setupForms(t)

	req := httptest.NewRequest(http.MethodGet, "/examples/forms", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Forms",
		"GET /examples/forms must render the Forms teaching page")
	assert.Contains(t, body, `id="forms-shell"`,
		"page must include the form shell HTMX targets")
	assert.Contains(t, body, `hx-post="/examples/forms/save"`,
		"form must POST to /examples/forms/save")
}

func TestForms_Save_ValidationErrorsPreserveValues(t *testing.T) {
	e := setupForms(t)
	rec := postForm(t, e, "/examples/forms/save", url.Values{
		"name":  []string{"Ada Lovelace"},
		"email": []string{""}, // missing → 422
		"color": []string{"blue"},
	})

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code,
		"missing email must return 422")
	body := rec.Body.String()
	assert.Contains(t, body, "Email is required.",
		"missing email must produce the inline error message")
	assert.Contains(t, body, `value="Ada Lovelace"`,
		"submitted name must round-trip through the rerender")
	assert.Contains(t, body, "input-error",
		"failing field must light its border via input-error")
	assert.Contains(t, body, "focusFirstError",
		"422 must trigger after-settle focus event for keyboard recovery")
	assert.Contains(t, rec.Header().Get("HX-Trigger-After-Settle"), "focusFirstError",
		"server must emit HX-Trigger-After-Settle so the focusFirstError listener fires")
}

func TestForms_Save_ConflictRendersInlineWorkflowError(t *testing.T) {
	e := setupForms(t)
	rec := postForm(t, e, "/examples/forms/save", url.Values{
		"name":  []string{"Ada"},
		"email": []string{"conflict@example.com"},
	})

	require.Equal(t, http.StatusConflict, rec.Code,
		"conflict@example.com must return 409")
	body := rec.Body.String()
	assert.Contains(t, body, "already taken",
		"conflict must render an inline workflow error message")
	assert.Contains(t, body, `value="conflict@example.com"`,
		"conflict rerender must preserve the submitted email so the user can fix it")
	assert.Contains(t, rec.Header().Get("HX-Trigger-After-Settle"), "focusFirstError",
		"conflict must also focus the offending field after settle")
}

func TestForms_Save_GenericFailureDegradesToBanner(t *testing.T) {
	e := setupForms(t)
	rec := postForm(t, e, "/examples/forms/save", url.Values{
		"name":  []string{"Ada"},
		"email": []string{"boom@example.com"},
	})

	require.Equal(t, http.StatusInternalServerError, rec.Code,
		"boom@example.com must return 500 through the central pipeline")
	body := rec.Body.String()
	assert.Contains(t, body, "hx-swap-oob",
		"generic failure must degrade to the global banner OOB swap")
	assert.Contains(t, body, "Save failed",
		"banner must carry the handler-provided message")
	assert.NotContains(t, body, `id="forms-shell"`,
		"generic failure must NOT replace the form — banner is global chrome")
}

func TestForms_Save_SuccessReplacesFormWithSuccessFragment(t *testing.T) {
	e := setupForms(t)
	rec := postForm(t, e, "/examples/forms/save", url.Values{
		"name":  []string{"Ada Lovelace"},
		"email": []string{"ada@example.com"},
		"color": []string{"blue"},
	})

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "alert-success",
		"successful submit must render the success fragment")
	assert.Contains(t, body, "Ada Lovelace",
		"success fragment must echo the captured name")
	assert.Contains(t, body, "ada@example.com",
		"success fragment must echo the captured email")
	assert.Contains(t, body, "favorite color: blue",
		"success fragment must echo the captured color")
}
