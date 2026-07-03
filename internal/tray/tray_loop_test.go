package tray

import (
	"testing"
	"time"
	"whisper-voice-util/internal/config"
)

func TestUI_ActionLoops(t *testing.T) {
	cfg := &config.Config{}

	recordCalled := false
	quitCalled := false

	handlers := ActionHandlers{
		OnRecordStart: func() { recordCalled = true },
		OnQuit:        func() { quitCalled = true },
	}

	m := New(cfg, handlers)
	m.onReady()

	// Simulate clicks by sending to the ClickedCh of the menu items
	// This covers the go routines spawned in onReady

	m.mRecord.ClickedCh <- struct{}{}
	time.Sleep(50 * time.Millisecond)
	if !recordCalled {
		t.Error("Expected OnRecordStart to be called after mRecord click")
	}

	m.mQuit.ClickedCh <- struct{}{}
	time.Sleep(50 * time.Millisecond)
	if !quitCalled {
		t.Error("Expected OnQuit to be called after mQuit click")
	}
}

func TestUI_ReadClipboardLoop(t *testing.T) {
	cfg := &config.Config{}
	readCalled := false
	handlers := ActionHandlers{
		OnReadClipboard: func() { readCalled = true },
	}
	m := New(cfg, handlers)
	m.onReady()

	m.mRead.ClickedCh <- struct{}{}
	time.Sleep(50 * time.Millisecond)
	if !readCalled {
		t.Error("Expected OnReadClipboard to be called after mRead click")
	}
}

func TestUI_EngineSelectionLoops(t *testing.T) {
	cfg := &config.Config{}

	lastTransEngine := ""
	lastTTSEngine := ""

	handlers := ActionHandlers{
		OnSetTranscriptionEngine: func(e string) { lastTransEngine = e },
		OnSetTTSEngine:           func(e string) { lastTTSEngine = e },
	}

	m := New(cfg, handlers)
	m.onReady()

	// Simulate clicks on submenus
	for name, item := range m.mEnginesTrans {
		item.ClickedCh <- struct{}{}
		time.Sleep(20 * time.Millisecond)
		if lastTransEngine != name {
			t.Errorf("Expected transcription engine %s, got %s", name, lastTransEngine)
		}
	}

	for name, item := range m.mEnginesTTS {
		item.ClickedCh <- struct{}{}
		time.Sleep(20 * time.Millisecond)
		if lastTTSEngine != name {
			t.Errorf("Expected tts engine %s, got %s", name, lastTTSEngine)
		}
	}
}
func TestTray_StateUpdates(t *testing.T) {
	cfg := &config.Config{}
	m := New(cfg, ActionHandlers{})

	// Set icon logic (normally uses systray.SetIcon)
	// We just verify it doesn't panic
	m.SetState(StateRecording, "Recording...")
	m.SetState(StateIdle, "")
}
