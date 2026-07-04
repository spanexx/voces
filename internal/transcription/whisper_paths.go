/* Code Map: Whisper.cpp path resolution + validation
 * - Validate: reports missing binary/model
 * - resolvePathsIfNeeded: auto-discovers the managed /usr/local layout
 *   when configured paths are empty or stale
 * - resolveWhisperBinaryAndModel: the discovery function
 * - fileExists: cheap non-directory stat
 * - isRunningUnderGoTest: suppresses auto-discovery in `go test`
 *
 * Sibling files in this package:
 * - whisper_cpp.go:   core engine + Transcribe
 * - whisper_output.go: output parsing + error formatting
 *
 * CID Index:
 * CID:transcription-whisper-paths-001 -> Validate
 * CID:transcription-whisper-paths-002 -> resolvePathsIfNeeded
 * CID:transcription-whisper-paths-003 -> resolveWhisperBinaryAndModel
 * CID:transcription-whisper-paths-004 -> fileExists
 * CID:transcription-whisper-paths-005 -> isRunningUnderGoTest
 *
 * Quick lookup: rg -n "CID:transcription-whisper-paths-" internal/transcription/
 */
package transcription

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CID:transcription-whisper-paths-001 - Validate
// Purpose: checks if the whisper.cpp binary and model are accessible.
func (w *WhisperCPP) Validate() error {
	w.resolvePathsIfNeeded()

	if _, err := os.Stat(w.binaryPath); err != nil {
		return fmt.Errorf("whisper.cpp binary not found: %s", w.binaryPath)
	}
	if _, err := os.Stat(w.modelPath); err != nil {
		return fmt.Errorf("whisper model not found: %s", w.modelPath)
	}
	return nil
}

// CID:transcription-whisper-paths-002 - resolvePathsIfNeeded
func (w *WhisperCPP) resolvePathsIfNeeded() {
	if w == nil {
		return
	}
	if isRunningUnderGoTest() {
		return
	}

	binMissing := w.binaryPath == ""
	if !binMissing {
		_, err := os.Stat(w.binaryPath)
		binMissing = err != nil
	}

	modelMissing := w.modelPath == ""
	if !modelMissing {
		_, err := os.Stat(w.modelPath)
		modelMissing = err != nil
	}

	if !binMissing && !modelMissing {
		return
	}

	resolvedBin, resolvedModel := resolveWhisperBinaryAndModel(w.binaryPath, w.modelPath)
	if resolvedBin != "" {
		w.binaryPath = resolvedBin
	}
	if resolvedModel != "" {
		w.modelPath = resolvedModel
	}
}

// CID:transcription-whisper-paths-003 - resolveWhisperBinaryAndModel
func resolveWhisperBinaryAndModel(configuredBin, configuredModel string) (string, string) {
	managedBin := "/usr/local/share/voces/whisper.cpp/bin/whisper-cli"
	managedModel := "/usr/local/share/voces/whisper.cpp/models/ggml-base.en.bin"

	bin := configuredBin
	if bin == "" || !fileExists(bin) {
		if fileExists(managedBin) {
			bin = managedBin
		}
	}

	model := configuredModel
	if model == "" || !fileExists(model) {
		if fileExists(managedModel) {
			model = managedModel
		} else if bin != "" {
			binDir := filepath.Dir(bin)
			rel := []string{
				filepath.Join(binDir, "..", "models", "ggml-base.en.bin"),
				filepath.Join(binDir, "models", "ggml-base.en.bin"),
			}
			for _, c := range rel {
				if fileExists(c) {
					model = c
					break
				}
			}
		}
	}

	return bin, model
}

// CID:transcription-whisper-paths-004 - fileExists
func fileExists(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	if err != nil {
		return false
	}
	return !st.IsDir()
}

// CID:transcription-whisper-paths-005 - isRunningUnderGoTest
func isRunningUnderGoTest() bool {
	// `go test` builds an executable named `<pkg>.test`.
	// We don't want environment-dependent auto-discovery to change unit test behavior.
	return strings.HasSuffix(os.Args[0], ".test")
}
