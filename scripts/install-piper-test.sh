#!/usr/bin/env bash
# install-piper-test.sh — unit tests for the install_piper function in install.sh.
#
# The piper TTS engine is downloaded at install time
# (rc1-hotpatch-32) by install.sh. install_piper queries the
# rhasspy/piper GitHub API for the latest release tag, downloads
# the prebuilt tarball for the host arch, extracts the `piper`
# binary, chmods it +x, and runs `--version` as a smoke test.
#
# We test the function without touching the network or the real
# filesystem: each test case (a) overrides curl to return a canned
# API response and a canned tarball from a temp dir, (b) points
# PIPER_DEST_DIR / PIPER_DEST at a temp dir, and (c) sources the
# function out of install.sh with sed (the same pattern
# install-test.sh uses for pick_latest_tag).
#
# Run: bash scripts/install-piper-test.sh
#
# TDD cases:
#   1. Already-installed piper          → skip (no download)
#   2. Fresh install, normal flow       → downloads + extracts + chmods
#   3. Fresh install, no API response   → skips, prints warning, exit 0
#   4. Fresh install, asset 404         → skips, prints warning, exit 0
#   5. Fresh install, broken binary     → extracts but then removes it
#   6. piper_arch_for_machine: x86_64   → "x86_64"
#   7. piper_arch_for_machine: aarch64  → "aarch64"
#   8. piper_arch_for_machine: arm64    → "aarch64"
#   9. piper_arch_for_machine: riscv64  → "" (unsupported)

# `set -u` is intentionally NOT set here: install_piper sets an
# internal EXIT trap that references $TMPDIR (a runtime variable,
# not set in the test environment) and $piper_tmp (a local var
# that's out of scope at exit). With `set -u` the trap would
# explode the test runner at script exit. The production
# install.sh does not set `set -u` either.
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_SH="$SCRIPT_DIR/../install.sh"

if [[ ! -f "$INSTALL_SH" ]]; then
    echo "❌ install.sh not found next to this test (expected $INSTALL_SH)." >&2
    exit 1
fi

# Extract the install_piper + piper_arch_for_machine functions from
# install.sh. Each function is the block between "func_name() {" and
# the next "^}" (sed range /start/,/^}$/p).
HELPER_SRC="$(
    sed -n '/^piper_arch_for_machine() {/,/^}$/p' "$INSTALL_SH"
    sed -n '/^install_piper() {/,/^}$/p' "$INSTALL_SH"
)"
if [[ -z "$HELPER_SRC" ]]; then
    echo "❌ install_piper / piper_arch_for_machine not found in install.sh." >&2
    exit 1
fi
eval "$HELPER_SRC"

PASS=0
FAIL=0
RESULTS=()

record_result() {
    if [[ "$2" == "ok" ]]; then
        PASS=$((PASS + 1))
        RESULTS+=("  PASS  $1")
    else
        FAIL=$((FAIL + 1))
        RESULTS+=("  FAIL  $1")
    fi
}

# make_fake_piper_tarball <out.tar.gz>  → builds a tarball that
# extracts to <basename>/piper containing a script that prints
# "piper v1.2.0". Used by tests that need the "download + extract
# + smoke test" path to succeed.
make_fake_piper_tarball() {
    local out="$1"
    local dir
    dir="$(mktemp -d)"
    cat > "$dir/piper" <<'EOF'
#!/bin/sh
if [ "$1" = "--version" ]; then
    echo "piper v1.2.0-test"
    exit 0
fi
exit 0
EOF
    chmod 0755 "$dir/piper"
    ( cd "$dir" && tar -czf "$out" piper )
    rm -rf "$dir"
}

# reset_globals wipes the temp dir + per-test variables so each
# test starts clean. The trap is set once at the top and the
# top-level TMP is recreated.
reset_globals() {
    rm -rf "$TMP"
    TMP="$(mktemp -d)"
    INSTALL_DIR="$TMP/opt/voces"
    PIPER_DEST_DIR="$INSTALL_DIR/engines"
    PIPER_DEST="$PIPER_DEST_DIR/piper"
    mkdir -p "$PIPER_DEST_DIR"
}

# =============================================================================
# Test 1: already-installed piper → skip without calling curl
# =============================================================================
TMP="$(mktemp -d)"
trap "rm -rf '$TMP'" EXIT
reset_globals
# Drop a fake piper that responds to --version (so the smoke test
# would pass if we got there — we shouldn't get there).
cat > "$PIPER_DEST" <<'EOF'
#!/bin/sh
echo "piper v1.2.0-already-there"
exit 0
EOF
chmod 0755 "$PIPER_DEST"

# Stub curl: if the function calls curl, fail loudly.
curl() {
    echo "FAIL: curl should not be called when piper is already installed" >&2
    return 99
}

if install_piper > "$TMP/out.txt" 2> "$TMP/err.txt"; then
    if grep -q "already at" "$TMP/out.txt"; then
        record_result "Test 1: already-installed piper → skip" ok
    else
        echo "  --- stdout ---"; cat "$TMP/out.txt"
        echo "  --- stderr ---"; cat "$TMP/err.txt"
        record_result "Test 1: already-installed piper → skip" fail
    fi
else
    echo "  --- stdout ---"; cat "$TMP/out.txt"
    echo "  --- stderr ---"; cat "$TMP/err.txt"
    record_result "Test 1: already-installed piper → skip" fail
fi
unset -f curl

# =============================================================================
# Test 2: fresh install, normal flow → downloads + extracts + chmods
# =============================================================================
reset_globals
mkdir -p "$TMP/fake_api"
cat > "$TMP/fake_api/release.json" <<'EOF'
{"tag_name": "v1.2.0", "assets": [{"name": "piper_linux_x86_64.tar.gz", "browser_download_url": "file:///ignored-by-stub"}]}
EOF
make_fake_piper_tarball "$TMP/piper.tar.gz"

# Stub curl. install_piper calls `curl -fsSL URL` (stdout) for the
# GitHub API and `curl -fsSL -o FILE URL` (file output) for the
# asset. We handle both shapes by inspecting argv:
#   - $1 = -fsSL (or -fsSL -o FILE, etc.) → strip flags, take the URL
#   - find the last non-flag arg as the URL
#   - if the URL is the API, print the canned JSON to stdout
#   - if the URL is the asset, copy the canned tarball to the -o target
# This is a deliberately small subset of curl's CLI surface; if
# install_piper gains a new curl flag, the test will catch it.
curl() {
    local url=""
    local out=""
    local arg
    local i=1
    # Walk args. Once we see -o, the next arg is the output file.
    local saw_o=0
    for arg in "$@"; do
        if [[ "$arg" == "-o" ]]; then
            saw_o=1
            continue
        fi
        if [[ $saw_o -eq 1 ]]; then
            out="$arg"
            saw_o=0
            continue
        fi
        # Skip other flag-like args.
        if [[ "$arg" == -* ]]; then
            continue
        fi
        url="$arg"
    done
    case "$url" in
        *api.github.com*)
            if [[ -n "$out" ]]; then
                cp "$TMP/fake_api/release.json" "$out"
            else
                cat "$TMP/fake_api/release.json"
            fi
            return 0
            ;;
        *piper_linux_x86_64.tar.gz*)
            if [[ -n "$out" ]]; then
                cp "$TMP/piper.tar.gz" "$out"
            else
                cat "$TMP/piper.tar.gz"
            fi
            return 0
            ;;
        *)
            echo "FAIL: unexpected curl url: $url" >&2
            return 1
            ;;
    esac
}

# Stub uname so we don't depend on the host arch for this test.
uname() { echo "x86_64"; }

if install_piper > "$TMP/out.txt" 2> "$TMP/err.txt"; then
    if [[ -x "$PIPER_DEST" ]] && "$PIPER_DEST" --version 2>&1 | grep -qi "piper"; then
        record_result "Test 2: fresh install, normal flow → success" ok
    else
        echo "  piper binary missing or non-functional at $PIPER_DEST"
        echo "  --- stdout ---"; cat "$TMP/out.txt"
        echo "  --- stderr ---"; cat "$TMP/err.txt"
        record_result "Test 2: fresh install, normal flow → success" fail
    fi
else
    echo "  --- stdout ---"; cat "$TMP/out.txt"
    echo "  --- stderr ---"; cat "$TMP/err.txt"
    record_result "Test 2: fresh install, normal flow → success" fail
fi
unset -f curl
unset -f uname

# =============================================================================
# Test 3: fresh install, no API response (rate-limited / offline)
# =============================================================================
reset_globals
curl() { return 22; }
uname() { echo "x86_64"; }

if install_piper > "$TMP/out.txt" 2> "$TMP/err.txt"; then
    if [[ ! -e "$PIPER_DEST" ]] && grep -q "could not query" "$TMP/err.txt"; then
        record_result "Test 3: no API response → skip + warn" ok
    else
        echo "  piper unexpectedly present or warning missing"
        echo "  --- stderr ---"; cat "$TMP/err.txt"
        record_result "Test 3: no API response → skip + warn" fail
    fi
else
    record_result "Test 3: no API response → skip + warn" fail
fi
unset -f curl
unset -f uname

# =============================================================================
# Test 4: fresh install, asset download 404s
# =============================================================================
reset_globals
# Stub: API works, asset returns 22 (curl would do this for a 404).
curl() {
    local arg url=""
    for arg in "$@"; do
        if [[ "$arg" != -* ]]; then url="$arg"; fi
    done
    if [[ "$url" == *api.github.com* ]]; then
        echo '{"tag_name": "v1.2.0"}'
        return 0
    fi
    # asset URL → 404
    return 22
}
uname() { echo "x86_64"; }

if install_piper > "$TMP/out.txt" 2> "$TMP/err.txt"; then
    if [[ ! -e "$PIPER_DEST" ]] && grep -q "download failed" "$TMP/err.txt"; then
        record_result "Test 4: asset 404 → skip + warn" ok
    else
        echo "  piper unexpectedly present or warning missing"
        echo "  --- stderr ---"; cat "$TMP/err.txt"
        record_result "Test 4: asset 404 → skip + warn" fail
    fi
else
    record_result "Test 4: asset 404 → skip + warn" fail
fi
unset -f curl
unset -f uname

# =============================================================================
# Test 5: fresh install, broken binary (extracted but --version fails)
# =============================================================================
reset_globals
mkdir -p "$TMP/fake_api"
cat > "$TMP/fake_api/release.json" <<'EOF'
{"tag_name": "v1.2.0", "assets": [{"name": "piper_linux_x86_64.tar.gz"}]}
EOF
# Build a tarball whose piper script exits 0 but doesn't print
# "piper" — the smoke test (grep -qi piper on --version output)
# should reject it.
broken_dir="$(mktemp -d)"
cat > "$broken_dir/piper" <<'EOF'
#!/bin/sh
exit 0
EOF
chmod 0755 "$broken_dir/piper"
( cd "$broken_dir" && tar -czf "$TMP/piper.tar.gz" piper )
rm -rf "$broken_dir"

curl() {
    local url=""
    local out=""
    local arg
    local saw_o=0
    for arg in "$@"; do
        if [[ "$arg" == "-o" ]]; then
            saw_o=1
            continue
        fi
        if [[ $saw_o -eq 1 ]]; then
            out="$arg"
            saw_o=0
            continue
        fi
        if [[ "$arg" == -* ]]; then
            continue
        fi
        url="$arg"
    done
    if [[ "$url" == *api.github.com* ]]; then
        if [[ -n "$out" ]]; then
            cp "$TMP/fake_api/release.json" "$out"
        else
            cat "$TMP/fake_api/release.json"
        fi
    else
        # asset URL → return the (broken) tarball
        if [[ -n "$out" ]]; then
            cp "$TMP/piper.tar.gz" "$out"
        else
            cat "$TMP/piper.tar.gz"
        fi
    fi
    return 0
}
uname() { echo "x86_64"; }

if install_piper > "$TMP/out.txt" 2> "$TMP/err.txt"; then
    if [[ ! -e "$PIPER_DEST" ]] && grep -q "did not respond" "$TMP/err.txt"; then
        record_result "Test 5: broken binary → removed" ok
    else
        echo "  broken binary unexpectedly present"
        echo "  --- stderr ---"; cat "$TMP/err.txt"
        record_result "Test 5: broken binary → removed" fail
    fi
else
    record_result "Test 5: broken binary → removed" fail
fi
unset -f curl
unset -f uname

# =============================================================================
# Test 6-9: piper_arch_for_machine
# =============================================================================
uname() { echo "x86_64"; }
got="$(piper_arch_for_machine)"
[[ "$got" == "x86_64" ]] \
    && record_result "Test 6: arch x86_64" ok \
    || record_result "Test 6: arch x86_64 (got: $got)" fail
unset -f uname

uname() { echo "aarch64"; }
got="$(piper_arch_for_machine)"
[[ "$got" == "aarch64" ]] \
    && record_result "Test 7: arch aarch64" ok \
    || record_result "Test 7: arch aarch64 (got: $got)" fail
unset -f uname

uname() { echo "arm64"; }
got="$(piper_arch_for_machine)"
[[ "$got" == "aarch64" ]] \
    && record_result "Test 8: arch arm64 → aarch64" ok \
    || record_result "Test 8: arch arm64 (got: $got)" fail
unset -f uname

uname() { echo "riscv64"; }
got="$(piper_arch_for_machine)"
[[ -z "$got" ]] \
    && record_result "Test 9: arch riscv64 → unsupported" ok \
    || record_result "Test 9: arch riscv64 (got: $got)" fail
unset -f uname

# =============================================================================
# Summary
# =============================================================================
echo ""
echo "Results:"
for line in "${RESULTS[@]}"; do
    echo "$line"
done
echo ""
echo "Summary: $PASS pass, $FAIL fail"
if [[ $FAIL -gt 0 ]]; then
    exit 1
fi
exit 0
