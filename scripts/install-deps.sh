#!/bin/bash
# install-deps.sh — install runtime system dependencies for voces.
#
# Installs the libraries the App needs to run on a fresh Linux box:
#   - GTK 3 runtime (wizard window, tray menu)
#   - libayatana-appindicator3 (system tray icon — the menu in the panel)
#   - xclip, xdotool, xdg-utils (clipboard, hotkey simulation, open-folder)
#   - libx11, libxtst (X11 hotkey capture backend)
#   - espeak-ng (piper TTS uses it for phoneme generation)
#   - libasound2, libpulse0 (audio capture / playback via ALSA / PulseAudio)
#
# IMPL §8 / ADR-0001. Behavior:
#   - Detects /etc/os-release. Warns and exits non-zero on non-Debian/Ubuntu.
#   - If $EUID != 0, prepends sudo to the package install.
#   - Skips packages that dpkg reports as already installed.
#   - Exits 0 on success, non-zero on failure.
#
# Re-runnable. Safe to run after a failed attempt.

set -euo pipefail

# 1. Detect distro. Only Debian / Ubuntu supported for v1.
if [[ ! -f /etc/os-release ]]; then
    echo "❌ /etc/os-release not found. Cannot detect distro." >&2
    exit 1
fi
. /etc/os-release
case "${ID:-unknown}" in
    debian|ubuntu|pop|linuxmint|elementary|"kde neon"|zorin)
        echo "✅ Detected ${PRETTY_NAME:-${ID}} (Debian-family)."
        ;;
    *)
        echo "⚠️  Detected ${PRETTY_NAME:-${ID}}. This script is tested only on" >&2
        echo "   Debian-family distros. You'll need to translate the apt package" >&2
        echo "   list below to your distro's package manager." >&2
        echo "" >&2
        echo "   Required runtime packages (Debian names):" >&2
        echo "     libgtk-3-0 libayatana-appindicator3-1 xclip xdotool xdg-utils" >&2
        echo "     libx11-6 libxtst6 libasound2 libpulse0 espeak-ng" >&2
        exit 2
        ;;
esac

# 2. Pick sudo if we're not root.
SUDO=()
if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
    if ! command -v sudo >/dev/null 2>&1; then
        echo "❌ Not running as root and 'sudo' is not installed. Install sudo or run as root." >&2
        exit 1
    fi
    SUDO=(sudo)
    echo "🔒 Not running as root; will use sudo for package install."
fi

# 3. Runtime packages. Keep the list in lockstep with the comment at top.
PKGS=(
    libgtk-3-0
    libayatana-appindicator3-1
    xclip
    xdotool
    xdg-utils
    libx11-6
    libxtst6
    libasound2
    libpulse0
    espeak-ng
)

# 4. Filter out already-installed packages. `dpkg -s` exits 0 if installed.
TO_INSTALL=()
for pkg in "${PKGS[@]}"; do
    if dpkg -s "$pkg" >/dev/null 2>&1; then
        echo "  ✓ $pkg (already installed)"
    else
        TO_INSTALL+=("$pkg")
    fi
done

if [[ ${#TO_INSTALL[@]} -eq 0 ]]; then
    echo ""
    echo "✅ All dependencies are already installed. Nothing to do."
    exit 0
fi

echo ""
echo "📦 Installing ${#TO_INSTALL[@]} package(s): ${TO_INSTALL[*]}"
echo ""

# 5. Run apt. -y to assume yes; --no-install-recommends to keep it minimal.
if ! "${SUDO[@]}" apt-get install -y --no-install-recommends "${TO_INSTALL[@]}"; then
    echo "" >&2
    echo "❌ apt-get install failed. Try:" >&2
    echo "   ${SUDO[*]} apt-get update" >&2
    echo "   then re-run this script." >&2
    exit 3
fi

echo ""
echo "✅ Dependencies installed successfully."
echo ""
echo "Next: extract the tarball (or build from source) and run ./voces."
echo "On first run, the setup wizard will offer to download the AI model files."
