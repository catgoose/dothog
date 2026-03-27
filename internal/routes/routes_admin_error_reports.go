// setup:feature:demo

package routes

import (
	"strconv"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/hypermedia"
	"catgoose/dothog/internal/routes/params"
	"catgoose/dothog/web/views"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

const errorReportsBase = "/admin/error-reports"

type errorReportRoutes struct{ db *demo.DB }

func (ar *appRoutes) initAdminErrorReportsRoutes(db *demo.DB) {
	hypermedia.Ring(
		hypermedia.Rel("/admin/error-reports", "Error Reports"),
		hypermedia.Rel("/admin/health", "Health"),
		hypermedia.Rel("/admin/error-traces", "Error Traces"),
		hypermedia.Rel("/admin/sessions", "Sessions"),
		hypermedia.Rel("/admin/settings", "Control Panel"),
	)

	d := &errorReportRoutes{db: db}
	ar.e.GET(errorReportsBase, d.handleErrorReportsPage)
	ar.e.GET(errorReportsBase+"/table", d.handleErrorReportsTable)
	ar.e.POST(errorReportsBase+"/:id/resolve", d.handleResolveReport)
	ar.e.POST(errorReportsBase+"/:id/dismiss", d.handleDismissReport)
}

func (d *errorReportRoutes) handleErrorReportsPage(c echo.Context) error {
	bar, container, err := d.buildErrorReportsContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load error reports", err)
	}
	return handler.RenderBaseLayout(c, views.AdminErrorReportsPage(bar, container))
}

func (d *errorReportRoutes) handleErrorReportsTable(c echo.Context) error {
	_, container, err := d.buildErrorReportsContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load error reports", err)
	}
	setTableReplaceURL(c, errorReportsBase)
	return handler.RenderComponent(c, container)
}

func (d *errorReportRoutes) handleResolveReport(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid report ID", err)
	}
	if err := d.db.UpdateErrorReportStatus(c.Request().Context(), id, demo.ErrorReportStatusResolved); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to resolve report", err)
	}
	applyFilterFromCurrentURL(c)
	_, container, err := d.buildErrorReportsContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, errorReportsBase)
	return handler.RenderComponent(c, container)
}

func (d *errorReportRoutes) handleDismissReport(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid report ID", err)
	}
	if err := d.db.UpdateErrorReportStatus(c.Request().Context(), id, demo.ErrorReportStatusDismissed); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to dismiss report", err)
	}
	applyFilterFromCurrentURL(c)
	_, container, err := d.buildErrorReportsContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, errorReportsBase)
	return handler.RenderComponent(c, container)
}

func (d *errorReportRoutes) buildErrorReportsContent(c echo.Context) (hypermedia.FilterBar, templ.Component, error) {
	const perPage = 20
	p := parseErrorReportParams(c, perPage)

	reports, total, err := d.db.ListErrorReports(c.Request().Context(), p.Q, p.Status, p.Sort, p.Dir, p.Page, p.PerPage)
	if err != nil {
		return hypermedia.FilterBar{}, nil, err
	}

	target := "#error-reports-table-container"
	bar := hypermedia.NewFilterBar(errorReportsBase+"/table", target,
		hypermedia.SearchField("q", "Search reports\u2026", p.Q),
		hypermedia.SelectField("status", "Status", p.Status,
			hypermedia.SelectOptions(p.Status, errorReportStatusPairs()...)),
	)

	sortBase := buildSortBase(c)
	cols := []hypermedia.TableCol{
		hypermedia.SortableCol("created_at", "Timestamp", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		{Label: "Request ID"},
		{Label: "Description"},
		hypermedia.SortableCol("route", "Route", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		hypermedia.SortableCol("status_code", "Code", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		{Label: "User Agent"},
		hypermedia.SortableCol("status", "Status", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		{Label: "Actions"},
	}

	info := buildPageInfo(c, p.Page, p.PerPage, total, target)

	body := views.AdminErrorReportsBody(reports)
	container := views.AdminErrorReportsTableContainer(cols, body, info)
	return bar, container, nil
}

type errorReportParams struct {
	Q       string
	Status  string
	Sort    string
	Dir     string
	Page    int
	PerPage int
}

func parseErrorReportParams(c echo.Context, perPage int) errorReportParams {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	return errorReportParams{
		Q:       c.QueryParam("q"),
		Status:  c.QueryParam("status"),
		Sort:    c.QueryParam("sort"),
		Dir:     c.QueryParam("dir"),
		Page:    page,
		PerPage: perPage,
	}
}

func errorReportStatusPairs() []string {
	return []string{
		"", "All",
		string(demo.ErrorReportStatusPending), "Pending",
		string(demo.ErrorReportStatusResolved), "Resolved",
		string(demo.ErrorReportStatusDismissed), "Dismissed",
	}
}
