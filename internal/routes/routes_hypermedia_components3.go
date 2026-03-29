// setup:feature:demo

package routes

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/params"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

// feedItem represents a single item in the infinite-scroll feed.
type feedItem struct {
	ID    int
	Title string
	Body  string
	Tag   string
}

// favoriteItem represents an item that can be favorited (optimistic UI demo).
type favoriteItem struct {
	ID       int
	Title    string
	Favorited bool
}

// undoItem represents a soft-deletable item.
type undoItem struct {
	ID      int
	Name    string
	Deleted bool
}

// components3State holds mutable demo state for /hypermedia/components3.
type components3State struct {
	mu         sync.RWMutex
	feedItems  []feedItem
	favorites  []favoriteItem
	undoItems  []undoItem
	undoNextID int
}

func newComponents3State() *components3State {
	feed := make([]feedItem, 50)
	tags := []string{"htmx", "go", "templ", "daisyui", "hypermedia"}
	for i := range feed {
		feed[i] = feedItem{
			ID:    i + 1,
			Title: fmt.Sprintf("Post #%d", i+1),
			Body:  fmt.Sprintf("This is feed item %d demonstrating infinite scroll. Content loads automatically as you scroll down.", i+1),
			Tag:   tags[i%len(tags)],
		}
	}

	return &components3State{
		feedItems: feed,
		favorites: []favoriteItem{
			{ID: 1, Title: "Server-Driven UI", Favorited: false},
			{ID: 2, Title: "Hypermedia Controls", Favorited: true},
			{ID: 3, Title: "OOB Swaps", Favorited: false},
			{ID: 4, Title: "SSE Streaming", Favorited: false},
			{ID: 5, Title: "Type-Safe Templates", Favorited: true},
		},
		undoItems: []undoItem{
			{ID: 1, Name: "Project Alpha"},
			{ID: 2, Name: "Project Beta"},
			{ID: 3, Name: "Project Gamma"},
			{ID: 4, Name: "Project Delta"},
			{ID: 5, Name: "Project Epsilon"},
		},
		undoNextID: 6,
	}
}

const components3Base = hypermediaBase + "/components3"

func (ar *appRoutes) initComponents3Routes() {
	s := newComponents3State()

	ar.e.GET(components3Base, s.handleComponents3Page)

	// Infinite scroll
	ar.e.GET(components3Base+"/feed", s.handleFeedPage)

	// Optimistic UI
	ar.e.POST(components3Base+"/favorite/:id", s.handleFavoriteToggle)

	// Calculate (output element demo)
	ar.e.GET(components3Base+"/calculate", handleCalculate)

	// Undo / soft delete
	ar.e.DELETE(components3Base+"/undo/:id", s.handleUndoDelete)
	ar.e.POST(components3Base+"/undo/:id/restore", s.handleUndoRestore)
	ar.e.GET(components3Base+"/undo/list", s.handleUndoList)
}

// ─── Page handler ───────────────────────────────────────────────────────────────

func (s *components3State) handleComponents3Page(c echo.Context) error {
	s.mu.Lock()
	fresh := newComponents3State()
	s.feedItems = fresh.feedItems
	s.favorites = fresh.favorites
	s.undoItems = fresh.undoItems
	s.undoNextID = fresh.undoNextID

	firstPage := feedItemsToView(s.feedItems[:10])
	favs := favoriteItemsToView(s.favorites)
	undos := undoItemsToView(s.undoItems)
	totalFeed := len(s.feedItems)
	s.mu.Unlock()

	return handler.RenderBaseLayout(c, views.Components3Page(views.Components3PageData{
		FeedItems:  firstPage,
		TotalFeed:  totalFeed,
		PageSize:   10,
		Favorites:  favs,
		UndoItems:  undos,
	}))
}

// ─── Infinite scroll handler ─────────────────────────────────────────────────────

func (s *components3State) handleFeedPage(c echo.Context) error {
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 1 {
		page = 1
	}

	const pageSize = 10
	s.mu.RLock()
	total := len(s.feedItems)
	start := (page - 1) * pageSize
	if start >= total {
		s.mu.RUnlock()
		return c.NoContent(200)
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	items := feedItemsToView(s.feedItems[start:end])
	hasMore := end < total
	s.mu.RUnlock()

	return handler.RenderComponent(c, views.FeedItemsFragment(items, page, hasMore))
}

// ─── Optimistic UI handler ───────────────────────────────────────────────────────

func (s *components3State) handleFavoriteToggle(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid ID", err)
	}

	// Simulate network latency so the optimistic update is visible
	time.Sleep(800 * time.Millisecond)

	s.mu.Lock()
	idx := -1
	for i, f := range s.favorites {
		if f.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.mu.Unlock()
		return handler.HandleHypermediaError(c, 404, "Item not found", fmt.Errorf("id=%d", id))
	}
	s.favorites[idx].Favorited = !s.favorites[idx].Favorited
	item := s.favorites[idx]
	s.mu.Unlock()

	return handler.RenderComponent(c, views.FavoriteItemFragment(favoriteItemToView(item)))
}

// ─── Undo / soft delete handlers ─────────────────────────────────────────────────

func (s *components3State) handleUndoDelete(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid ID", err)
	}

	s.mu.Lock()
	idx := -1
	for i, it := range s.undoItems {
		if it.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.mu.Unlock()
		return handler.HandleHypermediaError(c, 404, "Item not found", fmt.Errorf("id=%d", id))
	}
	s.undoItems[idx].Deleted = true
	item := s.undoItems[idx]
	s.mu.Unlock()

	return handler.RenderComponent(c, views.UndoToastFragment(undoItemToView(item)))
}

func (s *components3State) handleUndoRestore(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid ID", err)
	}

	s.mu.Lock()
	idx := -1
	for i, it := range s.undoItems {
		if it.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.mu.Unlock()
		return handler.HandleHypermediaError(c, 404, "Item not found", fmt.Errorf("id=%d", id))
	}
	s.undoItems[idx].Deleted = false
	item := s.undoItems[idx]
	s.mu.Unlock()

	return handler.RenderComponent(c, views.UndoItemRowFragment(undoItemToView(item)))
}

func (s *components3State) handleUndoList(c echo.Context) error {
	s.mu.RLock()
	items := undoItemsToView(s.undoItems)
	s.mu.RUnlock()

	return handler.RenderComponent(c, views.UndoListFragment(items))
}

// ─── Calculate handler (output element demo) ────────────────────────────────────

func handleCalculate(c echo.Context) error {
	a := rand.Intn(100) + 1
	b := rand.Intn(100) + 1
	return c.HTML(200, fmt.Sprintf("%d + %d = <strong>%d</strong>", a, b, a+b))
}

// ─── helpers ────────────────────────────────────────────────────────────────────

func feedItemsToView(items []feedItem) []views.FeedItemData {
	out := make([]views.FeedItemData, len(items))
	for i, it := range items {
		out[i] = views.FeedItemData{ID: it.ID, Title: it.Title, Body: it.Body, Tag: it.Tag}
	}
	return out
}

func favoriteItemToView(f favoriteItem) views.FavoriteItemData {
	return views.FavoriteItemData{ID: f.ID, Title: f.Title, Favorited: f.Favorited}
}

func favoriteItemsToView(items []favoriteItem) []views.FavoriteItemData {
	out := make([]views.FavoriteItemData, len(items))
	for i, f := range items {
		out[i] = favoriteItemToView(f)
	}
	return out
}

func undoItemToView(it undoItem) views.UndoItemData {
	return views.UndoItemData{ID: it.ID, Name: it.Name, Deleted: it.Deleted}
}

func undoItemsToView(items []undoItem) []views.UndoItemData {
	out := make([]views.UndoItemData, len(items))
	for i, it := range items {
		out[i] = undoItemToView(it)
	}
	return out
}
