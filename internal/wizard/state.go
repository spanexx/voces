/* Code Map: Wizard State
 * - State: in-memory snapshot of the user's wizard selections
 * - NewState: returns a State with sensible defaults
 *
 * CID Index:
 * CID:wizard-state-001 -> State
 * CID:wizard-state-002 -> NewState
 *
 * Quick lookup: rg -n "CID:wizard-state-" internal/wizard/state.go
 */
package wizard

import (
	"voces/internal/setup"
)

// CID:wizard-state-001 - State
// Purpose: in-memory snapshot of the user's choices across the wizard
// steps. Each step's Capture() copies its widget values into this
// struct before the user advances. Phase 5 will translate this into
// a setup.State (persisted to state.json) plus a config.yaml.
type State struct {
	// Language is the ISO 639-1 code the user picked, e.g. "en", "de".
	// "en" is the default; the IMPL requires English to be first in
	// the picker.
	Language string
	// HotkeyPreset is one of the setup.HotkeyPreset* constants or
	// setup.HotkeyPresetCustom when CustomHotkey is set.
	HotkeyPreset string
	// CustomHotkey is the captured key combination when HotkeyPreset
	// is "custom". Format mirrors what hotkey.ParseKeys accepts.
	CustomHotkey string
	// TTSEnabled is true when the user opted in to text-to-speech.
	// Only consulted when Language != "en"; the TTS step is skipped
	// in the chain for English.
	TTSEnabled bool
	// TTSVoice is the piper voice id the user picked, e.g.
	// "en_US-lessac-medium". Empty when TTSEnabled is false.
	// For English (rc1-hotpatch-18) the TTS step is still
	// skipped, but the wizard auto-fills this with the
	// English default ("en_US-lessac-medium") so the piper
	// voice is downloaded and "Read clipboard" can speak the
	// transcript even when the user did not opt in.
	TTSVoice string
	// Secondary hotkey fields (rc1-hotpatch-14). The wizard's
	// SecondaryHotkeys step lets the user customize the four
	// hotkeys bound to "read clipboard", "toggle TTS", "toggle
	// transcription", and the optional separate "stop recording"
	// key. Empty string means "use the runtime default
	// (<f10>/<f11>/<f12>/'' for the four fields respectively)".
	StopRecordingKey       string
	ReadClipboardKey       string
	ToggleTTSKey           string
	ToggleTranscriptionKey string
}

// CID:wizard-state-002 - NewState
// Purpose: returns a State with the same defaults the welcome step
// presents (English, ctrl-space, no TTS, runtime defaults for the
// four secondary hotkeys). The hotkey constants are pulled from
// the setup package so wizard + persistence agree. Autostart was
// removed in rc1-hotpatch-18 — the behavior block is hardcoded.
func NewState() *State {
	return &State{
		Language:               "en",
		HotkeyPreset:           setup.HotkeyPresetCtrlSpace,
		TTSEnabled:             false,
		StopRecordingKey:       "",
		ReadClipboardKey:       "<f10>",
		ToggleTTSKey:           "<f11>",
		ToggleTranscriptionKey: "<f12>",
	}
}

// CID:wizard-state-003 - Getters
// Purpose: methods that implement steps.StateReader. Defined here
// (rather than on State directly via fields) so steps does not need
// to import this package. Go's structural typing means *State
// automatically satisfies the interface.
func (s *State) LanguageCode() string             { return s.Language }
func (s *State) Hotkey() string                   { return s.HotkeyPreset }
func (s *State) Custom() string                   { return s.CustomHotkey }
func (s *State) TTS() bool                        { return s.TTSEnabled }
func (s *State) StopRecordingKeyCode() string     { return s.StopRecordingKey }
func (s *State) ReadClipboardKeyCode() string     { return s.ReadClipboardKey }
func (s *State) ToggleTTSKeyCode() string         { return s.ToggleTTSKey }
func (s *State) ToggleTranscriptionKeyCode() string {
	return s.ToggleTranscriptionKey
}

// CID:wizard-state-004 - Setters
// Purpose: methods that implement steps.StateSetter. Steps call
// these from their Capture closure to commit the user's choice.
// Each setter is a no-op when the value is empty/false so a step
// that has not been reached does not overwrite the State.
func (s *State) SetLanguageCode(code string) {
	if code != "" {
		s.Language = code
	}
}
func (s *State) SetHotkey(preset, custom string) {
	if preset == "" {
		return
	}
	s.HotkeyPreset = preset
	s.CustomHotkey = custom
}
func (s *State) SetTTS(enabled bool) {
	s.TTSEnabled = enabled
}
func (s *State) SetSecondaryHotkeys(stop, read, toggleTTS, toggleTranscription string) {
	if stop != "" {
		s.StopRecordingKey = stop
	}
	if read != "" {
		s.ReadClipboardKey = read
	}
	if toggleTTS != "" {
		s.ToggleTTSKey = toggleTTS
	}
	if toggleTranscription != "" {
		s.ToggleTranscriptionKey = toggleTranscription
	}
}
