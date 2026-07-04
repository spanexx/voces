/* Code Map: System Dependency Checker
 * - Dep: a single required system dependency (binary or shared library)
 * - MissingDep: result of a probe (name, fix command, required flag)
 * - Run: probe all deps and return the missing ones
 * - FixCommandFor: maps dep name to the apt install line
 *
 * CID Index:
 * CID:depcheck-001 -> Dep
 * CID:depcheck-002 -> MissingDep
 * CID:depcheck-003 -> allDeps
 * CID:depcheck-004 -> Run
 * CID:depcheck-005 -> FixCommandFor
 * CID:depcheck-006 -> probeBinary
 * CID:depcheck-007 -> probeLibrary
 *
 * Quick lookup: rg -n "CID:depcheck-" internal/depcheck/
 */
package depcheck

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// libSearchDirs is the canonical list of shared-library search paths on
// Debian/Ubuntu-derived distros. We do not consult ldconfig; this is the
// subset that covers the binaries built on this project.
var libSearchDirs = []string{
	"/usr/lib/x86_64-linux-gnu",
	"/usr/lib64",
	"/usr/lib",
	"/lib/x86_64-linux-gnu",
	"/lib64",
	"/lib",
}

// execLookPath is the function used to resolve a binary on PATH.
// Tests may override it to simulate a missing or present binary without
// mutating the real process environment.
var execLookPath = exec.LookPath

// libDirsFn returns the list of directories the library probe should search.
// Tests may override it to point at a temp dir.
var libDirsFn = func() []string { return libSearchDirs }

// CID:depcheck-001 - Dep
// Purpose: One required system dependency, either a binary (probed via PATH)
// or a shared library (probed by file existence under libSearchDirs).
type Dep struct {
	Name    string // human label shown in error UI
	Kind    string // "binary" or "lib"
	Probe   string // binary name (for Kind=binary) or library file name (for Kind=lib)
	AptPkg  string // apt package to install
	Reason  string // one-line user-facing explanation
	Required bool  // true = wizard blocks; false = warning only
}

// CID:depcheck-002 - MissingDep
// Purpose: A single failed probe. Returned to the wizard for display.
type MissingDep struct {
	Name       string
	FixCommand string
	Required   bool
	Reason     string
}

// CID:depcheck-003 - allDeps
// Purpose: The full set of required system dependencies for the App.
// Declared as a var (not a func) so tests can substitute a fixture
// without mutating the package's compiled code.
var allDeps = func() []Dep {
	return []Dep{
		{
			Name:    "xclip",
			Kind:    "binary",
			Probe:   "xclip",
			AptPkg:  "xclip",
			Reason:  "copy transcription to the clipboard",
			Required: true,
		},
		{
			Name:    "xdotool",
			Kind:    "binary",
			Probe:   "xdotool",
			AptPkg:  "xdotool",
			Reason:  "auto-type the transcription into the focused window",
			Required: true,
		},
		{
			Name:    "libayatana-appindicator3",
			Kind:    "lib",
			Probe:   "libayatana-appindicator3.so.1",
			AptPkg:  "libayatana-appindicator3-dev",
			Reason:  "show the system tray icon",
			Required: true,
		},
		{
			Name:    "libX11",
			Kind:    "lib",
			Probe:   "libX11.so.6",
			AptPkg:  "libx11-dev",
			Reason:  "capture global hotkeys on X11",
			Required: true,
		},
		{
			Name:    "libXtst",
			Kind:    "lib",
			Probe:   "libXtst.so.6",
			AptPkg:  "libxtst-dev",
			Reason:  "synthesize key events for the hotkey handler",
			Required: true,
		},
	}
}

// CID:depcheck-004 - Run
// Purpose: Probe every required dep and return the missing ones.
// The result is the union across binaries and libraries. Order matches
// allDeps() so the UI is stable.
func Run() ([]MissingDep, error) {
	var missing []MissingDep
	for _, d := range allDeps() {
		var ok bool
		var err error
		switch d.Kind {
		case "binary":
			ok, err = probeBinary(d.Probe)
		case "lib":
			ok, err = probeLibrary(d.Probe)
		default:
			return nil, fmt.Errorf("unknown dep kind %q for %s", d.Kind, d.Name)
		}
		if err != nil {
			return nil, fmt.Errorf("probe %s: %w", d.Name, err)
		}
		if !ok {
			missing = append(missing, MissingDep{
				Name:       d.Name,
				FixCommand: "sudo apt install " + d.AptPkg,
				Required:   d.Required,
				Reason:     d.Reason,
			})
		}
	}
	return missing, nil
}

// CID:depcheck-005 - FixCommandFor
// Purpose: Returns the apt install line for a named dep. Empty string
// if the dep is unknown.
func FixCommandFor(name string) string {
	for _, d := range allDeps() {
		if d.Name == name {
			return "sudo apt install " + d.AptPkg
		}
	}
	return ""
}

// CID:depcheck-006 - probeBinary
// Purpose: Returns true if the binary is found on PATH (via execLookPath).
// The current process's PATH is used; the wizard runs in the user's
// environment, so this is the right scope. Tests override execLookPath.
func probeBinary(name string) (bool, error) {
	_, err := execLookPath(name)
	if err == nil {
		return true, nil
	}
	if strings.Contains(err.Error(), "executable file not found") {
		return false, nil
	}
	return false, err
}

// CID:depcheck-007 - probeLibrary
// Purpose: Returns true if the shared library file exists in any of
// libDirsFn(). We use the SONAME form (e.g. "libfoo.so.1") because
// that's what the dynamic linker resolves to. Tests override libDirsFn.
func probeLibrary(name string) (bool, error) {
	for _, dir := range libDirsFn() {
		candidate := filepath.Join(dir, name)
		if exists, err := fileExists(candidate); err != nil {
			return false, err
		} else if exists {
			return true, nil
		}
	}
	return false, nil
}

// fileExists returns (true, nil) if path exists and is not a directory,
// (false, nil) if the path does not exist, and a real error otherwise.
func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}
