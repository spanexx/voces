/* Code Map: rc31 fix tests — reject non-TTS pipers
 * (rc1-hotpatch-31: libratbag naming collision)
 *
 * - TestIsPiperTTS_AcceptsRHasspyPiper: a binary that prints
 *   "piper vX.Y.Z" on --version is identified as the TTS
 *   engine. This is the path forward when the user installs
 *   the rhasspy/piper binary from GitHub releases.
 * - TestIsPiperTTS_RejectsLibratbagPiper: the libratbag
 *   gaming-mouse GUI prints "Unknown option --version" (and
 *   GTK help text) on --version. isPiperTTS must return false
 *   so FindPiperBinary does not surface it.
 * - TestIsPiperTTS_RejectsGApplicationHelp: a candidate that
 *   prints GApplication / --help-gtk (a different GTK app
 *   that happens to call itself "piper") is also rejected.
 *   Belt-and-suspenders: we don't just check for "Unknown
 *   option" — we also check for the GTK / GApplication
 *   markers.
 * - TestIsPiperTTS_RejectsNonZeroExit: a binary that exits
 *   non-zero on --version (crash, missing library) is
 *   rejected. Rhasspy/piper always exits 0 on --version.
 * - TestIsPiperTTS_RejectsHang: a binary that hangs (sleeps
 *   forever) on --version is killed by the 2s context
 *   timeout and rejected. Belt-and-suspenders against a
 *   misbehaving candidate.
 * - TestFindPiperBinary_SkipsLibratbagPiper: integration
 *   test — when piperCandidatePaths points at a libratbag
 *   piper, FindPiperBinary returns empty. The wizard then
 *   shows the install hint, where rc31's libratbag warning
 *   guides the user to the right download.
 * - TestPiperInstallHintLinux_MentionsLibratbag: the linux
 *   install hint must explicitly warn Debian/Ubuntu users
 *   about the libratbag collision (rc1-hotpatch-31) and
 *   give the rhasspy/piper install commands. The wizard
 *   tests under empty DISPLAY already check that the
 *   piper_status step renders correctly; this test is the
 *   source-of-truth check on the hint text itself.
 *
 * CID Index:
 * CID:setup-rc31test-001 -> TestIsPiperTTS_AcceptsRHasspyPiper
 * CID:setup-rc31test-002 -> TestIsPiperTTS_RejectsLibratbagPiper
 * CID:setup-rc31test-003 -> TestIsPiperTTS_RejectsGApplicationHelp
 * CID:setup-rc31test-004 -> TestIsPiperTTS_RejectsNonZeroExit
 * CID:setup-rc31test-005 -> TestIsPiperTTS_RejectsHang
 * CID:setup-rc31test-006 -> TestFindPiperBinary_SkipsLibratbagPiper
 * CID:setup-rc31test-007 -> TestPiperInstallHintLinux_MentionsLibratbag
 */
package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeScript drops a shell script at path with the given
// content and the +x bit. The piper binary in libratbag is a
// Python script with a shebang, but for tests we just need a
// script that answers --version the way the real libratbag
// piper does. Real file system, no fakes (per the precommit
// gate that bans fake paths in tests).
func writeScript(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script %s: %v", path, err)
	}
}

// CID:setup-rc31test-001 - TestIsPiperTTS_AcceptsRHasspyPiper
// Purpose: a binary that prints "piper vX.Y.Z" on --version
// is the rhasspy/piper TTS engine and must be accepted.
// Covers the happy path of the rc31 fix.
func TestIsPiperTTS_AcceptsRHasspyPiper(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "piper")
	writeScript(t, bin, "#!/bin/sh\necho \"piper v1.2.0\"\nexit 0\n")
	if !isPiperTTS(bin) {
		t.Errorf("isPiperTTS(%q): got false, want true (rhasspy/piper prints version)", bin)
	}
}

// CID:setup-rc31test-002 - TestIsPiperTTS_RejectsLibratbagPiper
// Purpose: the libratbag gaming-mouse GUI (Debian package
// "piper") prints "Unknown option --version" on stderr and
// exits 0. isPiperTTS must return false so we don't pick
// it up as the rhasspy/piper TTS engine.
//
// This is the regression test for the user's "Unknown option
// -m" error after rc30 — the rc31 fix means the libratbag
// piper never gets past the detection stage.
func TestIsPiperTTS_RejectsLibratbagPiper(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "piper")
	// The real libratbag piper writes "Unknown option --version"
	// to stderr and exits 0. We mirror that here.
	writeScript(t, bin, "#!/bin/sh\necho \"Unknown option --version\" >&2\nexit 0\n")
	if isPiperTTS(bin) {
		t.Errorf("isPiperTTS(%q): got true, want false (libratbag piper should be rejected)", bin)
	}
}

// CID:setup-rc31test-003 - TestIsPiperTTS_RejectsGApplicationHelp
// Purpose: a candidate that prints the GApplication / GTK
// help text is a GTK app (could be libratbag piper, could be
// some other program that happens to be named "piper"). The
// rc31 detection checks for the GTK markers explicitly so
// that a future GTK-based impostor (e.g. someone repackaging
// piper with a different shell) is also rejected.
func TestIsPiperTTS_RejectsGApplicationHelp(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "piper")
	// Mimic libratbag piper's --help output (truncated).
	helpText := "Usage:\n  piper [OPTION…]\n\nHelp Options:\n  -h, --help                 Show help options\n  --help-all                 Show all help options\n  --help-gapplication        Show GApplication options\n  --help-gtk                 Show GTK+ Options\n"
	writeScript(t, bin, "#!/bin/sh\necho \""+helpText+"\"\nexit 0\n")
	if isPiperTTS(bin) {
		t.Errorf("isPiperTTS(%q): got true, want false (GTK app should be rejected)", bin)
	}
}

// CID:setup-rc31test-004 - TestIsPiperTTS_RejectsNonZeroExit
// Purpose: a binary that exits non-zero on --version (crash,
// missing library, anything that prevents the version string
// from reaching us) is not the rhasspy/piper TTS engine.
// The rhasspy binary always exits 0 on --version.
func TestIsPiperTTS_RejectsNonZeroExit(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "piper")
	writeScript(t, bin, "#!/bin/sh\necho \"piper v1.2.0 (crash)\" >&2\nexit 1\n")
	if isPiperTTS(bin) {
		t.Errorf("isPiperTTS(%q): got true, want false (non-zero exit on --version is rejected)", bin)
	}
}

// CID:setup-rc31test-005 - TestIsPiperTTS_RejectsHang
// Purpose: a binary that hangs on --version (the user has
// installed some custom wrapper that blocks indefinitely)
// is killed by the 2s context timeout and rejected. Better
// a false negative that the user can fix than a wizard that
// hangs for 30s on every show.
func TestIsPiperTTS_RejectsHang(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "piper")
	writeScript(t, bin, "#!/bin/sh\nsleep 30\nexit 0\n")
	start := time.Now()
	got := isPiperTTS(bin)
	elapsed := time.Since(start)
	if got {
		t.Errorf("isPiperTTS(hang): got true, want false")
	}
	// 2s timeout + a generous slack for process startup
	// and kill propagation. 4s is the upper bound.
	if elapsed > 4*time.Second {
		t.Errorf("isPiperTTS(hang): elapsed %v, want <= 4s (context timeout is 2s)", elapsed)
	}
}

// CID:setup-rc31test-006 - TestFindPiperBinary_SkipsLibratbagPiper
// Purpose: end-to-end check. piperCandidatePaths points at a
// libratbag piper (the user's reported setup). FindPiperBinary
// must return empty, not the libratbag path. The wizard's
// piper-status step then shows the install hint with the
// libratbag warning.
func TestFindPiperBinary_SkipsLibratbagPiper(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "piper")
	writeScript(t, bin, "#!/bin/sh\necho \"Unknown option --version\" >&2\nexit 0\n")
	origCandidates := piperCandidatePaths
	piperCandidatePaths = []string{bin}
	t.Cleanup(func() { piperCandidatePaths = origCandidates })
	t.Setenv("PATH", t.TempDir())

	if got := FindPiperBinary(); got != "" {
		t.Errorf("FindPiperBinary (libratbag): got %q, want empty string", got)
	}
}

// CID:setup-rc31test-007 - TestPiperInstallHintLinux_MentionsLibratbag
// Purpose: the install hint the wizard shows on Debian/Ubuntu
// must explicitly call out the libratbag naming collision
// (rc1-hotpatch-31) and give the rhasspy/piper install
// commands. Without this, a user who lands on the
// "piper not installed" screen would just run "apt install
// piper" and end up with the wrong binary again.
func TestPiperInstallHintLinux_MentionsLibratbag(t *testing.T) {
	hint := PiperInstallHintLinux
	mustContain := []string{
		"libratbag",      // names the wrong project
		"rhasspy/piper",  // names the right project
		"github.com/rhasspy/piper/releases", // gives the install URL
	}
	for _, s := range mustContain {
		if !strings.Contains(hint, s) {
			t.Errorf("PiperInstallHintLinux: missing %q\n---hint---\n%s\n---", s, hint)
		}
	}
	// And the old "sudo apt-get install piper" line must be
	// gone (or at minimum, the "DO NOT" framing must appear).
	if strings.Contains(hint, "apt-get install piper\n") ||
		strings.Contains(hint, "apt install piper\n") {
		t.Errorf("PiperInstallHintLinux: still suggests 'apt install piper' as a valid path (rc31 regression):\n%s", hint)
	}
}
