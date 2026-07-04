#!/bin/bash
# build-release.sh — Build the whisper-voice-util release tarball.
#
# IMPL §8: produces builds/whisper-voice-util-vX.Y.Z-linux-amd64.tar.gz
# containing both Go binaries + bundled engines + docs + install-deps.sh.
#
# Usage:
#   scripts/build-release.sh v0.2.0
#   scripts/build-release.sh v0.2.0-rc1
#
# Layout produced (matches ADR-0001):
#   whisper-voice-util-vX.Y.Z/
#     whisper-voice-util          # Go binary (uses -ldflags for Version)
#     whisper-voice-overlay       # Go binary (overlay window)
#     engines/
#       whisper-cli               # from vendor/whisper.cpp/build/bin/
#       piper                     # from vendor/piper/build/ (if built)
#       models.json               # engine manifest (URLs, sizes, hashes)
#     README.md
#     USAGE.md
#     install-deps.sh
#     config.yaml.example
#
# Re-runnable. Wipes the per-version build dir on each run.

set -euo pipefail

cd "$(dirname "$0")/.."  # repo root

# 1. Parse VERSION arg.
VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version>    e.g. $0 v0.2.0" >&2
    exit 1
fi
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    echo "❌ Version must look like vX.Y.Z or vX.Y.Z-rc1, got: $VERSION" >&2
    exit 1
fi

OS="linux"
ARCH="amd64"
APP_NAME="whisper-voice-util"
BUILD_DIR="builds"
RELEASE_DIR="${BUILD_DIR}/${APP_NAME}-${VERSION}"
TAR_NAME="${APP_NAME}-${VERSION}-${OS}-${ARCH}.tar.gz"

echo "======================================"
echo " Packaging: ${APP_NAME} ${VERSION}    "
echo "======================================"

# 2. Reset the release dir.
rm -rf "${RELEASE_DIR}"
mkdir -p "${RELEASE_DIR}/engines"

# 3. Build Go binaries with -ldflags for Version injection.
#
# -s -w      : strip debug info (smaller binary)
# -X main.Version=$VERSION : inject the version string into the
#              `var Version = "dev"` placeholder in cmd/whisper-voice-util/main.go.
#              The same ldflags pattern should be applied to whisper-voice-overlay
#              once it has a Version var.
LDFLAGS="-s -w -X main.Version=${VERSION}"

echo "🔨 Building Go binaries (Version=${VERSION})..."
mkdir -p bin
go build -mod=vendor -ldflags="${LDFLAGS}" -o "${RELEASE_DIR}/${APP_NAME}"           ./cmd/${APP_NAME}
go build -mod=vendor -ldflags="${LDFLAGS}" -o "${RELEASE_DIR}/whisper-voice-overlay"  ./cmd/whisper-voice-overlay
# Keep the same artifacts in bin/ for `make install` to use.
cp "${RELEASE_DIR}/${APP_NAME}"            bin/${APP_NAME}
cp "${RELEASE_DIR}/whisper-voice-overlay"  bin/whisper-voice-overlay

for f in "${RELEASE_DIR}/${APP_NAME}" "${RELEASE_DIR}/whisper-voice-overlay"; do
    if [[ ! -f "$f" ]]; then
        echo "❌ Build failed: $f not produced" >&2
        exit 1
    fi
done
echo "✅ Go binaries built."

# 4. Engine: whisper-cli. Make target is idempotent.
echo "🔨 Building whisper.cpp (may take a few minutes on first run)..."
if make -s whispercpp-build 2>&1 | tail -20; then
    if [[ -f vendor/whisper.cpp/build/bin/whisper-cli ]]; then
        install -m 0755 vendor/whisper.cpp/build/bin/whisper-cli "${RELEASE_DIR}/engines/whisper-cli"
        strip "${RELEASE_DIR}/engines/whisper-cli" || true
        echo "✅ whisper-cli bundled (and stripped)."
    else
        echo "⚠️  whisper.cpp build said OK but no binary at expected path." >&2
    fi
else
    echo "❌ whisper.cpp build failed. Tarball will ship without whisper-cli." >&2
    echo "   User will need to install whisper.cpp system-wide for transcription." >&2
fi

# 5. Engine: piper. Optional — piper has heavy build deps (ONNX runtime, etc.).
#    The Makefile target is a no-op stub that prints a TODO if piper isn't vendored.
echo "🔨 Building piper (optional, may be skipped)..."
if make -s piper-build 2>&1 | tail -10; then
    if [[ -f vendor/piper/build/piper ]]; then
        install -m 0755 vendor/piper/build/piper "${RELEASE_DIR}/engines/piper"
        strip "${RELEASE_DIR}/engines/piper" || true
        echo "✅ piper bundled (and stripped)."
    fi
else
    echo "⚠️  piper build skipped or failed. TTS will require a system install." >&2
fi

# 6. Engine manifest. Ship the one in engines/ if present, else the default.
if [[ -f engines/models.json ]]; then
    cp engines/models.json "${RELEASE_DIR}/engines/models.json"
    echo "✅ Bundled engines/models.json (${APP_NAME} live manifest)."
else
    echo "⚠️  engines/models.json not found in repo. Tarball ships without it." >&2
    echo "   The App will fall back to its built-in default manifest at runtime." >&2
fi

# 7. Copy docs and helpers.
echo "📁 Copying docs + install-deps.sh + config.yaml.example..."
for f in README.md USAGE.md; do
    if [[ -f "$f" ]]; then
        cp "$f" "${RELEASE_DIR}/"
    else
        echo "⚠️  $f missing — skipping." >&2
    fi
done

if [[ -f scripts/install-deps.sh ]]; then
    cp scripts/install-deps.sh "${RELEASE_DIR}/"
    chmod +x "${RELEASE_DIR}/install-deps.sh"
fi

if [[ -f config.yaml.example ]]; then
    cp config.yaml.example "${RELEASE_DIR}/"
else
    # Touch empty file so the App doesn't fail looking for it.
    : > "${RELEASE_DIR}/config.yaml.example"
    echo "⚠️  config.yaml.example missing — created empty stub." >&2
fi

# 8. Show what we're about to package.
echo ""
echo "📦 Tarball contents:"
find "${RELEASE_DIR}" -type f -printf "  %P  (%s bytes)\n" | sort

# 9. Tar + gzip.
echo ""
echo "🗜️  Creating ${TAR_NAME}..."
mkdir -p "${BUILD_DIR}"
tar -C "${BUILD_DIR}" -czf "${BUILD_DIR}/${TAR_NAME}" "${APP_NAME}-${VERSION}/"

# 10. Final size + path.
TAR_SIZE=$(du -h "${BUILD_DIR}/${TAR_NAME}" | cut -f1)
echo ""
echo "======================================"
echo "✅ Release packaged successfully        "
echo "📂 Location: ${BUILD_DIR}/${TAR_NAME}   "
echo "📦 Size:     ${TAR_SIZE}                 "
echo "======================================"
echo ""
echo "Next: upload to GitHub Releases as a ${VERSION/-rc/.rc} pre-release (rc) or"
echo "latest (non-rc). The App's auto-updater will then find it automatically."
