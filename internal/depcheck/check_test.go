package depcheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// CID:depcheck-test-001 - test helpers
// swapLookPath swaps execLookPath for the duration of the test.
// It is restored automatically via t.Cleanup.
func swapLookPath(t *testing.T, fn func(name string) (string, error)) {
	t.Helper()
	orig := execLookPath
	execLookPath = fn
	t.Cleanup(func() { execLookPath = orig })
}

// swapLibDirs swaps libDirsFn for the duration of the test.
func swapLibDirs(t *testing.T, dirs []string) {
	t.Helper()
	orig := libDirsFn
	libDirsFn = func() []string { return dirs }
	t.Cleanup(func() { libDirsFn = orig })
}

// fakeLookPath builds a LookPath replacement that treats the named
// binaries as missing and reports all others as present at /usr/bin/<name>.
func fakeLookPath(missing map[string]bool) func(string) (string, error) {
	return func(name string) (string, error) {
		if missing[name] {
			return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
		}
		return "/usr/bin/" + name, nil
	}
}

// CID:depcheck-test-002 - TestRun_ShapeIsValid
// Purpose: Run() returns without error and every reported missing dep
// carries a non-empty Name and FixCommand. The test passes whether the
// host is fully provisioned or not — we are not asserting "zero missing",
// only "if missing is reported, it is well-formed".
func TestRun_ShapeIsValid(t *testing.T) {
	missing, err := Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, m := range missing {
		if m.Name == "" {
			t.Errorf("missing dep has empty Name")
		}
		if m.FixCommand == "" {
			t.Errorf("missing dep %q has empty FixCommand", m.Name)
		}
		if !strings.HasPrefix(m.FixCommand, "sudo apt install ") {
			t.Errorf("missing dep %q FixCommand = %q, want it to start with %q",
				m.Name, m.FixCommand, "sudo apt install ")
		}
	}
}

// CID:depcheck-test-003 - TestRun_FakePathMissingXclip
// Purpose: with xclip absent from PATH, Run() reports a MissingDep whose
// name is "xclip" and whose fix command is "sudo apt install xclip".
func TestRun_FakePathMissingXclip(t *testing.T) {
	swapLookPath(t, fakeLookPath(map[string]bool{"xclip": true}))

	missing, err := Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := findMissing(t, missing, "xclip")
	if got.FixCommand != "sudo apt install xclip" {
		t.Errorf("xclip FixCommand = %q, want %q", got.FixCommand, "sudo apt install xclip")
	}
	if !got.Required {
		t.Errorf("xclip should be Required")
	}
	if got.Reason == "" {
		t.Errorf("xclip should have a Reason")
	}
}

// CID:depcheck-test-004 - TestRun_FakePathMissingXdotool
// Purpose: same as xclip but for xdotool.
func TestRun_FakePathMissingXdotool(t *testing.T) {
	swapLookPath(t, fakeLookPath(map[string]bool{"xdotool": true}))

	missing, err := Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := findMissing(t, missing, "xdotool")
	if got.FixCommand != "sudo apt install xdotool" {
		t.Errorf("xdotool FixCommand = %q, want %q", got.FixCommand, "sudo apt install xdotool")
	}
	if !got.Required {
		t.Errorf("xdotool should be Required")
	}
}

// CID:depcheck-test-005 - TestRun_FakeLibPathMissingAayatana
// Purpose: with the lib search path pointed at an empty temp dir,
// libayatana-appindicator3 is reported missing with the dev package name.
func TestRun_FakeLibPathMissingAayatana(t *testing.T) {
	swapLibDirs(t, []string{t.TempDir()})

	missing, err := Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := findMissing(t, missing, "libayatana-appindicator3")
	if got.FixCommand != "sudo apt install libayatana-appindicator3-dev" {
		t.Errorf("fix = %q, want apt-dev pkg", got.FixCommand)
	}
	if !got.Required {
		t.Errorf("libayatana-appindicator3 should be Required")
	}
}

// CID:depcheck-test-006 - TestRun_FakeLibPathMissingX11AndXtst
// Purpose: both X11 libraries are missing when the search dir is empty.
func TestRun_FakeLibPathMissingX11AndXtst(t *testing.T) {
	swapLibDirs(t, []string{t.TempDir()})

	missing, err := Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	x11 := findMissing(t, missing, "libX11")
	if x11.FixCommand != "sudo apt install libx11-dev" {
		t.Errorf("libX11 fix = %q", x11.FixCommand)
	}

	xtst := findMissing(t, missing, "libXtst")
	if xtst.FixCommand != "sudo apt install libxtst-dev" {
		t.Errorf("libXtst fix = %q", xtst.FixCommand)
	}
}

// CID:depcheck-test-007 - TestRun_FakePathAndLib_AllMissing
// Purpose: when both PATH and lib dir are empty, every required dep is
// reported missing in the order allDeps() declares them.
func TestRun_FakePathAndLib_AllMissing(t *testing.T) {
	swapLookPath(t, fakeLookPath(allBinariesAsMissing()))
	swapLibDirs(t, []string{t.TempDir()})

	missing, err := Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(missing) != len(allDeps()) {
		names := make([]string, 0, len(missing))
		for _, m := range missing {
			names = append(names, m.Name)
		}
		t.Fatalf("expected %d missing, got %d: %v", len(allDeps()), len(missing), names)
	}
	for i, m := range missing {
		if m.Name != allDeps()[i].Name {
			t.Errorf("missing[%d].Name = %q, want %q (order must match allDeps)", i, m.Name, allDeps()[i].Name)
		}
	}
}

// CID:depcheck-test-008 - TestRun_UnknownKind_ReturnsError
// Purpose: a malformed Dep with an unknown Kind propagates as an error,
// not as a silent miss.
func TestRun_UnknownKind_ReturnsError(t *testing.T) {
	// We can't reach allDeps() from outside; inject a broken entry by
	// temporarily replacing the package-level list.
	origDeps := allDeps
	allDeps = func() []Dep {
		return []Dep{{Name: "broken", Kind: "wat", Probe: "x"}}
	}
	t.Cleanup(func() { allDeps = origDeps })

	if _, err := Run(); err == nil {
		t.Errorf("expected error for unknown Kind, got nil")
	}
}

// CID:depcheck-test-009 - TestFixCommandFor
// Purpose: every known dep maps to a stable install line; unknown
// names return the empty string.
func TestFixCommandFor(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"xclip", "sudo apt install xclip"},
		{"xdotool", "sudo apt install xdotool"},
		{"libayatana-appindicator3", "sudo apt install libayatana-appindicator3-dev"},
		{"libX11", "sudo apt install libx11-dev"},
		{"libXtst", "sudo apt install libxtst-dev"},
		{"does-not-exist", ""},
		{"", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FixCommandFor(c.name); got != c.want {
				t.Errorf("FixCommandFor(%q) = %q, want %q", c.name, got, c.want)
			}
		})
	}
}

// CID:depcheck-test-010 - TestProbeBinary_RespectsOverride
// Purpose: the override hook makes the probe return (false, nil) for a
// missing binary and (true, nil) for a present one.
func TestProbeBinary_RespectsOverride(t *testing.T) {
	swapLookPath(t, fakeLookPath(map[string]bool{"missing-bin": true}))

	if ok, err := probeBinary("missing-bin"); err != nil || ok {
		t.Errorf("probeBinary(missing) = (%v, %v), want (false, nil)", ok, err)
	}
	if ok, err := probeBinary("present-bin"); err != nil || !ok {
		t.Errorf("probeBinary(present) = (%v, %v), want (true, nil)", ok, err)
	}
}

// CID:depcheck-test-011 - TestProbeLibrary_FindsInFirstMatchingDir
// Purpose: the library probe returns true as soon as it finds the file
// in any dir, and false when no dir has it.
func TestProbeLibrary_FindsInFirstMatchingDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "myfake.so.1"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	swapLibDirs(t, []string{"/nonexistent-a", dir, "/nonexistent-b"})

	if ok, err := probeLibrary("myfake.so.1"); err != nil || !ok {
		t.Errorf("probeLibrary(found) = (%v, %v), want (true, nil)", ok, err)
	}
	if ok, err := probeLibrary("not-there.so.1"); err != nil || ok {
		t.Errorf("probeLibrary(missing) = (%v, %v), want (false, nil)", ok, err)
	}
}

// CID:depcheck-test-012 - TestProbeLibrary_RejectsDirectory
// Purpose: a directory with the same name as the SONAME must not be
// reported as a present library.
func TestProbeLibrary_RejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "afake.so.1"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	swapLibDirs(t, []string{dir})

	if ok, err := probeLibrary("afake.so.1"); err != nil || ok {
		t.Errorf("probeLibrary(dir) = (%v, %v), want (false, nil)", ok, err)
	}
}

// findMissing looks up a missing dep by name and fails the test if absent.
func findMissing(t *testing.T, missing []MissingDep, name string) MissingDep {
	t.Helper()
	for _, m := range missing {
		if m.Name == name {
			return m
		}
	}
	names := make([]string, 0, len(missing))
	for _, m := range missing {
		names = append(names, m.Name)
	}
	t.Fatalf("dep %q not in missing list (got %v)", name, names)
	return MissingDep{}
}

// allBinariesAsMissing returns a "missing" set covering every binary in
// allDeps(). Used by the all-missing test.
func allBinariesAsMissing() map[string]bool {
	out := map[string]bool{}
	for _, d := range allDeps() {
		if d.Kind == "binary" {
			out[d.Probe] = true
		}
	}
	return out
}
