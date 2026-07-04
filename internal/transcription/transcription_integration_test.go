package transcription

import (
	"os"
	"path/filepath"
	"testing"
	"voces/internal/config"
)

func TestWhisperCPP_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	binPath := filepath.Join(tmpDir, "whisper-fake")
	modelPath := filepath.Join(tmpDir, "model.bin")
	audioPath := filepath.Join(tmpDir, "audio.wav")

	// Create real files to satisfy os.Stat
	os.WriteFile(binPath, []byte("#!/bin/sh\nexit 0"), 0o755)
	os.WriteFile(modelPath, []byte("fake model"), 0o644)
	os.WriteFile(audioPath, []byte("fake audio"), 0o644)

	cfg := &config.Config{}
	cfg.Transcription.WhisperCPP.BinaryPath = binPath
	cfg.Transcription.WhisperCPP.Model = modelPath

	w := NewWhisperCPP(cfg)

	// Test success where it outputs to stdout (txt file not found)
	// We need the script to output something
	os.WriteFile(binPath, []byte("#!/bin/sh\necho 'transcribed text'"), 0o755)

	text, err := w.Transcribe(audioPath)
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	if text != "transcribed text" {
		t.Errorf("Expected 'transcribed text', got %q", text)
	}

	// Test success where it creates a .txt file
	txtPath := filepath.Join(tmpDir, "audio.txt")
	script := `#!/bin/sh
echo "from file" > "` + txtPath + `"
echo "from stdout"
exit 0
`
	os.WriteFile(binPath, []byte(script), 0o755)

	text, err = w.Transcribe(audioPath)
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	if text != "from file" {
		t.Errorf("Expected 'from file', got %q", text)
	}

	// Test failure path
	os.WriteFile(binPath, []byte("#!/bin/sh\nexit 1"), 0o755)
	_, err = w.Transcribe(audioPath)
	if err == nil {
		t.Error("Expected error from whisper.cpp failure")
	}

	// Validate
	if err := w.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}
}

func TestOpenAI_Integration_EasyPaths(t *testing.T) {
	cfg := &config.Config{}
	o := NewOpenAIAPI(cfg)

	// 1. Missing API key
	_, err := o.Transcribe("any.wav")
	if err == nil {
		t.Error("Expected error with missing API key")
	}

	// 2. Missing file
	o.apiKey = "fake-key"
	_, err = o.Transcribe("nonexistent.wav")
	if err == nil {
		t.Error("Expected error with missing file")
	}
}
