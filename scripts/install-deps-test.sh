#!/usr/bin/env bash
# install-deps-test.sh — smoke tests for the wait_for_apt_lock helper
# in install-deps.sh.
#
# Verifies the three behaviours the helper promises:
#   1. Returns 0 immediately when no lock is held.
#   2. Returns 1 after the timeout when the lock is held indefinitely.
#   3. Returns 0 within the timeout when the holder releases.
#
# We extract the helper from install-deps.sh with sed and source it
# into a fresh bash so the test runs in isolation. The test stubs
# SUDO=() (root) and uses a temp lock file path so we don't need to
# touch /var/lib/apt or /var/lib/dpkg on the build host.
#
# Run: bash scripts/install-deps-test.sh

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DEPS="$SCRIPT_DIR/install-deps.sh"

if [[ ! -f "$INSTALL_DEPS" ]]; then
    echo "❌ install-deps.sh not found next to this test." >&2
    exit 1
fi

# Extract the wait_for_apt_lock function from install-deps.sh. The
# helper is the only function between "wait_for_apt_lock() {" and
# the next blank-line-then-"#" comment block, so a sed range works.
HELPER_SRC="$(sed -n '/^wait_for_apt_lock() {/,/^}$/p' "$INSTALL_DEPS")"
if [[ -z "$HELPER_SRC" ]]; then
    echo "❌ wait_for_apt_lock not found in install-deps.sh." >&2
    exit 1
fi
eval "$HELPER_SRC"

# Test harness. SUDO=() because we run as the current user and
# don't need privileged fuser — the lock holder in the test is
# the same user.
SUDO=()
PASS=0
FAIL=0

assert() {
    local name="$1"; shift
    if "$@"; then
        echo "  PASS  $name"
        PASS=$((PASS+1))
    else
        echo "  FAIL  $name"
        FAIL=$((FAIL+1))
    fi
}

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"; kill $HOLDER_PID 2>/dev/null || true' EXIT

# 1. No lock held -> returns 0 immediately
LOCK="$TMPDIR/lock1"
rm -f "$LOCK"
START=$(date +%s)
if wait_for_apt_lock "test" "$LOCK"; then
    END=$(date +%s)
    ELAPSED=$((END - START))
    if [[ $ELAPSED -le 1 ]]; then
        echo "  PASS  no lock -> exit 0 in ${ELAPSED}s"
        PASS=$((PASS+1))
    else
        echo "  FAIL  no lock -> exit 0 but elapsed=${ELAPSED}s (expected <1s)"
        FAIL=$((FAIL+1))
    fi
else
    echo "  FAIL  no lock -> expected 0, got 1"
    FAIL=$((FAIL+1))
fi

# 2. Lock held indefinitely -> returns 1 after timeout
LOCK="$TMPDIR/lock2"
APT_LOCK_TIMEOUT=3
(
    flock 9
    sleep 10
) 9>"$LOCK" &
HOLDER_PID=$!
sleep 0.5   # let the holder grab the lock
START=$(date +%s)
if wait_for_apt_lock "test" "$LOCK"; then
    echo "  FAIL  held lock -> expected timeout, got 0"
    FAIL=$((FAIL+1))
else
    END=$(date +%s)
    ELAPSED=$((END - START))
    if [[ $ELAPSED -ge 2 && $ELAPSED -le 5 ]]; then
        echo "  PASS  held lock -> exit 1 after ${ELAPSED}s (timeout 3s)"
        PASS=$((PASS+1))
    else
        echo "  FAIL  held lock -> exit 1 but elapsed=${ELAPSED}s (expected ~3s)"
        FAIL=$((FAIL+1))
    fi
fi
kill $HOLDER_PID 2>/dev/null || true
wait $HOLDER_PID 2>/dev/null || true
rm -f "$LOCK"
APT_LOCK_TIMEOUT=""

# 3. Lock held briefly then released -> returns 0 within timeout
LOCK="$TMPDIR/lock3"
APT_LOCK_TIMEOUT=5
(
    flock 9
    sleep 1
) 9>"$LOCK" &
HOLDER_PID=$!
sleep 0.3
START=$(date +%s)
if wait_for_apt_lock "test" "$LOCK"; then
    END=$(date +%s)
    ELAPSED=$((END - START))
    if [[ $ELAPSED -ge 1 && $ELAPSED -le 4 ]]; then
        echo "  PASS  short lock -> exit 0 after ${ELAPSED}s"
        PASS=$((PASS+1))
    else
        echo "  FAIL  short lock -> exit 0 but elapsed=${ELAPSED}s (expected 1-3s)"
        FAIL=$((FAIL+1))
    fi
else
    echo "  FAIL  short lock -> expected 0, got 1"
    FAIL=$((FAIL+1))
fi
wait $HOLDER_PID 2>/dev/null || true
rm -f "$LOCK"

# 4. Multiple lock files: any held => wait, all free => return 0
LOCK_A="$TMPDIR/lock4a"
LOCK_B="$TMPDIR/lock4b"
(
    flock 9
    sleep 0.8
) 9>"$LOCK_A" &
HOLDER_A=$!
sleep 0.3
START=$(date +%s)
if wait_for_apt_lock "test" "$LOCK_A" "$LOCK_B"; then
    END=$(date +%s)
    ELAPSED=$((END - START))
    if [[ $ELAPSED -ge 1 && $ELAPSED -le 3 ]]; then
        echo "  PASS  multi-lock (A held, B free) -> exit 0 after ${ELAPSED}s"
        PASS=$((PASS+1))
    else
        echo "  FAIL  multi-lock -> elapsed=${ELAPSED}s (expected 1-3s)"
        FAIL=$((FAIL+1))
    fi
else
    echo "  FAIL  multi-lock -> expected 0, got 1"
    FAIL=$((FAIL+1))
fi
wait $HOLDER_A 2>/dev/null || true
rm -f "$LOCK_A" "$LOCK_B"

echo ""
echo "Results: $PASS pass, $FAIL fail"
exit $FAIL
