/* Code Map: Whisper.cpp output parsing
 * - stripWhisperTimestamps: removes whisper's "[hh:mm:ss --> hh:mm:ss]"
 *   prefix lines and collapses to a single line.
 * - stripWhisperMainDeprecationWarning: removes the 3-line deprecation
 *   notice from deprecated 'main' binary output.
 * - isMainDeprecationWarning: detects the warning in stdout/stderr
 *   to decide whether the exit code is "real" or "warning".
 * - resolveWhisperTxtOutput: prefers the expected .txt path; falls
 *   back to cmd-relative output for older builds.
 * - formatWhisperError: wraps the underlying error with stdout/stderr
 *   for debugging.
 * - shouldFallbackToMain: decides whether to retry with the legacy
 *   'main' binary when whisper-cli fails to load (exit 127 / missing
 *   shared libs).
 * - withPrependedEnvPath: prepends a directory to a colon-separated
 *   env var (LD_LIBRARY_PATH) without dropping existing entries.
 *
 * Sibling files in this package:
 * - whisper_cpp.go:  core engine + Transcribe
 * - whisper_paths.go: path resolution + validation
 *
 * CID Index:
 * CID:transcription-whisper-output-001 -> stripWhisperTimestamps
 * CID:transcription-whisper-output-002 -> stripWhisperMainDeprecationWarning
 * CID:transcription-whisper-output-003 -> isMainDeprecationWarning
 * CID:transcription-whisper-output-004 -> resolveWhisperTxtOutput
 * CID:transcription-whisper-output-005 -> formatWhisperError
 * CID:transcription-whisper-output-006 -> shouldFallbackToMain
 * CID:transcription-whisper-output-007 -> withPrependedEnvPath
 *
 * Quick lookup: rg -n "CID:transcription-whisper-output-" internal/transcription/
 */
package transcription

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CID:transcription-whisper-output-001 - stripWhisperTimestamps
var whisperTimestampPrefixRE = regexp.MustCompile(`(?m)^\[[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3} --> [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}\]\s*`)

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

// CID:transcription-whisper-output-002 - stripWhisperMainDeprecationWarning
var whisperMainDeprecationLineRE = regexp.MustCompile(`(?m)^(WARNING: The binary 'main' is deprecated\.|Please use 'whisper-cli' instead\.|See https://github\.com/ggerganov/whisper\.cpp/tree/master/examples/deprecation-warning/README\.md for more information\.)\s*$`)

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

// CID:transcription-whisper-output-003 - isMainDeprecationWarning
func isMainDeprecationWarning(stdout, stderr string) bool {
	out := stdout + "\n" + stderr
	return strings.Contains(out, "The binary 'main' is deprecated")
}

// CID:transcription-whisper-output-004 - resolveWhisperTxtOutput
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

// CID:transcription-whisper-output-005 - formatWhisperError
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

// CID:transcription-whisper-output-008 - formatWhisperEmptyOutput
// Purpose: build the "no transcription text was produced" error.
// Distinguishes two cases the previous single-line error
// conflated:
//   - "no speech detected" — stdout and stderr are both empty.
//     whisper.cpp returned 0 and wrote no .txt file. The
//     microphone captured silence or the user released the
//     hotkey without speaking. This is benign; the tray
//     notification should say "I didn't catch that — try
//     again" rather than "transcription failed".
//   - "binary failed" — stdout is empty but stderr is not.
//     whisper.cpp likely returned non-zero (model load error,
//     unsupported audio format, OOM, etc.). The underlying
//     error is the stderr text, which the user actually
//     needs to see to debug. Truncates to 200 chars so the
//     notification does not get spammed by a 4 MB backtrace.
//
// rc1-hotpatch-14 R3: replaces the previous
// "whisper.cpp produced no transcription output (binary=...)"
// string which gave the user no actionable information.
func formatWhisperEmptyOutput(binary, stdout, stderr string) error {
	errMsg := strings.TrimSpace(stderr)
	if errMsg == "" {
		return ErrNoSpeechDetected
	}
	const max = 200
	if len(errMsg) > max {
		errMsg = errMsg[:max] + "... (truncated)"
	}
	return fmt.Errorf("whisper.cpp produced no transcription output (binary=%s): %s", binary, errMsg)
}

// CID:transcription-whisper-output-006 - shouldFallbackToMain
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

// CID:transcription-whisper-output-007 - withPrependedEnvPath
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
