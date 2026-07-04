/* Code Map: Application Lifecycle
 * - New: Initializes the application tree
 * - Run: Main execution loop
 * - shutdown: Graceful termination logic
 *
 * CID Index:
 * CID:app-lifecycle-001 -> New
 * CID:app-lifecycle-002 -> Run
 *
 * Quick lookup: rg -n "CID:app-lifecycle-" internal/app/lifecycle.go
 */
package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"voces/internal/audio"
	"voces/internal/config"
	"voces/internal/hotkey"
	"voces/internal/input"
	"voces/internal/notify"
	"voces/internal/overlay"
	"voces/internal/transcription"
	"voces/internal/tray"
	"voces/internal/tts"
)

// CID:app-lifecycle-001 - New
// Purpose: Bootstraps the application, loads configuration, and initializes all modules.
func New() (*Application, error) {
	log.Println("Initializing Voces...")

	// 1. Single Instance Lock
	cleanupLock, err := CheckAndLockSingleInstance()
	if err != nil {
		return nil, err
	}

	// 2. Load Config
	cfg, err := config.Load()
	if err != nil {
		cleanupLock()
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 3. Sync Autostart state
	log.Printf("Autostart: desired=%v", cfg.Behavior.Autostart)
	if err := SyncAutostartState(cfg.Behavior.Autostart); err != nil {
		log.Printf("Warning: failed to sync autostart state (desired=%v): %v\n", cfg.Behavior.Autostart, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	a := &Application{
		Config:      cfg,
		ctx:         ctx,
		cancel:      cancel,
		cleanupLock: cleanupLock,

		Recorder:    audio.NewRecorder(),
		Player:      audio.NewPlayer(),
		Transcriber: transcription.New(cfg),
		TTS:         tts.New(cfg),
		AutoTyper:   input.NewAutoTyper(cfg),
		Clipboard:   input.NewClipboard(),
		Overlay:     overlay.New(),
	}

	// Initialize subsystems
	a.Notifier = notify.New(cfg)
	a.Hotkeys = hotkey.NewManager(cfg, a.buildHotkeyHandlers())
	a.Tray = tray.New(cfg, a.buildTrayHandlers())

	return a, nil
}

// NewTestApp creates an application instance suitable for unit tests.
// It skips the single-instance lock and uses provided dependencies.
func NewTestApp(cfg *config.Config) *Application {
	ctx, cancel := context.WithCancel(context.Background())
	a := &Application{
		Config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
	// Caller is responsible for setting other fields like Notifier, Tray, etc.
	return a
}

// CID:app-lifecycle-002 - Run
// Purpose: Starts background services and waits for termination signals or tray quit.
func (a *Application) Run() {
	a.Notifier.Start()
	if err := a.Hotkeys.Start(); err != nil {
		log.Printf("Hotkeys failed to start: %v\n", err)
		a.Notifier.Error("Hotkeys Disabled", err.Error())
	}

	a.Notifier.Info("Voces Ready", "Application is running in the background.")

	// Capture OS signals for graceful termination (Ctrl+C, kill).
	// Started before the tray so a SIGTERM during systray init
	// still unblocks Run().
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			log.Println("Received termination signal")
			a.cancel()
		case <-a.ctx.Done():
		}
	}()

	// Start the DBus Tray (this blocks the main thread by design until Quit is called)
	// We run it in a goroutine so we can wait on the context completion
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.Tray.Run() // blocks until tray.Quit()
	}()

	// Kick off the background update check. Silent on the no-update
	// path; shows a single notification on the update-available path.
	// Phase 7: IMPL §7 — initial check 10 s after Run() returns.
	a.startAutoCheck()

	<-a.ctx.Done() // Wait until application is requested to stop

	// Perform graceful cleanup
	a.shutdown()
}

// shutdown reverses the initialization and cleans up locks/routines.
func (a *Application) shutdown() {
	log.Println("Shutting down Voces...")

	a.Notifier.Info("Shutting Down", "Exiting Voces")

	// 1. Stop background inputs
	a.Hotkeys.Stop()
	if a.Overlay != nil {
		a.Overlay.Stop()
	}

	// 2. Stop notification bus
	a.Notifier.Stop()

	// 3. Stop systray UI
	a.Tray.Quit()
	a.wg.Wait()

	// 4. Release single instance lock
	if a.cleanupLock != nil {
		a.cleanupLock()
	}

	log.Println("Shutdown complete. Goodbye.")
}

func (a *Application) saveConfigAsync() {
	go func() {
		if err := config.Save(a.Config); err != nil {
			log.Printf("Failed to save configuration: %v\n", err)
		}
	}()
}
