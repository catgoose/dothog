package htmxutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsHTMX(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	assert.False(t, IsHTMX(r))
	r.Header.Set("HX-Request", "true")
	assert.True(t, IsHTMX(r))
}

func TestIsBoosted(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	assert.False(t, IsBoosted(r))
	r.Header.Set("HX-Boosted", "true")
	assert.True(t, IsBoosted(r))
}

func TestCurrentURL_ParsesHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Current-URL", "https://example.com/apps/inventory?search=widget&page=3")
	got, ok := CurrentURL(r)
	require.True(t, ok)
	assert.Equal(t, "/apps/inventory", got.Path)
	assert.Equal(t, "search=widget&page=3", got.RawQuery)
}

func TestCurrentURL_MissingHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	_, ok := CurrentURL(r)
	assert.False(t, ok, "missing HX-Current-URL must not yield a parsed URL")
}

func TestCurrentURL_Malformed(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Current-URL", "://not-a-url")
	_, ok := CurrentURL(r)
	assert.False(t, ok, "malformed HX-Current-URL must not panic and must report not-ok")
}

func TestCurrentRawQuery(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Current-URL", "https://example.com/apps/inventory?search=foo&page=2")
	assert.Equal(t, "search=foo&page=2", CurrentRawQuery(r))

	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	assert.Empty(t, CurrentRawQuery(r2))
}

func TestResponse_ReplaceURL(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NoError(t, New().ReplaceURL("/apps/inventory?search=foo").Write(rec))
	assert.Equal(t, "/apps/inventory?search=foo", rec.Header().Get("HX-Replace-Url"))
}

func TestResponse_Refresh(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NoError(t, New().Refresh().Write(rec))
	assert.Equal(t, "true", rec.Header().Get("HX-Refresh"))
}

func TestResponse_TriggerPlain(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NoError(t, New().Trigger("showReportModal").Write(rec))
	assert.Equal(t, "showReportModal", rec.Header().Get("HX-Trigger"))
}

func TestResponse_TriggerDetail(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NoError(t, New().TriggerDetail("showAlert", "Failed to submit report.").Write(rec))
	got := rec.Header().Get("HX-Trigger")
	assert.JSONEq(t, `{"showAlert":"Failed to submit report."}`, got)
}

func TestResponse_ReswapNoneAndTriggerCompose(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NoError(t, New().
		TriggerDetail("showAlert", "Issue reported.").
		ReswapNone().
		Write(rec))
	assert.JSONEq(t, `{"showAlert":"Issue reported."}`, rec.Header().Get("HX-Trigger"))
	assert.Equal(t, "none", rec.Header().Get("HX-Reswap"))
}
