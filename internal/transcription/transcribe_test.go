package transcription

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"voces/internal/config"
)

// TestTranscribe_WhisperCPP_BinaryNotFound tests transcribe with missing binary
func TestTranscribe_WhisperCPP_BinaryNotFound(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "/nonexistent/binary",
				Model:      "/nonexistent/model",
			},
		},
	}

	transcriber := New(cfg)
	_, err := transcriber.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error when binary not found")
	}
}

// TestTranscribe_WhisperCPP_ModelNotFound tests transcribe with missing model
func TestTranscribe_WhisperCPP_ModelNotFound(t *testing.T) {
	// Create temp binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "whisper")
	os.WriteFile(binaryPath, []byte("binary"), 0o755)

	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: binaryPath,
				Model:      "/nonexistent/model",
			},
		},
	}

	transcriber := New(cfg)
	_, err := transcriber.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error when model not found")
	}
}

// TestTranscribe_OpenAIAPI_NoAPIKey tests transcribe without API key
func TestTranscribe_OpenAIAPI_NoAPIKey(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "openai_api",
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)
	_, err := transcriber.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error when API key not configured")
	}
}

// TestFormatWhisperEmptyOutput (rc1-hotpatch-14 R3) verifies
// the helper that splits the "empty output" case into the
// no-speech-detected sentinel vs a wrapped error with the
// underlying stderr text. The old single-line error conflated
// the two and was impossible for the tray to branch on.
func TestFormatWhisperEmptyOutput(t *testing.T) {
	// Case 1: empty stdout and empty stderr → no-speech sentinel
	err := formatWhisperEmptyOutput("/bin/whisper", "", "")
	if !errors.Is(err, ErrNoSpeechDetected) {
		t.Errorf("empty/empty → want ErrNoSpeechDetected, got %v", err)
	}
	// Case 2: empty stdout but stderr has the real error → wrapped
	// error with the stderr text (truncated to 200 chars for
	// the notification).
	stderr := "model load failed: cannot open ggml-small.en.bin"
	err = formatWhisperEmptyOutput("/bin/whisper", "", stderr)
	if errors.Is(err, ErrNoSpeechDetected) {
		t.Errorf("empty/non-empty → should NOT be ErrNoSpeechDetected, got %v", err)
	}
	if !strings.Contains(err.Error(), stderr) {
		t.Errorf("stderr text should appear in error message: got %q", err.Error())
	}
	// Case 3: truncation. A 500-char stderr is cut to 200 + "...".
	long := strings.Repeat("a", 500)
	err = formatWhisperEmptyOutput("/bin/whisper", "", long)
	if !strings.Contains(err.Error(), "... (truncated)") {
		t.Errorf("long stderr should be truncated: got %q", err.Error())
	}
}
