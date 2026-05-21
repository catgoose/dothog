package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"catgoose/dothog/internal/routes/middleware"
	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// recordingStore is a minimal promolog.Storer that records every Promote call.
// Only Promote is exercised by the HTTP error handler; the rest are no-op stubs.
type recordingStore struct {
	onPromote func(promolog.TraceSummary)
	traces    []promolog.Trace
	mu        sync.Mutex
}

func (s *recordingStore) InitSchema() error                           { return nil }
func (s *recordingStore) SetOnPromote(fn func(promolog.TraceSummary)) { s.onPromote = fn }
func (s *recordingStore) PromoteAt(_ context.Context, t promolog.Trace, _ time.Time) error {
	return s.Promote(context.Background(), t)
}
func (s *recordingStore) Get(_ context.Context, _ string) (*promolog.Trace, error) {
	return nil, nil
}
func (s *recordingStore) ListTraces(_ context.Context, _ promolog.TraceFilter) ([]promolog.TraceSummary, int, error) {
	return nil, 0, nil
}
func (s *recordingStore) AvailableFilters(_ context.Context, _ promolog.TraceFilter) (promolog.FilterOptions, error) {
	return promolog.FilterOptions{}, nil
}
func (s *recordingStore) DeleteTrace(_ context.Context, _ string) error                    { return nil }
func (s *recordingStore) StartCleanup(_ context.Context, _ time.Duration, _ time.Duration) {}
func (s *recordingStore) CreateRule(_ context.Context, r promolog.FilterRule) (promolog.FilterRule, error) {
	return r, nil
}
func (s *recordingStore) ListRules(_ context.Context) ([]promolog.FilterRule, error) { return nil, nil }
func (s *recordingStore) UpdateRule(_ context.Context, _ promolog.FilterRule) error  { return nil }
func (s *recordingStore) DeleteRule(_ context.Context, _ int) error                  { return nil }
func (s *recordingStore) CreateRetentionRule(_ context.Context, r promolog.RetentionRule) (promolog.RetentionRule, error) {
	return r, nil
}
func (s *recordingStore) ListRetentionRules(_ context.Context) ([]promolog.RetentionRule, error) {
	return nil, nil
}
func (s *recordingStore) UpdateRetentionRule(_ context.Context, _ promolog.RetentionRule) error {
	return nil
}
func (s *recordingStore) DeleteRetentionRule(_ context.Context, _ int) error { return nil }
func (s *recordingStore) Aggregate(_ context.Context, _ promolog.AggregateFilter) ([]promolog.AggregateResult, error) {
	return nil, nil
}

func (s *recordingStore) Promote(_ context.Context, t promolog.Trace) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.traces = append(s.traces, t)
	return nil
}

func (s *recordingStore) snapshot() []promolog.Trace {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]promolog.Trace, len(s.traces))
	copy(out, s.traces)
	return out
}

var _ promolog.Storer = (*recordingStore)(nil)

// newEchoWithPromotion returns an Echo configured like the production stack
// for the 404 path: correlation middleware (so requests carry an ID), the
// production HTTPErrorHandler (which is what calls Promote), and the
// production RouteNotFound handler.
func newEchoWithPromotion(store promolog.Storer) *echo.Echo {
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.HTTPErrorHandler = middleware.NewHTTPErrorHandler(store)
	e.RouteNotFound("/*", HandleNotFound)
	return e
}

// TestHandleNotFound_NonHTMX_PromotesTrace pins down the regression where
// a direct unmatched-route visit rendered the 404 page but never made it
// into the promolog error-trace store.
func TestHandleNotFound_NonHTMX_PromotesTrace(t *testing.T) {
	store := &recordingStore{}
	e := newEchoWithPromotion(store)

	req := httptest.NewRequest(http.MethodGet, "/fakeroute/asdf", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)

	traces := store.snapshot()
	require.Len(t, traces, 1, "unmatched non-HTMX route should promote exactly one trace")
	require.Equal(t, http.StatusNotFound, traces[0].StatusCode)
	require.Equal(t, "/fakeroute/asdf", traces[0].Route)
	require.Equal(t, http.MethodGet, traces[0].Method)
	require.NotEmpty(t, traces[0].RequestID, "promoted trace should carry the correlation request ID")
}

// TestHandleNotFound_HTMX_PromotesTrace guards the pre-existing HTMX path:
// it should still reach the promote step.
func TestHandleNotFound_HTMX_PromotesTrace(t *testing.T) {
	store := &recordingStore{}
	e := newEchoWithPromotion(store)

	req := httptest.NewRequest(http.MethodGet, "/fakeroute/asdf", http.NoBody)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	traces := store.snapshot()
	require.Len(t, traces, 1)
	require.Equal(t, http.StatusNotFound, traces[0].StatusCode)
	require.Equal(t, "/fakeroute/asdf", traces[0].Route)
}

// TestHandleNotFound_NonHTMX_RendersBody protects the existing UX: non-HTMX
// 404 still emits a non-empty HTML body (not just an OOB swap).
func TestHandleNotFound_NonHTMX_RendersBody(t *testing.T) {
	store := &recordingStore{}
	e := newEchoWithPromotion(store)

	req := httptest.NewRequest(http.MethodGet, "/fakeroute/asdf", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.NotEmpty(t, rec.Body.String(), "non-HTMX 404 body should not be empty")
}
