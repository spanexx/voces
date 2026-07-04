#!/usr/bin/env bash
# Whisper Voice Utility one-line installer.
#
# Usage:
#   curl -fsSL https://github.com/yourusername/whisper-voice-util/releases/latest/download/install.sh | bash
#
# What it does:
#   1. Detects the latest published release from GitHub.
#   2. Downloads the linux-amd64 tarball into a temp dir.
#   3. Extracts it into /opt/whisper-voice-util/.
#   4. Runs install-deps.sh to install the system libraries.
#   5. Symlinks the binaries into /usr/local/bin so they are on $PATH.
#   6. Installs the .desktop file so the app shows in the app menu.
#
# Uninstall:
#   sudo rm -rf /opt/whisper-voice-util
#   sudo rm -f /usr/local/bin/whisper-voice-util /usr/local/bin/whisper-voice-overlay
#   sudo rm -f /usr/local/share/applications/whisper-voice-util.desktop

set -euo pipefail

REPO="yourusername/whisper-voice-util"
INSTALL_DIR="/opt/whisper-voice-util"

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
echo "Whisper Voice Utility installer"
echo "  Repo:   $REPO"
echo "  Target: $INSTALL_DIR"
echo ""

API_URL="https://api.github.com/repos/${REPO}/releases/latest"
LATEST_ASSET_URL="$(
    curl -fsSL "$API_URL" \
        | grep '"browser_download_url"' \
        | grep 'linux-amd64\.tar\.gz' \
        | head -n 1 \
        | cut -d'"' -f4
)"

if [ -z "${LATEST_ASSET_URL:-}" ]; then
    echo "Error: could not find a linux-amd64.tar.gz asset on the latest release." >&2
    echo "  Check: https://github.com/${REPO}/releases/latest" >&2
    exit 1
fi

echo "Latest release asset: $LATEST_ASSET_URL"

# --- 2. Download to a temp dir ----------------------------------------------
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading tarball..."
curl -fsSL -o "$TMPDIR/whisper-voice-util.tar.gz" "$LATEST_ASSET_URL"

# --- 3. Extract to /opt/whisper-voice-util ---------------------------------
echo "Installing to $INSTALL_DIR..."
$SUDO mkdir -p "$INSTALL_DIR"
$SUDO tar xzf "$TMPDIR/whisper-voice-util.tar.gz" -C "$INSTALL_DIR" --strip-components=1

# --- 4. Install system dependencies ----------------------------------------
echo "Installing system dependencies..."
$SUDO "$INSTALL_DIR/install-deps.sh"

# --- 5. Symlink binaries ----------------------------------------------------
echo "Linking binaries to /usr/local/bin..."
$SUDO ln -sf "$INSTALL_DIR/whisper-voice-util" /usr/local/bin/whisper-voice-util
$SUDO ln -sf "$INSTALL_DIR/whisper-voice-overlay" /usr/local/bin/whisper-voice-overlay

# --- 6. Install .desktop file ----------------------------------------------
if [ -f "$INSTALL_DIR/whisper-voice-util.desktop" ]; then
    echo "Installing app menu entry..."
    $SUDO mkdir -p /usr/local/share/applications
    $SUDO cp "$INSTALL_DIR/whisper-voice-util.desktop" /usr/local/share/applications/
    $SUDO update-desktop-database /usr/local/share/applications 2>/dev/null || true
fi

# --- Done -------------------------------------------------------------------
cat <<EOF

✅ Whisper Voice Utility installed!

  Run it from anywhere:   whisper-voice-util
  Open the app menu:      search for "Whisper Voice Utility"

  On first launch a setup wizard will open to:
    - pick your language
    - download the speech recognition "brain" (model)
    - pick a hotkey
    - (optionally) download a TTS voice

  See https://github.com/${REPO} for docs and troubleshooting.
EOF
