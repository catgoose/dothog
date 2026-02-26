// setup:feature:avatar

package graph

import (
	"context"
	"errors"
	"fmt"

	odataerrors "github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
)

// FetchUserPhoto downloads the profile photo for the given Azure user ID.
// Returns nil, nil when the user has no photo (404).
func (c *Client) FetchUserPhoto(ctx context.Context, azureID string) ([]byte, error) {
	data, err := c.client.Users().ByUserId(azureID).Photo().Content().Get(ctx, nil)
	if err != nil {
		if isNotFoundError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch photo for %s: %w", azureID, err)
	}
	return data, nil
}

// isNotFoundError checks whether a Graph SDK error is a 404.
func isNotFoundError(err error) bool {
	var odataErr *odataerrors.ODataError
	if errors.As(err, &odataErr) {
		if odataErr.GetStatusCode() == 404 {
			return true
		}
	}
	return false
}
