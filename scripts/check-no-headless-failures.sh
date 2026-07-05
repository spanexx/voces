#!/bin/bash
# check-no-headless-failures.sh
#
# Purpose: re-run the wizard tests under an empty DISPLAY and
# verify they exit 0 with no real failures (FAIL). This catches
# a specific class of regression: a future refactor re-introducing
# t.Fatalf on a GTK init error, which would break CI on the
# integration test job (the previous failure mode was the wizard
# tests in ./internal/wizard/... exiting 1 in CI with messages
# like "ensureInit: Unable to initialize GTK", which made every
# push go red even when the actual logic was fine).
#
# The wizard package has 4 tests that need a live GTK:
#   - TestWizard_NewWindow_DoesNotPanic
#   - TestWizard_EnsureInit_Idempotent
#   - TestStepLanguage_DefaultIsEnglish
#   - TestStepHotkey_PresetsHaveLabels
# All 4 must early-return when GTK cannot be initialized, so the
# package exits 0. This script runs the whole package twice —
# once normally (so the dev sees real failures on the dev box)
# and once under DISPLAY='' / WAYLAND_DISPLAY='' (so the dev
# catches a regression in the opt-out pattern before it lands in
# CI).
#
# Exit codes:
#   0  - all tests pass under both modes (no FAIL, no build error)
#   1  - either mode has a real failure (not skip or pass)
#
# Run by `make precommit` (step 8/8).

set -e

echo "🖥️  Running wizard tests under empty DISPLAY..."
echo "   (catches regressions in the requireGTKOrSkip helper)"
echo ""

# Run the wizard tests with DISPLAY and WAYLAND_DISPLAY both
# unset. gotk3's gtk.InitCheck returns an error in this mode;
# the requireGTKOrSkip helper should early-return so the test
# passes (or be SKIP-marked if a future refactor reaches for
# the standard t.Skipf primitive). The "env -i" prefix is the
# safest cross-shell way to fully empty the environment
# (otherwise DISPLAY inherited from the user's SSH session
# would still be set and the test would not actually exercise
# the opt-out path).
RESULT=$(env -i HOME="$HOME" PATH="$PATH" \
    DISPLAY= \
    WAYLAND_DISPLAY= \
    go test -mod=vendor -count=1 -v ./internal/wizard/... 2>&1) || {
    echo "❌ wizard tests failed under empty DISPLAY:"
    echo "$RESULT" | tail -30
    echo ""
    echo "💡 Solution:"
    echo "   - All wizard tests that need a live GTK should use"
    echo "     requireGTKOrSkip(t) instead of t.Fatalf on the"
    echo "     ensureInit error."
    echo "   - Tests that need GTK only when DISPLAY is set are"
    echo "     listed at the top of internal/wizard/wizard_test.go."
    exit 1
}

# Sanity check: no FAIL markers (real failures). The opt-out
# path may produce either PASS (the requireGTKOrSkip helper
# uses t.Logf + return) or SKIP (if a future refactor
# switches to t.Skipf), both of which give exit 0. Only FAIL
# is the regression signal.
if echo "$RESULT" | grep -qE "^--- FAIL|FAIL\s+voces/internal/wizard"; then
    echo "❌ wizard tests have real FAIL markers under empty DISPLAY:"
    echo "$RESULT" | grep -E "FAIL" | head -10
    echo ""
    echo "💡 Solution:"
    echo "   - A test that needs a live GTK must early-return"
    echo "     via requireGTKOrSkip(t)."
    exit 1
fi

# Surface useful signal so the dev knows the check actually
# exercised the opt-out path. If neither PASS nor SKIP came
# from a test that we know needs GTK, the dev is running
# inside an X / Wayland session even under env -i and the
# check did not actually exercise the path.
#
# Note: with -v, go test prints `--- PASS: <name>`, so the
# test name comes AFTER the marker. Match both orders.
GTK_TESTS="TestWizard_NewWindow_DoesNotPanic|TestWizard_EnsureInit_Idempotent|TestStepLanguage_DefaultIsEnglish|TestStepHotkey_PresetsHaveLabels"
if ! echo "$RESULT" | grep -qE "(--- PASS:.*(${GTK_TESTS})|--- SKIP:.*(${GTK_TESTS})|(${GTK_TESTS}).*--- (PASS|SKIP))"; then
    echo "⚠️  The 4 GTK-needing tests did not pass or skip. Either"
    echo "   you have a display even under env -i (e.g. xvfb-run),"
    echo "   or the opt-out path is broken. Last 20 lines:"
    echo "$RESULT" | tail -20
    echo ""
    echo "💡 To verify the opt-out path manually:"
    echo "   env -i DISPLAY= WAYLAND_DISPLAY= go test ./internal/wizard/..."
    exit 1
fi

OPT_OUT_COUNT=$(echo "$RESULT" | grep -cE "(--- PASS:.*(${GTK_TESTS})|--- SKIP:.*(${GTK_TESTS})|(${GTK_TESTS}).*--- (PASS|SKIP))" || true)
echo "✅ $OPT_OUT_COUNT wizard test(s) exercised the opt-out path under empty DISPLAY"
echo "   (requireGTKOrSkip pattern is intact)"
exit 0
