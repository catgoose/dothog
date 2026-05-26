// setup:feature:avatar

package graph

import (
	"context"
	"time"

	"catgoose/dothog/internal/logger"
)

// photoCheckFreshness bounds how often photo_sync re-queries Graph for a
// given user. Users with recent LastCheckedAt are skipped regardless of
// whether the cached row has bytes or is a checked-and-missing marker — so
// repeated startups don't hammer Graph for users with no photo.
const photoCheckFreshness = 24 * time.Hour

// SyncPhotos downloads profile photos for the given users into cache. Skips
// users whose LastCheckedAt is within photoCheckFreshness unless force is
// true. Graph "no photo" responses persist a missing-marker row via
// MarkMissing so the next sync within the window can skip the user without
// re-querying. Requests are throttled to avoid hitting Graph rate limits.
func SyncPhotos(ctx context.Context, client *Client, cache *PhotoCache, users []User, force bool) error {
	log := logger.WithContext(ctx)
	var downloaded, skipped, noPhoto, errCount int

	now := time.Now()
	for _, u := range users {
		select {
		case <-ctx.Done():
			log.Info("Photo sync cancelled", "downloaded", downloaded, "skipped", skipped, "noPhoto", noPhoto, "errors", errCount)
			return ctx.Err()
		default:
		}

		if u.AzureID == "" {
			continue
		}

		if !force {
			if last, found, err := cache.LastChecked(ctx, u.AzureID); err == nil && found && now.Sub(last) < photoCheckFreshness {
				skipped++
				continue
			}
		}

		data, err := client.FetchUserPhoto(ctx, u.AzureID)
		if err != nil {
			log.Error("Failed to fetch photo", "azureID", u.AzureID, "error", err)
			errCount++
			continue
		}
		if len(data) == 0 {
			if err := cache.MarkMissing(ctx, u.AzureID); err != nil {
				log.Error("Failed to record missing photo", "azureID", u.AzureID, "error", err)
				errCount++
				continue
			}
			noPhoto++
		} else if err := cache.Save(ctx, u.AzureID, "image/jpeg", data); err != nil {
			log.Error("Failed to save photo", "azureID", u.AzureID, "error", err)
			errCount++
			continue
		} else {
			downloaded++
		}

		// Throttle between requests
		select {
		case <-ctx.Done():
			log.Info("Photo sync cancelled", "downloaded", downloaded, "skipped", skipped, "noPhoto", noPhoto, "errors", errCount)
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	log.Info("Photo sync complete", "downloaded", downloaded, "skipped", skipped, "noPhoto", noPhoto, "errors", errCount)
	return nil
}
