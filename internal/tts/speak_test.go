package tts

import (
	"os"
	"path/filepath"
	"testing"

	"whisper-voice-util/internal/config"
)

// TestSpeak_Piper_BinaryNotFound tests Piper.Speak with missing binary
func TestSpeak_Piper_BinaryNotFound(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  "/nonexistent/piper",
				Model:       "/nonexistent/model.onnx",
				VoiceConfig: "/nonexistent/config.json",
			},
		},
	}

	p := NewPiper(cfg)
	err := p.Speak("Hello world")
	if err == nil {
		t.Error("Expected error when piper binary not found")
	}
}

// TestSpeak_Piper_ModelNotFound tests Piper.Speak with missing model
func TestSpeak_Piper_ModelNotFound(t *testing.T) {
	// Create temp binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "piper")
	os.WriteFile(binaryPath, []byte("binary"), 0o755)

	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  binaryPath,
				Model:       "/nonexistent/model.onnx",
				VoiceConfig: "/nonexistent/config.json",
			},
		},
	}

	p := NewPiper(cfg)
	err := p.Speak("Hello world")
	if err == nil {
		t.Error("Expected error when model not found")
	}
}

// TestSpeak_ElevenLabs_NoAPIKey tests ElevenLabs.Speak without API key
func TestSpeak_ElevenLabs_NoAPIKey(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "",
				VoiceID: "test-voice",
				Model:   "eleven_monolingual_v1",
			},
		},
	}

	e := NewElevenLabs(cfg)
	err := e.Speak("Hello world")
	if err == nil {
		t.Error("Expected error when API key not configured")
	}
}

// TestSpeak_TTS_Engine tests TTS.Speak through the manager
func TestSpeak_TTS_Engine(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "elevenlabs",
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "test-key",
				VoiceID: "test-voice",
				Model:   "eleven_monolingual_v1",
			},
		},
	}

	tts := New(cfg)
	// This will fail because it's trying to call the real API
	err := tts.Speak("Hello")
	if err == nil {
		t.Error("Expected error when calling API in tests")
	}
}
