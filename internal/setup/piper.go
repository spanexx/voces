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
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CID:setup-piper-001 - FindPiperBinary
// Purpose: locate the piper binary on the system, returning the
// absolute path of the first match that is the rhasspy/piper
// TTS engine, or empty string if none is found. Search order:
//
//   1. $PATH via os/exec.LookPath (covers the "user installed
//      piper via their distro's package manager and it's on
//      PATH" case — e.g. `apt install piper-tts` on Debian).
//   2. piperCandidatePaths (default order: /opt/voces/engines/piper,
//      /usr/local/bin/piper, /usr/bin/piper).
//
// Returning the absolute path (not just "found" or "not found")
// lets the wizard print it in the success state so the user can
// see where Voces expects piper to be. Each candidate is also
// validated by isPiperTTS so we don't pick up the libratbag
// "piper" Debian package (rc1-hotpatch-31), which is a Python
// GTK app for configuring gaming mice that happens to share
// the binary name. The candidate list is a package-level
// variable (not a constant) so tests can swap it for a
// temp-dir path (see piper_test.go).
//
// Real-file check, not a mock (per the no-mocks gate).
func FindPiperBinary() string {
	if p, err := exec.LookPath("piper"); err == nil {
		if isExecutable(p) && isPiperTTS(p) {
			return p
		}
	}
	for _, c := range piperCandidatePaths {
		if isExecutable(c) && isPiperTTS(c) {
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

// CID:setup-piper-003 - ResolvePiperBinaryPath
// Purpose: pick the piper binary path the wizard writes into
// the generated config.yaml. Two-tier strategy:
//
//  1. FindPiperBinary() — the same detection the wizard's
//     piper-status step uses, so the config matches what
//     the user just saw. Catches the "I installed piper
//     system-wide via apt/dnf/pacman and it's on $PATH"
//     case (rc30) where the bundled <engines>/piper path
//     doesn't exist.
//  2. <engines>/piper — the bundled default. Kept as the
//     fallback so the release tarball still works out of
//     the box on a system where piper is only present in
//     the voces engines dir (Phase 8.1: bundled piper).
//
// No mocking: a real file is the source of truth (per the
// no-fakes gate). Used by setup.defaultConfigFor; not
// directly exported to other packages — only the wizard
// commit step needs it.
func ResolvePiperBinaryPath(bundledEnginesDir string) string {
	if p := FindPiperBinary(); p != "" {
		return p
	}
	return filepath.Join(bundledEnginesDir, "piper")
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

// CID:setup-piper-004 - isPiperTTS (rc1-hotpatch-31)
// Purpose: distinguish the rhasspy/piper TTS engine from other
// "piper" binaries that happen to share the binary name. The
// Debian package "piper" (libratbag/piper) is a Python GTK
// app for configuring gaming mice; it has no `-m`/`--model`
// flag and printing "Unknown option -m" on first Speak() call
// is the symptom that prompted rc31.
//
// Detection strategy: run the candidate binary with `--version`
// and check the response. The rhasspy/piper binary prints
// something like "piper v1.2.0" to stdout and exits 0. The
// libratbag piper binary prints "Unknown option --version" to
// stderr and exits 0. A binary that doesn't even recognise
// `--version` is not the rhasspy/piper we want.
//
// Edge cases handled:
//   - binary that crashes on --version: returns false (treated
//     as "not piper"). The fallback in piper_status.go shows
//     the install hint, and the user can install the real one.
//   - binary that hangs on --version: the 2s context timeout
//     kills it and we return false (better a false negative
//     that the user can fix than a wizard that hangs).
//   - binary that exits non-zero: returns false. The rhasspy
//     binary always exits 0 on --version.
//
// Real subprocess invocation, no mocks. The piper binary is
// the system-under-test, so the test that exercises this
// path is piper_test.go (drops a fake libratbag script in
// t.TempDir() and verifies isPiperTTS returns false).
func isPiperTTS(path string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	// Cancel (Go 1.20+) sends SIGKILL to the child process
	// when the context is cancelled. Without it, the parent
	// shell exits on context timeout but `sleep` / the
	// hung binary keeps running — `cmd.CombinedOutput()`
	// then blocks until the child finishes, which is the
	// exact bug TestIsPiperTTS_RejectsHang guards against.
	cmd.Cancel = func() error {
		return cmd.Process.Kill()
	}
	cmd.WaitDelay = 500 * time.Millisecond
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return false
	}
	if err != nil {
		return false
	}
	text := strings.ToLower(string(out))
	// libratbag piper signals "wrong binary" with the GTK help
	// text (gapplication / --help-gtk) and "Unknown option
	// --version". Reject either shape.
	if strings.Contains(text, "gapplication") ||
		strings.Contains(text, "help-gtk") ||
		strings.Contains(text, "help-gapplication") ||
		strings.Contains(text, "unknown option") {
		return false
	}
	// rhasspy/piper prints "piper vX.Y.Z" or similar on
	// --version. Accept the candidate when the output mentions
	// "piper" and doesn't have any of the rejection signals.
	// We keep this lenient (just "piper" in the text) because
	// the upstream version string format has changed over the
	// years — strict matching would be brittle.
	return strings.Contains(text, "piper")
}

// PiperInstallHint and the platform-specific variants live
// in piper_install_hint.go (extracted for the 250-line cap).
// PiperInstallHintForOS is also there.
