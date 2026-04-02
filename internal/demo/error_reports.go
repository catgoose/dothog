// setup:feature:demo

package demo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ErrorReportStatus represents the lifecycle state of an error report.
type ErrorReportStatus string

// Error report status constants.
const (
	ErrorReportStatusPending   ErrorReportStatus = "pending"
	ErrorReportStatusResolved  ErrorReportStatus = "resolved"
	ErrorReportStatusDismissed ErrorReportStatus = "dismissed"
)

// ErrorReport represents a user-submitted error report stored in the demo DB.
type ErrorReport struct {
	ID          int
	RequestID   string
	Description string
	Route       string
	StatusCode  int
	UserAgent   string
	LogEntries  string // JSON text of captured log entries
	Status      ErrorReportStatus
	CreatedAt   time.Time
	ResolvedAt  sql.NullTime
}

// CreatedAtFormatted returns the created_at timestamp in a human-readable format.
func (r ErrorReport) CreatedAtFormatted() string {
	return r.CreatedAt.Format("2006-01-02 15:04:05")
}

// ResolvedAtFormatted returns the resolved_at timestamp formatted, or "" if null.
func (r ErrorReport) ResolvedAtFormatted() string {
	if !r.ResolvedAt.Valid {
		return ""
	}
	return r.ResolvedAt.Time.Format("2006-01-02 15:04:05")
}

// DescriptionPreview returns a truncated description for table display.
func (r ErrorReport) DescriptionPreview() string {
	if len(r.Description) <= 80 {
		return r.Description
	}
	return r.Description[:77] + "..."
}

// UserAgentSnippet returns a truncated user agent string for table display.
func (r ErrorReport) UserAgentSnippet() string {
	if len(r.UserAgent) <= 40 {
		return r.UserAgent
	}
	return r.UserAgent[:37] + "..."
}

// ErrorReportStatuses is the list of valid statuses for filters.
var ErrorReportStatuses = []string{
	string(ErrorReportStatusPending),
	string(ErrorReportStatusResolved),
	string(ErrorReportStatusDismissed),
}

// allowedErrorReportSort maps query-param sort keys to safe SQL column names.
var allowedErrorReportSort = map[string]string{
	"created_at":  "created_at",
	"status_code": "status_code",
	"status":      "status",
	"route":       "route",
}

// InsertErrorReport stores a new error report and returns it with the assigned ID.
func (d *DB) InsertErrorReport(ctx context.Context, r ErrorReport) (ErrorReport, error) {
	res, err := d.db.ExecContext(ctx,
		`INSERT INTO error_reports (request_id, description, route, status_code, user_agent, log_entries, status, created_at)
		 VALUES (@RequestID, @Description, @Route, @StatusCode, @UserAgent, @LogEntries, @Status, @CreatedAt)`,
		sql.Named("RequestID", r.RequestID),
		sql.Named("Description", r.Description),
		sql.Named("Route", r.Route),
		sql.Named("StatusCode", r.StatusCode),
		sql.Named("UserAgent", r.UserAgent),
		sql.Named("LogEntries", r.LogEntries),
		sql.Named("Status", string(ErrorReportStatusPending)),
		sql.Named("CreatedAt", time.Now().UTC().Format(time.RFC3339)),
	)
	if err != nil {
		return ErrorReport{}, fmt.Errorf("insert error report: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ErrorReport{}, fmt.Errorf("get last insert id: %w", err)
	}
	r.ID = int(id)
	r.Status = ErrorReportStatusPending
	return r, nil
}

// GetErrorReport returns a single error report by ID.
func (d *DB) GetErrorReport(ctx context.Context, id int) (ErrorReport, error) {
	var r ErrorReport
	var status string
	var createdAt string
	var resolvedAt sql.NullString
	err := d.db.QueryRowContext(ctx,
		`SELECT id, request_id, description, route, status_code, user_agent, log_entries, status, created_at, resolved_at
		 FROM error_reports WHERE id = @ID`,
		sql.Named("ID", id),
	).Scan(&r.ID, &r.RequestID, &r.Description, &r.Route, &r.StatusCode, &r.UserAgent, &r.LogEntries, &status, &createdAt, &resolvedAt)
	if err != nil {
		return ErrorReport{}, fmt.Errorf("get error report %d: %w", id, err)
	}
	r.Status = ErrorReportStatus(status)
	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		r.CreatedAt = t
	} else {
		r.CreatedAt = time.Now()
	}
	if resolvedAt.Valid {
		if t, err := time.Parse(time.RFC3339, resolvedAt.String); err == nil {
			r.ResolvedAt = sql.NullTime{Time: t, Valid: true}
		} else {
			r.ResolvedAt = sql.NullTime{Time: time.Now(), Valid: true}
		}
	}
	return r, nil
}

// ListErrorReports queries error reports with optional filters, sort, and pagination.
func (d *DB) ListErrorReports(ctx context.Context, search, status, sortBy, sortDir string, page, perPage int) ([]ErrorReport, int, error) {
	col, ok := allowedErrorReportSort[sortBy]
	if !ok {
		col = "created_at"
		sortDir = "desc"
	}
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	var conds []string
	var args []any
	if search != "" {
		conds = append(conds, "(request_id LIKE @Search OR description LIKE @Search OR route LIKE @Search)")
		args = append(args, sql.Named("Search", "%"+search+"%"))
	}
	if status != "" {
		conds = append(conds, "status = @Status")
		args = append(args, sql.Named("Status", status))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	if err := d.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM error_reports %s", where), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count error reports: %w", err)
	}

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage
	query := fmt.Sprintf(
		`SELECT id, request_id, description, route, status_code, user_agent, log_entries, status, created_at, resolved_at
		 FROM error_reports %s ORDER BY %s %s LIMIT @Limit OFFSET @Offset`,
		where, col, sortDir)
	la := make([]any, len(args), len(args)+2)
	copy(la, args)
	la = append(la, sql.Named("Limit", perPage), sql.Named("Offset", offset))

	rows, err := d.db.QueryContext(ctx, query, la...)
	if err != nil {
		return nil, 0, fmt.Errorf("list error reports: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var reports []ErrorReport
	for rows.Next() {
		var r ErrorReport
		var statusStr string
		var createdAt string
		var resolvedAt sql.NullString
		if err := rows.Scan(&r.ID, &r.RequestID, &r.Description, &r.Route, &r.StatusCode, &r.UserAgent, &r.LogEntries, &statusStr, &createdAt, &resolvedAt); err != nil {
			return nil, 0, err
		}
		r.Status = ErrorReportStatus(statusStr)
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			r.CreatedAt = t
		} else {
			r.CreatedAt = time.Now()
		}
		if resolvedAt.Valid {
			if t, err := time.Parse(time.RFC3339, resolvedAt.String); err == nil {
				r.ResolvedAt = sql.NullTime{Time: t, Valid: true}
			} else {
				r.ResolvedAt = sql.NullTime{Time: time.Now(), Valid: true}
			}
		}
		reports = append(reports, r)
	}
	return reports, total, rows.Err()
}

// UpdateErrorReportStatus changes the status of an error report.
func (d *DB) UpdateErrorReportStatus(ctx context.Context, id int, status ErrorReportStatus) error {
	var resolvedAt any
	if status == ErrorReportStatusResolved || status == ErrorReportStatusDismissed {
		resolvedAt = time.Now().UTC().Format(time.RFC3339)
	}
	res, err := d.db.ExecContext(ctx,
		"UPDATE error_reports SET status = @Status, resolved_at = @ResolvedAt WHERE id = @ID",
		sql.Named("Status", string(status)),
		sql.Named("ResolvedAt", resolvedAt),
		sql.Named("ID", id),
	)
	if err != nil {
		return fmt.Errorf("update error report %d status: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("update error report %d: no rows affected", id)
	}
	return nil
}

// initErrorReports creates the error_reports table if it does not exist.
func (d *DB) initErrorReports() error {
	_, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS error_reports (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		request_id  TEXT    NOT NULL DEFAULT '',
		description TEXT    NOT NULL DEFAULT '',
		route       TEXT    NOT NULL DEFAULT '',
		status_code INTEGER NOT NULL DEFAULT 0,
		user_agent  TEXT    NOT NULL DEFAULT '',
		log_entries TEXT    NOT NULL DEFAULT '[]',
		status      TEXT    NOT NULL DEFAULT 'pending',
		created_at  TEXT    NOT NULL,
		resolved_at TEXT
	)`)
	if err != nil {
		return fmt.Errorf("create error_reports table: %w", err)
	}
	return nil
}
