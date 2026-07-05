/* Code Map: Update Notifier — Semver Compare
 * Companion to updates.go. IsNewer is the only consumer-facing
 * function; parseSemver / parseSemverFull / compareSemver are
 * internal helpers.
 *
 * CID Index:
 * CID:updates-005 -> IsNewer
 * CID:updates-005b -> parseSemver / parseSemverFull / compareSemver
 * CID:updates-005c -> IsPreRelease
 *
 * Quick lookup: rg -n "CID:updates-005" internal/updates/semver.go
 */
package updates

import (
	"strconv"
	"strings"
)

// semverParts is the structured form of a version string. The
// "num" array holds [major, minor, patch]. The "hasPre" flag +
// "preNum" hold any "-rcN" prerelease suffix (e.g. 0.2.0-rc5
// has num=[0,2,0], hasPre=true, preNum=5). Versions without
// a prerelease suffix are "stable" — semver §11 says a
// prerelease is strictly LESS than the corresponding stable
// version (0.2.0-rc5 < 0.2.0).
type semverParts struct {
	num    [3]int
	hasPre bool
	preNum int
}

// CID:updates-005 - IsNewer
// Purpose: Semver-style compare. Returns true if r.TagName sorts
// strictly greater than currentVersion.
//
// "v" prefix on either side is stripped. Prerelease suffixes
// (e.g. "0.2.0-rc5") are honoured per semver §11: 0.2.0-rc6 >
// 0.2.0-rc5 > 0.2.0-rc2, and 0.2.0 > 0.2.0-rc6. Non-numeric
// components or unparseable input return false (we never
// claim an update is available from garbage input).
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
	return compareSemver(r.TagName, currentVersion) > 0
}

// CID:updates-005b - compareSemver
// Purpose: full semver-aware compare. Returns -1, 0, or +1.
// Handles prerelease suffixes so 0.2.0-rc6 > 0.2.0-rc5 and
// 0.2.0 > 0.2.0-rc6. Unparseable input returns 0 (we never
// claim an ordering on garbage).
func compareSemver(a, b string) int {
	pa := parseSemverFull(a)
	pb := parseSemverFull(b)
	if pa == nil || pb == nil {
		return 0
	}
	for i := 0; i < 3; i++ {
		if pa.num[i] > pb.num[i] {
			return 1
		}
		if pa.num[i] < pb.num[i] {
			return -1
		}
	}
	// Major/minor/patch equal — compare the prerelease suffix.
	// Per semver §11, a version with a prerelease has LOWER
	// precedence than the same version without one, so:
	//   !hasPre && !hasPre  → equal
	//   !hasPre &&  hasPre  → a wins (stable > prerelease)
	//    hasPre && !hasPre  → b wins
	//    hasPre &&  hasPre  → higher rcNum wins
	if !pa.hasPre && !pb.hasPre {
		return 0
	}
	if !pa.hasPre {
		return 1
	}
	if !pb.hasPre {
		return -1
	}
	if pa.preNum > pb.preNum {
		return 1
	}
	if pa.preNum < pb.preNum {
		return -1
	}
	return 0
}

// CID:updates-005b - parseSemver
// Purpose: backwards-compat shim. Returns just the [major,
// minor, patch] tuple (the prerelease suffix is stripped).
// Used by callers that don't care about the rc number —
// currently nothing, but kept so external callers of the
// internal helper don't break if they were to import it.
// New code should call parseSemverFull or compareSemver.
func parseSemver(s string) *[3]int {
	p := parseSemverFull(s)
	if p == nil {
		return nil
	}
	return &p.num
}

// CID:updates-005b - parseSemverFull
// Purpose: Extract [major, minor, patch] and any "rcN"
// prerelease suffix from a version string. Returns nil on
// unparseable input.
//
// Accepted forms: "1.2.3", "v1.2", "1.2.3-rc1", "v0.2.0-rc5",
// "0.2.0.4" (4th component ignored). The prerelease suffix
// must look like "rc<digits>" — "alpha", "beta", etc. are
// not recognised; the version parses as stable in that case.
func parseSemverFull(s string) *semverParts {
	v := strings.TrimPrefix(s, "v")
	parts := strings.SplitN(v, ".", 4)
	if len(parts) < 1 {
		return nil
	}
	var p semverParts
	for i := 0; i < 3; i++ {
		if i >= len(parts) {
			break // missing components default to 0
		}
		// strip prerelease from the patch component:
		// "0-rc1" → "0"
		head := parts[i]
		if idx := strings.IndexAny(head, "-+"); idx >= 0 {
			head = head[:idx]
		}
		n, err := strconv.Atoi(head)
		if err != nil {
			return nil
		}
		p.num[i] = n
	}
	// Prerelease extraction. We look for "rc<digits>" anywhere
	// in the version (after the major). Forms accepted:
	//   "0.2.0-rc1"        → preNum=1
	//   "v0.2.0-rc5"       → preNum=5
	//   "0.2.0"            → no prerelease
	//   "0.2.0-alpha"      → no prerelease (alpha is not a
	//                         recognised pre-id; we leave
	//                         hasPre=false and the version
	//                         sorts as stable 0.2.0)
	if idx := strings.Index(v, "-"); idx >= 0 {
		rcStr := v[idx+1:]
		if strings.HasPrefix(rcStr, "rc") {
			rcStr = rcStr[2:]
			if n, err := strconv.Atoi(rcStr); err == nil {
				p.hasPre = true
				p.preNum = n
			}
		}
	}
	return &p
}

// CID:updates-005b - IsPreRelease
// Purpose: returns true when the version is pre-1.0 — either
// because the major is 0 (e.g. "0.2.0", "0.2.0-rc1") or the
// version itself carries a pre-release suffix (e.g. "1.0.0-rc1").
// Used by LatestSuitableRelease to decide whether the user
// should be offered prereleases as "updates":
//   - pre-1.0 user → prereleases OK (rc1 → rc6)
//   - stable user → prereleases skipped (1.0.0 → 1.1.0, skip 1.1.0-rc1)
//
// "dev", empty, or unparseable input returns false (we treat
// "dev" as not-pre-1.0 because it's a local build, not a release
// tag; the caller never reaches this function for "dev" anyway
// — LatestSuitableRelease short-circuits earlier).
func IsPreRelease(version string) bool {
	v := strings.TrimPrefix(version, "v")
	if v == "" || v == "dev" {
		return false
	}
	// Pre-release suffix is the most obvious signal.
	if strings.ContainsAny(v, "-+") {
		return true
	}
	// Pre-1.0 major counts as pre-release per semver convention.
	parts := strings.SplitN(v, ".", 4)
	if len(parts) == 0 {
		return false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	return major == 0
}
