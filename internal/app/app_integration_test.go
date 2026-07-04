package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestEnvironment sets up a temporary directory with a config file
// and intercepts XDG_CONFIG_HOME so config loading works without modifying real dotfiles.
func setupTestEnvironment(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cfgDir := filepath.Join(dir, "voces")
	err := os.MkdirAll(cfgDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	cfgContent := `
hotkeys:
  record_and_type: "Super+Alt+R"
audio:
  sample_rate: 16000
  channels: 1
transcription:
  default_engine: "whisper_cpp"
tts:
  default_engine: "piper"
behavior:
  notifications: false
  autostart: false
`
	err = os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(cfgContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	lockFilePath = filepath.Join(dir, "voces.lock")

	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

// TestApplication_Integration_FullLifecycle creates a real application using New(),
// starts it via Run() in a goroutine, triggers some handlers, and then shuts it down.
func TestApplication_Integration_FullLifecycle(t *testing.T) {
	setupTestEnvironment(t)

	// Since New() creates real file locks,
	// ensure any stale lock from previous failed runs is removed.
	os.Remove(lockFilePath)

	app, err := New()
	if err != nil {
		t.Fatalf("Failed to initialize full application: %v", err)
	}

	// We are going to start the application in the background
	_, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	// WaitGroup to monitor run return
	errCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Prevent tests hanging on a panic
				errCh <- fmt.Errorf("Panic in Run(): %v", r)
			}
		}()
		app.Run()
		close(errCh)
	}()

	// Give the app time to bootstrap
	time.Sleep(200 * time.Millisecond)

	// Inject a piece of logic to simulate user triggering handlers
	// Note: these are the REAL handlers built by the app.
	// Since X11 might not be present in testing environments, we directly trigger the action
	// functions mapped to the tray/hotkeys without pressing physical keys.
	// But the app logic that follows is entirely 100% real.
	trayHandlers := app.buildTrayHandlers()
	hotkeyHandlers := app.buildHotkeyHandlers()

	// 1. Toggle settings
	trayHandlers.OnSetTranscriptionEngine("test-engine")
	hotkeyHandlers.OnToggleTTS()

	// 2. Quit the app gracefully after giving async routines a brief moment
	time.Sleep(200 * time.Millisecond)
	trayHandlers.OnQuit()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Application run failure: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Application did not shut down in time after OnQuit")
	}

	// Make sure the lock file was cleaned up by the graceful shutdown
	if _, err := os.Stat(lockFilePath); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Expected lock file to be removed, but got error: %v", err)
	}
}
