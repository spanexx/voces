/* Code Map: Tray Types
 * - State: Enumeration of visual app states
 * - ActionHandlers: Callbacks for tray menu events
 *
 * CID Index:
 * CID:tray-types-001 -> State
 * CID:tray-types-002 -> ActionHandlers
 *
 * Quick lookup: rg -n "CID:tray-types-" internal/tray/types.go
 */
package tray

// CID:tray-types-001 - State
// Purpose: Defines the possible visual and operational states of the tray icon.
type State int

const (
	StateIdle State = iota
	StateRecording
	StateProcessing
	StateError
	StateDisabled
)

// string returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateRecording:
		return "Recording"
	case StateProcessing:
		return "Processing"
	case StateError:
		return "Error"
	case StateDisabled:
		return "Disabled"
	default:
		return "Unknown"
	}
}

// CID:tray-types-002 - ActionHandlers
// Purpose: Interface for the tray to communicate user interactions back to the app logic.
//
// The OnRunSetup / OnCheckUpdates / OnOpenDataDir handlers (Phase 6)
// live alongside the legacy record / engine handlers. They are
// optional — when nil the corresponding menu item is a no-op.
//
// OnApplyUpdate (Phase 7) fires when the user clicks the dynamic
// "Update available (vX.Y.Z)" menu item. It is also optional.
type ActionHandlers struct {
	OnRecordStart            func()
	OnReadClipboard          func()
	OnSetTranscriptionEngine func(engine string)
	OnSetTTSEngine           func(engine string)
	OnRunSetup               func()
	OnCheckUpdates           func()
	OnApplyUpdate            func()
	OnOpenDataDir            func()
	OnQuit                   func()
}
