#!/bin/bash
# Pre-commit hook: Check for secrets/credentials
# Must not commit any secrets, API keys, or credentials

set -e

echo "🔒 Checking for secrets and credentials..."

ERRORS=0
VIOLATIONS=""

# Patterns to detect secrets
PATTERNS=(
    "api[_-]?key\s*[=:]\s*['\"][^'\"]+['\"]"
    "api[_-]?secret\s*[=:]\s*['\"][^'\"]+['\"]"
    "password\s*[=:]\s*['\"][^'\"]+['\"]"
    "secret\s*[=:]\s*['\"][^'\"]+['\"]"
    "token\s*[=:]\s*['\"][^'\"]+['\"]"
    "access[_-]?token\s*[=:]\s*['\"][^'\"]+['\"]"
    "private[_-]?key\s*[=:]\s*['\"][^'\"]+['\"]"
    "AWS_SECRET"
    "AWS_ACCESS_KEY"
    "sk-[a-zA-Z0-9]+"
    "ghp_[a-zA-Z0-9]+"
    "xox[baprs]-[a-zA-Z0-9]+"
)

# Files to check (staged files)
if git rev-parse --verify HEAD >/dev/null 2>&1; then
    # Get staged files
    FILES=$(git diff --cached --name-only --diff-filter=ACM 2>/dev/null || true)
else
    # Initial commit - check all files
    FILES=$(find . -type f -name "*.go" -o -name "*.yaml" -o -name "*.yml" -o -name "*.json" -o -name "*.env*" 2>/dev/null | grep -v vendor | grep -v .git || true)
fi

if [ -z "$FILES" ]; then
    echo "✅ No files to check"
    exit 0
fi

for file in $FILES; do
    if [ ! -f "$file" ]; then
        continue
    fi

    # Skip test files, example configs, and pre-commit config
    if echo "$file" | grep -qE "(test\.go|_test\.go|\.example\.|config\.example|\.pre-commit-config\.yaml)"; then
        continue
    fi

    for pattern in "${PATTERNS[@]}"; do
        if grep -qiE "$pattern" "$file" 2>/dev/null; then
            # Check if it's a placeholder or environment variable.
            # Use single quotes so the backslashes survive shell
            # processing - in double quotes \$\{ becomes ${
            # which is invalid ERE ($ is end-of-line, { starts a
            # quantifier) and the env-var bypass never fires.
            if ! grep -qiE '\$\{|os\.Getenv|flag\.String|default.*""|TODO|FIXME|xxx|placeholder' "$file"; then
                VIOLATIONS="$VIOLATIONS\n  - $file: Potential secret detected (pattern: $pattern)"
                ERRORS=$((ERRORS + 1))
                break
            fi
        fi
    done
done

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "❌ Found $ERRORS file(s) with potential secrets:"
    echo -e "$VIOLATIONS"
    echo ""
    echo "💡 Solution:"
    echo "   - Use environment variables: os.Getenv(\"API_KEY\")"
    echo "   - Use config files with .example suffix"
    echo "   - Never commit actual credentials"
    exit 1
fi

echo "✅ No secrets detected"
exit 0
