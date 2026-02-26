// setup:feature:avatar

package graph

import (
	"context"
	"time"

	"catgoose/go-htmx-demo/internals/domain"
	"catgoose/go-htmx-demo/internals/logger"
)

// SyncPhotos downloads profile photos for the given users into store.
// It skips users who already have a cached photo unless force is true.
// Requests are throttled to avoid hitting Graph rate limits.
func SyncPhotos(ctx context.Context, client *Client, store *PhotoStore, users []domain.GraphUser, force bool) error {
	log := logger.Get()
	var downloaded, skipped, noPhoto, errCount int

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

		if !force && store.HasPhoto(u.AzureID) {
			skipped++
			continue
		}

		data, err := client.FetchUserPhoto(ctx, u.AzureID)
		if err != nil {
			log.Error("Failed to fetch photo", "azureID", u.AzureID, "error", err)
			errCount++
			continue
		}
		if data == nil {
			noPhoto++
			continue
		}

		if err := store.Save(u.AzureID, data); err != nil {
			log.Error("Failed to save photo", "azureID", u.AzureID, "error", err)
			errCount++
			continue
		}
		downloaded++

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
