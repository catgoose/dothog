// setup:feature:graph

package graph

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"catgoose/dothog/internal/logger"

	"github.com/catgoose/chuck/dbrepo"
	"github.com/jmoiron/sqlx"
)

// Directory sync state keys stored in MetaTable.
const (
	syncKindUsers = "users"
)

// Directory is the persistent Graph-owned directory snapshot: a SQLite cache
// of Azure users (UsersTable) co-located with optional avatar blobs
// (PhotosTable). Open via OpenDirectory; the underlying DB handle is hidden
// so callers consume narrow accessors (PhotoCache via Photos(), user query
// methods directly on Directory) rather than the raw pool.
type Directory struct {
	db *sqlx.DB
}

// Close releases the backing DB handle. Safe on a nil Directory.
func (d *Directory) Close() error {
	if d == nil || d.db == nil {
		return nil
	}
	return d.db.Close()
}

// setup:feature:avatar:start

// Photos returns a PhotoCache view over this Directory's underlying DB.
// Both share the same connection pool; routes/handlers should hold the
// returned *PhotoCache rather than the parent Directory.
func (d *Directory) Photos() *PhotoCache {
	return &PhotoCache{db: d.db}
}

// setup:feature:avatar:end

// InsertOrUpdateUsers upserts each Graph user in a single transaction,
// rolling back on the first row that fails.
func (d *Directory) InsertOrUpdateUsers(ctx context.Context, users []GraphUser) error {
	log := logger.WithContext(ctx)

	if err := d.EnsureSchema(ctx); err != nil {
		return fmt.Errorf("failed to ensure schema: %w", err)
	}

	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Error("Failed to rollback transaction", "rollback_error", rollbackErr, "original_error", err)
			}
		}
	}()

	query := `
		INSERT INTO Users (AzureId, GivenName, Surname, DisplayName, UserPrincipalName, Mail, JobTitle, OfficeLocation, Department, CompanyName, AccountName, UpdatedAt)
		VALUES (:AzureId, :GivenName, :Surname, :DisplayName, :UserPrincipalName, :Mail, :JobTitle, :OfficeLocation, :Department, :CompanyName, :AccountName, CURRENT_TIMESTAMP)
		ON CONFLICT(AzureId) DO UPDATE SET
		  AzureId = excluded.AzureId
			, GivenName = excluded.GivenName
			, Surname = excluded.Surname
			, DisplayName = excluded.DisplayName
			, UserPrincipalName = excluded.UserPrincipalName
			, Mail = excluded.Mail
			, JobTitle = excluded.JobTitle
			, OfficeLocation = excluded.OfficeLocation
			, Department = excluded.Department
			, CompanyName = excluded.CompanyName
			, AccountName = excluded.AccountName
			, UpdatedAt = CURRENT_TIMESTAMP;
	`

	for _, user := range users {
		if _, err = tx.NamedExecContext(ctx, query, user); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Error("Failed to rollback transaction after insert/update error", "rollback_error", rollbackErr, "original_error", err, "user_id", user.AzureID)
			}
			return fmt.Errorf("failed to insert/update user %s: %w", user.AzureID, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			log.Error("Failed to rollback transaction after commit error", "rollback_error", rollbackErr, "commit_error", err)
		}
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

const userSelect = "SELECT AzureId, GivenName, Surname, DisplayName, UserPrincipalName, Mail, JobTitle, OfficeLocation, Department, CompanyName, AccountName"

// SearchUsers ANDs LIKE clauses across GivenName/Surname/DisplayName/AccountName
// for each term and returns up to limit rows ordered by DisplayName.
func (d *Directory) SearchUsers(ctx context.Context, terms []string, limit int) ([]GraphUser, error) {
	if len(terms) == 0 {
		return nil, fmt.Errorf("no search terms provided")
	}
	var conditions []string
	var args []any

	for i, term := range terms {
		searchPattern := "%" + term + "%"
		paramName := fmt.Sprintf("Search%d", i)
		conditions = append(conditions, fmt.Sprintf("(GivenName LIKE @%s OR Surname LIKE @%s OR DisplayName LIKE @%s OR AccountName LIKE @%s)", paramName, paramName, paramName, paramName))
		args = append(args, sql.Named(paramName, searchPattern))
	}
	whereClause := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`
	  %s
		FROM Users
		WHERE (%s)
		ORDER BY DisplayName
		LIMIT @Limit
	`, userSelect, whereClause)

	args = append(args, sql.Named("Limit", limit))

	var users []GraphUser
	if err := d.db.SelectContext(ctx, &users, query, args...); err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	return users, nil
}

// AllUsers dumps the entire Users table (DisplayName-sorted). Intended for
// one-off admin views; large directories should prefer SearchUsers.
func (d *Directory) AllUsers(ctx context.Context) ([]GraphUser, error) {
	query := userSelect + " FROM Users ORDER BY DisplayName"
	var users []GraphUser
	if err := d.db.SelectContext(ctx, &users, query); err != nil {
		return nil, fmt.Errorf("failed to retrieve all users: %w", err)
	}
	return users, nil
}

// UserByAzureID wraps sql.ErrNoRows in a contextual error when the row is missing.
func (d *Directory) UserByAzureID(ctx context.Context, azureID string) (*GraphUser, error) {
	query := userSelect + " FROM Users WHERE AzureId = @AzureId"
	var user GraphUser
	if err := d.db.GetContext(ctx, &user, query, sql.Named("AzureId", azureID)); err != nil {
		return nil, fmt.Errorf("failed to retrieve user by Azure ID: %w", err)
	}
	return &user, nil
}

// UsersTableExists probes sqlite_master so callers can decide whether to
// run EnsureSchema before queries.
func (d *Directory) UsersTableExists(ctx context.Context) (bool, error) {
	var count int
	if err := d.db.GetContext(ctx, &count, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='Users'"); err != nil {
		return false, err
	}
	return count > 0, nil
}

// EnsureSchema is idempotent: re-runs the Graph cache materializer against
// the bound DB so both UsersTable and PhotosTable exist. OpenDirectory
// already does this at open time; callers that re-enter EnsureSchema across
// long-lived runtimes can invoke it defensively.
func (d *Directory) EnsureSchema(ctx context.Context) error {
	return materializeCacheTables(ctx, d.db)
}

// UserCount is the row count of Users (cached and persisted).
func (d *Directory) UserCount(ctx context.Context) (int, error) {
	var count int
	if err := d.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM Users"); err != nil {
		return 0, fmt.Errorf("failed to get user count: %w", err)
	}
	return count, nil
}

// ReplaceUsers upserts the fetched users and deletes any local rows whose
// AzureId isn't in the fetched set. This is the snapshot-replace path used
// by full Graph syncs so users disabled/removed in Graph drop out of the
// local directory rather than living on forever. A successful empty fetch
// is a valid snapshot — it clears the cache. Degraded fetches show up as a
// non-nil error from fetchUsersFunc, not as an empty users slice, so the
// caller can distinguish "Graph said nothing" from "Graph is empty".
func (d *Directory) ReplaceUsers(ctx context.Context, users []GraphUser) error {
	if len(users) == 0 {
		// Wipe the table — successful empty snapshot means no Graph users.
		res, err := d.db.ExecContext(ctx, "DELETE FROM Users")
		if err != nil {
			return fmt.Errorf("clear users for empty snapshot: %w", err)
		}
		if removed, rerr := res.RowsAffected(); rerr == nil && removed > 0 {
			logger.WithContext(ctx).Info("Cleared directory for empty Graph snapshot", "count", removed)
		}
		return nil
	}
	if err := d.InsertOrUpdateUsers(ctx, users); err != nil {
		return err
	}
	// Build the keep-set via a parameterized IN clause. Names are @id0…@idN
	// to play nicely with sqlx's named-arg substitution.
	placeholders := make([]string, len(users))
	args := make([]any, 0, len(users))
	for i, u := range users {
		name := fmt.Sprintf("id%d", i)
		placeholders[i] = "@" + name
		args = append(args, sql.Named(name, u.AzureID))
	}
	query := fmt.Sprintf("DELETE FROM Users WHERE AzureId NOT IN (%s)", strings.Join(placeholders, ", "))
	res, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("prune stale users: %w", err)
	}
	if removed, rerr := res.RowsAffected(); rerr == nil && removed > 0 {
		logger.WithContext(ctx).Info("Pruned stale users from directory", "count", removed)
	}
	return nil
}

// LastSuccessfulSync returns when the named sync kind last completed
// successfully. The bool is false when no row exists yet (never synced).
func (d *Directory) LastSuccessfulSync(ctx context.Context, kind string) (time.Time, bool, error) {
	var t sql.NullTime
	err := d.db.GetContext(ctx, &t,
		"SELECT LastSuccessAt FROM "+MetaTable.Name+" WHERE Kind = @Kind",
		sql.Named("Kind", kind),
	)
	if err == sql.ErrNoRows {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("read last successful sync for %s: %w", kind, err)
	}
	if !t.Valid {
		return time.Time{}, true, nil
	}
	return t.Time, true, nil
}

// MarkSuccessfulSync records the current time as the last successful run
// for kind. Subsequent reads via LastSuccessfulSync compare against this
// timestamp to decide whether the next sync can skip a refresh.
func (d *Directory) MarkSuccessfulSync(ctx context.Context, kind string) error {
	now := dbrepo.GetNow()
	_, err := d.db.ExecContext(ctx,
		"INSERT INTO "+MetaTable.Name+" (Kind, LastSuccessAt, CreatedAt, UpdatedAt) "+
			"VALUES (@Kind, @LastSuccessAt, @CreatedAt, @UpdatedAt) "+
			"ON CONFLICT(Kind) DO UPDATE SET "+
			"LastSuccessAt = excluded.LastSuccessAt, "+
			"UpdatedAt = excluded.UpdatedAt",
		sql.Named("Kind", kind),
		sql.Named("LastSuccessAt", now),
		sql.Named("CreatedAt", now),
		sql.Named("UpdatedAt", now),
	)
	if err != nil {
		return fmt.Errorf("mark successful sync for %s: %w", kind, err)
	}
	return nil
}
