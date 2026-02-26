// setup:feature:demo

package routes

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"catgoose/go-htmx-demo/internals/routes/handler"
	"catgoose/go-htmx-demo/internals/routes/hypermedia"
	"catgoose/go-htmx-demo/web/views"

	"github.com/labstack/echo/v4"
)

// controlsGalleryState holds mutable demo state for the /hypermedia/controls/* endpoints.
type controlsGalleryState struct {
	mu sync.RWMutex

	// Resource demo (section 3)
	resourceName    string
	resourceDesc    string
	resourceDeleted bool

	// Items demo (section 5)
	galleryItems []views.GalleryItem
	nextItemID   int

	// Row items (sections 11-12)
	rowItems []galleryRowItem

	// Error recovery demos (section 13)
	transientAttempts int
	staleVersion      int
	staleName         string
	cascadeDeleted    bool
}

// galleryRowItem is a table row for the controls gallery row demos.
type galleryRowItem struct {
	Name     string
	Category string
	Price    string
	Active   bool
	ID       int
}

func newControlsGalleryState() *controlsGalleryState {
	gs := &controlsGalleryState{}
	gs.resetLocked()
	return gs
}

// resetLocked reinitialises all mutable state. Caller must NOT hold the lock.
func (gs *controlsGalleryState) resetLocked() {
	gs.resourceName = "Hypermedia Widget"
	gs.resourceDesc = "A server-driven UI component"
	gs.resourceDeleted = false
	gs.galleryItems = nil
	gs.nextItemID = 1
	gs.rowItems = []galleryRowItem{
		{ID: 1, Name: "Example Item", Category: "Electronics", Price: "99.99", Active: true},
		{ID: 2, Name: "Second Item", Category: "Books", Price: "24.50", Active: true},
	}
	gs.transientAttempts = 0
	gs.staleVersion = 1
	gs.staleName = "Widget Pro"
	gs.cascadeDeleted = false
}

// initControlsGalleryRoutes registers the controls page and all its interactive endpoints.
func (ar *appRoutes) initControlsGalleryRoutes() {
	gs := newControlsGalleryState()
	base := hypermediaBase + "/controls"

	// Page (resets state on each load)
	ar.e.GET(base, gs.handleControlsPage)

	// Stateless demos
	ar.e.POST(base+"/echo", gs.handleEcho)
	ar.e.GET(base+"/retry", gs.handleRetry)
	ar.e.POST(base+"/action", gs.handleAction)
	ar.e.GET(base+"/dismiss-reset", gs.handleDismissReset)

	// Resource demo
	ar.e.GET(base+"/resource", gs.handleResourceView)
	ar.e.GET(base+"/resource/edit", gs.handleResourceEdit)
	ar.e.PUT(base+"/resource", gs.handleResourceSave)
	ar.e.DELETE(base+"/resource", gs.handleResourceDelete)

	// Form demo
	ar.e.POST(base+"/form", gs.handleFormSubmit)
	ar.e.GET(base+"/form/reset", gs.handleFormReset)

	// Empty state / items demo
	ar.e.POST(base+"/items", gs.handleItemCreate)
	ar.e.GET(base+"/items/reset", gs.handleItemsReset)

	// Filter demo
	ar.e.GET(base+"/filter", gs.handleFilter)

	// Row demo
	ar.e.GET(base+"/rows/:id", gs.handleRowView)
	ar.e.GET(base+"/rows/:id/edit", gs.handleRowEdit)
	ar.e.PUT(base+"/rows/:id", gs.handleRowSave)
	ar.e.DELETE(base+"/rows/:id", gs.handleRowDelete)

	// Error recovery demos
	ar.e.POST(base+"/errors/transient", gs.handleErrTransient)
	ar.e.POST(base+"/errors/validate", gs.handleErrValidate)
	ar.e.GET(base+"/errors/validate/fix", gs.handleErrValidateFix)
	ar.e.POST(base+"/errors/conflict", gs.handleErrConflict)
	ar.e.PUT(base+"/errors/conflict/update", gs.handleErrConflictUpdate)
	ar.e.POST(base+"/errors/conflict/copy", gs.handleErrConflictCopy)
	ar.e.POST(base+"/errors/stale", gs.handleErrStale)
	ar.e.GET(base+"/errors/stale/refresh", gs.handleErrStaleRefresh)
	ar.e.POST(base+"/errors/stale/force", gs.handleErrStaleForce)
	ar.e.DELETE(base+"/errors/cascade", gs.handleErrCascade)
	ar.e.GET(base+"/errors/cascade/reassign", gs.handleErrCascadeReassign)
	ar.e.POST(base+"/errors/cascade/reassign", gs.handleErrCascadeReassignSubmit)
	ar.e.DELETE(base+"/errors/cascade/force", gs.handleErrCascadeForce)
}

// ─── Controls page ──────────────────────────────────────────────────────────

func (gs *controlsGalleryState) handleControlsPage(c echo.Context) error {
	gs.mu.Lock()
	gs.resetLocked()
	gs.mu.Unlock()
	return handler.RenderBaseLayout(c, views.HypermediaControlsPage())
}

// ─── Section 1: Variant Echo ────────────────────────────────────────────────

func (gs *controlsGalleryState) handleEcho(c echo.Context) error {
	variant := c.QueryParam("v")
	if variant == "" {
		variant = "secondary"
	}
	ts := time.Now().Format("15:04:05")
	return handler.RenderComponent(c, views.VariantEchoFragment(variant, ts))
}

// ─── Section 2: Retry / Action ──────────────────────────────────────────────

func (gs *controlsGalleryState) handleRetry(c echo.Context) error {
	ts := time.Now().Format("15:04:05")
	return handler.RenderComponent(c, views.RetryResultFragment(ts))
}

func (gs *controlsGalleryState) handleAction(c echo.Context) error {
	ts := time.Now().Format("15:04:05")
	return handler.RenderComponent(c, views.ActionResultFragment(ts))
}

func (gs *controlsGalleryState) handleDismissReset(c echo.Context) error {
	return handler.RenderComponent(c, views.DismissDemoFragment())
}

// ─── Section 3: Resource ────────────────────────────────────────────────────

func (gs *controlsGalleryState) handleResourceView(c echo.Context) error {
	gs.mu.Lock()
	if gs.resourceDeleted {
		gs.resourceDeleted = false
	}
	name, desc := gs.resourceName, gs.resourceDesc
	gs.mu.Unlock()
	return handler.RenderComponent(c, views.ResourceViewFragment(name, desc))
}

func (gs *controlsGalleryState) handleResourceEdit(c echo.Context) error {
	gs.mu.RLock()
	name, desc := gs.resourceName, gs.resourceDesc
	gs.mu.RUnlock()
	return handler.RenderComponent(c, views.ResourceEditFragment(name, desc))
}

func (gs *controlsGalleryState) handleResourceSave(c echo.Context) error {
	name := c.FormValue("name")
	desc := c.FormValue("desc")
	gs.mu.Lock()
	gs.resourceName = name
	gs.resourceDesc = desc
	gs.resourceDeleted = false
	gs.mu.Unlock()
	return handler.RenderComponent(c, views.ResourceViewFragment(name, desc))
}

func (gs *controlsGalleryState) handleResourceDelete(c echo.Context) error {
	gs.mu.Lock()
	gs.resourceDeleted = true
	gs.mu.Unlock()
	return handler.RenderComponent(c, views.ResourceDeletedFragment())
}

// ─── Section 4: Form ────────────────────────────────────────────────────────

func (gs *controlsGalleryState) handleFormSubmit(c echo.Context) error {
	label := c.FormValue("label")
	value := c.FormValue("value")
	return handler.RenderComponent(c, views.FormSubmitResultFragment(label, value))
}

func (gs *controlsGalleryState) handleFormReset(c echo.Context) error {
	return handler.RenderComponent(c, views.FormDemoFragment())
}

// ─── Section 5: Items / Empty State ─────────────────────────────────────────

func (gs *controlsGalleryState) handleItemCreate(c echo.Context) error {
	gs.mu.Lock()
	item := views.GalleryItem{ID: gs.nextItemID, Name: fmt.Sprintf("Item #%d", gs.nextItemID)}
	gs.galleryItems = append(gs.galleryItems, item)
	gs.nextItemID++
	items := make([]views.GalleryItem, len(gs.galleryItems))
	copy(items, gs.galleryItems)
	gs.mu.Unlock()
	return handler.RenderComponent(c, views.ItemsListFragment(items))
}

func (gs *controlsGalleryState) handleItemsReset(c echo.Context) error {
	gs.mu.Lock()
	gs.galleryItems = nil
	gs.nextItemID = 1
	gs.mu.Unlock()
	return handler.RenderComponent(c, views.EmptyStateFragment())
}

// ─── Section 9: Filter ──────────────────────────────────────────────────────

var filterDataset = []views.FilterDemoItem{
	{Name: "Wireless Headphones", Category: "Electronics", Price: 250, Active: true},
	{Name: "Running Shoes", Category: "Sports", Price: 120, Active: true},
	{Name: "Cookbook: Italian", Category: "Books", Price: 35, Active: false},
	{Name: "Cotton T-Shirt", Category: "Clothing", Price: 25, Active: true},
	{Name: "Yoga Mat", Category: "Sports", Price: 45, Active: true},
	{Name: "Bluetooth Speaker", Category: "Electronics", Price: 180, Active: false},
	{Name: "Novel: Sci-Fi", Category: "Books", Price: 15, Active: true},
	{Name: "Winter Jacket", Category: "Clothing", Price: 200, Active: true},
}

func (gs *controlsGalleryState) handleFilter(c echo.Context) error {
	q := strings.ToLower(c.QueryParam("q"))
	cat := c.QueryParam("cat")
	maxPrice := c.QueryParam("price")
	active := c.QueryParam("active")

	var results []views.FilterDemoItem
	for _, item := range filterDataset {
		if q != "" && !strings.Contains(strings.ToLower(item.Name), q) {
			continue
		}
		if cat != "" && item.Category != cat {
			continue
		}
		if maxPrice != "" {
			var mp int
			if _, err := fmt.Sscanf(maxPrice, "%d", &mp); err == nil && item.Price > mp {
				continue
			}
		}
		if active == "true" && !item.Active {
			continue
		}
		results = append(results, item)
	}
	return handler.RenderComponent(c, views.FilterResultsFragment(results, len(filterDataset)))
}

// ─── Sections 11-12: Row CRUD ───────────────────────────────────────────────

func (gs *controlsGalleryState) handleRowView(c echo.Context) error {
	id, err := parseGalleryRowID(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid ID")
	}
	gs.mu.RLock()
	row, found := gs.findRow(id)
	gs.mu.RUnlock()
	if !found {
		return c.String(http.StatusNotFound, "Row not found")
	}
	return handler.RenderComponent(c, views.RowViewFragment(views.GalleryRowItem{
		ID: row.ID, Name: row.Name, Category: row.Category, Price: row.Price, Active: row.Active,
	}))
}

func (gs *controlsGalleryState) handleRowEdit(c echo.Context) error {
	id, err := parseGalleryRowID(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid ID")
	}
	gs.mu.RLock()
	row, found := gs.findRow(id)
	gs.mu.RUnlock()
	if !found {
		return c.String(http.StatusNotFound, "Row not found")
	}
	return handler.RenderComponent(c, views.RowEditFragment(views.GalleryRowItem{
		ID: row.ID, Name: row.Name, Category: row.Category, Price: row.Price, Active: row.Active,
	}))
}

func (gs *controlsGalleryState) handleRowSave(c echo.Context) error {
	id, err := parseGalleryRowID(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid ID")
	}
	gs.mu.Lock()
	idx := gs.findRowIndex(id)
	if idx < 0 {
		gs.mu.Unlock()
		return c.String(http.StatusNotFound, "Row not found")
	}
	gs.rowItems[idx].Name = c.FormValue("name")
	gs.rowItems[idx].Category = c.FormValue("category")
	gs.rowItems[idx].Price = c.FormValue("price")
	gs.rowItems[idx].Active = c.FormValue("active") == "true"
	row := gs.rowItems[idx]
	gs.mu.Unlock()
	return handler.RenderComponent(c, views.RowViewFragment(views.GalleryRowItem{
		ID: row.ID, Name: row.Name, Category: row.Category, Price: row.Price, Active: row.Active,
	}))
}

func (gs *controlsGalleryState) handleRowDelete(c echo.Context) error {
	id, err := parseGalleryRowID(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid ID")
	}
	gs.mu.Lock()
	idx := gs.findRowIndex(id)
	if idx >= 0 {
		gs.rowItems = append(gs.rowItems[:idx], gs.rowItems[idx+1:]...)
	}
	gs.mu.Unlock()
	return c.NoContent(http.StatusOK)
}

// ─── helpers ────────────────────────────────────────────────────────────────

func parseGalleryRowID(c echo.Context) (int, error) {
	var id int
	_, err := fmt.Sscanf(c.Param("id"), "%d", &id)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("invalid id %q", c.Param("id"))
	}
	return id, nil
}

func (gs *controlsGalleryState) findRow(id int) (galleryRowItem, bool) {
	for _, r := range gs.rowItems {
		if r.ID == id {
			return r, true
		}
	}
	return galleryRowItem{}, false
}

func (gs *controlsGalleryState) findRowIndex(id int) int {
	for i, r := range gs.rowItems {
		if r.ID == id {
			return i
		}
	}
	return -1
}

// ─── Section 13: HATEOAS Error Recovery ─────────────────────────────────────

// Section 13 result-div IDs, reused across triggers, error controls, and success panels.
const (
	resultIDTransient = "transient-result"
	resultIDValidate  = "validate-result"
	resultIDConflict  = "conflict-result"
	resultIDStale     = "stale-result"
	resultIDCascade   = "cascade-result"
)

// Scenario 1: Transient error — odd attempts fail, even succeed, then reset.
func (gs *controlsGalleryState) handleErrTransient(c echo.Context) error {
	gs.mu.Lock()
	gs.transientAttempts++
	attempt := gs.transientAttempts
	gs.mu.Unlock()

	if attempt%2 == 1 {
		ec := hypermedia.ErrorContext{
			StatusCode: 500,
			Message:    fmt.Sprintf("Save failed — transient network error (attempt %d)", attempt),
			Route:      c.Request().URL.Path,
			Closable:   true,
			Controls: []hypermedia.Control{
				hypermedia.RetryButton("Retry Save", hypermedia.HxMethodPost,
					"/hypermedia/controls/errors/transient", "#"+resultIDTransient).
					WithErrorTarget("#" + resultIDTransient),
			},
		}
		c.Response().WriteHeader(http.StatusInternalServerError)
		return handler.RenderComponent(c, views.GalleryErrorPanel("transient-error", ec))
	}
	return handler.RenderComponent(c, views.ErrorRecoverySuccess(
		"Saved!", fmt.Sprintf("Record saved on attempt %d.", attempt), resultIDTransient))
}

// Scenario 2: Validation — checks name length and price sign.
func (gs *controlsGalleryState) handleErrValidate(c echo.Context) error {
	name := c.FormValue("name")
	price := c.FormValue("price")

	var errs []string
	if len(name) < 3 {
		errs = append(errs, "name must be at least 3 characters")
	}
	if price == "" || strings.HasPrefix(price, "-") {
		errs = append(errs, "price must be a positive number")
	}

	if len(errs) > 0 {
		fixURL := fmt.Sprintf("/hypermedia/controls/errors/validate/fix?name=%s&price=%s",
			url.QueryEscape(name), url.QueryEscape(price))
		ec := hypermedia.ErrorContext{
			StatusCode: 422,
			Message:    "Validation failed",
			Err:        errors.New(strings.Join(errs, "; ")),
			Route:      c.Request().URL.Path,
			Closable:   true,
			Controls: []hypermedia.Control{
				{
					Kind:        hypermedia.ControlKindHTMX,
					Label:       "Fix & Resubmit",
					Variant:     hypermedia.VariantPrimary,
					ErrorTarget: "#" + resultIDValidate,
					HxRequest:   hypermedia.HxGet(fixURL, "#"+resultIDValidate),
				},
			},
		}
		c.Response().WriteHeader(http.StatusUnprocessableEntity)
		return handler.RenderComponent(c, views.GalleryErrorPanel("validate-error", ec))
	}

	return handler.RenderComponent(c, views.ErrorRecoverySuccess(
		"Saved!", fmt.Sprintf("'%s' at $%s saved successfully.", name, price), resultIDValidate))
}

func (gs *controlsGalleryState) handleErrValidateFix(c echo.Context) error {
	name := c.QueryParam("name")
	price := c.QueryParam("price")
	return handler.RenderComponent(c, views.ValidationFixForm(name, price))
}

// Scenario 3: Conflict — record already exists, offer update or copy.
func (gs *controlsGalleryState) handleErrConflict(c echo.Context) error {
	ec := hypermedia.ErrorContext{
		StatusCode: 409,
		Message:    "'Widget Alpha' already exists (ID: 42)",
		Err:        errors.New("unique constraint violated on column 'name'"),
		Route:      c.Request().URL.Path,
		Closable:   true,
		Controls: []hypermedia.Control{
			{
				Kind:        hypermedia.ControlKindHTMX,
				Label:       "Update Existing",
				Variant:     hypermedia.VariantPrimary,
				ErrorTarget: "#" + resultIDConflict,
				HxRequest: hypermedia.HxRequestConfig{
					Method: hypermedia.HxMethodPut,
					URL:    "/hypermedia/controls/errors/conflict/update",
					Target: "#" + resultIDConflict,
				},
			},
			{
				Kind:        hypermedia.ControlKindHTMX,
				Label:       "Create as Copy",
				Variant:     hypermedia.VariantSecondary,
				ErrorTarget: "#" + resultIDConflict,
				HxRequest:   hypermedia.HxPost("/hypermedia/controls/errors/conflict/copy", "#"+resultIDConflict),
			},
		},
	}
	c.Response().WriteHeader(http.StatusConflict)
	return handler.RenderComponent(c, views.GalleryErrorPanel("conflict-error", ec))
}

func (gs *controlsGalleryState) handleErrConflictUpdate(c echo.Context) error {
	return handler.RenderComponent(c, views.ErrorRecoverySuccess(
		"Updated!", "Existing 'Widget Alpha' (ID: 42) updated with your data.", resultIDConflict))
}

func (gs *controlsGalleryState) handleErrConflictCopy(c echo.Context) error {
	return handler.RenderComponent(c, views.ErrorRecoverySuccess(
		"Created!", "'Widget Alpha (copy)' created as a new record.", resultIDConflict))
}

// Scenario 4: Stale data — version mismatch, offer refresh or force save.
func (gs *controlsGalleryState) handleErrStale(c echo.Context) error {
	name := c.FormValue("name")
	if name == "" {
		name = "Widget Pro (stale edit)"
	}
	var sv int
	fmt.Sscanf(c.FormValue("version"), "%d", &sv)

	gs.mu.RLock()
	currentVersion := gs.staleVersion
	gs.mu.RUnlock()

	if sv < currentVersion {
		forceURL := fmt.Sprintf("/hypermedia/controls/errors/stale/force?name=%s",
			url.QueryEscape(name))
		ec := hypermedia.ErrorContext{
			StatusCode: 412,
			Message:    fmt.Sprintf("Record modified by another user (your v%d, server v%d)", sv, currentVersion),
			Err:        errors.New("optimistic lock failed — row version mismatch"),
			Route:      c.Request().URL.Path,
			Closable:   true,
			Controls: []hypermedia.Control{
				{
					Kind:        hypermedia.ControlKindHTMX,
					Label:       "Load Fresh Data",
					Variant:     hypermedia.VariantPrimary,
					ErrorTarget: "#" + resultIDStale,
					HxRequest:   hypermedia.HxGet("/hypermedia/controls/errors/stale/refresh", "#"+resultIDStale),
				},
				{
					Kind:        hypermedia.ControlKindHTMX,
					Label:       "Force Save",
					Variant:     hypermedia.VariantDanger,
					Confirm:     "Override the other user's changes?",
					ErrorTarget: "#" + resultIDStale,
					HxRequest: hypermedia.HxRequestConfig{
						Method: hypermedia.HxMethodPost,
						URL:    forceURL,
						Target: "#" + resultIDStale,
					},
				},
			},
		}
		c.Response().WriteHeader(http.StatusPreconditionFailed)
		return handler.RenderComponent(c, views.GalleryErrorPanel("stale-error", ec))
	}

	// Version matches — save succeeds
	gs.mu.Lock()
	gs.staleName = name
	gs.staleVersion++
	v := gs.staleVersion
	gs.mu.Unlock()
	return handler.RenderComponent(c, views.ErrorRecoverySuccess(
		"Saved!", fmt.Sprintf("'%s' saved (now v%d).", name, v), resultIDStale))
}

func (gs *controlsGalleryState) handleErrStaleRefresh(c echo.Context) error {
	gs.mu.RLock()
	name := gs.staleName
	version := gs.staleVersion
	gs.mu.RUnlock()
	return handler.RenderComponent(c, views.StaleRefreshForm(name, version))
}

func (gs *controlsGalleryState) handleErrStaleForce(c echo.Context) error {
	name := c.QueryParam("name")
	gs.mu.Lock()
	if name != "" {
		gs.staleName = name
	}
	gs.staleVersion++
	v := gs.staleVersion
	gs.mu.Unlock()
	return handler.RenderComponent(c, views.ErrorRecoverySuccess(
		"Force Saved!", fmt.Sprintf("'%s' saved as v%d — other user's changes overwritten.", name, v), resultIDStale))
}

// Scenario 5: Cascade — cannot delete because of dependencies.
var cascadeDependents = []string{"Wireless Headphones", "Bluetooth Speaker", "USB Charger"}

func (gs *controlsGalleryState) handleErrCascade(c echo.Context) error {
	gs.mu.RLock()
	deleted := gs.cascadeDeleted
	gs.mu.RUnlock()

	if deleted {
		return handler.RenderComponent(c, views.ErrorRecoverySuccess(
			"Already Deleted", "Category was previously removed. Reload page to reset.", resultIDCascade))
	}

	ec := hypermedia.ErrorContext{
		StatusCode: 409,
		Message:    fmt.Sprintf("Cannot delete 'Electronics' — %d items depend on it", len(cascadeDependents)),
		Err:        errors.New("foreign key constraint: items.category_id → categories.id"),
		Route:      c.Request().URL.Path,
		Closable:   true,
		Controls: []hypermedia.Control{
			{
				Kind:        hypermedia.ControlKindHTMX,
				Label:       "Reassign Items",
				Variant:     hypermedia.VariantPrimary,
				ErrorTarget: "#" + resultIDCascade,
				HxRequest:   hypermedia.HxGet("/hypermedia/controls/errors/cascade/reassign", "#"+resultIDCascade),
			},
			hypermedia.ConfirmAction("Force Delete All", hypermedia.HxMethodDelete,
				"/hypermedia/controls/errors/cascade/force", "#"+resultIDCascade,
				fmt.Sprintf("Delete 'Electronics' AND all %d items?", len(cascadeDependents))).
				WithErrorTarget("#" + resultIDCascade),
		},
	}
	c.Response().WriteHeader(http.StatusConflict)
	return handler.RenderComponent(c, views.GalleryErrorPanel("cascade-error", ec))
}

func (gs *controlsGalleryState) handleErrCascadeReassign(c echo.Context) error {
	return handler.RenderComponent(c, views.CascadeReassignForm(cascadeDependents))
}

func (gs *controlsGalleryState) handleErrCascadeReassignSubmit(c echo.Context) error {
	newCat := c.FormValue("new-category")
	gs.mu.Lock()
	gs.cascadeDeleted = true
	gs.mu.Unlock()
	return handler.RenderComponent(c, views.ErrorRecoverySuccess(
		"Done!",
		fmt.Sprintf("%d items moved to '%s', then 'Electronics' deleted.", len(cascadeDependents), newCat),
		resultIDCascade))
}

func (gs *controlsGalleryState) handleErrCascadeForce(c echo.Context) error {
	gs.mu.Lock()
	gs.cascadeDeleted = true
	gs.mu.Unlock()
	return handler.RenderComponent(c, views.ErrorRecoverySuccess(
		"Force Deleted!",
		fmt.Sprintf("'Electronics' and all %d dependent items permanently removed.", len(cascadeDependents)),
		resultIDCascade))
}
