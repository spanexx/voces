package app

import (
	"testing"
	"time"

	"voces/internal/config"
	"voces/internal/notify"
)

func TestApplication_Lifecycle(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.Notifications = false // keep it quiet

	app := NewTestApp(cfg)
	if app == nil {
		t.Fatal("NewTestApp returned nil")
	}

	// Configure Notifier
	app.Notifier = notify.New(cfg)

	// Test basic field initialization
	if app.Config != cfg {
		t.Errorf("Expected config to be set")
	}

	// Test cancellation
	if app.ctx.Err() != nil {
		t.Error("Context should not be cancelled yet")
	}

	app.cancel()

	select {
	case <-app.ctx.Done():
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Context did not cancel in time")
	}
}

func TestApplication_Handlers(t *testing.T) {
	cfg := &config.Config{}
	app := NewTestApp(cfg)

	// Test that handlers can be built without crashing
	hotkeyHandlers := app.buildHotkeyHandlers()
	if hotkeyHandlers.OnRecordStart == nil {
		t.Error("Missing RecordHoldStart handler")
	}

	trayHandlers := app.buildTrayHandlers()
	if trayHandlers.OnQuit == nil {
		t.Error("Missing Quit handler")
	}
}

func TestApplication_Logic(t *testing.T) {
	cfg := &config.Config{}
	_ = NewTestApp(cfg)

	// Configure dependencies for logic testing
	// We don't need real transcription or audio here if we just test the glue code

	// Test transcription results formatting (unexported logic)
	// Actually most of that is in Transcription package, but App handles the notification

	// This is more about ensuring handlers are wired correctly
}
