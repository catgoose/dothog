package requestlog

import (
	"context"
	"testing"
	"time"

	"catgoose/dothog/internal/database/dialect"
	"catgoose/dothog/internal/database/schema"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	d := dialect.SQLiteDialect{}
	for _, stmt := range schema.ErrorTracesTable.CreateIfNotExistsSQL(d) {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}
	return db
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(openTestDB(t))
}

func sampleTrace(requestID string, statusCode int, method string) ErrorTrace {
	return ErrorTrace{
		RequestID:  requestID,
		ErrorChain: "something went wrong",
		StatusCode: statusCode,
		Route:      "/api/test",
		Method:     method,
		UserAgent:  "TestAgent/1.0",
		RemoteIP:   "127.0.0.1",
		UserID:     "user-42",
		Entries: []Entry{
			{Time: time.Now(), Level: "ERROR", Message: "test error", Attrs: "key=val"},
		},
	}
}

// --- Buffer context tests ---

func TestNewBufferContext_GetBuffer_Roundtrip(t *testing.T) {
	ctx := NewBufferContext(context.Background())
	buf := GetBuffer(ctx)
	require.NotNil(t, buf)
	assert.Empty(t, buf.Entries)
}

func TestGetBuffer_PlainContext_ReturnsNil(t *testing.T) {
	buf := GetBuffer(context.Background())
	assert.Nil(t, buf)
}

// --- Store.Promote / Get tests ---

func TestPromote_AndGet_Roundtrip(t *testing.T) {
	store := newTestStore(t)
	trace := sampleTrace("req-1", 500, "GET")
	store.Promote(trace)

	got := store.Get("req-1")
	require.NotNil(t, got)
	assert.Equal(t, "req-1", got.RequestID)
	assert.Equal(t, 500, got.StatusCode)
	assert.Equal(t, "GET", got.Method)
	assert.Equal(t, "/api/test", got.Route)
	assert.Equal(t, "something went wrong", got.ErrorChain)
	assert.Equal(t, "TestAgent/1.0", got.UserAgent)
	assert.Equal(t, "127.0.0.1", got.RemoteIP)
	assert.Equal(t, "user-42", got.UserID)
	require.Len(t, got.Entries, 1)
	assert.Equal(t, "test error", got.Entries[0].Message)
}

func TestPromote_FiresOnPromoteCallback(t *testing.T) {
	store := newTestStore(t)

	var received TraceSummary
	called := false
	store.SetOnPromote(func(ts TraceSummary) {
		called = true
		received = ts
	})

	trace := sampleTrace("req-cb", 503, "POST")
	store.Promote(trace)

	require.True(t, called)
	assert.Equal(t, "req-cb", received.RequestID)
	assert.Equal(t, 503, received.StatusCode)
	assert.Equal(t, "POST", received.Method)
}

func TestPromote_EmptyEntries(t *testing.T) {
	store := newTestStore(t)
	trace := sampleTrace("req-empty", 500, "GET")
	trace.Entries = nil

	store.Promote(trace)

	got := store.Get("req-empty")
	require.NotNil(t, got)
	assert.Empty(t, got.Entries)
}

func TestGet_UnknownRequestID_ReturnsNil(t *testing.T) {
	store := newTestStore(t)
	got := store.Get("nonexistent")
	assert.Nil(t, got)
}

// --- ListTraces tests ---

func TestListTraces_BasicPagination(t *testing.T) {
	store := newTestStore(t)

	for i := 0; i < 5; i++ {
		store.Promote(sampleTrace(
			"req-"+string(rune('a'+i)),
			500, "GET",
		))
	}

	rows, total, err := store.ListTraces(TraceFilter{Page: 1, PerPage: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, rows, 2)

	rows2, _, err := store.ListTraces(TraceFilter{Page: 2, PerPage: 2})
	require.NoError(t, err)
	assert.Len(t, rows2, 2)
	assert.NotEqual(t, rows[0].RequestID, rows2[0].RequestID)
}

func TestListTraces_SearchFilter(t *testing.T) {
	store := newTestStore(t)
	store.Promote(sampleTrace("req-alpha", 500, "GET"))

	t2 := sampleTrace("req-beta", 404, "POST")
	t2.Route = "/api/special"
	store.Promote(t2)

	// Search by route
	rows, total, err := store.ListTraces(TraceFilter{Q: "special", Page: 1, PerPage: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "req-beta", rows[0].RequestID)

	// Search by request ID
	rows, total, err = store.ListTraces(TraceFilter{Q: "alpha", Page: 1, PerPage: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "req-alpha", rows[0].RequestID)
}

func TestListTraces_StatusFilter(t *testing.T) {
	store := newTestStore(t)
	store.Promote(sampleTrace("req-400", 400, "GET"))
	store.Promote(sampleTrace("req-404", 404, "GET"))
	store.Promote(sampleTrace("req-500", 500, "GET"))
	store.Promote(sampleTrace("req-502", 502, "GET"))

	// 4xx filter
	rows, total, err := store.ListTraces(TraceFilter{Status: "4xx", Page: 1, PerPage: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)

	// 5xx filter
	rows, total, err = store.ListTraces(TraceFilter{Status: "5xx", Page: 1, PerPage: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)

	// Specific status code
	rows, total, err = store.ListTraces(TraceFilter{Status: "404", Page: 1, PerPage: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "req-404", rows[0].RequestID)
}

func TestListTraces_MethodFilter(t *testing.T) {
	store := newTestStore(t)
	store.Promote(sampleTrace("req-get", 500, "GET"))
	store.Promote(sampleTrace("req-post", 500, "POST"))

	rows, total, err := store.ListTraces(TraceFilter{Method: "POST", Page: 1, PerPage: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "req-post", rows[0].RequestID)
}

func TestListTraces_Sorting(t *testing.T) {
	store := newTestStore(t)

	t1 := sampleTrace("req-1", 500, "GET")
	t1.Route = "/b"
	store.PromoteAt(t1, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	t2 := sampleTrace("req-2", 400, "POST")
	t2.Route = "/a"
	store.PromoteAt(t2, time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC))

	// Sort by Route ascending
	rows, _, err := store.ListTraces(TraceFilter{Sort: "Route", Dir: "asc", Page: 1, PerPage: 10})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "/a", rows[0].Route)
	assert.Equal(t, "/b", rows[1].Route)

	// Sort by StatusCode descending (default dir)
	rows, _, err = store.ListTraces(TraceFilter{Sort: "StatusCode", Page: 1, PerPage: 10})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, 500, rows[0].StatusCode)
	assert.Equal(t, 400, rows[1].StatusCode)
}

// --- DeleteTrace tests ---

func TestDeleteTrace(t *testing.T) {
	store := newTestStore(t)
	store.Promote(sampleTrace("req-del", 500, "GET"))

	require.NotNil(t, store.Get("req-del"))

	err := store.DeleteTrace("req-del")
	require.NoError(t, err)

	assert.Nil(t, store.Get("req-del"))
}

// --- PromoteAt tests ---

func TestPromoteAt_StoresCustomTimestamp(t *testing.T) {
	store := newTestStore(t)
	ts := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)
	store.PromoteAt(sampleTrace("req-ts", 500, "GET"), ts)

	got := store.Get("req-ts")
	require.NotNil(t, got)
	assert.Contains(t, got.CreatedAt, "2024-06-15")
}
