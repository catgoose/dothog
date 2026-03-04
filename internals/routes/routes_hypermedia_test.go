package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func crudContext(method, path string, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestHandleCRUDCreate_WithName(t *testing.T) {
	s := newHypermediaState()
	c, rec := crudContext(http.MethodPost, "/hypermedia/crud/items", "name=TestItem&notes=hello")

	err := s.handleCRUDCreate(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "TestItem")
}

func TestHandleCRUDCreate_WithoutName(t *testing.T) {
	s := newHypermediaState()
	c, rec := crudContext(http.MethodPost, "/hypermedia/crud/items", "name=&notes=hello")

	err := s.handleCRUDCreate(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	// Should default to "New Item".
	s.mu.RLock()
	last := s.items[len(s.items)-1]
	s.mu.RUnlock()
	assert.Equal(t, "New Item", last.Name)
}

func TestHandleCRUDUpdate_EmptyNameReturns400(t *testing.T) {
	s := newHypermediaState()
	c, _ := crudContext(http.MethodPut, "/hypermedia/crud/items/1", "name=&notes=updated")
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := s.handleCRUDUpdate(c)
	// The handler returns an HTTPError via HandleHypermediaError.
	require.Error(t, err)
}

func TestHandleCRUDUpdate_ValidName(t *testing.T) {
	s := newHypermediaState()
	c, rec := crudContext(http.MethodPut, "/hypermedia/crud/items/1", "name=Updated&notes=new")
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := s.handleCRUDUpdate(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	s.mu.RLock()
	item, found := s.findItem(1)
	s.mu.RUnlock()
	require.True(t, found)
	assert.Equal(t, "Updated", item.Name)
}

func TestHandleCRUDDelete_ExistingID(t *testing.T) {
	s := newHypermediaState()
	c, rec := crudContext(http.MethodDelete, "/hypermedia/crud/items/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := s.handleCRUDDelete(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	s.mu.RLock()
	_, found := s.findItem(1)
	s.mu.RUnlock()
	assert.False(t, found)
}

func TestHandleCRUDDelete_NonExistingID(t *testing.T) {
	s := newHypermediaState()
	c, rec := crudContext(http.MethodDelete, "/hypermedia/crud/items/999", "")
	c.SetParamNames("id")
	c.SetParamValues("999")

	err := s.handleCRUDDelete(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleCRUDPatchToggle(t *testing.T) {
	s := newHypermediaState()
	// Item 1 starts as "active".
	c, rec := crudContext(http.MethodPatch, "/hypermedia/crud/items/1/toggle", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := s.handleCRUDPatchToggle(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	s.mu.RLock()
	item, found := s.findItem(1)
	s.mu.RUnlock()
	require.True(t, found)
	assert.Equal(t, "inactive", item.Status)
}

func TestConcurrentHypermediaAccess(t *testing.T) {
	s := newHypermediaState()
	var wg sync.WaitGroup

	for range 20 {
		wg.Add(3)
		go func() {
			defer wg.Done()
			c, _ := crudContext(http.MethodPost, "/hypermedia/crud/items", "name=Concurrent&notes=test")
			_ = s.handleCRUDCreate(c)
		}()
		go func() {
			defer wg.Done()
			c, _ := crudContext(http.MethodPatch, "/hypermedia/crud/items/1/toggle", "")
			c.SetParamNames("id")
			c.SetParamValues("1")
			_ = s.handleCRUDPatchToggle(c)
		}()
		go func() {
			defer wg.Done()
			c, _ := crudContext(http.MethodGet, "/hypermedia/crud/items", "")
			_ = s.handleCRUDItems(c)
		}()
	}
	wg.Wait()
}

func TestCrudItemToView(t *testing.T) {
	item := crudItem{ID: 42, Name: "Test", Status: "active", Notes: "notes"}
	v := item.toView()
	assert.Equal(t, 42, v.ID)
	assert.Equal(t, "Test", v.Name)
	assert.Equal(t, "active", v.Status)
	assert.Equal(t, "notes", v.Notes)
}
