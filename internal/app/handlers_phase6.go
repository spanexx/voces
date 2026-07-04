/* Code Map: Application Handlers — Phase 6
 * - runSetupSubprocess: Re-execs the binary with --wizard-only so the
 *   tray can continue running while the wizard is shown.
 * - openDataDir: Resolves the XDG data dir and asks xdg-open to
 *   reveal it in the user's file manager.
 * - notifySetupFinished / notifySetupError: Centralised tray
 *   notifications for the wizard subprocess lifecycle.
 *
 * CID Index:
 * CID:app-handlers-phase6-001 -> runSetupSubprocess
 * CID:app-handlers-phase6-002 -> openDataDir
 * CID:app-handlers-phase6-003 -> notifySetupFinished
 * CID:app-handlers-phase6-004 -> notifySetupError
 *
 * Quick lookup: rg -n "CID:app-handlers-phase6-" internal/app/handlers_phase6.go
 */
package app

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/exec"
	"time"

	"whisper-voice-util/internal/paths"
)

// CID:app-handlers-phase6-001 - runSetupSubprocess
// Purpose: Re-execs the current binary with --wizard-only and waits
// for it to finish. The parent tray continues running independently.
// On exit we notify the user; new config is picked up on the next
// user action (no automatic hot-reload — see IMPL §6).
func (a *Application) runSetupSubprocess() {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("runSetupSubprocess: os.Executable: %v", err)
		a.notifySetupError("Failed to launch setup", err)
		return
	}
	// Bound the subprocess so a hung wizard doesn't block the tray
	// forever. 30 min matches the wizard's own internal timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, exe, "--wizard-only")
	cmd.Env = os.Environ() // inherit XDG_*, WVU_*, DISPLAY, etc.
	var tail bytes.Buffer
	cmd.Stdout = &tail
	cmd.Stderr = &tail

	log.Printf("runSetupSubprocess: starting %s --wizard-only", exe)
	if err := cmd.Run(); err != nil {
		log.Printf("runSetupSubprocess: %s failed: %v (tail: %s)", exe, err, tail.String())
		a.notifySetupError("Setup did not complete", err)
		return
	}
	log.Printf("runSetupSubprocess: finished (tail: %s)", tail.String())
	a.notifySetupFinished()
}

// CID:app-handlers-phase6-002 - openDataDir
// Purpose: Resolves the XDG data directory and asks xdg-open to
// reveal it in the user's file manager. Runs in a goroutine so the
// tray click handler is non-blocking.
func (a *Application) openDataDir() {
	dir, err := paths.DataDir()
	if err != nil {
		log.Printf("openDataDir: paths.DataDir: %v", err)
		a.notifySetupError("Failed to resolve data folder", err)
		return
	}
	cmd := exec.Command("xdg-open", dir)
	if err := cmd.Start(); err != nil {
		log.Printf("openDataDir: xdg-open %q: %v", dir, err)
		a.notifySetupError("Failed to open data folder", err)
		return
	}
	// Reap the process so it doesn't leak a zombie.
	go func() { _ = cmd.Wait() }()
}

// CID:app-handlers-phase6-003 - notifySetupFinished
// Purpose: Posts a friendly notification when the wizard subprocess
// returns successfully.
func (a *Application) notifySetupFinished() {
	if a.Notifier == nil {
		return
	}
	a.Notifier.Info("Setup complete", "Restart the app or use a tray action to load the new config")
}

// CID:app-handlers-phase6-004 - notifySetupError
// Purpose: Surfaces a setup-failure notification. Used by both the
// subprocess launcher and openDataDir so the user always sees a tray
// notification on failure (not just log lines).
func (a *Application) notifySetupError(title string, err error) {
	if a.Notifier == nil {
		return
	}
	a.Notifier.Error(title, err.Error())
}
