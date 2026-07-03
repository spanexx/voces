#!/bin/bash
# Pre-commit hook: Check for mocked/stubbed test implementations
# All tests must test real implementations, no stubs or placeholders

echo "🔍 Checking for mocked/stubbed test implementations..."

# Find all test files
TEST_FILES=$(find . -name "*_test.go" -type f 2>/dev/null | grep -v vendor | grep -v .git || true)

if [ -z "$TEST_FILES" ]; then
    echo "✅ No test files found"
    exit 0
fi

ERRORS=0
VIOLATIONS=""

# Patterns that indicate stubbed/mocked tests
PATTERNS=(
    "t\\.Skip"
    "t\\.Skipf"
    "//.*stub"
    "//.*Stub"
    "//.*STUB"
    "//.*TODO"
    "//.*FIXME"
    "//.*XXX"
    "//.*mock"
    "//.*Mock"
    "//.*MOCK"
    "//.*placeholder"
    "//.*Placeholder"
    "//.*PLACEHOLDER"
    "//.*for now"
    "//.*For now"
    "//.*For Now"
    "//.*in real"
    "//.*In real"
    "//.*actual implementation"
    "//.*real implementation"
    "//.*will be implemented"
    "//.*to be implemented"
    "//.*not yet implemented"
    "//.*NYI"
    "//.*WIP"
    "panic.*not implemented"
    "panic.*TODO"
    "panic.*stub"
    "return.*nil.*//.*TODO"
    "return.*nil.*//.*stub"
    "return.*errors\\.New.*TODO"
    "return.*errors\\.New.*stub"
)

for file in $TEST_FILES; do
    FILE_ERRORS=0

    for pattern in "${PATTERNS[@]}"; do
        MATCHES=$(grep -niE "$pattern" "$file" 2>/dev/null || true)
        if [ -n "$MATCHES" ]; then
            # Check context - is it actually a stub or just mentioning the word?
            while IFS= read -r match; do
                LINE_NUM=$(echo "$match" | cut -d: -f1)
                LINE_CONTENT=$(echo "$match" | cut -d: -f2-)

                # Skip if it's in a variable name or string literal that's not a comment
                if echo "$LINE_CONTENT" | grep -qE '^\s*//|^\s*\*|^\s*/\*'; then
                    # It's in a comment - likely a stub marker
                    VIOLATIONS="$VIOLATIONS\n  - $file:$LINE_NUM: $LINE_CONTENT"
                    FILE_ERRORS=$((FILE_ERRORS + 1))
                elif echo "$LINE_CONTENT" | grep -qiE 't\.Skip|panic.*TODO|panic.*stub|panic.*not implemented'; then
                    # It's actual stub code
                    VIOLATIONS="$VIOLATIONS\n  - $file:$LINE_NUM: $LINE_CONTENT"
                    FILE_ERRORS=$((FILE_ERRORS + 1))
                fi
            done <<< "$MATCHES"
        fi
    done

    # Check for empty or minimal test functions
    while IFS= read -r func_line; do
        FUNC_NAME=$(echo "$func_line" | sed -E 's/.*func (Test[A-Za-z0-9_]+).*/\1/')
        LINE_NUM=$(echo "$func_line" | cut -d: -f1)

        # Extract the function body (find closing brace)
        FUNC_BODY=$(sed -n "${LINE_NUM},\$p" "$file" | awk '
            BEGIN { braces=0; started=0 }
            /{/ { braces++; started=1 }
            started { print }
            /}/ { braces--; if (braces==0 && started) exit }
        ')

        # Count non-comment, non-empty lines
        REAL_LINES=$(echo "$FUNC_BODY" | grep -v '^\s*$' | grep -v '^\s*//' | grep -v '^\s*\*' | wc -l)

        if [ "$REAL_LINES" -lt 3 ]; then
            VIOLATIONS="$VIOLATIONS\n  - $file:$LINE_NUM: Test function $FUNC_NAME has only $REAL_LINES real lines (likely stub)"
            FILE_ERRORS=$((FILE_ERRORS + 1))
        fi
    done < <(grep -n "^func Test" "$file" 2>/dev/null || true)

    if [ $FILE_ERRORS -gt 0 ]; then
        ERRORS=$((ERRORS + 1))
    fi
done

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "❌ Found $ERRORS test file(s) with mocked/stubbed implementations:"
    echo -e "$VIOLATIONS"
    echo ""
    echo "💡 Solution:"
    echo "   - Remove t.Skip() calls and implement the test"
    echo "   - Replace TODO/stub comments with real test code"
    echo "   - Implement actual test logic instead of placeholders"
    echo "   - Tests must verify real behavior, not mock expectations"
    exit 1
fi

echo "✅ No mocked/stubbed test implementations found"
exit 0
