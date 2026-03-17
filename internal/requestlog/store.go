// Package requestlog provides per-request log capture with promote-on-error
// semantics. Each request buffers its slog records locally; only when an error
// occurs is the buffer promoted to a SQLite-backed Store for later retrieval.
package requestlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	dbrepo "catgoose/dothog/internal/database/repository"
	"catgoose/dothog/internal/database/schema"

	"github.com/jmoiron/sqlx"
)

// Entry is a single captured log record.
type Entry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"msg"`
	Attrs   string    `json:"attrs,omitempty"`
}

// Buffer is a per-request log buffer stored in the request context.
// It is not thread-safe — each request is handled by a single goroutine.
type Buffer struct {
	Entries []Entry
}

type bufferKey struct{}

// NewBufferContext returns a new context with an empty Buffer attached.
func NewBufferContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, bufferKey{}, &Buffer{})
}

// GetBuffer retrieves the per-request Buffer from the context, or nil.
func GetBuffer(ctx context.Context) *Buffer {
	buf, _ := ctx.Value(bufferKey{}).(*Buffer)
	return buf
}

// ErrorTrace contains all the information captured when a request errors.
type ErrorTrace struct {
	RequestID  string
	ErrorChain string
	StatusCode int
	Route      string
	Method     string
	UserAgent  string
	RemoteIP   string
	UserID     string
	Entries    []Entry
	CreatedAt  string
}

// TraceSummary is a lightweight row for list views (no log entries).
type TraceSummary struct {
	RequestID  string `db:"RequestID"`
	ErrorChain string `db:"ErrorChain"`
	StatusCode int    `db:"StatusCode"`
	Route      string `db:"Route"`
	Method     string `db:"Method"`
	RemoteIP   string `db:"RemoteIP"`
	UserID     string `db:"UserID"`
	CreatedAt  string `db:"CreatedAt"`
}

var tableName = schema.ErrorTracesTable.Name

// Store is a SQLite-backed store of error request log entries.
// Only requests that encounter errors are promoted here.
type Store struct {
	db        *sqlx.DB
	onPromote func(TraceSummary) // optional callback fired after each promote
}

// NewStore creates a Store backed by the given database connection.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// SetOnPromote registers a callback that is invoked after each successful promote.
// Used to broadcast new error traces via SSE.
func (s *Store) SetOnPromote(fn func(TraceSummary)) {
	s.onPromote = fn
}

// Promote persists an error trace to the database. This should only
// be called when the request resulted in an error.
func (s *Store) Promote(trace ErrorTrace) {
	s.promoteAt(trace, dbrepo.GetNow())
}

// PromoteAt persists an error trace with a specific timestamp. Used for seeding.
func (s *Store) PromoteAt(trace ErrorTrace, createdAt time.Time) {
	s.promoteAt(trace, createdAt)
}

func (s *Store) promoteAt(trace ErrorTrace, createdAt time.Time) {
	data, err := json.Marshal(trace.Entries)
	if err != nil {
		return
	}
	insertCols := schema.ErrorTracesTable.InsertColumns()
	query := dbrepo.InsertInto(tableName, insertCols...)
	_, execErr := s.db.Exec(query,
		sql.Named("RequestID", trace.RequestID),
		sql.Named("ErrorChain", trace.ErrorChain),
		sql.Named("StatusCode", trace.StatusCode),
		sql.Named("Route", trace.Route),
		sql.Named("Method", trace.Method),
		sql.Named("UserAgent", trace.UserAgent),
		sql.Named("RemoteIP", trace.RemoteIP),
		sql.Named("UserID", trace.UserID),
		sql.Named("Entries", string(data)),
		sql.Named("CreatedAt", createdAt),
		sql.Named("UpdatedAt", createdAt),
	)
	if execErr == nil && s.onPromote != nil {
		s.onPromote(TraceSummary{
			RequestID:  trace.RequestID,
			ErrorChain: trace.ErrorChain,
			StatusCode: trace.StatusCode,
			Route:      trace.Route,
			Method:     trace.Method,
			RemoteIP:   trace.RemoteIP,
			UserID:     trace.UserID,
			CreatedAt:  createdAt.Format("2006-01-02 15:04:05"),
		})
	}
}

// errorTraceRow maps to a row in the ErrorTraces table.
type errorTraceRow struct {
	RequestID  string `db:"RequestID"`
	ErrorChain string `db:"ErrorChain"`
	StatusCode int    `db:"StatusCode"`
	Route      string `db:"Route"`
	Method     string `db:"Method"`
	UserAgent  string `db:"UserAgent"`
	RemoteIP   string `db:"RemoteIP"`
	UserID     string `db:"UserID"`
	Entries    string `db:"Entries"`
	CreatedAt  string `db:"CreatedAt"`
}

var selectCols = dbrepo.Columns(
	"RequestID", "ErrorChain", "StatusCode", "Route", "Method",
	"UserAgent", "RemoteIP", "UserID", "Entries", "CreatedAt",
)

var summaryCols = dbrepo.Columns(
	"RequestID", "ErrorChain", "StatusCode", "Route", "Method",
	"RemoteIP", "UserID", "CreatedAt",
)

// TraceFilter holds all filter parameters for ListTraces.
type TraceFilter struct {
	Q      string
	Status string
	Method string
	Sort   string
	Dir    string
	Page   int
	PerPage int
}

// FilterOptions holds the distinct values available for filter dropdowns,
// computed from rows that match the *other* active filters.
type FilterOptions struct {
	StatusCodes []int
	Methods     []string
}

// AvailableFilters returns the distinct status codes and methods present in
// the store. Each facet is filtered by the *other* facet's selection so the
// dropdowns only show values that would produce results.
//   - Available status codes: filtered by search + method
//   - Available methods: filtered by search + status
func (s *Store) AvailableFilters(f TraceFilter) (FilterOptions, error) {
	// Status codes: apply search + method (not status itself)
	sw := s.buildSearchWhere(f)
	applyMethodWhere(sw, f)
	statusQ := fmt.Sprintf("SELECT DISTINCT StatusCode FROM %s %s ORDER BY StatusCode", tableName, sw.String())
	var codes []int
	if err := s.db.Select(&codes, statusQ, sw.Args()...); err != nil {
		return FilterOptions{}, fmt.Errorf("available status codes: %w", err)
	}

	// Methods: apply search + status (not method itself)
	mw := s.buildSearchWhere(f)
	applyStatusWhere(mw, f)
	methodQ := fmt.Sprintf("SELECT DISTINCT Method FROM %s %s ORDER BY Method", tableName, mw.String())
	var methods []string
	if err := s.db.Select(&methods, methodQ, mw.Args()...); err != nil {
		return FilterOptions{}, fmt.Errorf("available methods: %w", err)
	}

	return FilterOptions{StatusCodes: codes, Methods: methods}, nil
}

// applyStatusWhere adds the status filter clause to w.
func applyStatusWhere(w *dbrepo.WhereBuilder, f TraceFilter) {
	if f.Status == "" {
		return
	}
	switch f.Status {
	case "4xx":
		w.And("StatusCode >= 400 AND StatusCode < 500")
	case "5xx":
		w.And("StatusCode >= 500")
	default:
		code := 0
		fmt.Sscanf(f.Status, "%d", &code)
		if code > 0 {
			w.And("StatusCode = @StatusCode", sql.Named("StatusCode", code))
		}
	}
}

// applyMethodWhere adds the method filter clause to w.
func applyMethodWhere(w *dbrepo.WhereBuilder, f TraceFilter) {
	if f.Method != "" {
		w.And("Method = @Method", sql.Named("Method", f.Method))
	}
}

// buildSearchWhere builds the WHERE clause for the search query only (no
// status/method filters) so it can be reused by AvailableFilters.
func (s *Store) buildSearchWhere(f TraceFilter) *dbrepo.WhereBuilder {
	w := dbrepo.NewWhere()
	if f.Q != "" {
		for i, term := range strings.Fields(f.Q) {
			param := fmt.Sprintf("Q%d", i)
			pattern := "%" + term + "%"
			clause := fmt.Sprintf(
				"(Route LIKE @%s OR ErrorChain LIKE @%s OR RequestID LIKE @%s OR UserID LIKE @%s OR RemoteIP LIKE @%s OR CAST(StatusCode AS TEXT) LIKE @%s OR Method LIKE @%s)",
				param, param, param, param, param, param, param)
			w.And(clause, sql.Named(param, pattern))
		}
	}
	return w
}

// ListTraces returns a page of trace summaries matching the given filters.
func (s *Store) ListTraces(f TraceFilter) ([]TraceSummary, int, error) {
	w := s.buildSearchWhere(f)
	applyStatusWhere(w, f)
	applyMethodWhere(w, f)

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", tableName, w.String())
	var total int
	if err := s.db.Get(&total, countQuery, w.Args()...); err != nil {
		return nil, 0, fmt.Errorf("count error traces: %w", err)
	}

	// Sort
	orderCol := "CreatedAt"
	orderDir := "DESC"
	validSorts := map[string]bool{"CreatedAt": true, "StatusCode": true, "Route": true, "Method": true}
	if validSorts[f.Sort] {
		orderCol = f.Sort
	}
	if f.Dir == "asc" {
		orderDir = "ASC"
	}

	offset := (f.Page - 1) * f.PerPage
	dataQuery := fmt.Sprintf("SELECT %s FROM %s %s ORDER BY %s %s LIMIT @Limit OFFSET @Offset",
		summaryCols, tableName, w.String(), orderCol, orderDir)
	args := append(w.Args(), sql.Named("Limit", f.PerPage), sql.Named("Offset", offset))

	var rows []TraceSummary
	if err := s.db.Select(&rows, dataQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("list error traces: %w", err)
	}
	return rows, total, nil
}

// DeleteTrace removes a single trace by request ID.
func (s *Store) DeleteTrace(requestID string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE RequestID = @RequestID", tableName)
	_, err := s.db.Exec(query, sql.Named("RequestID", requestID))
	return err
}

// Get returns the full error trace for a request ID, or nil if not found.
func (s *Store) Get(requestID string) *ErrorTrace {
	w := dbrepo.NewWhere().And("RequestID = @RequestID", sql.Named("RequestID", requestID))
	query, args := dbrepo.NewSelect(tableName, selectCols).Where(w).Build()

	var row errorTraceRow
	if err := s.db.Get(&row, query, args...); err != nil {
		return nil
	}
	var entries []Entry
	if err := json.Unmarshal([]byte(row.Entries), &entries); err != nil {
		entries = nil
	}
	return &ErrorTrace{
		RequestID:  row.RequestID,
		ErrorChain: row.ErrorChain,
		StatusCode: row.StatusCode,
		Route:      row.Route,
		Method:     row.Method,
		UserAgent:  row.UserAgent,
		RemoteIP:   row.RemoteIP,
		UserID:     row.UserID,
		Entries:    entries,
		CreatedAt:  row.CreatedAt,
	}
}

// StartCleanup runs a background goroutine that deletes entries older than ttl.
// It stops when ctx is cancelled.
func (s *Store) StartCleanup(ctx context.Context, ttl time.Duration, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.deleteOlderThan(ttl)
			}
		}
	}()
}

func (s *Store) deleteOlderThan(ttl time.Duration) {
	cutoff := time.Now().Add(-ttl)
	query := fmt.Sprintf("DELETE FROM %s WHERE CreatedAt < @Cutoff", tableName)
	_, _ = s.db.Exec(query, sql.Named("Cutoff", cutoff))
}
