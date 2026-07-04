package app

import (
	"context"
	"testing"

	"voces/internal/audio"
	"voces/internal/config"
	"voces/internal/hotkey"
	"voces/internal/input"
	"voces/internal/notify"
	"voces/internal/transcription"
	"voces/internal/tray"
	"voces/internal/tts"
)

func TestApplication_HandlersExecution(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.Notifications = false

	app := NewTestApp(cfg)

	// Configure test dependencies
	// We just need them to exist so we don't panic on dereference
	app.Notifier = notify.New(cfg)
	app.Tray = tray.New(cfg, tray.ActionHandlers{})
	app.Recorder = audio.NewRecorder()
	app.Player = audio.NewPlayer()

	app.Transcriber = transcription.New(cfg)
	app.TTS = tts.New(cfg)

	// Override clipboard to not block on xclip
	app.Clipboard = input.NewClipboard()

	hotkeyHandlers := app.buildHotkeyHandlers()

	hotkeyHandlers.OnToggleTTS()
	hotkeyHandlers.OnToggleTranscription()

	hotkeyHandlers.OnRecordStop()

	trayHandlers := app.buildTrayHandlers()
	trayHandlers.OnSetTranscriptionEngine("test-engine")
	trayHandlers.OnSetTTSEngine("test-engine")
	trayHandlers.OnQuit()
}

func TestApplication_Shutdown(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.Notifications = false

	app := NewTestApp(cfg)

	app.Notifier = notify.New(cfg)
	app.Tray = tray.New(cfg, tray.ActionHandlers{})
	app.Hotkeys = hotkey.NewManager(cfg, hotkey.ActionHandlers{})

	ctx, cancel := context.WithCancel(context.Background())
	app.ctx = ctx
	app.cancel = cancel
	app.cleanupLock = func() {}

	app.shutdown()
}
