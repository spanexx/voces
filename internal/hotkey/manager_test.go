package hotkey

import (
	"testing"
	"voces/internal/config"
)

func TestManager_Lifecycle(t *testing.T) {
	cfg := &config.Config{
		Hotkeys: config.HotkeysConfig{
			RecordAndType:       "ctrl+space",
			ReadClipboard:       "f10",
			ToggleTTS:           "f11",
			ToggleTranscription: "f12",
		},
	}

	handlers := ActionHandlers{
		OnRecordStart:         func() {},
		OnRecordStop:          func() {},
		OnReadClipboard:       func() {},
		OnToggleTTS:           func() {},
		OnToggleTranscription: func() {},
	}

	m := NewManager(cfg, handlers)
	if m.IsRunning() {
		t.Error("Manager should not be running yet")
	}

	err := m.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !m.IsRunning() {
		t.Error("Manager should be running")
	}

	// Test double start
	err = m.Start()
	if err == nil {
		t.Error("Expected error on double start")
	}

	m.Stop()
	if m.IsRunning() {
		t.Error("Manager should not be running after stop")
	}

	// Test double stop (should not panic)
	m.Stop()
}
