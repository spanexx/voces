#!/bin/bash
# Pre-commit hook: Check for mock implementations in code
# All code must be real implementations, no placeholders or stubs

set -e

echo "🔍 Checking for mock implementations in code..."

# Find all Go files (exclude test files)
GO_FILES=$(find . -name "*.go" -type f ! -name "*_test.go" 2>/dev/null | grep -v vendor | grep -v .git)

if [ -z "$GO_FILES" ]; then
    echo "✅ No Go files found"
    exit 0
fi

ERRORS=0
VIOLATIONS=""

# Patterns that indicate mock/placeholder implementations in production code
PATTERNS=(
    "//.*In real"
    "//.*in real"
    "//.*For now"
    "//.*for now"
    "//.*TODO"
    "//.*FIXME"
    "//.*XXX"
    "//.*Stub"
    "//.*stub"
    "//.*STUB"
    "//.*placeholder"
    "//.*Placeholder"
    "//.*PLACEHOLDER"
    "//.*mock implementation"
    "//.*Mock implementation"
    "//.*fake implementation"
    "//.*Fake implementation"
    "//.*temporary"
    "//.*Temporary"
    "//.*will be replaced"
    "//.*to be implemented"
    "//.*not yet implemented"
    "//.*NYI"
    "//.*WIP"
    "panic.*not implemented"
    "panic.*TODO"
    "panic.*stub"
    "panic.*placeholder"
    "return.*nil.*//.*TODO"
    "return.*nil.*//.*stub"
    "return.*errors\.New.*TODO"
    "return.*errors\.New.*stub"
    "return.*fmt\.Errorf.*TODO"
    "return.*fmt\.Errorf.*stub"
)

for file in $GO_FILES; do
    FILE_ERRORS=0

    for pattern in "${PATTERNS[@]}"; do
        if grep -qiE "$pattern" "$file" 2>/dev/null; then
            # Get the matching lines
            MATCHES=$(grep -niE "$pattern" "$file" 2>/dev/null || true)
            if [ -n "$MATCHES" ]; then
                while IFS= read -r match; do
                    LINE_NUM=$(echo "$match" | cut -d: -f1)
                    LINE_CONTENT=$(echo "$match" | cut -d: -f2-)

                    # Check if it's in a comment (most likely) or actual code
                    if echo "$LINE_CONTENT" | grep -qE '^\s*//|^\s*\*|^\s*/\*'; then
                        # It's in a comment indicating placeholder code
                        VIOLATIONS="$VIOLATIONS\n  - $file:$LINE_NUM: $LINE_CONTENT"
                        FILE_ERRORS=$((FILE_ERRORS + 1))
                    elif echo "$LINE_CONTENT" | grep -qiE 'panic.*TODO|panic.*stub|panic.*not implemented|panic.*placeholder'; then
                        # It's actual placeholder code with panic
                        VIOLATIONS="$VIOLATIONS\n  - $file:$LINE_NUM: $LINE_CONTENT"
                        FILE_ERRORS=$((FILE_ERRORS + 1))
                    fi
                done <<< "$MATCHES"
            fi
        fi
    done

    if [ $FILE_ERRORS -gt 0 ]; then
        ERRORS=$((ERRORS + 1))
    fi
done

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "❌ Found $ERRORS file(s) with mock/placeholder implementations:"
    echo -e "$VIOLATIONS"
    echo ""
    echo "💡 Solution:"
    echo "   - Replace placeholder comments with real implementation"
    echo "   - Remove TODO/stub markers and implement the actual logic"
    echo "   - Implement complete functionality before committing"
    echo "   - Use feature branches for work-in-progress code"
    exit 1
fi

echo "✅ No mock/placeholder implementations found"
exit 0
