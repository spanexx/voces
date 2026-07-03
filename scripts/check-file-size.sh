#!/bin/bash
# Pre-commit hook: Check file line count
# All Go files must be below 250 lines (except tests and markdown)

set -e

echo "📏 Checking file sizes (max 250 lines for .go files)..."

MAX_LINES=250
ERRORS=0
VIOLATIONS=""

# Find all Go files (exclude test files and vendor)
GO_FILES=$(find . -name "*.go" -type f ! -name "*_test.go" 2>/dev/null | grep -v vendor | grep -v .git)

if [ -z "$GO_FILES" ]; then
    echo "✅ No Go files found"
    exit 0
fi

for file in $GO_FILES; do
    # Count lines
    LINES=$(wc -l < "$file")

    if [ "$LINES" -gt "$MAX_LINES" ]; then
        VIOLATIONS="$VIOLATIONS\n  - $file ($LINES lines)"
        ERRORS=$((ERRORS + 1))
    fi
done

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "❌ Found $ERRORS file(s) exceeding $MAX_LINES lines:"
    echo -e "$VIOLATIONS"
    echo ""
    echo "💡 Solution: Use go-modularizer workflow to split into smaller packages"
    echo "   Reference: .task-manager/workflows/go-modularizer.md"
    exit 1
fi

echo "✅ All Go files are under $MAX_LINES lines"
exit 0
