/* Code Map: Application Handlers — Phase 7 (update notifier)
 * - checkForUpdates: Single entry point for both the auto-check on
 *   startup and the tray "Check for updates" click. Centralises the
 *   state-update + UI-badge + notification logic so the two paths
 *   stay in sync.
 * - applyUpdate: Download the staged release and exec into it.
 *   Replaces the running process on success.
 * - startAutoCheck: Background goroutine that runs the first check
 *   10 s after Run() so the tray and notification bus have time to
 *   settle.
 *
 * CID Index:
 * CID:app-handlers-phase7-001 -> checkForUpdates
 * CID:app-handlers-phase7-002 -> applyUpdate
 * CID:app-handlers-phase7-003 -> startAutoCheck
 * CID:app-handlers-phase7-004 -> summarizeRelease
 *
 * Quick lookup: rg -n "CID:app-handlers-phase7-" internal/app/handlers_phase7.go
 */
package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"whisper-voice-util/internal/updates"
)

// autoCheckDelay is the time we wait after Run() before kicking off
// the background GitHub check. 10 s is enough for the systray to be
// fully initialised and for the user to have closed the welcome
// notification.
const autoCheckDelay = 10 * time.Second

// checkTimeout caps the GitHub check so a stalled connection cannot
// hold the tray click handler open longer than 10 s. 5 s is the
// per-request timeout inside updates.LatestRelease; we add 5 s of
// slack for DNS / TCP setup on slow networks.
const checkTimeout = 10 * time.Second

// CID:app-handlers-phase7-001 - checkForUpdates
// Purpose: Query GitHub for the latest release, store the result on
// the Application, and surface a tray notification + badge if a
// newer version is available. Used by both the background auto-check
// (silent on the no-update path) and the tray menu click (always
// shows a notification).
//
// showNotification controls the user-facing summary:
//   - true  → always show a notification, even when "you're up to date"
//   - false → only show a notification when a new version is found
func (a *Application) checkForUpdates(showNotification bool) {
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	rel, err := updates.LatestRelease(ctx)
	if err != nil {
		// 404 is the "no releases yet" case for a fresh repo — don't
		// treat it as an error the user needs to see.
		if err == updates.ErrNoRelease {
			log.Printf("checkForUpdates: no releases published yet")
			return
		}
		log.Printf("checkForUpdates: %v", err)
		if showNotification && a.Notifier != nil {
			a.Notifier.Error("Update check failed", err.Error())
		}
		return
	}
	a.setLatestRelease(rel)

	newer := rel.IsNewer(a.Version)
	a.applyUpdateBadge(rel, newer)

	if !newer {
		log.Printf("checkForUpdates: up to date (latest=%s, current=%s)", rel.TagName, a.Version)
		if showNotification && a.Notifier != nil {
			a.Notifier.Info("Up to date", "Latest is "+rel.TagName)
		}
		return
	}

	log.Printf("checkForUpdates: update available (latest=%s, current=%s)", rel.TagName, a.Version)
	if a.Notifier != nil {
		a.Notifier.Info("Update available", summarizeRelease(rel))
	}
}

// applyUpdateBadge pushes the (release, isNewer) decision into the
// tray, which shows or hides the "Update available (vX.Y.Z)" menu
// item. Safe to call when a.Tray is nil (e.g. in headless tests).
func (a *Application) applyUpdateBadge(rel *updates.Release, isNewer bool) {
	if a.Tray == nil {
		return
	}
	if isNewer {
		a.Tray.SetUpdateBadge(rel)
	} else {
		a.Tray.ClearUpdateBadge()
	}
}

// CID:app-handlers-phase7-002 - applyUpdate
// Purpose: Download the staged release asset to <bin>.new and exec
// into it. Called from the tray "Update available" menu item click.
// Runs in a goroutine so the click handler returns immediately;
// the syscall.Exec will replace the process image, so we never
// re-enter the tray loop.
func (a *Application) applyUpdate() {
	rel := a.GetLatestRelease()
	if rel == nil {
		log.Printf("applyUpdate: no release cached; running a fresh check first")
		a.checkForUpdates(false)
		rel = a.GetLatestRelease()
		if rel == nil {
			if a.Notifier != nil {
				a.Notifier.Error("Update failed", "Could not fetch the latest release")
			}
			return
		}
	}
	if !rel.IsNewer(a.Version) {
		log.Printf("applyUpdate: cached release %s is not newer than %s; refreshing",
			rel.TagName, a.Version)
		a.checkForUpdates(false)
		rel = a.GetLatestRelease()
		if rel == nil || !rel.IsNewer(a.Version) {
			if a.Notifier != nil {
				a.Notifier.Info("Up to date", "You are already running the latest version")
			}
			return
		}
	}

	exe, err := os.Executable()
	if err != nil {
		log.Printf("applyUpdate: os.Executable: %v", err)
		if a.Notifier != nil {
			a.Notifier.Error("Update failed", err.Error())
		}
		return
	}

	if a.Notifier != nil {
		a.Notifier.Info("Downloading update", rel.TagName)
	}

	dlCtx, dlCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer dlCancel()
	staged, err := rel.Download(dlCtx, exe)
	if err != nil {
		log.Printf("applyUpdate: Download: %v", err)
		if a.Notifier != nil {
			a.Notifier.Error("Download failed", err.Error())
		}
		return
	}
	// The staged file lives at <exe>.new; the running binary is at
	// <exe>. Restart() does the swap + syscall.Exec and replaces the
	// process image. If it returns, an error occurred and the old
	// binary is still in place.
	if a.Notifier != nil {
		a.Notifier.Info("Restarting", "Restarting into "+rel.TagName)
	}
	if err := updates.Restart(exe, os.Args); err != nil {
		log.Printf("applyUpdate: Restart: %v (staged at %s)", err, staged)
		if a.Notifier != nil {
			a.Notifier.Error("Restart failed",
				fmt.Sprintf("New binary is at %s. %v", staged, err))
		}
	}
}

// CID:app-handlers-phase7-003 - startAutoCheck
// Purpose: Spawn the background check goroutine. Tracked by a.wg so
// shutdown can wait for it. The auto-check is silent on the
// no-update path and shows a single notification on the
// update-available path.
func (a *Application) startAutoCheck() {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		// Small delay so we don't race the systray init or fire
		// notifications before the user can see them.
		select {
		case <-a.ctx.Done():
			return
		case <-time.After(autoCheckDelay):
		}
		if a.ctx.Err() != nil {
			return
		}
		a.checkForUpdates(false)
	}()
}

// CID:app-handlers-phase7-004 - summarizeRelease
// Purpose: Build a one-line tray notification body for a release.
// Keeps the wording in one place so the auto-check and the click
// handler produce identical messages.
func summarizeRelease(rel *updates.Release) string {
	if rel == nil {
		return ""
	}
	short := strings.SplitN(rel.Name, "\n", 2)[0]
	if short == "" {
		short = rel.TagName
	}
	return fmt.Sprintf("%s — click 'Update available' to install", short)
}

// trayIconLabelFor builds the tray menu label "Update available
// (vX.Y.Z)". Extracted so tests and the tray can share the format.
func trayIconLabelFor(rel *updates.Release) string {
	if rel == nil {
		return "Update available"
	}
	return "Update available (" + rel.TagName + ")"
}
