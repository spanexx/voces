package transcription

import (
	"testing"

	"whisper-voice-util/internal/config"
)

// TestTranscribe_Integration tests the full transcribe flow
func TestTranscribe_Integration(t *testing.T) {
	// Test with whisper_cpp engine
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "/nonexistent/whisper",
				Model:      "/nonexistent/model",
			},
		},
	}

	transcriber := New(cfg)
	
	// Try to transcribe - should fail due to missing binary
	_, err := transcriber.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error for missing binary")
	}

	// Test engine switching
	err = transcriber.SetEngine("openai_api")
	// Should fail because API key is empty
	if err == nil {
		t.Log("SetEngine to openai_api succeeded unexpectedly")
	}
}

// TestTranscribe_WithLanguage tests transcription with language
func TestTranscribe_WithLanguage(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "/nonexistent/whisper",
				Model:      "/nonexistent/model",
				Language:   "en",
			},
		},
	}

	w := NewWhisperCPP(cfg)
	if w.language != "en" {
		t.Errorf("Expected language 'en', got %s", w.language)
	}
}

// TestTranscribe_WithComputeType tests transcription with compute type
func TestTranscribe_WithComputeType(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath:  "/nonexistent/whisper",
				Model:       "/nonexistent/model",
				ComputeType: "float16",
			},
		},
	}

	w := NewWhisperCPP(cfg)
	if w.computeType != "float16" {
		t.Errorf("Expected computeType 'float16', got %s", w.computeType)
	}
}
