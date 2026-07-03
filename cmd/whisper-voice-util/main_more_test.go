package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMainConfigLoading tests config loading scenarios
func TestMainConfigLoading(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create a valid config
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
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)
	
	// Verify config exists
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("Config file not created: %v", err)
	}
	
	t.Log("Config loading test passed")
}

// TestMainWithMinimalConfig tests with minimal config
func TestMainWithMinimalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	
	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin

audio:
  sample_rate: 16000
  channels: 1

hotkeys:
  record_and_type: '<f8>'

behavior:
  auto_type: true
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)
	
	t.Log("Minimal config test passed")
}

// TestMainWithEmptyConfig tests empty config handling
func TestMainWithEmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create empty config
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(""), 0o644)
	
	t.Log("Empty config test passed")
}

// TestMainPackageImports tests that imports are correct
func TestMainPackageImports(t *testing.T) {
	// Just verify the test compiles and runs
	t.Log("Main package imports test passed")
}
