// setup:feature:demo

package routes

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/hypermedia"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

const hypermediaBase = "/hypermedia"

// crudItem is the in-memory demo item for the CRUD page.
type crudItem struct {
	Name   string
	Status string
	Notes  string
	ID     int
}

// hypermediaState holds all mutable demo state for the /hypermedia/* pages.
type hypermediaState struct {
	items     []crudItem
	comments  []string
	nextID    int
	likeCount int
	mu        sync.RWMutex
	toggleOn  bool
}

func newHypermediaState() *hypermediaState {
	s := &hypermediaState{nextID: 1}
	s.items = []crudItem{
		{ID: s.nextID, Name: "Widget Alpha", Status: "active", Notes: "First demo item"},
		{ID: s.nextID + 1, Name: "Gadget Beta", Status: "inactive", Notes: "Second demo item"},
		{ID: s.nextID + 2, Name: "Doohickey Gamma", Status: "active", Notes: "Third demo item"},
	}
	s.nextID = 4
	return s
}

func (ar *appRoutes) initHypermediaRoutes() {
	// Links demo page
	ar.e.GET(hypermediaBase+"/links", func(c echo.Context) error {
		links := hypermedia.AllLinks()
		return handler.RenderBaseLayout(c, views.HypermediaLinksPage(links))
	})

	s := newHypermediaState()

	// CRUD page
	ar.e.GET(hypermediaBase+"/crud", s.handleCRUDPage)
	ar.e.GET(hypermediaBase+"/crud/items", s.handleCRUDItems)
	ar.e.POST(hypermediaBase+"/crud/items", s.handleCRUDCreate)
	ar.e.GET(hypermediaBase+"/crud/items/:id", s.handleCRUDItemRow)
	ar.e.GET(hypermediaBase+"/crud/items/:id/edit", s.handleCRUDEditForm)
	ar.e.PUT(hypermediaBase+"/crud/items/:id", s.handleCRUDUpdate)
	ar.e.PATCH(hypermediaBase+"/crud/items/:id/toggle", s.handleCRUDPatchToggle)
	ar.e.DELETE(hypermediaBase+"/crud/items/:id", s.handleCRUDDelete)

	// Lists page
	ar.e.GET(hypermediaBase+"/lists", s.handleListsPage)
	ar.e.GET(hypermediaBase+"/lists/items", handleListsItems)

	// Interactions page
	ar.e.GET(hypermediaBase+"/interactions", s.handleInteractionsPage)
	ar.e.GET(hypermediaBase+"/interactions/modal", s.handleInteractionsModal)
	ar.e.POST(hypermediaBase+"/interactions/submit", s.handleInteractionsSubmit)
	ar.e.POST(hypermediaBase+"/interactions/preview", s.handleInteractionsPreview)
	ar.e.POST(hypermediaBase+"/interactions/comment", s.handleInteractionsComment)

	// State page
	ar.e.GET(hypermediaBase+"/state", s.handleStatePage)
	ar.e.POST(hypermediaBase+"/state/like", s.handleStateLike)
	ar.e.POST(hypermediaBase+"/state/toggle", s.handleStateToggle)
	ar.e.GET(hypermediaBase+"/state/panel", s.handleStatePanel)
}

// ─── CRUD handlers ────────────────────────────────────────────────────────────

func (s *hypermediaState) handleCRUDPage(c echo.Context) error {
	s.mu.RLock()
	items := make([]crudItem, len(s.items))
	copy(items, s.items)
	s.mu.RUnlock()
	return handler.RenderBaseLayout(c, views.CRUDPage(crudItemsToView(items)))
}

func (s *hypermediaState) handleCRUDItems(c echo.Context) error {
	s.mu.RLock()
	items := make([]crudItem, len(s.items))
	copy(items, s.items)
	s.mu.RUnlock()
	return handler.RenderComponent(c, views.CRUDItemsTable(crudItemsToView(items)))
}

func (s *hypermediaState) handleCRUDCreate(c echo.Context) error {
	name := c.FormValue("name")
	notes := c.FormValue("notes")
	if name == "" {
		name = "New Item"
	}
	s.mu.Lock()
	item := crudItem{ID: s.nextID, Name: name, Status: "active", Notes: notes}
	s.items = append(s.items, item)
	s.nextID++
	s.mu.Unlock()
	return handler.RenderComponent(c, views.CRUDItemRow(item.toView()))
}

func (s *hypermediaState) handleCRUDItemRow(c echo.Context) error {
	id, err := parseCRUDID(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", err)
	}
	s.mu.RLock()
	item, found := s.findItem(id)
	s.mu.RUnlock()
	if !found {
		return handler.HandleHypermediaError(c, 404, "Item not found", fmt.Errorf("id=%d", id))
	}
	return handler.RenderComponent(c, views.CRUDItemRow(item.toView()))
}

func (s *hypermediaState) handleCRUDEditForm(c echo.Context) error {
	id, err := parseCRUDID(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", err)
	}
	s.mu.RLock()
	item, found := s.findItem(id)
	s.mu.RUnlock()
	if !found {
		return handler.HandleHypermediaError(c, 404, "Item not found", fmt.Errorf("id=%d", id))
	}
	return handler.RenderComponent(c, views.CRUDEditRow(item.toView()))
}

func (s *hypermediaState) handleCRUDUpdate(c echo.Context) error {
	id, err := parseCRUDID(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", err)
	}
	name := c.FormValue("name")
	if name == "" {
		return handler.HandleHypermediaError(c, 400, "Name is required", fmt.Errorf("empty name for id=%d", id))
	}
	notes := c.FormValue("notes")
	s.mu.Lock()
	idx := s.findIndex(id)
	if idx < 0 {
		s.mu.Unlock()
		return handler.HandleHypermediaError(c, 404, "Item not found", fmt.Errorf("id=%d", id))
	}
	s.items[idx].Name = name
	s.items[idx].Notes = notes
	item := s.items[idx]
	s.mu.Unlock()
	return handler.RenderComponent(c, views.CRUDItemRow(item.toView()))
}

func (s *hypermediaState) handleCRUDPatchToggle(c echo.Context) error {
	id, err := parseCRUDID(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", err)
	}
	s.mu.Lock()
	idx := s.findIndex(id)
	if idx < 0 {
		s.mu.Unlock()
		return handler.HandleHypermediaError(c, 404, "Item not found", fmt.Errorf("id=%d", id))
	}
	if s.items[idx].Status == "active" {
		s.items[idx].Status = "inactive"
	} else {
		s.items[idx].Status = "active"
	}
	item := s.items[idx]
	s.mu.Unlock()
	return handler.RenderComponent(c, views.CRUDItemRow(item.toView()))
}

func (s *hypermediaState) handleCRUDDelete(c echo.Context) error {
	id, err := parseCRUDID(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", err)
	}
	s.mu.Lock()
	idx := s.findIndex(id)
	if idx >= 0 {
		s.items = append(s.items[:idx], s.items[idx+1:]...)
	}
	s.mu.Unlock()
	return c.NoContent(200)
}

// ─── Lists handlers ────────────────────────────────────────────────────────────

const (
	listsDemoTotal   = 47
	listsDemoPerPage = 5
)

func (s *hypermediaState) handleListsPage(c echo.Context) error {
	items := generateListsDemoItems(1, listsDemoPerPage, listsDemoTotal)
	info := listsPageInfo(1)
	return handler.RenderBaseLayout(c, views.ListsPage(items, info))
}

func handleListsItems(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	items := generateListsDemoItems(page, listsDemoPerPage, listsDemoTotal)
	info := listsPageInfo(page)
	return handler.RenderComponent(c, views.ListsDemoTable(items, info))
}

func listsPageInfo(page int) hypermedia.PageInfo {
	return hypermedia.PageInfo{
		Page:       page,
		PerPage:    listsDemoPerPage,
		TotalItems: listsDemoTotal,
		TotalPages: hypermedia.ComputeTotalPages(listsDemoTotal, listsDemoPerPage),
		BaseURL:    hypermediaBase + "/lists/items",
		Target:     "#lists-table-container",
	}
}

func generateListsDemoItems(page, perPage, total int) []views.ListsDemoItem {
	categories := []string{"Electronics", "Clothing", "Food", "Books"}
	start := (page - 1) * perPage
	end := start + perPage
	if end > total {
		end = total
	}
	if start >= total {
		return nil
	}
	items := make([]views.ListsDemoItem, 0, end-start)
	for i := start; i < end; i++ {
		id := i + 1
		items = append(items, views.ListsDemoItem{
			ID:       id,
			Name:     fmt.Sprintf("Item %d", id),
			Category: categories[i%len(categories)],
		})
	}
	return items
}

// ─── Interactions handlers ─────────────────────────────────────────────────────

func (s *hypermediaState) handleInteractionsPage(c echo.Context) error {
	s.mu.RLock()
	comments := make([]string, len(s.comments))
	copy(comments, s.comments)
	s.mu.RUnlock()
	return handler.RenderBaseLayout(c, views.InteractionsPage(comments))
}

func (s *hypermediaState) handleInteractionsModal(c echo.Context) error {
	return handler.RenderComponent(c, views.InteractionsModalFragment())
}

func (s *hypermediaState) handleInteractionsSubmit(c echo.Context) error {
	name := c.FormValue("contact-name")
	email := c.FormValue("contact-email")
	msg := c.FormValue("contact-message")
	return handler.RenderComponent(c, views.InteractionsSubmitResult(name, email, msg))
}

func (s *hypermediaState) handleInteractionsPreview(c echo.Context) error {
	text := c.FormValue("preview-text")
	return handler.RenderComponent(c, views.InteractionsPreviewFragment(text))
}

func (s *hypermediaState) handleInteractionsComment(c echo.Context) error {
	text := c.FormValue("comment-text")
	if text == "" {
		return c.NoContent(200)
	}
	s.mu.Lock()
	s.comments = append(s.comments, text)
	comment := text
	s.mu.Unlock()
	return handler.RenderComponent(c, views.InteractionsCommentItem(comment))
}

// ─── State handlers ────────────────────────────────────────────────────────────

func (s *hypermediaState) handleStatePage(c echo.Context) error {
	s.mu.RLock()
	like := s.likeCount
	tog := s.toggleOn
	s.mu.RUnlock()
	return handler.RenderBaseLayout(c, views.StatePage(like, tog))
}

func (s *hypermediaState) handleStateLike(c echo.Context) error {
	s.mu.Lock()
	s.likeCount++
	count := s.likeCount
	s.mu.Unlock()
	return handler.RenderComponent(c, views.LikeButtonFragment(count))
}

func (s *hypermediaState) handleStateToggle(c echo.Context) error {
	s.mu.Lock()
	s.toggleOn = !s.toggleOn
	on := s.toggleOn
	s.mu.Unlock()
	return handler.RenderComponent(c, views.ToggleFragment(on))
}

func (s *hypermediaState) handleStatePanel(c echo.Context) error {
	return handler.RenderComponent(c, views.LazyPanelFragment())
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func parseCRUDID(c echo.Context) (int, error) {
	var id int
	_, err := fmt.Sscanf(c.Param("id"), "%d", &id)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("invalid id %q", c.Param("id"))
	}
	return id, nil
}

func (s *hypermediaState) findItem(id int) (crudItem, bool) {
	for _, it := range s.items {
		if it.ID == id {
			return it, true
		}
	}
	return crudItem{}, false
}

func (s *hypermediaState) findIndex(id int) int {
	for i, it := range s.items {
		if it.ID == id {
			return i
		}
	}
	return -1
}

func (ci crudItem) toView() views.CRUDViewItem {
	return views.CRUDViewItem{ID: ci.ID, Name: ci.Name, Status: ci.Status, Notes: ci.Notes}
}

func crudItemsToView(items []crudItem) []views.CRUDViewItem {
	out := make([]views.CRUDViewItem, len(items))
	for i, it := range items {
		out[i] = it.toView()
	}
	return out
}

func (ar *appRoutes) incrementPollCount() int64 {
	return atomic.AddInt64(&ar.pollCount, 1)
}
