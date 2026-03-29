// setup:feature:demo

package routes

import "catgoose/dothog/internal/routes/hypermedia"

// initLinkRelations registers all link relation declarations for the app.
// Hubs define parent→child discovery pages. Rings define peer groups.
// This is the single source of truth for the app's navigation topology.
func (ar *appRoutes) initLinkRelations() {
	// ── Hubs (discovery / index pages) ──────────────────────────────

	hypermedia.Hub("/demo", "Demo",
		hypermedia.Rel("/demo/inventory", "Inventory"),
		hypermedia.Rel("/demo/catalog", "Catalog"),
		hypermedia.Rel("/demo/people", "People"),
		hypermedia.Rel("/demo/kanban", "Kanban"),
		hypermedia.Rel("/demo/approvals", "Approvals"),
		hypermedia.Rel("/demo/vendors", "Vendors"),
		hypermedia.Rel("/demo/feed", "Feed"),
		hypermedia.Rel("/demo/canvas", "Canvas"),
		hypermedia.Rel("/demo/settings", "Settings"),
		hypermedia.Rel("/demo/bulk", "Bulk"),
		hypermedia.Rel("/demo/logging", "Logging"),
		hypermedia.Rel("/demo/repository", "Repository"),
		hypermedia.Rel("/dashboard", "Dashboard"),
		hypermedia.Rel("/pwa", "PWA Offline"),
	)

	hypermedia.Hub("/hypermedia", "Hypermedia",
		hypermedia.Rel("/hypermedia/controls", "Controls"),
		hypermedia.Rel("/hypermedia/crud", "CRUD"),
		hypermedia.Rel("/hypermedia/lists", "Lists"),
		hypermedia.Rel("/hypermedia/interactions", "Interactions"),
		hypermedia.Rel("/hypermedia/state", "State"),
		hypermedia.Rel("/hypermedia/errors", "Errors"),
		hypermedia.Rel("/hypermedia/components", "Components"),
		hypermedia.Rel("/hypermedia/components2", "Components 2"),
		hypermedia.Rel("/hypermedia/components3", "Components 3"),
		hypermedia.Rel("/hypermedia/realtime", "Realtime"),
		hypermedia.Rel("/hypermedia/links", "Links"),
		hypermedia.Rel("/hypermedia/standards", "Standards"),
	)

	hypermedia.Hub("/admin", "Admin",
		hypermedia.Rel("/admin/health", "Health"),
		hypermedia.Rel("/admin/sessions", "Sessions"),
		hypermedia.Rel("/admin/settings", "Control Panel"),
		hypermedia.Rel("/admin/error-traces", "Error Traces"),
		hypermedia.Rel("/admin/error-reports", "Error Reports"),
		hypermedia.Rel("/admin/system", "System"),
		hypermedia.Rel("/admin/config", "Config"),
	)

	hypermedia.Hub("/dashboard", "Dashboard",
		hypermedia.Rel("/demo/inventory", "Inventory"),
		hypermedia.Rel("/demo/people", "People"),
		hypermedia.Rel("/demo/kanban", "Kanban"),
		hypermedia.Rel("/demo/approvals", "Approvals"),
		hypermedia.Rel("/demo/vendors", "Vendors"),
		hypermedia.Rel("/demo/feed", "Feed"),
	)

	// ── Rings (peer groups) ─────────────────────────────────────────

	// Demo: data management pages
	hypermedia.Ring("Data",
		hypermedia.Rel("/demo/inventory", "Inventory"),
		hypermedia.Rel("/demo/catalog", "Catalog"),
		hypermedia.Rel("/demo/bulk", "Bulk Ops"),
		hypermedia.Rel("/demo/people", "People"),
		hypermedia.Rel("/demo/vendors", "Vendors"),
	)

	// Demo: workflow and process pages
	hypermedia.Ring("Workflow",
		hypermedia.Rel("/demo/kanban", "Kanban"),
		hypermedia.Rel("/demo/approvals", "Approvals"),
		hypermedia.Rel("/demo/feed", "Feed"),
	)

	// Demo: tools and utilities
	hypermedia.Ring("Utility",
		hypermedia.Rel("/demo/logging", "Logging"),
		hypermedia.Rel("/demo/canvas", "Canvas"),
		hypermedia.Rel("/demo/settings", "Settings"),
		hypermedia.Rel("/demo/repository", "Repository"),
	)

	// Demo: dashboard children are also peers
	hypermedia.Ring("Dashboard",
		hypermedia.Rel("/demo/inventory", "Inventory"),
		hypermedia.Rel("/demo/people", "People"),
		hypermedia.Rel("/demo/kanban", "Kanban"),
		hypermedia.Rel("/demo/approvals", "Approvals"),
		hypermedia.Rel("/demo/vendors", "Vendors"),
		hypermedia.Rel("/demo/feed", "Feed"),
	)

	// Hypermedia: pattern pages
	hypermedia.Ring("Patterns",
		hypermedia.Rel("/hypermedia/controls", "Controls"),
		hypermedia.Rel("/hypermedia/crud", "CRUD"),
		hypermedia.Rel("/hypermedia/lists", "Lists"),
		hypermedia.Rel("/hypermedia/interactions", "Interactions"),
		hypermedia.Rel("/hypermedia/state", "State"),
		hypermedia.Rel("/hypermedia/errors", "Errors"),
		hypermedia.Rel("/hypermedia/realtime", "Realtime"),
		hypermedia.Rel("/hypermedia/links", "Links"),
		hypermedia.Rel("/hypermedia/standards", "Standards"),
	)

	// Hypermedia: component gallery pages
	hypermedia.Ring("Components",
		hypermedia.Rel("/hypermedia/components", "Components"),
		hypermedia.Rel("/hypermedia/components2", "Components 2"),
		hypermedia.Rel("/hypermedia/components3", "Components 3"),
	)

	// Admin: operational pages
	hypermedia.Ring("Admin Ops",
		hypermedia.Rel("/admin/health", "Health"),
		hypermedia.Rel("/admin/error-traces", "Error Traces"),
		hypermedia.Rel("/admin/error-reports", "Error Reports"),
		hypermedia.Rel("/admin/sessions", "Sessions"),
		hypermedia.Rel("/admin/settings", "Control Panel"),
	)

	// Admin: system introspection
	hypermedia.Ring("System",
		hypermedia.Rel("/admin/system", "System"),
		hypermedia.Rel("/admin/config", "Config"),
		hypermedia.Rel("/admin/health", "Health"),
		hypermedia.Rel("/admin/error-traces", "Error Traces"),
	)

	// ── Settings ────────────────────────────────────────────────────

	hypermedia.Link("/settings", "related", "/user/settings", "Preferences")
	hypermedia.Link("/settings", "related", "/admin/config", "Admin Config")
	hypermedia.Link("/settings", "related", "/admin/settings", "Control Panel")

	// ── Action relations ────────────────────────────────────────────

	// List pages with create forms
	hypermedia.Link("/demo/inventory", "create-form", "/demo/inventory/items/new", "New Item")
	hypermedia.Link("/demo/repository", "create-form", "/demo/repository/tasks", "New Task")
}
