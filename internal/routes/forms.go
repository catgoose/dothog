package routes

import (
	"net/http"
	"strings"

	"catgoose/dothog/internal/htmxutil"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"
	corecomponents "catgoose/dothog/web/components/core"

	"github.com/labstack/echo/v4"
)

// initFormsRoutes wires the scaffold-facing /examples/forms teaching
// surface. Demonstrates the preferred dothog pattern for server-driven HTMX
// validation: the form rerenders itself on 422 with per-field errors and
// submitted values preserved, conflict-style workflow errors stay inline,
// and generic non-validation failures degrade through the central error
// pipeline to a banner. Registered as a child of the /examples hub by
// initExamplesRoutes so the link registry covers the parent/child link.
func (ar *AppRoutes) initFormsRoutes() {
	es := ar.e.Group("/examples/forms")
	es.GET("", handler.HandleComponent(views.FormsPage(views.FormsState{})))
	es.POST("/save", handleFormsSave)
}

// handleFormsSave demonstrates three response shapes from one endpoint:
//
//  1. Validation 422: returns the form shell with inline field errors and the
//     submitted values preserved. The page-local rerender is the right tool —
//     a central SurfaceError would lose the form context.
//  2. Workflow 409: a conflict (email already taken) is also an inline
//     rerender — semantically different from validation but the recovery UX
//     is the same.
//  3. Generic 500: a non-validation failure (database down, etc.) routes
//     through handler.NewSurfaceError with SurfaceBanner so the central
//     pipeline degrades to the global banner OOB swap. The form stays on
//     screen; the user sees an app-level error overlay.
func handleFormsSave(c echo.Context) error {
	state := views.FormsState{
		Name:  strings.TrimSpace(c.FormValue("name")),
		Email: strings.TrimSpace(c.FormValue("email")),
		Color: strings.TrimSpace(c.FormValue("color")),
	}

	state.Errors = map[string]string{}
	if state.Name == "" {
		state.Errors["name"] = "Name is required."
	}
	if state.Email == "" {
		state.Errors["email"] = "Email is required."
	} else if !strings.Contains(state.Email, "@") {
		state.Errors["email"] = "Email must contain @."
	}

	// Synthetic conflict trigger so the teaching surface can demonstrate a
	// workflow error that isn't a validation miss.
	if len(state.Errors) == 0 && state.Email == "conflict@example.com" {
		state.Errors["email"] = "This email is already taken. Try another."
		c.Response().Status = http.StatusConflict
		// Focus the offending field after the swap settles so keyboard users
		// land on the right control without a roundtrip back through the
		// form. _hyperscript inline on the input listens for focusFirstError.
		_ = htmxutil.New().TriggerAfterSettle("focusFirstError").Write(c.Response())
		return handler.RenderComponent(c, views.FormsShell(state))
	}

	// Synthetic generic failure trigger. The non-validation failure goes
	// through the central pipeline as a banner so the form stays on screen
	// and the error appears as global app chrome, not as a fake field error.
	if state.Email == "boom@example.com" {
		return handler.NewSurfaceError(c, corecomponents.SurfaceBanner,
			http.StatusInternalServerError,
			"Save failed",
			"The orders database returned an unrecoverable error.",
			nil)
	}

	if len(state.Errors) > 0 {
		c.Response().Status = http.StatusUnprocessableEntity
		_ = htmxutil.New().TriggerAfterSettle("focusFirstError").Write(c.Response())
		return handler.RenderComponent(c, views.FormsShell(state))
	}

	return handler.RenderComponent(c, views.FormsSuccess(state))
}
