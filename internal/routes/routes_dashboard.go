// setup:feature:demo

package routes

import (
	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/hypermedia"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

type dashboardRoutes struct {
	db      *demo.DB
	board   *demo.KanbanBoard
	queue   *demo.ApprovalQueue
	actLog  *demo.ActivityLog
}

func (ar *appRoutes) initDashboardRoutes(db *demo.DB, board *demo.KanbanBoard, queue *demo.ApprovalQueue, actLog *demo.ActivityLog) {
	hypermedia.Link("/dashboard", "related", "/demo/inventory", "Inventory")
	hypermedia.Link("/dashboard", "related", "/demo/people", "People")
	hypermedia.Link("/dashboard", "related", "/demo/kanban", "Kanban")
	hypermedia.Link("/dashboard", "related", "/demo/approvals", "Approvals")
	hypermedia.Link("/dashboard", "related", "/demo/vendors", "Vendors")
	hypermedia.Link("/dashboard", "related", "/demo/feed", "Feed")

	d := &dashboardRoutes{db: db, board: board, queue: queue, actLog: actLog}
	ar.e.GET("/dashboard", d.handleDashboard)
}

func (d *dashboardRoutes) handleDashboard(c echo.Context) error {
	ctx := c.Request().Context()

	// Inventory stats
	_, itemTotal, _ := d.db.ListItems(ctx, "", "", "", "", "", 1, 1)
	_, itemActive, _ := d.db.ListItems(ctx, "", "", "true", "", "", 1, 1)

	// People stats
	_, peopleTotal, _ := d.db.ListPeople(ctx, "", "", "", "", 1, 1)

	// Vendor stats
	vendors, _ := d.db.ListVendors(ctx, "", "")
	vendorCount := len(vendors)

	// Kanban stats
	tasks := d.board.AllTasks()
	kanbanByStatus := make(map[string]int)
	for _, t := range tasks {
		kanbanByStatus[t.Status]++
	}

	// Approval stats
	requests := d.queue.AllRequests()
	approvalByStatus := make(map[string]int)
	var pendingAmount float64
	for _, r := range requests {
		approvalByStatus[r.Status]++
		if r.Status == "pending" {
			pendingAmount += r.Amount
		}
	}

	// Activity stats
	recentEvents := d.actLog.Recent(8)

	stats := views.DashboardStats{
		ItemTotal:        itemTotal,
		ItemActive:       itemActive,
		PeopleTotal:      peopleTotal,
		VendorCount:      vendorCount,
		TaskTotal:        len(tasks),
		KanbanByStatus:   kanbanByStatus,
		ApprovalTotal:    len(requests),
		ApprovalByStatus: approvalByStatus,
		PendingAmount:    pendingAmount,
		RecentEvents:     recentEvents,
	}

	return handler.RenderBaseLayout(c, views.DashboardPage(stats))
}
