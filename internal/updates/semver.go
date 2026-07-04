/* Code Map: Update Notifier — Semver Compare
 * Companion to updates.go. IsNewer is the only consumer-facing
 * function; parseSemver is internal.
 *
 * CID Index:
 * CID:updates-005 -> IsNewer
 * CID:updates-005b -> parseSemver
 *
 * Quick lookup: rg -n "CID:updates-005" internal/updates/semver.go
 */
package updates

import (
	"strconv"
	"strings"
)

// CID:updates-005 - IsNewer
// Purpose: Semver-style compare. Returns true if r.TagName sorts
// strictly greater than currentVersion.
//
// "v" prefix on either side is stripped. Non-numeric components are
// skipped. Missing components are treated as 0. If either side is
// empty or unparseable, returns false (we never claim an update is
// available from garbage input).
//
// The "dev" build (Version constant in main.go) returns false — a
// development build should not be told to upgrade itself over the
// network.
func (r *Release) IsNewer(currentVersion string) bool {
	if r == nil || r.TagName == "" {
		return false
	}
	if currentVersion == "" || currentVersion == "dev" {
		// "dev" builds are unreleased; a published tag is "newer" by
		// definition, but we never auto-upgrade a dev build. Return
		// false so the tray does not surface a misleading badge.
		return false
	}
	cur := parseSemver(strings.TrimPrefix(currentVersion, "v"))
	rel := parseSemver(strings.TrimPrefix(r.TagName, "v"))
	if cur == nil || rel == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		if rel[i] > cur[i] {
			return true
		}
		if rel[i] < cur[i] {
			return false
		}
	}
	return false // equal versions
}

// CID:updates-005b - parseSemver
// Purpose: Extract [major, minor, patch] from "1.2.3", "v1.2",
// "1.2.3-rc1", or "1.2.3.4". Returns nil on unparseable input. Any
// non-digit prefix is stripped from each component.
func parseSemver(s string) *[3]int {
	out := [3]int{}
	parts := strings.SplitN(s, ".", 4)
	for i := 0; i < 3 && i < len(parts); i++ {
		// strip pre-release suffix: "1-rc1" → "1"
		head := parts[i]
		if idx := strings.IndexAny(head, "-+"); idx >= 0 {
			head = head[:idx]
		}
		n, err := strconv.Atoi(head)
		if err != nil {
			return nil
		}
		out[i] = n
	}
	return &out
}
