/* Code Map: Whisper.cpp Engine
 * - WhisperCPP: Subprocess-based local transcription engine
 * - NewWhisperCPP: Factory for creating a WhisperCPP engine
 * - Transcribe: Converts audio to text via whisper.cpp binary
 *
 * Sibling files in this package:
 * - whisper_output.go: output parsing + error formatting
 * - whisper_paths.go:  path resolution + validation
 *
 * CID Index:
 * CID:transcription-whisper-001 -> WhisperCPP
 * CID:transcription-whisper-002 -> NewWhisperCPP
 * CID:transcription-whisper-003 -> Transcribe
 *
 * Quick lookup: rg -n "CID:transcription-whisper-" internal/transcription/
 */
package transcription

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"whisper-voice-util/internal/config"
)

// CID:transcription-whisper-001 - WhisperCPP
// Purpose: Implements the Engine interface for the local whisper.cpp utility.
type WhisperCPP struct {
	binaryPath  string
	modelPath   string
	language    string
	computeType string
}

// CID:transcription-whisper-002 - NewWhisperCPP
// Purpose: Initializes a WhisperCPP engine with paths from application config.
func NewWhisperCPP(cfg *config.Config) *WhisperCPP {
	return &WhisperCPP{
		binaryPath:  cfg.Transcription.WhisperCPP.BinaryPath,
		modelPath:   cfg.Transcription.WhisperCPP.Model,
		language:    cfg.Transcription.WhisperCPP.Language,
		computeType: cfg.Transcription.WhisperCPP.ComputeType,
	}
}

// Name returns the engine name.
func (w *WhisperCPP) Name() string {
	return "whisper_cpp"
}

// CID:transcription-whisper-003 - Transcribe
// Purpose: Executes the whisper.cpp binary and parses its text output.
func (w *WhisperCPP) Transcribe(audioPath string) (string, error) {
	w.resolvePathsIfNeeded()

	// Check if binary exists
	if _, err := os.Stat(w.binaryPath); err != nil {
		return "", fmt.Errorf("whisper.cpp binary not found: %s", w.binaryPath)
	}

	// Check if model exists
	if _, err := os.Stat(w.modelPath); err != nil {
		return "", fmt.Errorf("whisper model not found: %s", w.modelPath)
	}

	// whisper.cpp outputs to a .txt file when using -otxt.
	// Some builds (notably the deprecated 'main') may emit output relative to the working directory,
	// so we set -of explicitly and later check both locations.
	outputBase := strings.TrimSuffix(audioPath, filepath.Ext(audioPath))

	// Build command
	args := []string{
		"-m", w.modelPath,
		"-f", audioPath,
		"-otxt",
		"-of", outputBase,
	}

	// Add language if specified
	if w.language != "" {
		args = append(args, "-l", w.language)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	outputFile := outputBase + ".txt"

	executedBinary := w.binaryPath
	cmd := exec.CommandContext(ctx, w.binaryPath, args...)
	cmd.Dir = filepath.Dir(w.binaryPath)
	binDir := cmd.Dir
	env := os.Environ()
	env = withPrependedEnvPath(env, "LD_LIBRARY_PATH", binDir)
	env = withPrependedEnvPath(env, "LD_LIBRARY_PATH", filepath.Join(binDir, "..", "lib"))
	env = withPrependedEnvPath(env, "LD_LIBRARY_PATH", filepath.Join(binDir, "lib"))
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fallbackBinary := filepath.Join(cmd.Dir, "main")
		fallbackErr := error(nil)
		if shouldFallbackToMain(err, stderr.String()) {
			if _, statErr := os.Stat(fallbackBinary); statErr == nil {
				stdout.Reset()
				stderr.Reset()
				executedBinary = fallbackBinary
				cmd2 := exec.CommandContext(ctx, fallbackBinary, args...)
				cmd2.Dir = cmd.Dir
				cmd2.Env = cmd.Env
				cmd2.Stdout = &stdout
				cmd2.Stderr = &stderr
				fallbackErr = cmd2.Run()
				if fallbackErr == nil {
					// Continue parsing below using stdout/stderr state from fallback run
				} else if isMainDeprecationWarning(stdout.String(), stderr.String()) {
					if resolved, _ := resolveWhisperTxtOutput(outputFile, cmd.Dir); resolved != "" {
						// Treat warning exit code as non-fatal if output was produced.
						fallbackErr = nil
					} else {
						return "", formatWhisperError(executedBinary, args, fallbackErr, stdout.String(), stderr.String())
					}
				} else {
					return "", formatWhisperError(executedBinary, args, fallbackErr, stdout.String(), stderr.String())
				}
			}
		}
		if fallbackErr == nil {
			if isMainDeprecationWarning(stdout.String(), stderr.String()) {
				if resolved, _ := resolveWhisperTxtOutput(outputFile, cmd.Dir); resolved != "" {
					// Treat warning exit code as non-fatal if output was produced.
					goto parse
				}
			}
			return "", formatWhisperError(executedBinary, args, err, stdout.String(), stderr.String())
		}
	}

parse:
	resolvedOutputFile, extraOutputFile := resolveWhisperTxtOutput(outputFile, cmd.Dir)

	// Try to read the output file
	text, err := os.ReadFile(resolvedOutputFile)
	if err != nil {
		// If no output file, try stdout
		text = stdout.Bytes()
	}

	// Clean up output file(s) if they exist
	if resolvedOutputFile != "" {
		if _, err := os.Stat(resolvedOutputFile); err == nil {
			os.Remove(resolvedOutputFile)
		}
	}
	if extraOutputFile != "" && extraOutputFile != resolvedOutputFile {
		if _, err := os.Stat(extraOutputFile); err == nil {
			os.Remove(extraOutputFile)
		}
	}

	clean := strings.TrimSpace(string(text))
	clean = stripWhisperMainDeprecationWarning(clean)
	clean = strings.TrimSpace(stripWhisperTimestamps(clean))
	if clean == "" {
		return "", fmt.Errorf("whisper.cpp produced no transcription output (binary=%s)", executedBinary)
	}
	return clean, nil
}
