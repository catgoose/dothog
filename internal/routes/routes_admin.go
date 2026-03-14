// setup:feature:demo

package routes

import (
	"fmt"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/ssebroker"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

type adminRoutes struct {
	db     *demo.DB
	actLog *demo.ActivityLog
	broker *ssebroker.SSEBroker
}

func (ar *appRoutes) initAdminRoutes(db *demo.DB, actLog *demo.ActivityLog, broker *ssebroker.SSEBroker) {
	a := &adminRoutes{db: db, actLog: actLog, broker: broker}
	ar.e.GET("/admin", a.handleAdminPage)
	ar.e.POST("/admin/db/reinit", a.handleReinit)
}

func (a *adminRoutes) handleAdminPage(c echo.Context) error {
	info, _ := a.db.GetSchemaInfo(c.Request().Context())
	return handler.RenderBaseLayout(c, views.AdminPage(info))
}

func (a *adminRoutes) handleReinit(c echo.Context) error {
	if err := a.db.Reset(); err != nil {
		info, _ := a.db.GetSchemaInfo(c.Request().Context())
		return handler.RenderComponent(c, views.AdminDBStatus(info, fmt.Sprintf("Reset failed: %s", err), true))
	}
	info, _ := a.db.GetSchemaInfo(c.Request().Context())

	evt := a.actLog.Record("created", "database", 0, "Database", "re-initialized with seed data")
	BroadcastActivity(a.broker, evt)

	return handler.RenderComponent(c, views.AdminDBStatus(info, "Database re-initialized with seed data", false))
}
