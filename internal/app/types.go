/* Code Map: Application Types
 * - Application: Root structure for dependency injection and lifecycle
 *
 * CID Index:
 * CID:app-types-001 -> Application
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

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	cleanupLock func()
}
