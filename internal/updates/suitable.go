/* Code Map: Update Notifier — Suitable Release Picker
 * - LatestSuitableRelease: returns the highest-semver release that
 *   is a valid update candidate for the user's current version.
 *   GitHub's /releases/latest endpoint excludes prereleases, so
 *   this function hits /releases and applies the rule:
 *
 *     - If the current version is pre-1.0 (IsPreRelease), any
 *       release that is strictly higher semver is a candidate —
 *       prereleases included.
 *     - If the current version is stable (>= 1.0.0), only stable
 *       releases are candidates. A user on v1.0.0 is never told
 *       "you should upgrade to v1.1.0-rc1".
 *
 *   Returns ErrNoRelease when nothing in the response is a valid
 *   candidate (caller logs "up to date"). Per IMPL §7 the request
 *   carries the same 5-second timeout as LatestRelease.
 *
 * This exists because rc1-hotpatch-20 hit a real bug: every RC
 * since rc2 was published via `gh release create --prerelease`,
 * so the auto-updater's "releases/latest" endpoint returned rc1
 * forever — users running rc1 saw "up to date" no matter what
 * the user published. The fix is in the updater, not in the
 * release tagging convention.
 *
 * CID Index:
 * CID:updates-006 -> LatestSuitableRelease
 *
 * Quick lookup: rg -n "CID:updates-006" internal/updates/
 */
package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// CID:updates-006 - LatestSuitableRelease
// Purpose: fetch the GitHub /releases endpoint (returns up to 30
// most recent releases, including prereleases) and return the
// best update candidate for the running version.
//
// Rule summary:
//   - current version "dev" / "" / unparseable → return ErrNoRelease.
//     We never auto-upgrade a dev build.
//   - current is pre-1.0 → highest IsNewer release wins
//     (prereleases included).
//   - current is stable (>= 1.0.0) → highest IsNewer stable
//     release wins (prereleases skipped).
//   - no candidate → return (nil, ErrNoRelease) so the caller
//     can log "up to date" the same way it does today.
//
// Special cases:
//   - HTTP 404 → (nil, ErrNoRelease). The repo exists but has
//     no published releases yet.
//   - any other non-2xx → typed error mentioning the status.
//   - network / parse error → returned as-is; caller logs and
//     moves on (no tray notification in that path).
//
// baseURL override is provided for tests via VOCES_GITHUB_API_BASE;
// releasesURL() reads it.
func LatestSuitableRelease(ctx context.Context, currentVersion string) (*Release, error) {
	if currentVersion == "" || currentVersion == "dev" {
		// "dev" builds are unreleased; never claim an update is
		// available. Returning ErrNoRelease keeps the tray silent
		// and lets checkForUpdates log a normal "up to date".
		return nil, ErrNoRelease
	}
	url := releasesURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("updates: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: defaultRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("updates: GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through
	case http.StatusNotFound:
		return nil, ErrNoRelease
	default:
		prefix, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("updates: GET %s: %s: %s",
			url, resp.Status, strings.TrimSpace(string(prefix)))
	}
	_ = ctx // reserved for future use

	var all []Release
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil, fmt.Errorf("updates: parse %s: %w", url, err)
	}
	if len(all) == 0 {
		// Empty list (or 200 with no body). Treat the same as 404
		// so the caller logs "up to date" instead of crashing.
		return nil, ErrNoRelease
	}

	currentIsPrerelease := IsPreRelease(currentVersion)
	var best *Release
	for i := range all {
		r := &all[i]
		if !r.IsNewer(currentVersion) {
			continue
		}
		// Stable users never get offered a prerelease.
		if !currentIsPrerelease && r.Prerelease {
			continue
		}
		if best == nil || higherSemver(r.TagName, best.TagName) {
			best = r
		}
	}
	if best == nil {
		return nil, ErrNoRelease
	}
	return best, nil
}

// releasesURL returns the absolute GitHub API URL for the list-
// releases endpoint. Per_page is bumped to 30 (the default is
// also 30 but pinning it makes the request deterministic for
// the test mock).
func releasesURL() string {
	owner, repo := OwnerRepo()
	return fmt.Sprintf("%s/repos/%s/%s/releases?per_page=30", apiBase(), owner, repo)
}

// higherSemver reports whether a sorts strictly after b. Used
// to pick the best candidate inside LatestSuitableRelease.
// Delegates to compareSemver (which handles prerelease suffixes
// correctly: 0.2.0-rc6 > 0.2.0-rc5 > 0.2.0-rc2, and
// 0.2.0 > 0.2.0-rc6). Equal or unparseable returns false.
func higherSemver(a, b string) bool {
	return compareSemver(a, b) > 0
}
