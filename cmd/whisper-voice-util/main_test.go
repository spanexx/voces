// Package main provides the entry point for the Whisper Voice Utility.
package main

import (
	"os"
	"path/filepath"
	"testing"

	"whisper-voice-util/internal/config"
)

func TestMainFunction(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin
  openai_api:
    api_key: test-key
    model: whisper-1

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx
  elevenlabs:
    api_key: test-key
    voice_id: test-voice

audio:
  sample_rate: 16000
  channels: 1
  chunk_size: 1024
  max_duration: 300

hotkeys:
  record_and_type: '<f8>'
  read_clipboard: '<f9>'
  toggle_tts: '<f10>'
  toggle_transcription: '<f11>'

behavior:
  auto_type: true
  type_delay: 15
  sound_on_start: false
  sound_on_end: false
  notifications: true
`
	// Path for XDG config
	configDir := filepath.Join(tmpDir, "whisper-voice-util")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Errorf("Failed to load config in main test: %v", err)
	}
	if cfg == nil {
		t.Error("Config should not be nil")
	}
}

func TestMainFunction_MissingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Change to temp directory
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create go.mod to indicate this is a source directory
	os.WriteFile("go.mod", []byte("module test"), 0o644)

	// Since we are in source dir (go.mod exists), Load() should create config.yaml in CWD
	cfg, err := config.Load()
	if err != nil {
		t.Errorf("Load should create default config: %v", err)
	}
	if cfg == nil {
		t.Error("Config should not be nil after auto-creation")
	}
}

func TestConfigLoading_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin
    language: en
    compute_type: float
  openai_api:
    api_key: test-key
    model: whisper-1
    prompt: test prompt

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx
    voice_config: /opt/piper/models/en_US-lessac-medium.onnx.json
    output_device: hw:0,0
  elevenlabs:
    api_key: test-key
    voice_id: test-voice
    model: eleven_monolingual_v1
    stability: 0.5
    similarity_boost: 0.75

audio:
  sample_rate: 16000
  channels: 1
  chunk_size: 1024
  max_duration: 300

hotkeys:
  record_and_type: '<f8>'
  read_clipboard: '<f9>'
  toggle_tts: '<f10>'
  toggle_transcription: '<f11>'

behavior:
  auto_type: true
  type_delay: 15
  sound_on_start: false
  sound_on_end: false
  notifications: true
`
	configDir := filepath.Join(tmpDir, "whisper-voice-util")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load valid config: %v", err)
	}

	// Verify values
	if cfg.Transcription.DefaultEngine != "whisper_cpp" {
		t.Errorf("Expected whisper_cpp, got %s", cfg.Transcription.DefaultEngine)
	}
}
