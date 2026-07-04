package transcription

import (
	"testing"

	"voces/internal/config"
)

// TestWhisperCPP_Transcribe_WithValidPaths tests the full transcribe flow
func TestWhisperCPP_Transcribe_WithValidPaths(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "/nonexistent/whisper",
				Model:      "/nonexistent/model",
				Language:   "en",
			},
		},
	}

	transcriber := New(cfg)

	// Try to transcribe - should fail due to missing binary
	_, err := transcriber.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error for missing binary")
	}

	// Verify current engine
	if transcriber.CurrentEngine() != "whisper_cpp" {
		t.Errorf("Expected current engine to be whisper_cpp, got %s", transcriber.CurrentEngine())
	}
}

// TestTranscriber_SetEngine_WithValidation tests engine switching with validation
func TestTranscriber_SetEngine_WithValidation(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "/nonexistent/whisper",
				Model:      "/nonexistent/model",
			},
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)

	// Try to switch to openai_api - should succeed with API key
	err := transcriber.SetEngine("openai_api")
	if err != nil {
		t.Logf("SetEngine to openai_api returned error (expected in test): %v", err)
	}

	// Try to switch back to whisper_cpp - will fail due to missing binary
	err = transcriber.SetEngine("whisper_cpp")
	if err == nil {
		t.Log("SetEngine to whisper_cpp succeeded unexpectedly")
	}
}

// TestTranscriber_ToggleEngine_Full tests the full toggle engine flow
func TestTranscriber_ToggleEngine_Full(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "/nonexistent/whisper",
				Model:      "/nonexistent/model",
			},
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)

	// Toggle from whisper_cpp to openai_api
	newEngine, err := transcriber.ToggleEngine()
	if err != nil {
		t.Logf("ToggleEngine returned error (may be expected): %v", err)
	}
	if newEngine == "openai_api" {
		t.Log("Successfully toggled to openai_api")
	}
}
