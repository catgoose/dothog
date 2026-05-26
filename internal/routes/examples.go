package routes

import (
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/catgoose/linkwell"
)

// initExamplesRoutes wires the scaffold-facing /examples section.
//
// /examples is a real visitable resource: GET /examples serves a discovery
// page listing every scaffold-owned teaching surface as a ResourceCard.
// Child examples register themselves under /examples/<name> and the link
// registry treats /examples as their parent hub so breadcrumb, context bar,
// and site map all reflect the hierarchy.
//
// This is the scaffold-owned link-registry seam: relations registered here
// survive `mage setup` regardless of feature selection. Demo-only relations
// live in initLinkRelations (links.go). The two seams write to the same
// linkwell registry; LinkRelationsMiddleware reads whatever is there.
//
// Distinct from /patterns (demo-only component gallery) and /demo (demo
// feature content). Always-on: every derived app inherits /examples and its
// children after `mage setup`, regardless of feature selection.
func (ar *AppRoutes) initExamplesRoutes() {
	// Hub registration: /examples is the parent; each scaffold-facing
	// teaching page is a spoke. Hub is append-only across calls if more
	// children get added later from other initializers.
	linkwell.Hub("/examples", "Examples",
		linkwell.Rel("/examples/error-scenarios", "Error Scenarios"),
		linkwell.Rel("/examples/forms", "Forms"),
	)

	ar.e.GET("/examples", handler.HandleComponent(views.ExamplesIndexPage()))

	ar.initErrorScenariosRoutes()
	ar.initFormsRoutes()
}
