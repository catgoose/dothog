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

func TestResponse_Reswap(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NoError(t, New().Reswap("outerHTML").Write(rec))
	assert.Equal(t, "outerHTML", rec.Header().Get("HX-Reswap"))
}

func TestResponse_Retarget(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NoError(t, New().Retarget("#form-errors").Write(rec))
	assert.Equal(t, "#form-errors", rec.Header().Get("HX-Retarget"))
}

func TestResponse_Reselect(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NoError(t, New().Reselect(".error-summary").Write(rec))
	assert.Equal(t, ".error-summary", rec.Header().Get("HX-Reselect"))
}

func TestResponse_TriggerAfterSwap(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NoError(t, New().TriggerAfterSwap("focusFirstError").Write(rec))
	assert.Equal(t, "focusFirstError", rec.Header().Get("HX-Trigger-After-Swap"))
}

func TestResponse_TriggerAfterSwapDetail(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NoError(t, New().TriggerAfterSwapDetail("rowAdded", map[string]int{"id": 42}).Write(rec))
	assert.JSONEq(t, `{"rowAdded":{"id":42}}`, rec.Header().Get("HX-Trigger-After-Swap"))
}

func TestResponse_TriggerAfterSettle(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NoError(t, New().TriggerAfterSettle("toastReady").Write(rec))
	assert.Equal(t, "toastReady", rec.Header().Get("HX-Trigger-After-Settle"))
}

func TestResponse_TriggerAfterSettleDetail(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NoError(t, New().TriggerAfterSettleDetail("scrollTo", "#row-42").Write(rec))
	assert.JSONEq(t, `{"scrollTo":"#row-42"}`, rec.Header().Get("HX-Trigger-After-Settle"))
}

func TestResponse_RetargetReswapReselectCompose(t *testing.T) {
	// Validates the realistic "validation error rerouting" recipe: the
	// triggering element wanted to swap into its parent row, but the server
	// retargets the response into the form-level error pane, narrows the
	// returned fragment via Reselect, and replaces the destination's
	// innerHTML.
	rec := httptest.NewRecorder()
	require.NoError(t, New().
		Retarget("#form-errors").
		Reselect(".error-summary").
		Reswap("innerHTML").
		Write(rec))
	assert.Equal(t, "#form-errors", rec.Header().Get("HX-Retarget"))
	assert.Equal(t, ".error-summary", rec.Header().Get("HX-Reselect"))
	assert.Equal(t, "innerHTML", rec.Header().Get("HX-Reswap"))
}
