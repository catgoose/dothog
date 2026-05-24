// setup:feature:avatar
package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"catgoose/dothog/internal/service/graph"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// openAvatarTestCache opens a fresh on-disk Graph cache DB under t.TempDir
// and binds a PhotoCache to it. The DB closes via t.Cleanup so each test
// gets its own isolated directory file.
func openAvatarTestCache(t *testing.T) *graph.PhotoCache {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "graph_cache.db")
	dir, err := graph.OpenDirectory(ctx, dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = dir.Close() })
	return dir.Photos()
}

func TestHandleAvatar_ReturnsCachedBlobAndContentType(t *testing.T) {
	cache := openAvatarTestCache(t)
	body := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46} // JPEG magic + a few bytes
	require.NoError(t, cache.Save(context.Background(), "azure-abc", "image/jpeg", body))

	e := echo.New()
	RegisterAvatarRoutes(e, cache)

	req := httptest.NewRequest(http.MethodGet, "/api/avatar/azure-abc", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/jpeg", rec.Header().Get("Content-Type"))
	assert.Equal(t, body, rec.Body.Bytes(), "response body must be the bytes stored in the SQLite cache")
}

func TestHandleAvatar_PreservesNonDefaultContentType(t *testing.T) {
	cache := openAvatarTestCache(t)
	body := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a} // PNG magic prefix
	require.NoError(t, cache.Save(context.Background(), "azure-png", "image/png", body))

	e := echo.New()
	RegisterAvatarRoutes(e, cache)

	req := httptest.NewRequest(http.MethodGet, "/api/avatar/azure-png", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/png", rec.Header().Get("Content-Type"),
		"Content-Type column must round-trip through PhotoCache.Get to the response header")
}

func TestHandleAvatar_MissingRowReturns404(t *testing.T) {
	cache := openAvatarTestCache(t)

	e := echo.New()
	RegisterAvatarRoutes(e, cache)

	req := httptest.NewRequest(http.MethodGet, "/api/avatar/azure-missing", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Body.Bytes(), "missing-photo response must be empty per c.NoContent contract")
}

func TestHandleAvatar_DBErrorReturns500(t *testing.T) {
	// Close the directory before serving so cache reads return a non-
	// ErrNoRows error — exercises the 500 branch in handleAvatar.
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "graph_cache.db")
	dir, err := graph.OpenDirectory(ctx, dbPath)
	require.NoError(t, err)
	cache := dir.Photos()
	require.NoError(t, dir.Close())

	e := echo.New()
	RegisterAvatarRoutes(e, cache)

	req := httptest.NewRequest(http.MethodGet, "/api/avatar/azure-anything", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code,
		"a non-ErrNoRows cache failure (here: closed DB) must surface as 500, not 404")
}
