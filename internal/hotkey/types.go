/* Code Map: Hotkey Types
 * - ActionHandlers: Callbacks for keyboard events
 *
 * CID Index:
 * CID:hotkey-types-001 -> ActionHandlers
 *
 * Quick lookup: rg -n "CID:hotkey-types-" internal/hotkey/types.go
 */
package hotkey

import (
	"time"
)

// CID:hotkey-types-001 - ActionHandlers
// Purpose: Registry of functions to execute when specific hotkeys are triggered.
type ActionHandlers struct {
	OnRecordStart         func() // hold-key pressed
	OnRecordStop          func() // hold-key released
	OnReadClipboard       func() // read-clipboard key pressed
	OnToggleTTS           func() // toggle-TTS key pressed
	OnToggleTranscription func() // toggle-transcription key pressed
}

// HoldBinding watches a key combination and fires callbacks on press/release.
type HoldBinding struct {
	keys         []string
	rawHotkey    string
	onPress      func()
	onRelease    func()
	pollInterval time.Duration
}

// PressBinding fires a callback when a key combination is tapped.
type PressBinding struct {
	keys         []string
	rawHotkey    string
	onPress      func()
	pollInterval time.Duration
	debounce     time.Duration
}
