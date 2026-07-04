/* Code Map: App-Managed Paths
 * - DataDir: XDG data dir for the App (with $HOME fallback)
 * - ModelsDir: Models subdirectory under data
 * - WhisperModelPath: Canonical path for a whisper .bin file
 * - PiperVoicePath: Canonical path for a piper .onnx voice
 * - EnginesDir: Bundled engine binaries (env var, exe-walk, system path)
 *
 * CID Index:
 * CID:paths-001 -> DataDir
 * CID:paths-002 -> ModelsDir
 * CID:paths-003 -> WhisperModelPath
 * CID:paths-004 -> PiperVoicePath
 * CID:paths-005 -> EnginesDir
 *
 * Quick lookup: rg -n "CID:paths-" internal/paths/paths.go
 */
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// appDataDirName is the leaf directory under $XDG_DATA_HOME or $HOME.
const appDataDirName = "whisper-voice-util"

// enginesEnvVar overrides the engines directory when set.
const enginesEnvVar = "WVU_ENGINES_DIR"

// systemEnginesDir is the fallback path used by `make install`.
const systemEnginesDir = "/usr/local/share/whisper-voice-util/bin"

// enginesSubdir is the leaf name of the bundled engine directory
// (relative to the binary's location or the env-var root).
const enginesSubdir = "engines"

// CID:paths-001 - DataDir
// Purpose: Returns the App's XDG data directory, creating it if missing.
// Used by: setup.State, paths.ModelsDir, paths.WhisperModelPath, paths.PiperVoicePath.
// Resolves $XDG_DATA_HOME first, then $HOME/.local/share, then os.UserConfigDir fallback.
func DataDir() (string, error) {
	base, err := xdgDataHome()
	if err != nil {
		return "", fmt.Errorf("failed to resolve data dir: %w", err)
	}
	dir := filepath.Join(base, appDataDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create data dir %q: %w", dir, err)
	}
	return dir, nil
}

// CID:paths-002 - ModelsDir
// Purpose: Returns the App's models subdirectory, creating it if missing.
func ModelsDir() (string, error) {
	data, err := DataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(data, "models")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create models dir %q: %w", dir, err)
	}
	return dir, nil
}

// CID:paths-003 - WhisperModelPath
// Purpose: Returns the canonical absolute path for a whisper .bin file.
// Example: WhisperModelPath("ggml-small.en.bin") -> ~/.local/share/whisper-voice-util/models/whisper/ggml-small.en.bin
func WhisperModelPath(name string) (string, error) {
	models, err := ModelsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(models, "whisper", name), nil
}

// CID:paths-004 - PiperVoicePath
// Purpose: Returns the canonical absolute path for a piper .onnx voice file.
// Example: PiperVoicePath("en_US-lessac-medium") -> ~/.local/share/whisper-voice-util/models/piper/en_US-lessac-medium.onnx
func PiperVoicePath(base string) (string, error) {
	models, err := ModelsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(models, "piper", base+".onnx"), nil
}

// CID:paths-005 - EnginesDir
// Purpose: Returns the directory containing the bundled engine binaries.
// Resolution order:
//  1. $WVU_ENGINES_DIR (if set and non-empty)
//  2. <exec-parent>/engines (binary at <dir>/bin/whisper-voice-util)
//  3. <exec-dir>/engines (binary at <dir>/whisper-voice-util)
//  4. /usr/local/share/whisper-voice-util/bin (make install fallback)
// Returns an error if none of the above resolve to a directory that exists.
func EnginesDir() (string, error) {
	exe, err := os.Executable()
	if err == nil {
		if dir, ok := enginesDirFrom(exe); ok {
			return dir, nil
		}
	}
	return enginesDirFromSystem()
}

// enginesDirFrom searches parent directories of exe for an engines/ sibling.
// Returns (path, true) on hit, ("", false) on miss.
func enginesDirFrom(exe string) (string, bool) {
	if v := strings.TrimSpace(os.Getenv(enginesEnvVar)); v != "" {
		if info, err := os.Stat(v); err == nil && info.IsDir() {
			return v, true
		}
	}
	for _, parent := range enginesCandidates(exe) {
		candidate := filepath.Join(parent, enginesSubdir)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

// enginesDirFromSystem is the make install fallback.
func enginesDirFromSystem() (string, error) {
	if info, err := os.Stat(systemEnginesDir); err == nil && info.IsDir() {
		return systemEnginesDir, nil
	}
	return "", fmt.Errorf("engines directory not found (set %s or place it next to the binary)", enginesEnvVar)
}

// xdgDataHome returns $XDG_DATA_HOME, or $HOME/.local/share, or the
// os.UserConfigDir() parent as a last-resort fallback. This avoids the
// data/config confusion that os.UserConfigDir alone would cause.
func xdgDataHome() (string, error) {
	if v := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}

// enginesCandidates returns the parent directories to try for an engines/
// sibling of the running binary, in priority order. The binary is expected
// to live at <parent>/bin/whisper-voice-util (try parent) or
// <parent>/whisper-voice-util (try parent).
func enginesCandidates(exe string) []string {
	dir := filepath.Dir(exe)
	candidates := []string{dir}
	// If exe is in a bin/ subdir, the project root is the parent.
	if filepath.Base(dir) == "bin" {
		candidates = append(candidates, filepath.Dir(dir))
	}
	return candidates
}
