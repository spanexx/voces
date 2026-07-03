/* Code Map: Whisper.cpp Engine
 * - WhisperCPP: Subprocess-based local transcription engine
 * - NewWhisperCPP: Factory for creating a WhisperCPP engine
 * - Transcribe: Converts audio to text via whisper.cpp binary
 *
 * CID Index:
 * CID:transcription-whisper-001 -> WhisperCPP
 * CID:transcription-whisper-002 -> NewWhisperCPP
 * CID:transcription-whisper-003 -> Transcribe
 *
 * Quick lookup: rg -n "CID:transcription-whisper-" internal/transcription/whisper_cpp.go
 */
package transcription

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

var whisperTimestampPrefixRE = regexp.MustCompile(`(?m)^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3} --> [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}\]\s*`)

var whisperMainDeprecationLineRE = regexp.MustCompile(`(?m)^(WARNING: The binary 'main' is deprecated\.|Please use 'whisper-cli' instead\.|See https://github\.com/ggerganov/whisper\.cpp/tree/master/examples/deprecation-warning/README\.md for more information\.)\s*$`)

func stripWhisperTimestamps(s string) string {
	if s == "" {
		return s
	}
	s = whisperTimestampPrefixRE.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	clean := make([]string, 0, len(lines))
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		clean = append(clean, ln)
	}
	return strings.Join(clean, " ")
}

func stripWhisperMainDeprecationWarning(s string) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	clean := make([]string, 0, len(lines))
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if whisperMainDeprecationLineRE.MatchString(ln) {
			continue
		}
		clean = append(clean, ln)
	}
	return strings.Join(clean, "\n")
}

func isMainDeprecationWarning(stdout, stderr string) bool {
	out := stdout + "\n" + stderr
	return strings.Contains(out, "The binary 'main' is deprecated")
}

func resolveWhisperTxtOutput(expectedTxtPath, cmdDir string) (string, string) {
	// Prefer the expected location (next to the audio file).
	if fileExists(expectedTxtPath) {
		return expectedTxtPath, ""
	}
	// Some whisper.cpp builds write output relative to the working directory.
	if cmdDir != "" {
		alt := filepath.Join(cmdDir, filepath.Base(expectedTxtPath))
		if fileExists(alt) {
			return alt, expectedTxtPath
		}
	}
	return expectedTxtPath, ""
}

func formatWhisperError(binary string, args []string, err error, stdout, stderr string) error {
	msg := strings.TrimSpace(stderr)
	if msg == "" {
		msg = strings.TrimSpace(stdout)
	}
	if msg == "" {
		return fmt.Errorf("whisper.cpp failed: %w (binary=%s)", err, binary)
	}
	return fmt.Errorf("whisper.cpp failed: %w (binary=%s, args=%s), output: %s", err, binary, strings.Join(args, " "), msg)
}

func shouldFallbackToMain(err error, stderr string) bool {
	if err == nil {
		return false
	}
	// exit status 127 is commonly used when the loader cannot execute due to missing shared libs
	if strings.Contains(stderr, "error while loading shared libraries") {
		return true
	}
	if strings.Contains(err.Error(), "exit status 127") {
		return true
	}
	return false
}

func withPrependedEnvPath(env []string, key, dir string) []string {
	prefix := key + "="
	for i, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			cur := strings.TrimPrefix(kv, prefix)
			if cur == "" {
				env[i] = prefix + dir
			} else {
				env[i] = prefix + dir + string(os.PathListSeparator) + cur
			}
			return env
		}
	}
	return append(env, prefix+dir)
}

// Validate checks if the whisper.cpp binary and model are accessible.
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

func resolveWhisperBinaryAndModel(configuredBin, configuredModel string) (string, string) {
	managedBin := "/usr/local/share/whisper-voice-util/whisper.cpp/bin/whisper-cli"
	managedModel := "/usr/local/share/whisper-voice-util/whisper.cpp/models/ggml-base.en.bin"

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

func isRunningUnderGoTest() bool {
	// `go test` builds an executable named `<pkg>.test`.
	// We don't want environment-dependent auto-discovery to change unit test behavior.
	return strings.HasSuffix(os.Args[0], ".test")
}
