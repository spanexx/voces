/* Code Map: piper install-hint strings
 * (rc1-hotpatch-31: libratbag naming collision)
 *
 * This file holds the human-readable install hints the wizard
 * shows when FindPiperBinary returns empty. They're kept in
 * one place (rather than scattered across the wizard steps)
 * so future copy edits are a one-file diff and so the rc31
 * "libratbag" warning is colocated with the rest of the
 * Debian-specific guidance.
 *
 * Why the rc31 hint is more aggressive than the rc30 one:
 * the Debian "apt install piper" command looks like a
 * natural install path, and the libratbag "piper" package
 * is what the user gets if they run it. We pre-empt the
 * footgun by calling it out as the FIRST option and giving
 * the right command (rhasspy/piper tarball from GitHub
 * releases) right after.
 */
package setup

import "runtime"

// PiperInstallHint is the human-readable text the wizard shows
// when FindPiperBinary returns empty. Kept short and concrete
// — three commands the user can paste in a terminal, plus the
// GitHub releases link for the source build.
//
// rc1-hotpatch-31: the Debian/Ubuntu "apt install piper" line
// is intentionally a footgun now — that package is the
// libratbag gaming-mouse GUI, not the rhasspy/piper TTS. The
// hint is explicit about that so a user who lands on this
// screen and runs the first command doesn't end up with the
// wrong binary (which would pass the rc30 "is piper
// installed?" check but fail at runtime with "Unknown option
// -m" the first time they press Ctrl+U).
const PiperInstallHint = `Piper is a fast, local neural text-to-speech engine. Voces uses it to read transcriptions aloud (for example, when you press Ctrl+U to read the clipboard).

Piper is not installed on this system. Pick one of these options to enable text-to-speech:

  • Debian / Ubuntu / Linux Mint — IMPORTANT:
      Do NOT run "sudo apt install piper" — that's a Python
      GTK app for configuring gaming mice (libratbag/piper),
      not the rhasspy/piper TTS engine. They share the
      binary name and Voces would fail at runtime with
      "Unknown option -m".

      Install the rhasspy/piper binary from GitHub releases
      instead:
          sudo apt install libonnxruntime1 libespeak-ng1
          curl -fsSL -o /tmp/piper.tar.gz \
            https://github.com/rhasspy/piper/releases/latest/download/piper_linux_x86_64.tar.gz
          sudo tar -C /usr/local -xzf /tmp/piper.tar.gz piper
          rm /tmp/piper.tar.gz

  • Fedora / RHEL (no libratbag collision):
      dnf install onnxruntime espeak-ng
      then download piper from the same GitHub releases URL
      and put it in /usr/local/bin/piper

  • Arch / Manjaro (no libratbag collision):
      pacman -S piper-tts

  • Build from source:
      https://github.com/rhasspy/piper

After installing piper, go Back to this step and click Next again to re-check. If you want to skip text-to-speech for now, click "Next" — the rest of the wizard still works; you'll just see a "TTS Unavailable" notification if you press Ctrl+U.`

// PiperInstallHintLinux, PiperInstallHintDarwin, PiperInstallHintWindows
// are platform-specific variants of PiperInstallHint. The
// wizard picks one based on runtime.GOOS. We keep the strings
// as separate constants (not computed) because go vet / gofmt
// keep raw string literals readable in source. The
// build-from-source link is the same across platforms because
// piper ships a prebuilt tarball for all three.
//
// rc1-hotpatch-31: the libratbag naming collision is a
// Debian/Ubuntu problem (the apt repo ships a "piper" package
// that is the libratbag mouse GUI). On Darwin / Windows there
// is no such collision; the install hint there just links to
// the GitHub release.
const (
	PiperInstallHintLinux  = PiperInstallHint
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
