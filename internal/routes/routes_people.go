// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"github.com/catgoose/linkwell"
	"catgoose/dothog/internal/routes/params"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/internal/ssetopics"
	"github.com/catgoose/tavern"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

const peopleBase = "/demo/people"

type peopleRoutes struct {
	db     *demo.DB
	broker *tavern.SSEBroker
	actLog *demo.ActivityLog
}

func (ar *appRoutes) initPeopleRoutes(db *demo.DB, broker *tavern.SSEBroker, actLog *demo.ActivityLog) {
	p := &peopleRoutes{db: db, broker: broker, actLog: actLog}
	ar.e.GET(peopleBase, p.handlePeoplePage)
	ar.e.GET(peopleBase+"/list", p.handlePeopleList)
	ar.e.GET(peopleBase+"/:id", p.handlePersonProfile)
	ar.e.GET(peopleBase+"/:id/edit", p.handlePersonEdit)
	ar.e.GET(peopleBase+"/:id/card", p.handlePersonCard)
	ar.e.PUT(peopleBase+"/:id", p.handlePersonUpdate)
	ar.e.GET("/sse/people/:id", p.handlePersonSSE)
}

func (p *peopleRoutes) handlePeoplePage(c echo.Context) error {
	people, total, bar, cols, info, err := p.buildPeopleContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load people", err)
	}
	from := c.QueryParam("from")
	return handler.RenderBaseLayout(c, views.PeoplePage(people, total, bar, cols, info, from))
}

func (p *peopleRoutes) handlePeopleList(c echo.Context) error {
	people, _, _, cols, info, err := p.buildPeopleContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load people", err)
	}
	from := c.QueryParam("from")
	return handler.RenderComponent(c, views.PeopleTableContainer(cols, people, info, from))
}

func (p *peopleRoutes) handlePersonProfile(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid ID", err)
	}
	person, err := p.db.GetPerson(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Person not found", err)
	}
	from := c.QueryParam("from")
	handler.SetPageLabel(c, person.FullName())
	return handler.RenderBaseLayout(c, views.PersonProfilePage(person, from))
}

func (p *peopleRoutes) handlePersonEdit(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid ID", err)
	}
	person, err := p.db.GetPerson(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Person not found", err)
	}
	return handler.RenderComponent(c, views.PersonEditForm(person))
}

func (p *peopleRoutes) handlePersonCard(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid ID", err)
	}
	person, err := p.db.GetPerson(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Person not found", err)
	}
	return handler.RenderComponent(c, views.PersonProfileCard(person))
}

func (p *peopleRoutes) handlePersonUpdate(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid ID", err)
	}
	person := demo.Person{
		ID:         id,
		FirstName:  c.FormValue("first_name"),
		LastName:   c.FormValue("last_name"),
		Email:      c.FormValue("email"),
		Phone:      c.FormValue("phone"),
		City:       c.FormValue("city"),
		State:      c.FormValue("state"),
		Department: c.FormValue("department"),
		JobTitle:   c.FormValue("job_title"),
		Bio:        c.FormValue("bio"),
	}
	if err := p.db.UpdatePerson(c.Request().Context(), person); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to update person", err)
	}
	// Re-fetch to get full data including created_at
	person, err = p.db.GetPerson(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to re-fetch person after update", err)
	}

	// Record activity and broadcast to feed
	evt := p.actLog.Record("updated", "person", id, person.FullName(), "profile updated")
	BroadcastActivity(p.broker, evt)

	// Broadcast SSE update to all profile viewers
	p.broadcastPersonUpdate(person)

	return handler.RenderComponent(c, views.PersonProfileCard(person))
}

func (p *peopleRoutes) broadcastPersonUpdate(person demo.Person) {
	topic := fmt.Sprintf("%s-%d", ssetopics.TopicPeopleUpdate, person.ID)
	if !p.broker.HasSubscribers(topic) {
		return
	}
	buf := statsBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	if err := views.PersonProfileCardOOB(person).Render(shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "broadcast person update"), buf); err != nil {
		statsBufPool.Put(buf)
		return
	}
	msg := tavern.NewSSEMessage("person-update", buf.String()).String()
	statsBufPool.Put(buf)
	p.broker.Publish(topic, msg)
}

func (p *peopleRoutes) handlePersonSSE(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid person ID", err)
	}
	topic := fmt.Sprintf("%s-%d", ssetopics.TopicPeopleUpdate, id)

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	ch, unsub := p.broker.Subscribe(topic)
	defer unsub()

	ctx := c.Request().Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			fmt.Fprint(c.Response(), msg)
			flusher.Flush()
		}
	}
}

func (p *peopleRoutes) buildPeopleContent(c echo.Context) ([]demo.Person, int, linkwell.FilterBar, []linkwell.TableCol, linkwell.PageInfo, error) {
	const perPage = 20
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	search := c.QueryParam("q")
	department := c.QueryParam("department")
	sort := c.QueryParam("sort")
	dir := c.QueryParam("dir")

	people, total, err := p.db.ListPeople(c.Request().Context(), search, department, sort, dir, page, perPage)
	if err != nil {
		return nil, 0, linkwell.FilterBar{}, nil, linkwell.PageInfo{}, err
	}

	bar := linkwell.NewFilterBar(peopleBase+"/list", "#people-table-container",
		linkwell.SearchField("q", "Search people\u2026", search),
		linkwell.SelectField("department", "Department", department,
			deptOptions(department)),
	)

	sortBase := buildSortBase(c)
	cols := []linkwell.TableCol{
		linkwell.SortableCol("name", "Name", sort, dir, sortBase, "#people-table-container", "#filter-form"),
		linkwell.SortableCol("department", "Department", sort, dir, sortBase, "#people-table-container", "#filter-form"),
		{Label: "Title"},
		linkwell.SortableCol("city", "Location", sort, dir, sortBase, "#people-table-container", "#filter-form"),
		{Label: "Email"},
	}

	info := buildPageInfo(c, page, perPage, total, "#people-table-container")

	return people, total, bar, cols, info, nil
}

func deptOptions(current string) []linkwell.FilterOption {
	opts := []linkwell.FilterOption{
		{Value: "", Label: "All Departments", Selected: current == ""},
	}
	for _, d := range demo.Departments {
		opts = append(opts, linkwell.FilterOption{Value: d, Label: d, Selected: current == d})
	}
	return opts
}
