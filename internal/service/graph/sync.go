// setup:feature:graph

package graph

import (
	"context"
	"fmt"
	"time"

	"catgoose/dothog/internal/domain"
	"catgoose/dothog/internal/logger"
	"catgoose/dothog/internal/shared"

	"catgoose/dothog/internal/env"
)

// SyncType labels a user-cache refresh for telemetry; the constant value
// is the human-readable phrase logged alongside each sync.
type SyncType string

// Sync trigger labels written into logs and admin telemetry.
const (
	SyncTypeInitial   SyncType = "initial sync"
	SyncTypeManual    SyncType = "manual sync"
	SyncTypeScheduled SyncType = "scheduled sync"
	SyncTypePeriodic  SyncType = "periodic sync"
)

// InitAndSyncUserCache primes the Azure user cache; dev fetches only when empty,
// prod performs an initial sync then schedules a daily refresh at refreshHour.
func InitAndSyncUserCache(
	ctx context.Context,
	userCache *UserCache,
	refreshHour int,
	fetchUsersFunc func() ([]domain.GraphUser, error),
	afterSync func(ctx context.Context, users []domain.GraphUser),
) error {
	ctx = shared.WithContextIDAndDescription(ctx, shared.GenerateContextID(), "user cache init")
	log := logger.WithContext(ctx)
	isDev := env.Dev()

	if isDev {
		// In dev, only fetch when the cache is missing or empty to avoid hammering Azure on every restart.
		exists, err := userCache.UsersTableExists()
		if err != nil {
			log.Warn("Users table check failed, will fetch users from Azure", "error", err)
		} else if !exists {
			log.Info("Users table does not exist, fetching users from Azure")
		} else {
			userCount, err := userCache.GetUserCount()
			if err != nil {
				log.Warn("Failed to get user count, will fetch users from Azure", "error", err)
			} else if userCount == 0 {
				log.Info("Users table is empty, fetching users from Azure")
			} else {
				log.Info("Using existing user cache in development environment", "user_count", userCount)
				return nil
			}
		}
		log.Info("Development mode: fetching users from Azure")
		users, err := fetchUsersFunc()
		if err != nil {
			return fmt.Errorf("failed to fetch users from Azure: %w", err)
		}
		log.Info("Successfully fetched users from Azure", "count", len(users))
		err = userCache.InsertOrUpdateUsers(ctx, users)
		if err != nil {
			return fmt.Errorf("failed to insert users into cache: %w", err)
		}
		log.Info("Successfully synced users to cache")
		if afterSync != nil {
			afterSync(ctx, users)
		}
		return nil
	}

	log.Info("Production mode: performing initial user sync")
	users, err := fetchUsersFunc()
	if err != nil {
		return fmt.Errorf("failed to fetch users during initial sync: %w", err)
	}

	if err := userCache.InsertOrUpdateUsers(ctx, users); err != nil {
		return fmt.Errorf("failed to sync users during initial sync: %w", err)
	}
	log.Info("Successfully completed initial sync", "user_count", len(users))
	if afterSync != nil {
		afterSync(ctx, users)
	}

	doSync := func(syncType SyncType) {
		syncCtx := shared.WithContextIDAndDescription(ctx, shared.GenerateContextID(), string(syncType))
		syncLog := logger.WithContext(syncCtx)
		syncLog.Info("Starting user cache refresh", "type", syncType)
		users, err := fetchUsersFunc()
		if err != nil {
			syncLog.Error("Failed to fetch users during sync", "type", syncType, "error", err)
			return
		}
		if err := userCache.InsertOrUpdateUsers(syncCtx, users); err != nil {
			syncLog.Error("Failed to sync users during sync", "type", syncType, "error", err)
			return
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
			logger.WithContext(ctx).Info("Scheduled user cache refresh", "next_refresh", next, "wait_duration", time.Until(next))
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
