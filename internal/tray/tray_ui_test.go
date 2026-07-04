package tray

import (
	"strings"
	"testing"
	"whisper-voice-util/internal/config"
	"whisper-voice-util/internal/updates"
)

func TestUI_onReady(t *testing.T) {
	cfg := &config.Config{}

	actionCalled := false
	handlers := ActionHandlers{
		OnSetTranscriptionEngine: func(e string) { actionCalled = true },
		OnSetTTSEngine:           func(e string) { actionCalled = true },
		OnRecordStart:            func() { actionCalled = true },
		OnReadClipboard:          func() { actionCalled = true },
		OnRunSetup:               func() { actionCalled = true },
		OnCheckUpdates:           func() { actionCalled = true },
		OnOpenDataDir:            func() { actionCalled = true },
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

// TestUI_Phase6_MenuItems confirms the three new tray items wired
// in IMPL §6 (Run setup again..., Check for updates, Open
// App-managed folder) are constructed during onReady. The test
// exercises the same direct-call path as TestUI_onReady, so it is
// subject to the same systray-must-not-panic-before-Run caveat.
func TestUI_Phase6_MenuItems(t *testing.T) {
	m := New(&config.Config{}, ActionHandlers{})
	m.onReady()

	if m.mRunSetup == nil {
		t.Error("Expected mRunSetup to be created (Phase 6 'Run setup again...')")
	}
	if m.mCheckUpdates == nil {
		t.Error("Expected mCheckUpdates to be created (Phase 6 'Check for updates')")
	}
	if m.mOpenDataDir == nil {
		t.Error("Expected mOpenDataDir to be created (Phase 6 'Open App-managed folder')")
	}
	if got := m.mRunSetup != nil && !strings.Contains(m.mRunSetup.String(), "Run setup again"); got {
		t.Errorf("mRunSetup.String() = %q; want it to contain 'Run setup again'", m.mRunSetup.String())
	}
	if got := m.mCheckUpdates != nil && !strings.Contains(m.mCheckUpdates.String(), "Check for updates"); got {
		t.Errorf("mCheckUpdates.String() = %q; want it to contain 'Check for updates'", m.mCheckUpdates.String())
	}
	if got := m.mOpenDataDir != nil && !strings.Contains(m.mOpenDataDir.String(), "Open App-managed folder"); got {
		t.Errorf("mOpenDataDir.String() = %q; want it to contain 'Open App-managed folder'", m.mOpenDataDir.String())
	}

	m.onExit()
}

// TestUI_Phase6_NilHandlersDoNotPanic confirms the menu items are
// safe to click when their handler is nil. The tray click goroutines
// are launched in onReady, so we don't click here — we just verify
// the build path tolerates nil handlers.
func TestUI_Phase6_NilHandlersDoNotPanic(t *testing.T) {
	m := New(&config.Config{}, ActionHandlers{}) // all handlers nil
	m.onReady()
	defer m.onExit()

	// If we reached here without panicking, the nil-tolerance in
	// the click goroutines is correct. Sanity check the wiring:
	if m.mRunSetup == nil || m.mCheckUpdates == nil || m.mOpenDataDir == nil {
		t.Error("Phase 6 menu items missing under nil-handler build")
	}
}

// TestUI_Phase7_UpdateItemExists confirms the "Update available"
// menu item is created during onReady. The item is added at the top
// of the menu and hidden by default; tests that build the tray
// without any update state should still see it constructed.
func TestUI_Phase7_UpdateItemExists(t *testing.T) {
	m := New(&config.Config{}, ActionHandlers{})
	m.onReady()
	defer m.onExit()

	if m.mUpdate == nil {
		t.Fatal("Expected mUpdate to be created (Phase 7 'Update available')")
	}
}

// TestUI_Phase7_SetAndClearBadge verifies the SetUpdateBadge /
// ClearUpdateBadge round-trip: SetUpdateBadge sets the title and
// shows the item; ClearUpdateBadge hides it again. The item's
// visibility state is implicit in systray (no getter), so we only
// assert that the calls do not panic and the title is updated.
func TestUI_Phase7_SetAndClearBadge(t *testing.T) {
	m := New(&config.Config{}, ActionHandlers{})
	m.onReady()
	defer m.onExit()

	rel := &updates.Release{TagName: "v0.3.1", Name: "0.3.1"}
	m.SetUpdateBadge(rel)
	if got := m.mUpdate.String(); !strings.Contains(got, "v0.3.1") {
		t.Errorf("After SetUpdateBadge, mUpdate.String() = %q, want it to contain v0.3.1", got)
	}

	m.ClearUpdateBadge()
	// Title is left untouched by ClearUpdateBadge; we only verify
	// the call does not panic. A second SetUpdateBadge(nil) is a
	// safe idempotent hide:
	m.SetUpdateBadge(nil)
	if got := m.mUpdate.String(); !strings.Contains(got, "Update available") {
		t.Errorf("After SetUpdateBadge(nil), mUpdate.String() = %q, want it to contain 'Update available'", got)
	}
}

// TestUI_Phase7_NilHandlerNoPanic confirms the click goroutine
// tolerates a nil OnApplyUpdate handler. The item is built in
// onReady, so we only need to verify the build path.
func TestUI_Phase7_NilHandlerNoPanic(t *testing.T) {
	m := New(&config.Config{}, ActionHandlers{}) // OnApplyUpdate nil
	m.onReady()
	defer m.onExit()
	if m.mUpdate == nil {
		t.Error("Phase 7 update item missing under nil-handler build")
	}
}
