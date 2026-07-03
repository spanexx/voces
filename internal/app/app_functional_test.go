package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"whisper-voice-util/internal/audio"
	"whisper-voice-util/internal/config"
	"whisper-voice-util/internal/hotkey"
	"whisper-voice-util/internal/input"
	"whisper-voice-util/internal/notify"
	"whisper-voice-util/internal/transcription"
	"whisper-voice-util/internal/tray"
	"whisper-voice-util/internal/tts"
)

func TestApplication_Handlers_Logic(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.Notifications = true
	cfg.Behavior.AutoType = true

	// Setup dummy binaries in PATH for recorder/player/xclip
	tmpDir := t.TempDir()
	setupFakeBinaries(t, tmpDir, map[string]string{
		"arecord": "#!/bin/sh\nsleep 0.1\nexit 0",
		"aplay":   "#!/bin/sh\nexit 0",
		"xclip":   "#!/bin/sh\necho 'clipboard text'\nexit 0",
		"xdotool": "#!/bin/sh\nexit 0", // added for autotyper
	})

	app := NewTestApp(cfg)
	app.Notifier = notify.New(cfg)
	app.Tray = tray.New(cfg, tray.ActionHandlers{})
	app.Clipboard = input.NewClipboard()
	app.Recorder = audio.NewRecorder()
	app.Player = audio.NewPlayer()
	app.AutoTyper = input.NewAutoTyper(cfg)
	app.Transcriber = transcription.New(cfg)
	app.TTS = tts.New(cfg)

	trayHandlers := app.buildTrayHandlers()
	hotkeyHandlers := app.buildHotkeyHandlers()

	// 1. Test OnRecordStart (Tray)
	trayHandlers.OnRecordStart()
	time.Sleep(200 * time.Millisecond) // wait for async record

	// 2. Test OnRecordStart/Stop (Hotkey)
	hotkeyHandlers.OnRecordStart()
	if app.Tray.GetState() != tray.StateRecording {
		t.Errorf("Expected tray state Recording, got %v", app.Tray.GetState())
	}
	hotkeyHandlers.OnRecordStop()
	// Should move to Processing
	if app.Tray.GetState() != tray.StateProcessing {
		t.Errorf("Expected tray state Processing, got %v", app.Tray.GetState())
	}

	// 3. Test OnReadClipboard
	trayHandlers.OnReadClipboard()
	time.Sleep(100 * time.Millisecond)

	// 4. Test Engine Selection
	trayHandlers.OnSetTranscriptionEngine("openai_api")
	if app.Config.Transcription.DefaultEngine != "openai_api" {
		t.Error("Failed to set transcription engine")
	}

	trayHandlers.OnSetTTSEngine("elevenlabs")
	if app.Config.TTS.DefaultEngine != "elevenlabs" {
		t.Error("Failed to set TTS engine")
	}

	// 5. Test Quit
	quitCalled := false
	app.cancel = func() { quitCalled = true }
	trayHandlers.OnQuit()
	if !quitCalled {
		t.Error("OnQuit did not trigger context cancellation")
	}

	// 6. Test Toggle handlers
	hotkeyHandlers.OnToggleTTS()
	hotkeyHandlers.OnToggleTranscription()
	hotkeyHandlers.OnReadClipboard()
	time.Sleep(100 * time.Millisecond)

	// 7. Test Record toggle from hotkey repeatedly
	hotkeyHandlers.OnRecordStart()
	hotkeyHandlers.OnRecordStart()
	if app.Tray.GetState() != tray.StateProcessing {
		t.Errorf("Expected tray state Processing after toggle stop, got %v", app.Tray.GetState())
	}

	// 8. Test processTranscription edge cases
	// Empty data
	app.processTranscription([]byte{})
	if app.Tray.GetState() != tray.StateIdle {
		t.Error("Expected StateIdle after empty processing")
	}

	// File error (if we could force it, but let's try a real transcription error)
	setupFakeBinaries(t, tmpDir, map[string]string{
		"whisper-fake": "#!/bin/sh\nexit 1",
	})
	app.Transcriber = transcription.New(cfg)
	app.Config.Transcription.WhisperCPP.BinaryPath = filepath.Join(tmpDir, "whisper-fake")
	app.Config.Transcription.WhisperCPP.Model = filepath.Join(tmpDir, "ggml-model.bin")
	os.WriteFile(filepath.Join(tmpDir, "ggml-model.bin"), []byte("fake"), 0644)

	app.processTranscription([]byte("fake-audio-data"))
	if app.Tray.GetState() != tray.StateError {
		t.Error("Expected StateError after transcription failure")
	}
}

func TestApplication_FullShutdown(t *testing.T) {
	cfg := &config.Config{}
	app := NewTestApp(cfg)

	// Initialize subsystems to ensure Stop methods can be executed without panic
	app.Notifier = notify.New(cfg)
	app.Hotkeys = hotkey.NewManager(cfg, hotkey.ActionHandlers{})
	app.Tray = tray.New(cfg, tray.ActionHandlers{})

	// Ensure shutdown runs without panic
	app.shutdown()
}

func setupFakeBinaries(t *testing.T, dir string, commands map[string]string) {
	t.Helper()
	for name, script := range commands {
		path := filepath.Join(dir, name)
		os.WriteFile(path, []byte(script), 0o755)
	}
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+":"+oldPath)
}

func TestApplication_Autostart_Logic(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// 1. Enable
	err := EnableAutostart()
	if err != nil {
		t.Fatalf("EnableAutostart failed: %v", err)
	}

	path, _ := desktopEntryPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Desktop entry was not created")
	}

	// 2. Disable
	err = DisableAutostart()
	if err != nil {
		t.Fatalf("DisableAutostart failed: %v", err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("Desktop entry was not removed")
	}

	// 3. Sync
	SyncAutostartState(true)
	SyncAutostartState(false)
}

func TestApplication_Instance_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	lockFilePath = filepath.Join(tmpDir, "instance.lock")

	// 1. Invalid PID in lock file
	os.WriteFile(lockFilePath, []byte("invalid"), 0644)
	cleanup, err := CheckAndLockSingleInstance()
	if err != nil {
		t.Fatalf("Should have overwritten invalid PID: %v", err)
	}
	cleanup()

	// 2. Dead PID in lock file
	os.WriteFile(lockFilePath, []byte("999999"), 0644)
	cleanup, err = CheckAndLockSingleInstance()
	if err != nil {
		t.Fatalf("Should have overwritten dead PID: %v", err)
	}
	cleanup()

	// 3. processExists with 0 (should exist usually, but we check branch)
	processExists(0)
}

func TestApplication_HotkeyHandlers_Errors(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.Notifications = true
	tmpDir := t.TempDir()

	app := NewTestApp(cfg)
	app.Notifier = notify.New(cfg)
	app.Tray = tray.New(cfg, tray.ActionHandlers{})
	app.Clipboard = input.NewClipboard()

	// 1. Simulate recording error
	setupFakeBinaries(t, tmpDir, map[string]string{
		"arecord": "#!/bin/sh\nexit 1",
	})
	app.Recorder = audio.NewRecorder()

	handlers := app.buildHotkeyHandlers()
	handlers.OnRecordStart()
	time.Sleep(100 * time.Millisecond) // wait for goroutine

	if app.Tray.GetState() != tray.StateIdle {
		t.Errorf("Expected StateIdle after record error, got %v", app.Tray.GetState())
	}

	// 2. Simulate clipboard error
	setupFakeBinaries(t, tmpDir, map[string]string{
		"xclip": "#!/bin/sh\nexit 1",
	})
	app.Clipboard = input.NewClipboard()
	handlers.OnReadClipboard()
	time.Sleep(100 * time.Millisecond)

	if app.Tray.GetState() != tray.StateIdle {
		t.Errorf("Expected StateIdle after clipboard error, got %v", app.Tray.GetState())
	}
}
