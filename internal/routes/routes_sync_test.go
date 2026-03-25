package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleSync_EmptyBatch(t *testing.T) {
	e := echo.New()
	ar := &appRoutes{e: e}

	body := `{"operations":[],"schema_version":1}`
	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := ar.handleSync(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"results":[]`)
}

func TestHandleSync_SingleOperation(t *testing.T) {
	e := echo.New()
	ar := &appRoutes{e: e}

	body := `{
		"operations": [{
			"method": "PUT",
			"url": "/demo/repository/tasks/1",
			"body": "title=Updated&description=test",
			"content_type": "application/x-www-form-urlencoded",
			"version": 1,
			"queued_at": "2026-03-25T12:00:00Z"
		}],
		"schema_version": 1
	}`
	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := ar.handleSync(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"applied"`)
	assert.Contains(t, rec.Body.String(), `"index":0`)
}

func TestHandleSync_MultipleOperations(t *testing.T) {
	e := echo.New()
	ar := &appRoutes{e: e}

	body := `{
		"operations": [
			{"method": "POST", "url": "/tasks", "body": "title=New", "content_type": "application/x-www-form-urlencoded", "queued_at": "2026-03-25T12:00:00Z"},
			{"method": "PUT", "url": "/tasks/1", "body": "title=Edited", "content_type": "application/x-www-form-urlencoded", "version": 1, "queued_at": "2026-03-25T12:01:00Z"},
			{"method": "DELETE", "url": "/tasks/2", "body": "", "content_type": "", "version": 3, "queued_at": "2026-03-25T12:02:00Z"}
		],
		"schema_version": 1
	}`
	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := ar.handleSync(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	// Should have 3 results
	assert.Contains(t, rec.Body.String(), `"index":0`)
	assert.Contains(t, rec.Body.String(), `"index":1`)
	assert.Contains(t, rec.Body.String(), `"index":2`)
}

func TestHandleSync_InvalidJSON(t *testing.T) {
	e := echo.New()
	ar := &appRoutes{e: e}

	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := ar.handleSync(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
