#!/bin/bash
# Pre-commit hook: Check test coverage
# Must maintain MIN_COVERAGE % test coverage across non-GTK packages.
#
# Why the whitelist (COVERAGE_EXCLUDE):
#   Five packages are GTK-only and cannot be meaningfully unit-tested
#   without a real display server (Xvfb counts but is slow and flaky).
#   They are excluded from the coverage gate so that GTK window/embed
#   code does not drag down the metric for the rest of the codebase.
#   Manual smoke tests cover the GTK surface.
#
#   The excluded packages are:
#     - assets/icons                 (go:generate, embeds PNGs)
#     - cmd/whisper-voice-overlay    (main binary, requires display)
#     - internal/overlay             (GTK window)
#     - internal/wizard              (GTK wizard orchestrator)
#     - internal/wizard/steps        (GTK step widgets)

echo "📊 Checking test coverage (minimum ${MIN_COVERAGE}%)..."

MIN_COVERAGE=70

# Check if there are any test files
TEST_FILES=$(find . -name "*_test.go" -type f 2>/dev/null | grep -v vendor | grep -v .git || true)

if [ -z "$TEST_FILES" ]; then
    echo "⚠️  No test files found - skipping coverage check"
    echo "💡 Please add tests for your code"
    exit 0
fi

# Build the package list, excluding the GTK-only whitelist
COVERAGE_EXCLUDE='whisper-voice-util/assets/icons|whisper-voice-util/cmd/whisper-voice-overlay|whisper-voice-util/internal/overlay|whisper-voice-util/internal/wizard'
PKGS=$(go list -mod=vendor ./... 2>/dev/null | grep -vE "$COVERAGE_EXCLUDE" || true)

if [ -z "$PKGS" ]; then
    echo "⚠️  No packages to measure - skipping coverage check"
    exit 0
fi

# Run tests with coverage on the non-GTK packages
echo "Running tests with coverage (excluding GTK-only packages)..."
if ! go test -mod=vendor -coverprofile=coverage.out $PKGS; then
    echo "❌ Tests failed"
    rm -f coverage.out
    exit 1
fi

# Get coverage percentage
COVERAGE=$(go tool cover -func=coverage.out 2>/dev/null | grep total | awk '{print $3}' | sed 's/%//' || true)

# Clean up
rm -f coverage.out

if [ -z "$COVERAGE" ]; then
    echo "⚠️  Could not calculate coverage - skipping check"
    exit 0
fi

# Compare coverage (integer comparison)
COVERAGE_INT=${COVERAGE%.*}

echo "Current coverage: ${COVERAGE}%"

if [ "$COVERAGE_INT" -lt "$MIN_COVERAGE" ]; then
    echo ""
    echo "❌ Coverage ${COVERAGE}% is below minimum ${MIN_COVERAGE}%"
    echo ""
    echo "💡 Add more tests to improve coverage"
    echo "   Run: go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out"
    exit 1
fi

echo "✅ Coverage ${COVERAGE}% meets minimum ${MIN_COVERAGE}%"
exit 0
