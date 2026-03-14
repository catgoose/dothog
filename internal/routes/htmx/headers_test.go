package htmx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// newContext creates a fresh echo.Context backed by an httptest request/recorder.
func newContext() (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// ---------- IsHTMX ----------

func TestIsHTMX_True(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.True(t, IsHTMX(c))
}

func TestIsHTMX_Missing(t *testing.T) {
	c, _ := newContext()
	require.False(t, IsHTMX(c))
}

func TestIsHTMX_OtherValue(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("HX-Request", "yes")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.False(t, IsHTMX(c))
}

// ---------- Trigger ----------

func TestTrigger(t *testing.T) {
	c, rec := newContext()
	Trigger(c, `{"myEvent":"payload"}`)

	require.Equal(t, `{"myEvent":"payload"}`, rec.Header().Get(HeaderTrigger))
}

// ---------- TriggerEvent ----------

func TestTriggerEvent_WithPayload(t *testing.T) {
	c, rec := newContext()
	payload := map[string]string{"key": "value"}
	err := TriggerEvent(c, "testEvent", payload)
	require.NoError(t, err)

	raw := rec.Header().Get(HeaderTrigger)
	require.NotEmpty(t, raw)

	var decoded map[string]any
	err = json.Unmarshal([]byte(raw), &decoded)
	require.NoError(t, err)

	inner, ok := decoded["testEvent"]
	require.True(t, ok, "expected key 'testEvent' in JSON")
	innerMap, ok := inner.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "value", innerMap["key"])
}

func TestTriggerEvent_NilPayload(t *testing.T) {
	c, rec := newContext()
	err := TriggerEvent(c, "refresh", nil)
	require.NoError(t, err)

	raw := rec.Header().Get(HeaderTrigger)
	require.Equal(t, `{"refresh":null}`, raw)
}

func TestTriggerEvent_MarshalError(t *testing.T) {
	c, _ := newContext()
	// Channels cannot be marshaled to JSON.
	err := TriggerEvent(c, "bad", make(chan int))
	require.Error(t, err)
}

// ---------- TriggerAfterSettle ----------

func TestTriggerAfterSettle(t *testing.T) {
	c, rec := newContext()
	TriggerAfterSettle(c, `{"settled":true}`)

	require.Equal(t, `{"settled":true}`, rec.Header().Get(HeaderTriggerAfterSettle))
}

// ---------- Redirect ----------

func TestRedirect(t *testing.T) {
	c, rec := newContext()
	Redirect(c, "/dashboard")

	require.Equal(t, "/dashboard", rec.Header().Get(HeaderRedirect))
}

// ---------- PushURL ----------

func TestPushURL(t *testing.T) {
	c, rec := newContext()
	PushURL(c, "/users/42")

	require.Equal(t, "/users/42", rec.Header().Get(HeaderPushURL))
}

// ---------- ReplaceURL ----------

func TestReplaceURL(t *testing.T) {
	c, rec := newContext()
	ReplaceURL(c, "/settings?tab=profile")

	require.Equal(t, "/settings?tab=profile", rec.Header().Get(HeaderReplaceURL))
}

// ---------- Reswap ----------

func TestReswap_Strategies(t *testing.T) {
	strategies := []SwapStrategy{
		SwapInnerHTML,
		SwapOuterHTML,
		SwapBeforeBegin,
		SwapAfterBegin,
		SwapBeforeEnd,
		SwapAfterEnd,
		SwapDelete,
		SwapNone,
	}

	for _, s := range strategies {
		t.Run(string(s), func(t *testing.T) {
			c, rec := newContext()
			Reswap(c, s)
			require.Equal(t, string(s), rec.Header().Get(HeaderReswap))
		})
	}
}

// ---------- ReswapNone ----------

func TestReswapNone(t *testing.T) {
	c, rec := newContext()
	ReswapNone(c)

	require.Equal(t, "none", rec.Header().Get(HeaderReswap))
}

// ---------- Retarget ----------

func TestRetarget(t *testing.T) {
	c, rec := newContext()
	Retarget(c, "#error-panel")

	require.Equal(t, "#error-panel", rec.Header().Get(HeaderRetarget))
}

// ---------- Refresh ----------

func TestRefresh(t *testing.T) {
	c, rec := newContext()
	Refresh(c)

	require.Equal(t, "true", rec.Header().Get(HeaderRefresh))
}
