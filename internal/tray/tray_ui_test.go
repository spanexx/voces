package tray

import (
	"testing"
	"whisper-voice-util/internal/config"
)

func TestUI_onReady(t *testing.T) {
	cfg := &config.Config{}

	actionCalled := false
	handlers := ActionHandlers{
		OnSetTranscriptionEngine: func(e string) { actionCalled = true },
		OnSetTTSEngine:           func(e string) { actionCalled = true },
		OnRecordStart:            func() { actionCalled = true },
		OnReadClipboard:          func() { actionCalled = true },
		OnQuit:                   func() { actionCalled = true },
	}

	m := New(cfg, handlers)

	// Direct call to onReady to initialize the UI elements (systray allows this before Run in most cases)
	m.onReady()

	if m.mRecord == nil {
		t.Error("Expected mRecord to be created")
	}
	if m.mRead == nil {
		t.Error("Expected mRead to be created")
	}
	if len(m.mEnginesTrans) == 0 {
		t.Error("Expected translation engines to be populated")
	}
	if len(m.mEnginesTTS) == 0 {
		t.Error("Expected TTS engines to be populated")
	}

	// Test Update routines don't panic
	m.UpdateTranscriptionEngine("whisper_cpp")
	m.UpdateTTSEngine("piper")

	m.onExit()

	_ = actionCalled // suppress unused
}
