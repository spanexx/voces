#!/usr/bin/env bash
# Voces one-line installer.
#
# Usage:
#   curl -fsSL https://github.com/spanexx/voces/releases/latest/download/install.sh | bash
#   # or pin a specific version:
#   VOCES_VERSION=v0.2.0-rc9 curl -fsSL ... | bash
#
# Channel auto-detection (rc1-hotpatch-28):
#   The installer picks a release by walking the GitHub tags and
#   deciding which channel the user is on:
#     1. If VOCES_VERSION is set, that wins (escape hatch).
#     2. Else if /opt/voces/voces exists, parse its version
#        (`voces --version`) and stay on the same channel:
#        - installed is a prerelease (v0.2.0-rc12) → pick the
#          highest prerelease of the same base (v0.2.0-rc13).
#          If no prerelease of that base exists, stay on
#          what's installed (no auto-promotion to stable).
#        - installed is stable (v0.2.0) → pick the highest
#          stable of any base (v0.2.1). Stable never
#          downgrades to a prerelease of its own base.
#     3. Else (fresh install) → pick the highest stable
#        (or, if no stable exists, the highest prerelease
#        of the highest base).
#
#   Result: a user who installed rc12 doesn't have to remember
#   `VOCES_VERSION=v0.2.0-rc13` for the next update — a plain
#   `curl ... | bash` does the right thing. The escape hatch
#   (VOCES_VERSION) is still available for one-off pinning.
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

# pick_latest_tag <installed_version> <tags>
#
# Pure function: given the currently installed version (or empty
# string for a fresh install) and the newline-separated list of
# available tags, prints the tag the installer should fetch.
# Returns 0 on success, 1 on no candidate.
#
# Channel rules (rc1-hotpatch-28):
#   - installed is a prerelease (e.g. v0.2.0-rc12):
#       * Pick the highest prerelease of the same base
#         (v0.2.0-rc13 if it exists).
#       * If no prerelease of that base exists, stay on
#         what's installed (no auto-promote to stable —
#         the user opted into the prerelease channel and
#         should opt out explicitly via VOCES_VERSION).
#   - installed is stable (e.g. v0.2.0):
#       * Pick the highest stable of any base
#         (v0.2.1 wins over v0.2.0).
#       * Never downgrade to a prerelease.
#   - installed is empty (fresh install):
#       * Pick the highest stable, falling back to the
#         highest prerelease of the highest base if no
#         stable is published.
#
# The function is intentionally side-effect-free so that
# scripts/install-test.sh can source it and exercise every
# branch without touching the network or the filesystem.
pick_latest_tag() {
    local installed="$1"
    local tags="$2"

    if [ -z "$tags" ]; then
        return 1
    fi

    # iv = version without the leading "v"
    # iv_base = base (before the first "-")
    # iv = iv_base  => stable
    # iv != iv_base => prerelease
    local iv="${installed#v}"
    local iv_base="${iv%%-*}"

    if [ "$iv" != "$iv_base" ]; then
        # PRERELEASE channel. Stay on the same base; pick the
        # highest prerelease within it.
        if printf '%s\n' "$tags" | grep -qE "^v${iv_base}-"; then
            printf '%s\n' "$tags" \
                | grep -E "^v${iv_base}-" \
                | sort -V \
                | tail -n 1
            return 0
        fi
        # No prereleases of this base are published. Stay on
        # what's installed (no auto-promote to stable).
        if [ -n "$installed" ]; then
            echo "$installed"
            return 0
        fi
        # Empty installed + no rc of this base can't happen
        # (empty installed hits the stable branch below) but
        # fall through for safety.
    fi

    # STABLE channel (or fresh install). Pick the highest
    # stable tag of any base.
    local stable
    stable="$(printf '%s\n' "$tags" | grep -E '^v[0-9]+(\.[0-9]+){2}$' | sort -V | tail -n 1)"
    if [ -n "$stable" ]; then
        echo "$stable"
        return 0
    fi

    # No stable tag exists. Fall back to the highest prerelease
    # of the highest base (sort -V treats the suffix as a
    # build identifier that sorts after the base).
    local highest_base
    highest_base="$(printf '%s\n' "$tags" | sed 's/-.*//' | sort -V | tail -n 1)"
    if printf '%s\n' "$tags" | grep -q "^${highest_base}-"; then
        printf '%s\n' "$tags" | grep "^${highest_base}-" | sort -V | tail -n 1
        return 0
    fi
    echo "$highest_base"
    return 0
}

# detect_installed_version
#
# Prints the version of the currently installed /opt/voces/voces,
# or empty string if no install is found / the binary can't run.
# Used as the "installed" argument to pick_latest_tag.
detect_installed_version() {
    if [ ! -x "$INSTALL_DIR/voces" ]; then
        return 0
    fi
    # `voces --version` prints "Voces version v<X>" on its own line.
    # We use $SUDO because /opt/voces is typically root-owned.
    $SUDO "$INSTALL_DIR/voces" --version 2>/dev/null \
        | awk '/Voces version/ {print $3; exit}'
}

# --- 1. Find the latest release tarball URL ---------------------------------
echo "Voces installer"
echo "  Repo:   $REPO"
echo "  Target: $INSTALL_DIR"
echo ""

# Honour a pinned version when the caller exports VOCES_VERSION
# (lets users install a specific tag like v0.2.0-rc8, or pin to
# rc1 in CI). When unset, auto-detect the channel from the
# currently installed version (see pick_latest_tag above).
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
    TAGS="$(
        curl -fsSL "$API_URL" \
            | grep '"tag_name"' \
            | sed 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/' \
            | grep -E '^v[0-9]+'
    )"
    INSTALLED_VERSION="$(detect_installed_version || true)"
    if [ -n "$INSTALLED_VERSION" ]; then
        echo "  Installed: $INSTALLED_VERSION (auto-detected channel)"
    else
        echo "  Installed: (none — fresh install)"
    fi
    LATEST_TAG="$(pick_latest_tag "$INSTALLED_VERSION" "$TAGS" || true)"
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

# --- 4b. Install piper TTS engine (rc1-hotpatch-32) ------------------------
# Voces uses piper (https://github.com/rhasspy/piper) to read transcriptions
# aloud. The piper binary is distributed as a prebuilt tarball on GitHub
# releases, ~5 MB. We download it to /opt/voces/engines/piper (the path
# setup.FindPiperBinary checks first), chmod it +x, and that's it — no
# apt install, no build step. libonnxruntime1 (the only system dep) is
# already in install-deps.sh above.
#
# If a piper binary is already at $INSTALL_DIR/engines/piper, we skip
# (this lets a partial download from a previous failed run be retried
# without re-downloading, and lets the rc32+ release tarball ship a
# pre-downloaded binary that we just chmod + extract).
#
# Failure mode: any error here is logged but does NOT abort the install.
# The voces tarball itself, the whisper.cpp binary, and the voices
# downloader are independent of piper; a user who doesn't use TTS
# (Ctrl+U) is unaffected. A user who does use TTS will see the rc31
# install hint in the wizard and can install manually.
PIPER_REPO="rhasspy/piper"
PIPER_DEST_DIR="$INSTALL_DIR/engines"
PIPER_DEST="$PIPER_DEST_DIR/piper"

# piper_arch_for_machine maps `uname -m` output to the asset suffix
# upstream ships in their release tarball filenames. Empty string when
# the arch is not supported (the script skips the download in that case).
piper_arch_for_machine() {
    case "$(uname -m)" in
        x86_64)         echo "x86_64" ;;
        aarch64|arm64)  echo "aarch64" ;;
        armv7l|armhf)   echo "armv7l" ;;
        *)              echo "" ;;
    esac
}

install_piper() {
    if [ -x "$PIPER_DEST" ]; then
        echo "  ✓ piper already at $PIPER_DEST"
        return 0
    fi

    local arch
    arch="$(piper_arch_for_machine)"
    if [ -z "$arch" ]; then
        echo "  ⚠️  piper: no prebuilt binary for $(uname -m) — skipping." >&2
        echo "     Install piper manually: see the wizard's piper-status step." >&2
        return 0
    fi

    local piper_tmp
    piper_tmp="$(mktemp -d)"
    # Don't replace the script's EXIT trap here — install.sh
    # already has one for $TMPDIR. Just clean up piper_tmp
    # explicitly on failure paths and let the script-level
    # EXIT trap handle the success path. This is also
    # install-piper-test.sh friendly: tests run install_piper
    # many times and the cumulative trap state would get
    # unwieldy otherwise.
    local cleanup_piper_tmp="rm -rf '$piper_tmp'"

    echo "  Downloading piper (arch=$arch) from GitHub releases..."
    # Query the API for the latest release tag. Fall back to a known-good
    # tag if the API is rate-limited (60 req/h unauthenticated) or the
    # network is flaky; the user can always re-run the installer.
    local latest_tag
    latest_tag="$(
        curl -fsSL "https://api.github.com/repos/${PIPER_REPO}/releases/latest" 2>/dev/null \
            | grep '"tag_name"' \
            | sed 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/' \
            | head -n 1
    )"
    if [ -z "$latest_tag" ]; then
        eval "$cleanup_piper_tmp"
        echo "  ⚠️  piper: could not query GitHub API for latest release." >&2
        echo "     The TTS feature won't work until piper is installed manually." >&2
        return 0
    fi

    local asset="piper_linux_${arch}.tar.gz"
    local url="https://github.com/${PIPER_REPO}/releases/download/${latest_tag}/${asset}"
    if ! curl -fsSL -o "$piper_tmp/$asset" "$url"; then
        eval "$cleanup_piper_tmp"
        echo "  ⚠️  piper: download failed (tried $url)." >&2
        echo "     The TTS feature won't work until piper is installed manually." >&2
        return 0
    fi

    $SUDO mkdir -p "$PIPER_DEST_DIR"
    # The asset contains a `piper` binary at the top level (verified
    # against piper's 1.2.0 release). --no-same-owner because we may be
    # running as a non-root user invoking sudo tar.
    if ! $SUDO tar -xzf "$piper_tmp/$asset" -C "$piper_tmp" piper; then
        eval "$cleanup_piper_tmp"
        echo "  ⚠️  piper: extract failed — the asset may have changed layout." >&2
        return 0
    fi
    $SUDO install -m 0755 "$piper_tmp/piper" "$PIPER_DEST"
    # Smoke test: --version should print something containing "piper".
    # If it doesn't, the binary is broken (or the asset has changed
    # shape) and we don't want to leave a non-working binary in place.
    if ! "$PIPER_DEST" --version 2>/dev/null | grep -qi "piper"; then
        eval "$cleanup_piper_tmp"
        echo "  ⚠️  piper: downloaded binary did not respond to --version." >&2
        $SUDO rm -f "$PIPER_DEST"
        return 0
    fi
    eval "$cleanup_piper_tmp"
    echo "  ✓ piper installed at $PIPER_DEST"
}

install_piper

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

  Read-clipboard speech (Ctrl+U) needs the piper TTS engine. The
  installer downloaded it to /opt/voces/engines/piper if your
  machine is x86_64 / aarch64 / armv7l. If the download was
  skipped or failed, the wizard's piper-status step will guide
  you to the manual install.

  See https://github.com/${REPO} for docs and troubleshooting.
EOF
