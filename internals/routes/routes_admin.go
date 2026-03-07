// setup:feature:demo

package routes

import (
	"database/sql"
	"fmt"

	"catgoose/go-htmx-demo/internals/database/dialect"
	"catgoose/go-htmx-demo/internals/database/schema"
	"catgoose/go-htmx-demo/internals/demo"
	"catgoose/go-htmx-demo/internals/routes/handler"
	"catgoose/go-htmx-demo/web/views"

	"github.com/labstack/echo/v4"
)

const seedDBPath = "db/seed.db"

type adminRoutes struct {
	seed *demo.SeedDB
}

func (ar *appRoutes) initAdminRoutes(appDB *sql.DB) {
	a := &adminRoutes{seed: demo.NewSeedDB(appDB)}
	ar.e.GET("/admin", a.handleAdminPage)
	ar.e.POST("/admin/db/attach", a.handleAttach)
	ar.e.POST("/admin/db/init-schema", a.handleInitSchema)
	ar.e.POST("/admin/db/copy/:table", a.handleCopyTable)
	ar.e.DELETE("/admin/db/tables", a.handleDropAll)
	ar.e.DELETE("/admin/db/tables/:table", a.handleDropTable)
}

func (a *adminRoutes) handleAdminPage(c echo.Context) error {
	seedTables, mainTables := a.gatherTables(c)
	return handler.RenderBaseLayout(c, views.AdminPage(a.seed.IsAttached(), seedTables, mainTables))
}

func (a *adminRoutes) handleAttach(c echo.Context) error {
	ctx := c.Request().Context()
	if err := a.seed.Attach(ctx, seedDBPath); err != nil {
		return a.renderWithLog(c, fmt.Sprintf("Failed to attach: %s", err), true)
	}
	return a.renderWithLog(c, "Attached seed.db", false)
}

func (a *adminRoutes) handleInitSchema(c echo.Context) error {
	ctx := c.Request().Context()
	d := dialect.SQLiteDialect{}

	for _, stmt := range schema.UsersTable.CreateSQL(d) {
		if _, err := a.seed.ExecSQL(ctx, stmt); err != nil {
			return a.renderWithLog(c, fmt.Sprintf("Init schema failed: %s", err), true)
		}
	}

	return a.renderWithLog(c, "Schema initialized (Users table created)", false)
}

func (a *adminRoutes) handleCopyTable(c echo.Context) error {
	ctx := c.Request().Context()
	table := c.Param("table")
	rows, err := a.seed.CopyTable(ctx, table)
	if err != nil {
		return a.renderWithLog(c, fmt.Sprintf("Copy %s failed: %s", table, err), true)
	}
	return a.renderWithLog(c, fmt.Sprintf("Copied %s (%d rows)", table, rows), false)
}

func (a *adminRoutes) handleDropAll(c echo.Context) error {
	ctx := c.Request().Context()
	tables, err := a.seed.MainTables(ctx)
	if err != nil {
		return a.renderWithLog(c, fmt.Sprintf("List tables failed: %s", err), true)
	}
	for _, t := range tables {
		if err := a.seed.DropMainTable(ctx, t.Name); err != nil {
			return a.renderWithLog(c, fmt.Sprintf("Drop %s failed: %s", t.Name, err), true)
		}
	}
	return a.renderWithLog(c, fmt.Sprintf("Dropped %d tables", len(tables)), false)
}

func (a *adminRoutes) handleDropTable(c echo.Context) error {
	ctx := c.Request().Context()
	table := c.Param("table")
	if err := a.seed.DropMainTable(ctx, table); err != nil {
		return a.renderWithLog(c, fmt.Sprintf("Drop %s failed: %s", table, err), true)
	}
	return a.renderWithLog(c, fmt.Sprintf("Dropped table %s", table), false)
}

func (a *adminRoutes) renderWithLog(c echo.Context, message string, isError bool) error {
	seedTables, mainTables := a.gatherTables(c)
	// Render the main content + OOB log entry
	if err := handler.RenderComponent(c, views.AdminContent(a.seed.IsAttached(), seedTables, mainTables)); err != nil {
		return err
	}
	return handler.RenderComponent(c, views.AdminActionLog(message, isError))
}

func (a *adminRoutes) gatherTables(c echo.Context) ([]demo.SeedTableInfo, []demo.SeedTableInfo) {
	ctx := c.Request().Context()
	var seedTables []demo.SeedTableInfo
	if a.seed.IsAttached() {
		seedTables, _ = a.seed.SeedTables(ctx)
	}
	mainTables, _ := a.seed.MainTables(ctx)
	return seedTables, mainTables
}
