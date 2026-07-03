#!/bin/bash
# Pre-commit hook: Check that .task-manager and spec directories are not committed
# These directories should be tracked but not committed to remote

set -e

echo "📁 Checking for protected directories..."

# Get staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM 2>/dev/null || true)

if [ -z "$STAGED_FILES" ]; then
    echo "✅ No staged files"
    exit 0
fi

ERRORS=0
VIOLATIONS=""

# Check if any staged files are in protected directories
for file in $STAGED_FILES; do
    # Check if file is in .task-manager or spec directory
    if echo "$file" | grep -qE "^\.task-manager/|^spec/"; then
        VIOLATIONS="$VIOLATIONS\n  - $file"
        ERRORS=$((ERRORS + 1))
    fi
done

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "❌ Found $ERRORS file(s) from protected directories:"
    echo -e "$VIOLATIONS"
    echo ""
    echo "💡 Solution:"
    echo "   - .task-manager/ and spec/ are local-only directories"
    echo "   - They should NOT be committed to the repository"
    echo "   - Remove them from staging: git reset HEAD .task-manager/ spec/"
    echo "   - Or use: git restore --staged .task-manager/ spec/"
    exit 1
fi

echo "✅ No protected directory files staged"
exit 0
