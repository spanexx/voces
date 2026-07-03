package tts

import (
	"os"
	"path/filepath"
	"testing"

	"whisper-voice-util/internal/config"
)

// TestPiper_Speak_WithAllFilesMissing tests Piper.Speak when all required files are missing
func TestPiper_Speak_WithAllFilesMissing(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:   "/nonexistent/piper",
				Model:        "/nonexistent/model.onnx",
				VoiceConfig:  "/nonexistent/config.json",
				OutputDevice: "",
			},
		},
	}

	p := NewPiper(cfg)
	
	// Test Speak - should fail on binary check
	err := p.Speak("Hello world")
	if err == nil {
		t.Error("Expected error when binary not found")
	}
}

// TestPiper_Speak_WithConfigMissing tests Piper.Speak when voice config is missing
func TestPiper_Speak_WithConfigMissing(t *testing.T) {
	// Create temp binary and model
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "piper")
	modelPath := filepath.Join(tmpDir, "model.onnx")
	os.WriteFile(binaryPath, []byte("binary"), 0o755)
	os.WriteFile(modelPath, []byte("model"), 0o644)

	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  binaryPath,
				Model:       modelPath,
				VoiceConfig: "/nonexistent/config.json",
			},
		},
	}

	p := NewPiper(cfg)
	err := p.Speak("Hello world")
	if err == nil {
		t.Error("Expected error when voice config not found")
	}
}

// TestTTS_Speak_ThroughManager tests TTS.Speak through the manager
func TestTTS_Speak_ThroughManager(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "elevenlabs",
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:          "test-key",
				VoiceID:         "test-voice",
				Model:           "eleven_monolingual_v1",
				Stability:       0.5,
				SimilarityBoost: 0.75,
			},
		},
	}

	tts := New(cfg)
	
	// Try to speak - will fail because API call fails in test environment
	err := tts.Speak("Hello world")
	if err == nil {
		t.Error("Expected error when calling API in tests")
	}
}

// TestTTS_Speak_WithPiperEngine tests TTS.Speak with piper engine
func TestTTS_Speak_WithPiperEngine(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "piper",
			Piper: config.PiperConfig{
				BinaryPath:  "/nonexistent/piper",
				Model:       "/nonexistent/model.onnx",
				VoiceConfig: "/nonexistent/config.json",
			},
		},
	}

	tts := New(cfg)
	
	// Try to speak - will fail because binary doesn't exist
	err := tts.Speak("Hello world")
	if err == nil {
		t.Error("Expected error when piper binary not found")
	}
}
