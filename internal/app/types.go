/* Code Map: Application Types
 * - Application: Root structure for dependency injection and lifecycle
 *
 * CID Index:
 * CID:app-types-001 -> Application
 * CID:app-types-002 -> SetVersion
 * CID:app-types-003 -> GetLatestRelease
 * CID:app-types-004 -> setLatestRelease
 *
 * Quick lookup: rg -n "CID:app-types-" internal/app/types.go
 */
package app

import (
	"context"
	"sync"

	"whisper-voice-util/internal/audio"
	"whisper-voice-util/internal/config"
	"whisper-voice-util/internal/hotkey"
	"whisper-voice-util/internal/input"
	"whisper-voice-util/internal/notify"
	"whisper-voice-util/internal/overlay"
	"whisper-voice-util/internal/transcription"
	"whisper-voice-util/internal/tray"
	"whisper-voice-util/internal/tts"
	"whisper-voice-util/internal/updates"
)

// CID:app-types-001 - Application
// Purpose: Central registry for all service managers and application context.
type Application struct {
	Config   *config.Config
	Notifier *notify.Manager
	Tray     *tray.Manager
	Hotkeys  *hotkey.Manager
	Overlay  *overlay.Manager

	Recorder    *audio.Recorder
	Player      *audio.Player
	Transcriber *transcription.Transcriber
	TTS         *tts.TTS
	AutoTyper   *input.AutoTyper
	Clipboard   *input.Clipboard

	// Version is the build version string injected at compile time
	// (e.g. "0.2.0"). Defaults to "dev" for local builds. Set via
	// SetVersion; the update-check flow uses it to compare against
	// the latest GitHub release tag.
	Version string

	// LatestRelease holds the most recent GitHub release as seen by
	// the running process. nil until the first check completes.
	// Guarded by latestReleaseMu; use GetLatestRelease /
	// setLatestRelease to read and write.
	LatestRelease   *updates.Release
	latestReleaseMu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	cleanupLock func()
}

// CID:app-types-002 - SetVersion
// Purpose: Set the build version. Called by main right after New().
// Kept separate from New() so the lifecycle signature stays stable
// across Phase 7 / Phase 8 changes.
func (a *Application) SetVersion(v string) {
	if v == "" {
		v = "dev"
	}
	a.Version = v
}

// CID:app-types-003 - GetLatestRelease
// Purpose: Thread-safe read of the most recent release seen by the
// app. Returns nil if no check has completed yet. Used by the tray
// to decide whether to show the "Update available" menu item.
func (a *Application) GetLatestRelease() *updates.Release {
	a.latestReleaseMu.RLock()
	defer a.latestReleaseMu.RUnlock()
	return a.LatestRelease
}

// CID:app-types-004 - setLatestRelease
// Purpose: Thread-safe write of the latest-release pointer. Internal
// to the app package — only checkForUpdates and the auto-check
// goroutine should call it.
func (a *Application) setLatestRelease(r *updates.Release) {
	a.latestReleaseMu.Lock()
	defer a.latestReleaseMu.Unlock()
	a.LatestRelease = r
}
