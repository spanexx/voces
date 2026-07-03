package transcription

import (
	"os"
	"path/filepath"
	"testing"

	"whisper-voice-util/internal/config"
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
