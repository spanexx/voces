#!/bin/bash
set -e

# Variables
VERSION=${1:-"v1.0.0"}
OS="linux"
ARCH="amd64"
APP_NAME="whisper-voice-util"
BUILD_DIR="builds"
RELEASE_DIR="${BUILD_DIR}/${APP_NAME}-${VERSION}"
TAR_NAME="${APP_NAME}-${VERSION}-${OS}-${ARCH}.tar.gz"

echo "======================================"
echo " Packaging: ${APP_NAME} ${VERSION} "
echo "======================================"

# Clean up previous build directory if it exists
rm -rf "${RELEASE_DIR}"
mkdir -p "${RELEASE_DIR}"

# 1. Build the binary and inject the version via ldflags (Task 63)
echo "🔨 Compiling binary for ${OS}/${ARCH}..."
go build -mod=mod -ldflags="-s -w -X main.Version=${VERSION}" -o "${RELEASE_DIR}/${APP_NAME}" ./cmd/${APP_NAME}

# Check if build worked
if [ ! -f "${RELEASE_DIR}/${APP_NAME}" ]; then
    echo "❌ Build failed. Binary not generated."
    exit 1
fi

echo "✅ Binary compiled."

# 2. Add supplementary files (Task 65)
echo "📁 Copying assets..."
cp README.md "${RELEASE_DIR}/"

# Provide a sample config instead of blowing out theirs with config.yaml
if [ -f "config.yaml" ]; then
    cp config.yaml "${RELEASE_DIR}/config.yaml.example"
else
    # Touch empty file just to be safe if config hasn't been generated yet
    touch "${RELEASE_DIR}/config.yaml.example"
fi

# 3. Create Tarball
echo "📦 Creating release tarball..."
cd "${BUILD_DIR}"
tar -czvf "${TAR_NAME}" "${APP_NAME}-${VERSION}"/ > /dev/null
cd ..

echo "======================================"
echo "✅ Release Packaged Successfully"
echo "📂 Location: ${BUILD_DIR}/${TAR_NAME}"
echo "======================================"
