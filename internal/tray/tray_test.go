package tray

import (
	"testing"
	"whisper-voice-util/internal/config"
)

func TestManager_NewAndState(t *testing.T) {
	cfg := &config.Config{}
	handlers := ActionHandlers{
		OnQuit: func() {},
	}

	m := New(cfg, handlers)
	if m == nil {
		t.Fatal("Expected tray manager, got nil")
	}

	if m.state != StateIdle {
		t.Errorf("Expected initial state to be StateIdle, got %v", m.state)
	}

	// Test state transitions safely without starting the systray loop
	m.SetState(StateRecording, "Testing Recording")
	if m.state != StateRecording {
		t.Errorf("Expected state to be StateRecording")
	}

	m.SetState(StateProcessing, "Testing Processing")
	if m.state != StateProcessing {
		t.Errorf("Expected state to be StateProcessing")
	}

	m.SetState(StateError, "Testing Error")
	if m.state != StateError {
		t.Errorf("Expected state to be StateError")
	}

	m.SetState(StateDisabled, "Testing Disabled")
	if m.state != StateDisabled {
		t.Errorf("Expected state to be StateDisabled")
	}

	m.SetState(StateIdle, "")
	if m.state != StateIdle {
		t.Errorf("Expected state to be StateIdle")
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateIdle, "Idle"},
		{StateRecording, "Recording"},
		{StateProcessing, "Processing"},
		{StateError, "Error"},
		{StateDisabled, "Disabled"},
		{State(99), "Unknown"},
	}

	for _, tc := range tests {
		if tc.state.String() != tc.expected {
			t.Errorf("Expected string %q for state %d, got %q", tc.expected, tc.state, tc.state.String())
		}
	}
}
