// setup:feature:demo
package routes

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleResourceView_NotDeleted_ShowsResource(t *testing.T) {
	gs := newControlsGalleryState()
	c, rec := crudContext(http.MethodGet, "/patterns/controls/resource", "")

	require.NoError(t, gs.handleResourceView(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Hypermedia Widget")
}

func TestHandleResourceView_Deleted_RendersDeletedState(t *testing.T) {
	gs := newControlsGalleryState()
	gs.resourceDeleted = true
	c, rec := crudContext(http.MethodGet, "/patterns/controls/resource", "")

	require.NoError(t, gs.handleResourceView(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Resource deleted!")
	assert.Contains(t, rec.Body.String(), "/patterns/controls/resource/restore")

	// A safe view must not mutate state.
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	assert.True(t, gs.resourceDeleted)
}

func TestHandleResourceRestore_ClearsDeletedAndShowsResource(t *testing.T) {
	gs := newControlsGalleryState()
	gs.resourceDeleted = true
	c, rec := crudContext(http.MethodPost, "/patterns/controls/resource/restore", "")

	require.NoError(t, gs.handleResourceRestore(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Hypermedia Widget")

	gs.mu.RLock()
	defer gs.mu.RUnlock()
	assert.False(t, gs.resourceDeleted)
}
