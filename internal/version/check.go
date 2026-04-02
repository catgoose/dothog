// setup:feature:demo

package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// UpdateInfo holds the result of a GitHub release check.
type UpdateInfo struct {
	Latest     string // e.g. "v0.0.30"
	Current    string
	ReleaseURL string // HTML URL to the release page
	NewerAvail bool
}

const releaseAPI = "https://api.github.com/repos/catgoose/dothog/releases/latest"

// CheckLatest queries the GitHub API for the latest release and compares it
// against the running version.
func CheckLatest(ctx context.Context) (UpdateInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseAPI, nil)
	if err != nil {
		return UpdateInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return UpdateInfo{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return UpdateInfo{}, fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return UpdateInfo{}, err
	}

	info := UpdateInfo{
		Latest:     release.TagName,
		Current:    Version,
		ReleaseURL: release.HTMLURL,
		NewerAvail: IsNewer(release.TagName, Version),
	}
	return info, nil
}

// IsNewer reports whether latest is a higher semver than current.
// Both values may optionally have a "v" prefix.
func IsNewer(latest, current string) bool {
	latParts := parseSemver(latest)
	curParts := parseSemver(current)
	if latParts == nil || curParts == nil {
		return false
	}
	for i := range 3 {
		if latParts[i] > curParts[i] {
			return true
		}
		if latParts[i] < curParts[i] {
			return false
		}
	}
	return false
}

// parseSemver extracts [major, minor, patch] from a version string like "v1.2.3".
func parseSemver(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return nil
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		nums[i] = n
	}
	return nums
}
