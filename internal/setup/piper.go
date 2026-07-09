/* Code Map: Piper Binary Detection
 * - FindPiperBinary: returns the absolute path of the piper
 *   binary if it can be located on the system, or empty
 *   string if piper is not installed.
 * - PiperInstallInstructions: platform-specific install
 *   commands the wizard shows when piper is missing.
 *
 * rc1-hotpatch-29: the wizard's "Piper status" step uses
 * FindPiperBinary to decide whether to show a friendly
 * install-help block or just the voice picker. The handler
 * in internal/app/handlers.go already gates Piper.Speak on
 * tts.Available(); this is the wizard-side equivalent that
 * keeps the user out of the trap of "yes, install a voice"
 * without a working piper engine.
 *
 * CID Index:
 * CID:setup-piper-001 -> FindPiperBinary
 * CID:setup-piper-002 -> PiperInstallInstructions
 *
 * Quick lookup: rg -n "CID:setup-piper-" internal/setup/
 */
package setup

import (
	"os"
	"os/exec"
	"runtime"
)

// CID:setup-piper-001 - FindPiperBinary
// Purpose: locate the piper binary on the system, returning the
// absolute path to the first match, or empty string if piper is
// not installed. Search order:
//
//   1. $PATH via os/exec.LookPath (covers the "user installed
//      piper via their distro's package manager and it's on
//      PATH" case — e.g. `apt install piper-tts` on Debian).
//   2. piperCandidatePaths (default order: /opt/voces/engines/piper,
//      /usr/local/bin/piper, /usr/bin/piper).
//
// Returning the absolute path (not just "found" or "not found")
// lets the wizard print it in the success state so the user can
// see where Voces expects piper to be. The check is os.Stat +
// executable bit; we don't run the binary here — that's the
// tts.Available() call's job. The candidate list is a
// package-level variable (not a constant) so tests can swap it
// for a temp-dir path (see piper_test.go).
//
// Real-file check, not a mock (per the no-mocks gate).
func FindPiperBinary() string {
	if p, err := exec.LookPath("piper"); err == nil {
		if isExecutable(p) {
			return p
		}
	}
	for _, c := range piperCandidatePaths {
		if isExecutable(c) {
			return c
		}
	}
	return ""
}

// piperCandidatePaths is the documented fallback search list
// for FindPiperBinary. Tests override this (see piper_test.go)
// to point at t.TempDir() binaries. /opt/voces/engines/piper
// is the canonical install path the wizard's own model
// download step uses; the /usr/local/bin and /usr/bin
// entries cover the FHS-local and FHS-system cases.
var piperCandidatePaths = []string{
	"/opt/voces/engines/piper",
	"/usr/local/bin/piper",
	"/usr/bin/piper",
}

// isExecutable reports whether path exists and is a regular
// file with at least one executable bit set. Uses os.Stat —
// real filesystem, no mocks.
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	mode := info.Mode()
	return mode&0111 != 0
}

// PiperInstallHint is the human-readable text the wizard shows
// when FindPiperBinary returns empty. Kept short and concrete
// — three commands the user can paste in a terminal, plus the
// GitHub releases link for the source build.
const PiperInstallHint = `Piper is a fast, local neural text-to-speech engine. Voces uses it to read transcriptions aloud (for example, when you press Ctrl+U to read the clipboard).

Piper is not installed on this system. Pick one of these options to enable text-to-speech:

  • Debian / Ubuntu / Linux Mint:
      sudo apt-get install piper
      (or: sudo apt-get install piper-tts on newer releases)

  • Fedora / RHEL:
      sudo dnf install piper

  • Arch / Manjaro:
      sudo pacman -S piper

  • Build from source:
      https://github.com/rhasspy/piper/releases

After installing piper, go Back to this step and click Next again to re-check. If you want to skip text-to-speech for now, click "Next" — the rest of the wizard still works; you'll just see a "TTS Unavailable" notification if you press Ctrl+U.`

// PiperInstallHintLinux, PiperInstallHintDarwin, PiperInstallHintWindows
// are platform-specific variants of PiperInstallHint. The
// wizard picks one based on runtime.GOOS. We keep the strings
// as separate constants (not computed) because go vet / gofmt
// keep raw string literals readable in source. The
// build-from-source link is the same across platforms because
// piper ships a prebuilt tarball for all three.
const (
	PiperInstallHintLinux = PiperInstallHint
	PiperInstallHintDarwin = `Piper is a fast, local neural text-to-speech engine.

Piper is not installed on this system. The simplest install is via Homebrew:

  brew install piper

Or build from source:
  https://github.com/rhasspy/piper/releases

After installing, go Back and click Next again to re-check.`
	PiperInstallHintWindows = `Piper is a fast, local neural text-to-speech engine.

Piper is not installed on this system. Download the latest prebuilt Windows release from:

  https://github.com/rhasspy/piper/releases

Extract piper.exe somewhere on your PATH (or in C:\Program Files\piper) and re-run the wizard.`
)

// PiperInstallHintForOS returns the platform-appropriate install
// hint. Centralised here so the wizard step doesn't import
// runtime (and so a future change to add more platform hints
// is a one-file diff).
func PiperInstallHintForOS() string {
	switch runtime.GOOS {
	case "darwin":
		return PiperInstallHintDarwin
	case "windows":
		return PiperInstallHintWindows
	default:
		return PiperInstallHintLinux
	}
}
