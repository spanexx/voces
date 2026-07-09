#!/usr/bin/env bash
# install-test.sh — unit tests for the channel-picker in install.sh.
#
# The picker decides which release tag to install based on either:
#   - VOCES_VERSION (escape hatch — always wins)
#   - the currently installed /opt/voces/voces version
#   - "fresh install" → default to the latest stable
#
# The function under test is `pick_latest_tag`, which is a pure function
# over (installed_version, list_of_tags) → chosen_tag. We extract it from
# install.sh with sed (the same pattern install-deps-test.sh uses for
# wait_for_apt_lock) and exercise it without touching the network, the
# filesystem, or the package manager.
#
# Run: bash scripts/install-test.sh
#
# TDD cases:
#   1. Fresh install + stable available        → picks stable
#   2. Fresh install + only prereleases         → picks highest prerelease of highest base
#   3. Installed stable, newer stable exists    → upgrades to newer stable
#   4. Installed stable, only prereleases of same base  → stays on stable (don't downgrade)
#   5. Installed rc, newer rc of same base      → upgrades to newer rc
#   6. Installed rc, no newer rc of same base   → stays on installed (don't auto-promote)
#   7. Installed rc, multiple bases             → stays on the base the user is on
#   8. Empty tag list                           → returns empty (caller handles)
#   9. Tags unsorted (random order)             → still picks the right one (sort -V)

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_SH="$SCRIPT_DIR/../install.sh"

if [[ ! -f "$INSTALL_SH" ]]; then
    echo "❌ install.sh not found next to this test (expected $INSTALL_SH)." >&2
    exit 1
fi

# Extract the pick_latest_tag function from install.sh. The function is
# the only block between "pick_latest_tag() {" and the next blank-line-
# then-"#" comment block or EOF.
HELPER_SRC="$(sed -n '/^pick_latest_tag() {/,/^}$/p' "$INSTALL_SH")"
if [[ -z "$HELPER_SRC" ]]; then
    echo "❌ pick_latest_tag not found in install.sh." >&2
    echo "   Add a pick_latest_tag() { ... } function and re-run." >&2
    exit 1
fi
eval "$HELPER_SRC"

PASS=0
FAIL=0

assert_picks() {
    local name="$1"
    local installed="$2"
    local tags="$3"
    local want="$4"
    local got
    got="$(pick_latest_tag "$installed" "$tags")"
    if [ "$got" = "$want" ]; then
        echo "  PASS  $name"
        PASS=$((PASS+1))
    else
        echo "  FAIL  $name"
        echo "        installed=$installed"
        echo "        tags=$(printf '%s ' $tags)"
        echo "        want=$want"
        echo "        got =$got"
        FAIL=$((FAIL+1))
    fi
}

# 1. Fresh install + stable available → picks stable
assert_picks \
    "fresh install picks stable" \
    "" \
    "v0.2.0
v0.2.0-rc1
v0.2.0-rc12
v0.2.0-rc13" \
    "v0.2.0"

# 2. Fresh install + only prereleases → highest prerelease of highest base
assert_picks \
    "fresh install with no stable picks highest rc" \
    "" \
    "v0.2.0-rc1
v0.2.0-rc13
v0.1.0-rc5" \
    "v0.2.0-rc13"

# 3. Installed stable + newer stable → upgrade
assert_picks \
    "installed stable upgrades to newer stable" \
    "v0.2.0" \
    "v0.2.0
v0.2.1
v0.2.1-rc1" \
    "v0.2.1"

# 4. Installed stable + only prereleases of same base → stay stable
assert_picks \
    "installed stable doesn't downgrade to prerelease" \
    "v0.2.0" \
    "v0.2.0
v0.2.0-rc13
v0.2.0-rc14" \
    "v0.2.0"

# 5. Installed rc + newer rc of same base → upgrade
assert_picks \
    "installed rc upgrades to newer rc of same base" \
    "v0.2.0-rc12" \
    "v0.2.0
v0.2.0-rc12
v0.2.0-rc13" \
    "v0.2.0-rc13"

# 6. Installed rc + no newer rc of same base → stay (don't auto-promote)
assert_picks \
    "installed rc stays when no newer rc" \
    "v0.2.0-rc12" \
    "v0.2.0
v0.2.0-rc12
v0.2.0-rc11" \
    "v0.2.0-rc12"

# 7. Installed rc + multiple bases → stay on the user's base
assert_picks \
    "installed rc of 0.1.0 stays on 0.1.0 even with 0.2.0 rcs" \
    "v0.1.0-rc5" \
    "v0.1.0
v0.1.0-rc5
v0.2.0
v0.2.0-rc1
v0.2.0-rc13" \
    "v0.1.0-rc5"

# 8. Empty tag list → empty
assert_picks \
    "empty tag list returns empty" \
    "v0.2.0-rc12" \
    "" \
    ""

# 9. Tags unsorted (random order) → still picks the right one
assert_picks \
    "unsorted tags still picks highest rc" \
    "v0.2.0-rc5" \
    "v0.2.0-rc13
v0.2.0
v0.2.0-rc12
v0.2.0-rc1" \
    "v0.2.0-rc13"

# 10. Realistic current state: v0.2.0-rc13 is what's live, plus stable
#     v0.2.0. New user on rc12 re-runs the installer. Should land on rc13.
assert_picks \
    "rc12 -> rc13 with v0.2.0 stable present" \
    "v0.2.0-rc12" \
    "v0.2.0
v0.2.0-rc12
v0.2.0-rc13" \
    "v0.2.0-rc13"

echo ""
if [ "$FAIL" -eq 0 ]; then
    echo "Results: $PASS pass, $FAIL fail"
    exit 0
else
    echo "Results: $PASS pass, $FAIL fail"
    exit 1
fi
