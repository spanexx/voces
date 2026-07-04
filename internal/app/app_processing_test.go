package app

import (
	"os"
	"path/filepath"
	"testing"
	"voces/internal/config"
	"voces/internal/input"
	"voces/internal/notify"
	"voces/internal/transcription"
	"voces/internal/tray"
)

func TestApplication_ProcessTranscription_Full(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.Notifications = false
	cfg.Behavior.AutoType = true

	app := NewTestApp(cfg)

	// Initialize subsystems manually for the test app
	app.Notifier = notify.New(cfg)
	app.Tray = tray.New(cfg, tray.ActionHandlers{})
	app.AutoTyper = input.NewAutoTyper(cfg)

	// 1. Success path
	fakeAudio := []byte("fake-audio-data")

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "whisper-fake")
	modelPath := filepath.Join(tmpDir, "model.bin")
	os.WriteFile(binPath, []byte("#!/bin/sh\necho 'Transcribed Text'\nexit 0"), 0o755)
	os.WriteFile(modelPath, []byte("fake model"), 0o644)

	app.Config.Transcription.WhisperCPP.BinaryPath = binPath
	app.Config.Transcription.WhisperCPP.Model = modelPath
	app.Config.Transcription.DefaultEngine = "whisper_cpp"

	// Initialize real transcriber
	app.Transcriber = transcription.New(app.Config)

	// Intercept AutoTyper to verify it was called
	typeCalled := false
	app.AutoTyper.SetRunner(func(args ...string) error {
		typeCalled = true
		return nil
	})

	app.processTranscription(fakeAudio)

	if !typeCalled {
		t.Error("AutoTyper should have been called on success")
	}

	// 2. Error path (Transcription fails)
	os.WriteFile(binPath, []byte("#!/bin/sh\nexit 1"), 0o755)
	app.processTranscription(fakeAudio)

	if app.Tray.GetState() != tray.StateError {
		t.Errorf("Expected tray state StateError, got %v", app.Tray.GetState())
	}

	// 4. AutoType disabled
	cfg.Behavior.AutoType = false
	typeCalled = false
	os.WriteFile(binPath, []byte("#!/bin/sh\necho 'text'\nexit 0"), 0o755)
	app.processTranscription(fakeAudio)
	if typeCalled {
		t.Error("AutoTyper should not have been called when disabled")
	}
}
