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
#   - Runs `apt-get update` first so the t64 detection below is
#     reliable on a fresh box (without this, resolve_pkg returns the
#     base name when the cache is empty and apt fails on libasound2).
#   - Waits for any background refresher (Linux Mint's mint-refresh-ca,
#     Ubuntu's unattended-upgrades, Pop!_OS's pop-system-updater) to
#     release /var/lib/apt/lists/lock before calling apt-get
#     (rc1-hotpatch-25 — failed on Mint 22.3 with process 1277933
#     holding the lock for ~10s on every install attempt).
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
    echo "     libx11-6 libxtst6 libasound2 libpulse0 espeak-ng libonnxruntime1" >&2
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

# 3. Wait for any background process to release the apt lock.
#    Linux Mint's mint-refresh-ca (process 1277933 in the rc25
#    bug report), Ubuntu's unattended-upgrades, and Pop!_OS's
#    pop-system-updater all hold /var/lib/apt/lists/lock for
#    several seconds at a time. Calling apt-get update while
#    the lock is held fails immediately with E: Could not get
#    lock, and the user has to re-run the installer manually.
#    wait_for_apt_lock polls the lock file with fuser; when the
#    refresher releases the lock, the helper returns and we
#    proceed. 120s default timeout is plenty for any of the
#    refreshers above (they typically hold the lock for 5-15s).
#    The actual apt-get call is also wrapped so a refresher
#    that starts AFTER our wait has a second chance.
wait_for_apt_lock() {
    local label="${1:-apt}"
    local timeout="${APT_LOCK_TIMEOUT:-120}"
    local elapsed=0
    # Three locks cover both apt-get update (lists/lock) and
    # apt-get install (dpkg/lock + dpkg/lock-frontend). Any
    # held lock means we wait; the polling is per-call, not
    # per-lock, so the timeout is the worst-case wait time.
    # The lock paths default to the apt locations; tests
    # override via the second argument.
    local lock_files=("${@:2}")
    if [[ ${#lock_files[@]} -eq 0 ]]; then
        lock_files=(
            /var/lib/apt/lists/lock
            /var/lib/dpkg/lock
            /var/lib/dpkg/lock-frontend
        )
    fi
    while true; do
        local held=()
        for lock in "${lock_files[@]}"; do
            # SUDO=() in production means we're root; SUDO=("sudo")
            # means the script prepends sudo to privileged calls. fuser
            # can only inspect another user's processes as root or via
            # sudo. The test in scripts/install-deps-test.sh stubs
            # SUDO=() to disable the wrapping and exercise the helper
            # without root.
            if [[ -e "$lock" ]] && "${SUDO[@]}" fuser "$lock" >/dev/null 2>&1; then
                held+=("$lock")
            fi
        done
        if [[ ${#held[@]} -eq 0 ]]; then
            return 0
        fi
        if [[ $elapsed -ge $timeout ]]; then
            echo "" >&2
            echo "❌ Timed out waiting ${timeout}s for the apt lock to be released" >&2
            echo "   (needed by: $label)." >&2
            echo "   Held by: ${held[*]}" >&2
            echo "" >&2
            echo "   Work-around: ${SUDO[*]} fuser -k ${held[*]}" >&2
            echo "   (kills the refresher; safe to run while the GUI is up)" >&2
            echo "   then re-run the installer." >&2
            return 1
        fi
        if [[ $elapsed -eq 0 ]]; then
            echo "⏳ Another process holds the apt lock: ${held[*]}" >&2
            echo "   (needed by: $label). Waiting up to ${timeout}s for it to release..." >&2
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
}

# 4. Refresh the apt cache. On a fresh Linux box (or one where the
#    user hasn't run apt in a while) the cache is empty, so
#    `apt-cache show <name>` returns nothing and the resolve_pkg
#    helper below falls back to the base name (libasound2 etc.)
#    which then fails with "no installation candidate". Running
#    update once, up-front, makes the t64 detection reliable and
#    avoids the user having to manually apt-get update + re-run
#    this script after the first failure.
if ! wait_for_apt_lock "apt-get update"; then
    echo "" >&2
    echo "❌ apt-get update skipped because the lock is still held." >&2
    echo "   See the work-around above." >&2
    exit 4
fi
if ! "${SUDO[@]}" apt-get update -y; then
    # The refresher may have grabbed the lock between our wait
    # and the apt-get call. One more retry covers that.
    if wait_for_apt_lock "apt-get update (retry)"; then
        if ! "${SUDO[@]}" apt-get update -y; then
            echo "" >&2
            echo "❌ apt-get update failed. Check your network connection and" >&2
            echo "   the contents of /etc/apt/sources.list." >&2
            exit 4
        fi
    else
        exit 4
    fi
fi

# 5. Resolve package names. Some distros (Ubuntu 24.04+, Debian 13+,
#    Linux Mint 22+) moved libraries from <name> to <name>t64 to
#    signal the time_t=64-bit ABI transition. On those distros the
#    original <name> is either a transitional alias or a pure virtual
#    package that apt will not auto-pick (failing the install with
#    "Package 'X' has no installation candidate"). We prefer the t64
#    variant when the apt cache lists it as a real package; older
#    distros (Ubuntu 22.04, Debian 12) only have the base name and
#    the helper falls back to that.
#
#    The previous version used only `apt-cache show <t64> | grep
#    "^Package: ..."` to detect the t64 variant. That check is
#    fragile: when the apt cache is partially populated (e.g. one
#    repo has a GPG error and apt-get update returns a partial
#    lists/ tree), `apt-cache show` can return empty even though
#    the package is installable. The user hit this exact case on
#    Linux Mint 22.3 with a broken atlassian apt key — the cache
#    for the Ubuntu main repo was intact (apt-get install would
#    have worked) but `apt-cache show libasound2t64 | grep ^Package`
#    returned nothing, so resolve_pkg fell back to the base name
#    and the install failed with "Package 'libasound2' has no
#    installation candidate".
#
#    New strategy: try multiple detection methods in order, with a
#    hardcoded t64 list for the known-new distros as the final
#    fallback. Each check is fast and has a clear pass/fail.
resolve_pkg() {
    local base="$1"
    local t64="${base}t64"
    # (a) Already installed? Use it. This is the fast path and also
    #     handles "I just installed it in a previous attempt".
    if dpkg -s "$t64" >/dev/null 2>&1; then
        echo "$t64"
        return
    fi
    # (b) apt-cache madison: lists each available version as
    #     "<name> | <version> | <repo>". One line means at least one
    #     version is in the cache. More reliable than `apt-cache
    #     show` because it doesn't depend on the full control-file
    #     being present (a partial cache can still resolve the
    #     binary package name).
    if apt-cache madison "$t64" 2>/dev/null | grep -q "^${t64} |"; then
        echo "$t64"
        return
    fi
    # (c) apt-cache show: the original check, kept as a fallback.
    #     Works when the apt cache is fully populated.
    if apt-cache show "$t64" 2>/dev/null | grep -q "^Package: ${t64}$"; then
        echo "$t64"
        return
    fi
    # (d) Hardcoded fallback for known-new distros. The whole
    #     point of t64 is the time_t=64-bit ABI transition that
    #     started with Ubuntu 24.04 / Debian 13 / Linux Mint 22.
    #     For those distros, t64 is always the correct name; we
    #     don't need to ask apt.
    case "${ID:-unknown}:${VERSION_ID:-0}" in
        linuxmint:2[2-9]*|linuxmint:3[0-9]*) echo "$t64"; return ;;
        ubuntu:2[4-9].*|ubuntu:3[0-9]*|pop:2[4-9].*|pop:3[0-9]*|zorin:1[7-9]*|zorin:2[0-9]*|"kde neon":2[4-9]*|"kde neon":3[0-9]*|elementary:8*|elementary:9*) echo "$t64"; return ;;
        debian:1[3-9]*|debian:2[0-9]*) echo "$t64"; return ;;
    esac
    # (e) Last resort: the base name. On older distros this is
    #     the real package; on new distros where every check
    #     above failed (broken cache, no network, etc.) the
    #     install will fail with a clear error message rather
    #     than silently picking a wrong provider.
    echo "$base"
}

# 6. Runtime packages. Keep the list in lockstep with the comment at top.
#    rc1-hotpatch-32: libonnxruntime1 added — piper TTS loads ONNX
#    models and needs the runtime shared lib at runtime. The piper
#    binary (downloaded separately by install.sh) is linked against
#    libonnxruntime.so.1, so without this package piper fails with
#    "error while loading shared libraries: libonnxruntime.so.1".
#    libonnxruntime1 is in Debian's main repo since Bullseye, so no
#    extra apt source is needed.
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
    libonnxruntime1
)

# 7. Filter out already-installed packages. `dpkg -s` exits 0 if installed.
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

# 8. Run apt. -y to assume yes; --no-install-recommends to keep it minimal.
#    Same lock-collision concern as apt-get update: a background
#    refresher can grab /var/lib/dpkg/lock the moment our update
#    finishes. wait_for_apt_lock handles the polling; we then
#    re-poll on the rare failure path with one retry.
if ! wait_for_apt_lock "apt-get install"; then
    echo "" >&2
    echo "❌ apt-get install skipped because the lock is still held." >&2
    exit 3
fi
if ! "${SUDO[@]}" apt-get install -y --no-install-recommends "${TO_INSTALL[@]}"; then
    if wait_for_apt_lock "apt-get install (retry)"; then
        if ! "${SUDO[@]}" apt-get install -y --no-install-recommends "${TO_INSTALL[@]}"; then
            echo "" >&2
            echo "❌ apt-get install failed. Try:" >&2
            echo "   ${SUDO[*]} apt-get update" >&2
            echo "   then re-run this script." >&2
            exit 3
        fi
    else
        exit 3
    fi
fi

echo ""
echo "✅ Dependencies installed successfully."
echo ""
echo "Next: extract the tarball (or build from source) and run ./voces."
echo "On first run, the setup wizard will offer to download the AI model files."
