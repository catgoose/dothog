// setup:feature:graph

package graph

import (
	"context"
	"fmt"
	"time"

	"catgoose/dothog/internal/logger"
	"catgoose/dothog/internal/shared"

	"catgoose/dothog/internal/env"
)

// SyncType labels a directory refresh for telemetry; the constant value
// is the human-readable phrase logged alongside each sync.
type SyncType string

// Sync trigger labels written into logs and admin telemetry.
const (
	SyncTypeInitial   SyncType = "initial sync"
	SyncTypeManual    SyncType = "manual sync"
	SyncTypeScheduled SyncType = "scheduled sync"
	SyncTypePeriodic  SyncType = "periodic sync"
)

// devSyncFreshness bounds how long a dev-mode startup can reuse the cached
// directory before re-fetching from Graph. Without this, a non-empty cache
// would stay stale forever on every restart.
const devSyncFreshness = 6 * time.Hour

// InitAndSyncDirectory primes the Graph directory cache: dev fetches when
// the cache is empty OR the last successful sync is older than
// devSyncFreshness; prod performs an initial sync then schedules a daily
// refresh at refreshHour. fetchUsersFunc takes a context so each sync
// invocation can carry its own deadline/cancellation.
func InitAndSyncDirectory(
	ctx context.Context,
	dir *Directory,
	refreshHour int,
	fetchUsersFunc func(ctx context.Context) ([]GraphUser, error),
	afterSync func(ctx context.Context, users []GraphUser),
) error {
	ctx = shared.WithContextIDAndDescription(ctx, shared.GenerateContextID(), "directory init")
	log := logger.WithContext(ctx)
	isDev := env.Dev()

	if isDev {
		if reuse, reason := devSnapshotIsFresh(ctx, dir, log); reuse {
			log.Info("Using existing directory snapshot in development", "reason", reason)
			return nil
		}
		log.Info("Development mode: fetching users from Azure")
		users, err := fetchUsersFunc(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch users from Azure: %w", err)
		}
		log.Info("Successfully fetched users from Azure", "count", len(users))
		if err := dir.ReplaceUsers(ctx, users); err != nil {
			return fmt.Errorf("failed to replace users in cache: %w", err)
		}
		if err := dir.MarkSuccessfulSync(ctx, syncKindUsers); err != nil {
			log.Warn("Failed to record sync timestamp", "error", err)
		}
		log.Info("Successfully synced users to cache")
		if afterSync != nil {
			afterSync(ctx, users)
		}
		return nil
	}

	log.Info("Production mode: performing initial user sync")
	users, fetchErr := fetchUsersFunc(ctx)
	switch {
	case fetchErr != nil:
		// Snapshot-first: a usable local cache must survive a failed initial
		// refresh so the app can serve requests immediately while the
		// background loop retries. With no usable snapshot at all, surface
		// the error so the operator sees the empty-start failure.
		hasSnapshot, snapErr := hasUsableSnapshot(ctx, dir)
		if snapErr != nil {
			log.Warn("Failed to check existing snapshot after fetch error; surfacing fetch error", "snapshot_error", snapErr)
		}
		if !hasSnapshot {
			return fmt.Errorf("failed to fetch users during initial sync: %w", fetchErr)
		}
		log.Warn("Initial Graph fetch failed; continuing with existing snapshot — background refresh will retry", "error", fetchErr)
	default:
		if err := dir.ReplaceUsers(ctx, users); err != nil {
			return fmt.Errorf("failed to sync users during initial sync: %w", err)
		}
		if err := dir.MarkSuccessfulSync(ctx, syncKindUsers); err != nil {
			log.Warn("Failed to record sync timestamp", "error", err)
		}
		log.Info("Successfully completed initial sync", "user_count", len(users))
		if afterSync != nil {
			afterSync(ctx, users)
		}
	}

	doSync := func(syncType SyncType) {
		syncCtx := shared.WithContextIDAndDescription(ctx, shared.GenerateContextID(), string(syncType))
		syncLog := logger.WithContext(syncCtx)
		syncLog.Info("Starting directory refresh", "type", syncType)
		users, err := fetchUsersFunc(syncCtx)
		if err != nil {
			syncLog.Error("Failed to fetch users during sync", "type", syncType, "error", err)
			return
		}
		if err := dir.ReplaceUsers(syncCtx, users); err != nil {
			syncLog.Error("Failed to sync users during sync", "type", syncType, "error", err)
			return
		}
		if err := dir.MarkSuccessfulSync(syncCtx, syncKindUsers); err != nil {
			syncLog.Warn("Failed to record sync timestamp", "type", syncType, "error", err)
		}
		syncLog.Info("Successfully completed sync", "type", syncType, "user_count", len(users))
		if afterSync != nil {
			afterSync(syncCtx, users)
		}
	}

	nextRefreshTime := func() time.Time {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), refreshHour, 0, 0, 0, now.Location())
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		return next
	}

	go func() {
		for {
			next := nextRefreshTime()
			logger.WithContext(ctx).Info("Scheduled directory refresh", "next_refresh", next, "wait_duration", time.Until(next))
			timer := time.NewTimer(time.Until(next))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				doSync(SyncTypeScheduled)
			}
		}
	}()
	return nil
}

// hasUsableSnapshot reports whether the directory has a non-empty Users
// table. Used by the production startup path to decide whether a failed
// initial Graph fetch should error or fall back to the existing snapshot.
func hasUsableSnapshot(ctx context.Context, dir *Directory) (bool, error) {
	exists, err := dir.UsersTableExists(ctx)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	count, err := dir.UserCount(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// devSnapshotIsFresh reports whether the cached directory in dev is recent
// enough to reuse without re-fetching from Graph. The string reason is
// surfaced through the log line so operators can see why a startup chose
// to skip — or didn't.
func devSnapshotIsFresh(ctx context.Context, dir *Directory, log interface {
	Warn(msg string, args ...any)
}) (bool, string) {
	exists, err := dir.UsersTableExists(ctx)
	if err != nil {
		log.Warn("Users table check failed; will refresh from Azure", "error", err)
		return false, ""
	}
	if !exists {
		return false, "users table missing"
	}
	count, err := dir.UserCount(ctx)
	if err != nil {
		log.Warn("User count check failed; will refresh from Azure", "error", err)
		return false, ""
	}
	if count == 0 {
		return false, "cache empty"
	}
	last, ok, err := dir.LastSuccessfulSync(ctx, syncKindUsers)
	if err != nil {
		log.Warn("Read last successful sync failed; will refresh from Azure", "error", err)
		return false, ""
	}
	if !ok {
		return false, "no recorded sync timestamp"
	}
	if time.Since(last) > devSyncFreshness {
		return false, fmt.Sprintf("last sync %s ago (over %s window)", time.Since(last).Round(time.Minute), devSyncFreshness)
	}
	return true, fmt.Sprintf("last sync %s ago (within %s window)", time.Since(last).Round(time.Minute), devSyncFreshness)
}
