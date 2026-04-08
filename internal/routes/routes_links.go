// setup:feature:demo

package routes

import "github.com/catgoose/linkwell"

// initLinkRelations registers all link relation declarations for the app.
// Hubs define parent→child discovery pages. Rings define peer groups.
// This is the single source of truth for the app's navigation topology.
func (ar *appRoutes) initLinkRelations() {
	// ── Hubs (discovery / index pages) ──────────────────────────────

	linkwell.Hub("/apps", "Applications",
		linkwell.Rel("/apps/inventory", "Inventory"),
		linkwell.Rel("/apps/catalog", "Catalog"),
		linkwell.Rel("/apps/people", "People"),
		linkwell.Rel("/apps/kanban", "Kanban"),
		linkwell.Rel("/apps/approvals", "Approvals"),
		linkwell.Rel("/apps/vendors", "Vendors"),
		linkwell.Rel("/apps/bulk", "Bulk"),
	)

	linkwell.Hub("/platform", "Platform",
		linkwell.Rel("/platform/logging", "Logging"),
		linkwell.Rel("/platform/repository", "Repository"),
		linkwell.Rel("/platform/settings", "Settings"),
		linkwell.Rel("/platform/pwa", "PWA Offline"),
	)

	linkwell.Hub("/patterns", "Patterns",
		linkwell.Rel("/patterns/controls", "Controls"),
		linkwell.Rel("/patterns/crud", "CRUD"),
		linkwell.Rel("/patterns/lists", "Lists"),
		linkwell.Rel("/patterns/interactions", "Interactions"),
		linkwell.Rel("/patterns/state", "State"),
		linkwell.Rel("/patterns/errors", "Errors"),
	)

	linkwell.Hub("/components", "Components",
		linkwell.Rel("/components/widgets", "Widgets"),
		linkwell.Rel("/components/cards", "Cards & Data"),
		linkwell.Rel("/components/advanced", "Advanced"),
	)

	linkwell.Hub("/realtime", "Real-time",
		linkwell.Rel("/realtime/dashboard", "Dashboard"),
		linkwell.Rel("/realtime/feed", "Feed"),
		linkwell.Rel("/realtime/canvas", "Canvas"),
		linkwell.Rel("/realtime/tavern", "Tavern Gallery"),
	)

	linkwell.Hub("/api", "API",
		linkwell.Rel("/api/hal", "HAL"),
		linkwell.Rel("/api/links", "Link Relations"),
		linkwell.Rel("/api/standards", "Web Standards"),
	)

	linkwell.Hub("/admin", "Admin",
		linkwell.Rel("/admin/health", "Health"),
		linkwell.Rel("/admin/sessions", "Sessions"),
		linkwell.Rel("/admin/settings", "Control Panel"),
		linkwell.Rel("/admin/error-traces", "Error Traces"),
		linkwell.Rel("/admin/error-reports", "Error Reports"),
		linkwell.Rel("/admin/system", "System"),
		linkwell.Rel("/admin/config", "Config"),
	)

	linkwell.Hub("/dashboard", "Dashboard",
		linkwell.Rel("/apps/inventory", "Inventory"),
		linkwell.Rel("/apps/people", "People"),
		linkwell.Rel("/apps/kanban", "Kanban"),
		linkwell.Rel("/apps/approvals", "Approvals"),
		linkwell.Rel("/apps/vendors", "Vendors"),
		linkwell.Rel("/realtime/feed", "Feed"),
	)

	// ── Rings (peer groups) ─────────────────────────────────────────

	// Applications: data management pages
	linkwell.Ring("Data",
		linkwell.Rel("/apps/inventory", "Inventory"),
		linkwell.Rel("/apps/catalog", "Catalog"),
		linkwell.Rel("/apps/bulk", "Bulk Ops"),
		linkwell.Rel("/apps/people", "People"),
		linkwell.Rel("/apps/vendors", "Vendors"),
	)

	// Applications: workflow and process pages
	linkwell.Ring("Workflow",
		linkwell.Rel("/apps/kanban", "Kanban"),
		linkwell.Rel("/apps/approvals", "Approvals"),
		linkwell.Rel("/realtime/feed", "Feed"),
	)

	// Platform: tools and utilities
	linkwell.Ring("Utility",
		linkwell.Rel("/platform/logging", "Logging"),
		linkwell.Rel("/realtime/canvas", "Canvas"),
		linkwell.Rel("/platform/settings", "Settings"),
		linkwell.Rel("/platform/repository", "Repository"),
	)

	// Dashboard children are also peers
	linkwell.Ring("Dashboard",
		linkwell.Rel("/apps/inventory", "Inventory"),
		linkwell.Rel("/apps/people", "People"),
		linkwell.Rel("/apps/kanban", "Kanban"),
		linkwell.Rel("/apps/approvals", "Approvals"),
		linkwell.Rel("/apps/vendors", "Vendors"),
		linkwell.Rel("/realtime/feed", "Feed"),
	)

	// Patterns: pattern pages
	linkwell.Ring("Patterns",
		linkwell.Rel("/patterns/controls", "Controls"),
		linkwell.Rel("/patterns/crud", "CRUD"),
		linkwell.Rel("/patterns/lists", "Lists"),
		linkwell.Rel("/patterns/interactions", "Interactions"),
		linkwell.Rel("/patterns/state", "State"),
		linkwell.Rel("/patterns/errors", "Errors"),
	)

	// Real-time pages
	linkwell.Ring("Real-time",
		linkwell.Rel("/realtime/dashboard", "Dashboard"),
		linkwell.Rel("/realtime/feed", "Feed"),
		linkwell.Rel("/realtime/canvas", "Canvas"),
	)

	// API pages
	linkwell.Ring("API",
		linkwell.Rel("/api/hal", "HAL"),
		linkwell.Rel("/api/links", "Link Relations"),
		linkwell.Rel("/api/standards", "Web Standards"),
	)

	// Component gallery pages
	linkwell.Ring("Components",
		linkwell.Rel("/components/widgets", "Widgets"),
		linkwell.Rel("/components/cards", "Cards & Data"),
		linkwell.Rel("/components/advanced", "Advanced"),
	)

	// Admin: operational pages
	linkwell.Ring("Admin Ops",
		linkwell.Rel("/admin/health", "Health"),
		linkwell.Rel("/admin/error-traces", "Error Traces"),
		linkwell.Rel("/admin/error-reports", "Error Reports"),
		linkwell.Rel("/admin/sessions", "Sessions"),
		linkwell.Rel("/admin/settings", "Control Panel"),
	)

	// Admin: system introspection
	linkwell.Ring("System",
		linkwell.Rel("/admin/system", "System"),
		linkwell.Rel("/admin/config", "Config"),
		linkwell.Rel("/admin/health", "Health"),
		linkwell.Rel("/admin/error-traces", "Error Traces"),
	)

	// ── Settings ────────────────────────────────────────────────────

	linkwell.Link("/settings", "related", "/user/settings", "Preferences")
	linkwell.Link("/settings", "related", "/admin/config", "Admin Config")
	linkwell.Link("/settings", "related", "/admin/settings", "Control Panel")

	// ── Action relations ────────────────────────────────────────────

	// List pages with create forms
	linkwell.Link("/apps/inventory", "create-form", "/apps/inventory/items/new", "New Item")
	linkwell.Link("/platform/repository", "create-form", "/platform/repository/tasks", "New Task")
}
