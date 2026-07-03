#!/bin/bash
# Pre-commit hook: Check for proper comments
# All Go files must have package comments and exported symbols documented

set -e

echo "💬 Checking for proper comments..."

ERRORS=0
VIOLATIONS=""

# Find all Go files (exclude test files and vendor)
GO_FILES=$(find . -name "*.go" -type f ! -name "*_test.go" 2>/dev/null | grep -v vendor | grep -v .git)

if [ -z "$GO_FILES" ]; then
    echo "✅ No Go files found"
    exit 0
fi

for file in $GO_FILES; do
    FILE_ERRORS=0

    # Check for package comment (first comment should be package comment)
    if ! head -20 "$file" | grep -q "^// Package"; then
        # Check for alternative comment style
        if ! head -20 "$file" | grep -q "^/\*.*Code Map"; then
            VIOLATIONS="$VIOLATIONS\n  - $file: Missing package comment"
            FILE_ERRORS=$((FILE_ERRORS + 1))
        fi
    fi

    # Check for exported functions without comments
    # Look for func starting with capital letter
    while IFS= read -r line; do
        LINE_NUM=$(echo "$line" | cut -d: -f1)
        FUNC_NAME=$(echo "$line" | cut -d: -f2-)

        # Check if previous line is a comment
        PREV_LINE=$(sed -n "$((LINE_NUM - 1))p" "$file")
        if ! echo "$PREV_LINE" | grep -q "^//"; then
            # Also check for CID block
            PREV_LINES=$(sed -n "$((LINE_NUM - 3)),$((LINE_NUM - 1))p" "$file" | grep -c "CID:" || true)
            if [ "$PREV_LINES" -eq 0 ]; then
                VIOLATIONS="$VIOLATIONS\n  - $file:$LINE_NUM: Exported $FUNC_NAME missing comment"
                FILE_ERRORS=$((FILE_ERRORS + 1))
            fi
        fi
    done < <(grep -n "^func [A-Z]" "$file" 2>/dev/null || true)

    # Check for exported types without comments
    while IFS= read -r line; do
        LINE_NUM=$(echo "$line" | cut -d: -f1)
        TYPE_NAME=$(echo "$line" | cut -d: -f2-)

        # Check if previous line is a comment
        PREV_LINE=$(sed -n "$((LINE_NUM - 1))p" "$file")
        if ! echo "$PREV_LINE" | grep -q "^//"; then
            VIOLATIONS="$VIOLATIONS\n  - $file:$LINE_NUM: Exported $TYPE_NAME missing comment"
            FILE_ERRORS=$((FILE_ERRORS + 1))
        fi
    done < <(grep -n "^type [A-Z]" "$file" 2>/dev/null || true)

    if [ $FILE_ERRORS -gt 0 ]; then
        ERRORS=$((ERRORS + 1))
    fi
done

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "❌ Found $ERRORS file(s) with missing comments:"
    echo -e "$VIOLATIONS"
    echo ""
    echo "💡 Solution: Use commenter workflow to add Code Map and CID blocks"
    echo "   Reference: .task-manager/workflows/commenter.md"
    exit 1
fi

echo "✅ All Go files have proper comments"
exit 0
