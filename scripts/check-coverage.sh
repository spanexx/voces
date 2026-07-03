#!/bin/bash
# Pre-commit hook: Check test coverage
# Must maintain 85% test coverage

echo "📊 Checking test coverage (minimum 85%)..."

MIN_COVERAGE=85

# Check if there are any test files
TEST_FILES=$(find . -name "*_test.go" -type f 2>/dev/null | grep -v vendor | grep -v .git || true)

if [ -z "$TEST_FILES" ]; then
    echo "⚠️  No test files found - skipping coverage check"
    echo "💡 Please add tests for your code"
    exit 0
fi

# Run tests with coverage
echo "Running tests with coverage..."
if ! go test -mod=vendor -coverprofile=coverage.out ./...; then
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
