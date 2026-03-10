// setup:feature:demo

package routes

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"

	"catgoose/harmony/internals/demo"
	"catgoose/harmony/internals/routes/handler"
	"catgoose/harmony/internals/routes/hypermedia"
	"catgoose/harmony/web/views"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

const repoBase = "/demo/repository"

type repositoryRoutes struct {
	store *demo.TaskStore
}

func (ar *appRoutes) initRepositoryRoutes(db *demo.DB) {
	store := demo.NewTaskStore(db.RawDB())
	r := &repositoryRoutes{store: store}
	ar.e.GET(repoBase, r.handlePage)
	ar.e.GET(repoBase+"/tasks", r.handleTaskList)
	ar.e.GET(repoBase+"/tasks/new", r.handleNewTaskForm)
	ar.e.GET(repoBase+"/tasks/new/cancel", r.handleNewTaskCancel)
	ar.e.POST(repoBase+"/tasks", r.handleCreateTask)
	ar.e.GET(repoBase+"/tasks/:id", r.handleTaskRow)
	ar.e.GET(repoBase+"/tasks/:id/edit", r.handleEditTaskForm)
	ar.e.PUT(repoBase+"/tasks/:id", r.handleUpdateTask)
	ar.e.DELETE(repoBase+"/tasks/:id", r.handleDeleteTask)
	ar.e.POST(repoBase+"/tasks/:id/restore", r.handleRestoreTask)
	ar.e.POST(repoBase+"/tasks/:id/archive", r.handleArchiveTask)
	ar.e.POST(repoBase+"/tasks/:id/unarchive", r.handleUnarchiveTask)
}

func (r *repositoryRoutes) handlePage(c echo.Context) error {
	bar, container, err := r.buildContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load tasks", err)
	}
	return handler.RenderBaseLayout(c, views.RepositoryPage(bar, container))
}

func (r *repositoryRoutes) handleTaskList(c echo.Context) error {
	_, container, err := r.buildContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load tasks", err)
	}
	setTableReplaceURL(c, repoBase)
	return handler.RenderComponent(c, container)
}

func (r *repositoryRoutes) handleNewTaskForm(c echo.Context) error {
	filterQuery := filterQueryFromHXCurrentURL(c)
	saveURL := repoBase + "/tasks"
	if filterQuery != "" {
		saveURL += "?" + filterQuery
	}
	return handler.RenderComponent(c, views.RepositoryEditRow(demo.Task{}, true, saveURL, repoBase+"/tasks/new/cancel"))
}

func (r *repositoryRoutes) handleNewTaskCancel(c echo.Context) error {
	return handler.RenderComponent(c, views.NewTaskPlaceholder())
}

func (r *repositoryRoutes) handleCreateTask(c echo.Context) error {
	t := parseTaskForm(c)
	if err := r.store.CreateTask(c.Request().Context(), &t); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to create task", err)
	}
	_, container, err := r.buildContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, repoBase)
	return handler.RenderComponent(c, container)
}

func (r *repositoryRoutes) handleTaskRow(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid task ID", err)
	}
	task, err := r.store.GetTask(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Task not found", err)
	}
	return handler.RenderComponent(c, views.RepositoryTaskRow(task))
}

func (r *repositoryRoutes) handleEditTaskForm(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid task ID", err)
	}
	task, err := r.store.GetTask(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Task not found", err)
	}
	filterQuery := filterQueryFromHXCurrentURL(c)
	baseURL := fmt.Sprintf(repoBase+"/tasks/%d", id)
	saveURL := baseURL
	if filterQuery != "" {
		saveURL += "?" + filterQuery
	}
	return handler.RenderComponent(c, views.RepositoryEditRow(task, false, saveURL, baseURL))
}

func (r *repositoryRoutes) handleUpdateTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid task ID", err)
	}
	existing, err := r.store.GetTask(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Task not found", err)
	}
	t := parseTaskForm(c)
	t.ID = existing.ID
	t.Version = existing.Version
	t.CreatedAt = existing.CreatedAt
	t.ArchivedAt = existing.ArchivedAt
	t.ReplacedBy = existing.ReplacedBy
	t.DeletedAt = existing.DeletedAt
	if err := r.store.UpdateTask(c.Request().Context(), &t); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to update task", err)
	}
	_, container, err := r.buildContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, repoBase)
	return handler.RenderComponent(c, container)
}

func (r *repositoryRoutes) handleDeleteTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid task ID", err)
	}
	if err := r.store.SoftDeleteTask(c.Request().Context(), id); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to delete task", err)
	}
	applyFilterFromCurrentURL(c)
	_, container, err := r.buildContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, repoBase)
	return handler.RenderComponent(c, container)
}

func (r *repositoryRoutes) handleRestoreTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid task ID", err)
	}
	if err := r.store.RestoreTask(c.Request().Context(), id); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to restore task", err)
	}
	applyFilterFromCurrentURL(c)
	_, container, err := r.buildContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, repoBase)
	return handler.RenderComponent(c, container)
}

func (r *repositoryRoutes) handleArchiveTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid task ID", err)
	}
	if err := r.store.ArchiveTask(c.Request().Context(), id); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to archive task", err)
	}
	applyFilterFromCurrentURL(c)
	_, container, err := r.buildContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, repoBase)
	return handler.RenderComponent(c, container)
}

func (r *repositoryRoutes) handleUnarchiveTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid task ID", err)
	}
	if err := r.store.UnarchiveTask(c.Request().Context(), id); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to unarchive task", err)
	}
	applyFilterFromCurrentURL(c)
	_, container, err := r.buildContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, repoBase)
	return handler.RenderComponent(c, container)
}

func parseTaskForm(c echo.Context) demo.Task {
	sortOrder, _ := strconv.Atoi(c.FormValue("sortorder"))
	return demo.Task{
		Title:       c.FormValue("title"),
		Description: toNullString(c.FormValue("description")),
		Status:      c.FormValue("status"),
		SortOrder:   sortOrder,
		Notes:       toNullString(c.FormValue("notes")),
	}
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

type repoTableParams struct {
	Q            string
	Status       string
	ShowArchived string
	ShowDeleted  string
	Sort         string
	Dir          string
	Page         int
	PerPage      int
}

func parseRepoTableParams(c echo.Context, perPage int) repoTableParams {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	return repoTableParams{
		Q:            c.QueryParam("q"),
		Status:       c.QueryParam("status"),
		ShowArchived: c.QueryParam("archived"),
		ShowDeleted:  c.QueryParam("deleted"),
		Sort:         c.QueryParam("sort"),
		Dir:          c.QueryParam("dir"),
		Page:         page,
		PerPage:      perPage,
	}
}

func (r *repositoryRoutes) buildContent(c echo.Context) (hypermedia.FilterBar, templ.Component, error) {
	const perPage = 10
	p := parseRepoTableParams(c, perPage)

	tasks, total, err := r.store.ListTasks(
		c.Request().Context(),
		p.Q, p.Status, p.ShowArchived, p.ShowDeleted,
		p.Sort, p.Dir, p.Page, p.PerPage,
	)
	if err != nil {
		return hypermedia.FilterBar{}, nil, err
	}

	itemsURL := repoBase + "/tasks"
	target := "#repo-table-container"

	bar := hypermedia.NewFilterBar(itemsURL, target,
		hypermedia.SearchField("q", "Search tasks...", p.Q),
		hypermedia.SelectField("status", "Status", p.Status,
			hypermedia.SelectOptions(p.Status,
				"", "All",
				"draft", "Draft",
				"active", "Active",
				"done", "Done",
			)),
		hypermedia.CheckboxField("archived", "Show archived", p.ShowArchived),
		hypermedia.CheckboxField("deleted", "Show deleted", p.ShowDeleted),
	)

	sortBase := repoStripParams(c.Request().URL, "sort", "dir")
	cols := []hypermedia.TableCol{
		hypermedia.SortableCol("title", "Title", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		{Label: "Description"},
		hypermedia.SortableCol("status", "Status", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		hypermedia.SortableCol("sortorder", "Order", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		hypermedia.SortableCol("version", "Ver", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		{Label: "State"},
		hypermedia.SortableCol("updated", "Updated", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		{Label: "Actions"},
	}

	pageBase := repoStripParams(c.Request().URL, "page")
	info := hypermedia.PageInfo{
		Page:       p.Page,
		PerPage:    p.PerPage,
		TotalItems: total,
		TotalPages: hypermedia.ComputeTotalPages(total, p.PerPage),
		BaseURL:    pageBase,
		Target:     target,
		Include:    "#filter-form",
	}

	body := views.RepositoryTasksBody(tasks)
	container := views.RepositoryTableContainer(cols, body, info)
	return bar, container, nil
}

func repoStripParams(u *url.URL, params ...string) string {
	return stripParams(u, params...)
}
