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
#   - Resolves <name> → <name>t64 on Ubuntu 24.04+ / Debian 13+ /
#     Linux Mint 22+ where the original is a virtual package.
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

# 3. Resolve package names. Some distros (Ubuntu 24.04+, Debian 13+,
#    Linux Mint 22+) moved libraries from <name> to <name>t64 to
#    signal the time_t=64-bit ABI transition. On those distros the
#    original <name> is either a transitional alias or a pure virtual
#    package that apt will not auto-pick (failing the install with
#    "Package 'X' has no installation candidate"). We prefer the t64
#    variant when the apt cache lists it as a real package; older
#    distros (Ubuntu 22.04, Debian 12) only have the base name and
#    the helper falls back to that.
resolve_pkg() {
    local base="$1"
    local t64="${base}t64"
    if apt-cache show "$t64" 2>/dev/null \
            | grep -q "^Package: ${t64}$"; then
        echo "$t64"
    else
        echo "$base"
    fi
}

# 4. Runtime packages. Keep the list in lockstep with the comment at top.
PKGS=(
    "$(resolve_pkg libgtk-3-0)"
    libayatana-appindicator3-1
    xclip
    xdotool
    xdg-utils
    libx11-6
    libxtst6
    "$(resolve_pkg libasound2)"
    libpulse0
    espeak-ng
)

# 5. Filter out already-installed packages. `dpkg -s` exits 0 if installed.
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

# 6. Run apt. -y to assume yes; --no-install-recommends to keep it minimal.
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
