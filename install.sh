#!/usr/bin/env bash
# Voces one-line installer.
#
# Usage:
#   curl -fsSL https://github.com/spanexx/voces/releases/latest/download/install.sh | bash
#   # or pin a specific version:
#   VOCES_VERSION=v0.2.0-rc9 curl -fsSL ... | bash
#
# What it does:
#   1. Finds the latest published release from GitHub (including
#      prereleases — see the rc1-hotpatch-22 fix below).
#   2. Downloads the linux-amd64 tarball into a temp dir.
#   3. Extracts it into /opt/voces/.
#   4. Runs install-deps.sh to install the system libraries.
#   5. Symlinks the binaries into /usr/local/bin so they are on $PATH.
#   6. Installs the .desktop file so the app shows in the app menu.
#
# Uninstall:
#   sudo rm -rf /opt/voces
#   sudo rm -f /usr/local/bin/voces /usr/local/bin/voces-overlay
#   sudo rm -f /usr/local/share/applications/voces.desktop

set -euo pipefail

REPO="spanexx/voces"
INSTALL_DIR="/opt/voces"

# Sudo setup. Most distros already have sudo configured; for the
# rare case where it isn't, fall back to running as root.
SUDO=""
if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    else
        echo "Error: this installer needs root. Either run as root or install sudo." >&2
        exit 1
    fi
fi

# --- 1. Find the latest release tarball URL ---------------------------------
echo "Voces installer"
echo "  Repo:   $REPO"
echo "  Target: $INSTALL_DIR"
echo ""

# Honour a pinned version when the caller exports VOCES_VERSION
# (lets users install a specific tag like v0.2.0-rc8, or pin to
# rc1 in CI). When unset, pick the highest semver tag from the
# GitHub API list endpoint — which INCLUDES prereleases.
#
# rc1-hotpatch-22: the previous version used /releases/latest,
# which GitHub defines to exclude prereleases. Every voces
# release since rc1 has been published with --prerelease, so the
# endpoint silently kept returning the rc1 tarball. Switching to
# /releases?per_page=100 and picking the highest tag fixes the
# "install runs cleanly but I stay on rc1 forever" loop that
# burned us from rc2 onward.
if [ -n "${VOCES_VERSION:-}" ]; then
    LATEST_TAG="$VOCES_VERSION"
else
    API_URL="https://api.github.com/repos/${REPO}/releases?per_page=100"
    LATEST_TAG="$(
        curl -fsSL "$API_URL" \
            | grep '"tag_name"' \
            | sed 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/' \
            | grep -E '^v[0-9]+' \
            | sort -V \
            | tail -n 1
    )"
fi

if [ -z "${LATEST_TAG:-}" ]; then
    echo "Error: could not find a voces release tag on GitHub." >&2
    echo "  Check: https://github.com/${REPO}/releases" >&2
    echo "  You can also pin a version: VOCES_VERSION=v0.2.0-rc8 curl ... | bash" >&2
    exit 1
fi

# Construct the tarball URL directly from the tag. Asset naming
# convention is voces-${TAG}-linux-amd64.tar.gz (matches every
# release from rc1 onward — verified for rc1..rc8).
LATEST_ASSET_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/voces-${LATEST_TAG}-linux-amd64.tar.gz"

echo "Latest release tag:   $LATEST_TAG"
echo "Latest release asset: $LATEST_ASSET_URL"

# --- 2. Download to a temp dir ----------------------------------------------
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading tarball..."
curl -fsSL -o "$TMPDIR/voces.tar.gz" "$LATEST_ASSET_URL"

# --- 3. Extract to /opt/voces ---------------------------------
echo "Installing to $INSTALL_DIR..."
$SUDO mkdir -p "$INSTALL_DIR"
$SUDO tar xzf "$TMPDIR/voces.tar.gz" -C "$INSTALL_DIR" --strip-components=1

# --- 4. Install system dependencies ----------------------------------------
echo "Installing system dependencies..."
$SUDO "$INSTALL_DIR/install-deps.sh"

# --- 5. Symlink binaries ----------------------------------------------------
echo "Linking binaries to /usr/local/bin..."
$SUDO ln -sf "$INSTALL_DIR/voces" /usr/local/bin/voces
$SUDO ln -sf "$INSTALL_DIR/voces-overlay" /usr/local/bin/voces-overlay

# --- 6. Install .desktop file ----------------------------------------------
if [ -f "$INSTALL_DIR/voces.desktop" ]; then
    echo "Installing app menu entry..."
    $SUDO mkdir -p /usr/local/share/applications
    $SUDO cp "$INSTALL_DIR/voces.desktop" /usr/local/share/applications/
    $SUDO update-desktop-database /usr/local/share/applications 2>/dev/null || true
fi

# --- Done -------------------------------------------------------------------
cat <<EOF

✅ Voces ${LATEST_TAG} installed!

  Run it from anywhere:   voces
  Open the app menu:      search for "Voces"

  On first launch a setup wizard will open to:
    - pick your language
    - download the speech recognition "brain" (model)
    - pick a hotkey
    - (optionally) download a TTS voice

  See https://github.com/${REPO} for docs and troubleshooting.
EOF
