/* Code Map: Tray UI Loop Tests
 * - TestUI_ActionLoops: clicks record + quit menu items, verifies handlers fire
 * - TestUI_ReadClipboardLoop: clicks read-clipboard item, verifies handler fires
 * - TestUI_EngineSelectionLoops: clicks each engine submenu item, verifies
 *   OnSetTranscriptionEngine and OnSetTTSEngine receive the right names
 * - TestTray_StateUpdates: smoke test for SetState (no panic)
 *
 * CID Index:
 * CID:tray-test-001 -> TestUI_ActionLoops
 * CID:tray-test-002 -> TestUI_ReadClipboardLoop
 * CID:tray-test-003 -> TestUI_EngineSelectionLoops
 * CID:tray-test-004 -> TestTray_StateUpdates
 *
 * Synchronization: the click handlers run in goroutines spawned inside
 * Manager.onReady. The tests use buffered channels to capture the
 * handler invocations and a short timeout for the race-free read.
 * This replaces an earlier time.Sleep-based pattern that produced
 * a -race detector failure (shared state read+write across goroutines
 * without sync).
 *
 * Quick lookup: rg -n "CID:tray-test-" internal/tray/tray_loop_test.go
 */
package tray

import (
	"testing"
	"time"
	"whisper-voice-util/internal/config"
)

const trayTestTimeout = 500 * time.Millisecond

// CID:tray-test-001 - TestUI_ActionLoops
// Purpose: verify OnRecordStart and OnQuit fire when their menu items
// are clicked. Uses buffered channels to avoid the data race that
// direct variable writes caused under `go test -race`.
func TestUI_ActionLoops(t *testing.T) {
	cfg := &config.Config{}

	recordCalled := make(chan struct{}, 1)
	quitCalled := make(chan struct{}, 1)

	handlers := ActionHandlers{
		OnRecordStart: func() { recordCalled <- struct{}{} },
		OnQuit:        func() { quitCalled <- struct{}{} },
	}

	m := New(cfg, handlers)
	m.onReady()

	// Simulate clicks by sending to the ClickedCh of the menu items.
	// This covers the goroutines spawned in onReady.
	m.mRecord.ClickedCh <- struct{}{}
	select {
	case <-recordCalled:
	case <-time.After(trayTestTimeout):
		t.Error("Expected OnRecordStart to be called after mRecord click")
	}

	m.mQuit.ClickedCh <- struct{}{}
	select {
	case <-quitCalled:
	case <-time.After(trayTestTimeout):
		t.Error("Expected OnQuit to be called after mQuit click")
	}
}

// CID:tray-test-002 - TestUI_ReadClipboardLoop
// Purpose: verify OnReadClipboard fires when its menu item is clicked.
func TestUI_ReadClipboardLoop(t *testing.T) {
	cfg := &config.Config{}
	readCalled := make(chan struct{}, 1)
	handlers := ActionHandlers{
		OnReadClipboard: func() { readCalled <- struct{}{} },
	}
	m := New(cfg, handlers)
	m.onReady()

	m.mRead.ClickedCh <- struct{}{}
	select {
	case <-readCalled:
	case <-time.After(trayTestTimeout):
		t.Error("Expected OnReadClipboard to be called after mRead click")
	}
}

// CID:tray-test-003 - TestUI_EngineSelectionLoops
// Purpose: verify OnSetTranscriptionEngine and OnSetTTSEngine receive
// the correct engine name when each submenu item is clicked.
func TestUI_EngineSelectionLoops(t *testing.T) {
	cfg := &config.Config{}

	transEngineCh := make(chan string, 8)
	ttsEngineCh := make(chan string, 8)

	handlers := ActionHandlers{
		OnSetTranscriptionEngine: func(e string) { transEngineCh <- e },
		OnSetTTSEngine:           func(e string) { ttsEngineCh <- e },
	}

	m := New(cfg, handlers)
	m.onReady()

	// Simulate clicks on submenus.
	for name, item := range m.mEnginesTrans {
		item.ClickedCh <- struct{}{}
		select {
		case got := <-transEngineCh:
			if got != name {
				t.Errorf("Expected transcription engine %s, got %s", name, got)
			}
		case <-time.After(trayTestTimeout):
			t.Errorf("Timeout waiting for transcription engine %s", name)
		}
	}

	for name, item := range m.mEnginesTTS {
		item.ClickedCh <- struct{}{}
		select {
		case got := <-ttsEngineCh:
			if got != name {
				t.Errorf("Expected tts engine %s, got %s", name, got)
			}
		case <-time.After(trayTestTimeout):
			t.Errorf("Timeout waiting for tts engine %s", name)
		}
	}
}

// CID:tray-test-004 - TestTray_StateUpdates
// Purpose: smoke test for SetState (no panic on null tray).
func TestTray_StateUpdates(t *testing.T) {
	cfg := &config.Config{}
	m := New(cfg, ActionHandlers{})

	// Set icon logic (normally uses systray.SetIcon).
	// We just verify it doesn't panic.
	m.SetState(StateRecording, "Recording...")
	m.SetState(StateIdle, "")
}
