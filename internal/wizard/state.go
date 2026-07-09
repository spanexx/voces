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
	TTSVoice string
	// Autostart is the user's answer to the "start Voces when you
	// log in?" question (rc1-hotpatch-14). Wired into
	// config.Behavior.Autostart by defaultConfigFor.
	Autostart bool
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
	// Model is the whisper model file the user picked (or the
	// language-implied default if they did not reach the model
	// step). rc1-hotpatch-24 (docs/wizard-model-picker) — chosen
	// from the model step's tiny/base/small/medium matrix per
	// language scope. Examples: "ggml-base.en.bin",
	// "ggml-small.bin", "ggml-tiny.bin". Empty means "use
	// DefaultModelForLanguage(Language)".
	Model string
}

// CID:wizard-state-002 - NewState
// Purpose: returns a State with the same defaults the wizard presents
// (English, ctrl-space, no TTS, autostart enabled, runtime defaults
// for the four secondary hotkeys, small.en for the model). The
// hotkey constants are pulled from the setup package so wizard +
// persistence agree. The model default follows ADR-0004 (English →
// small.en, anything else → base).
func NewState() *State {
	return &State{
		Language:               "en",
		HotkeyPreset:           setup.HotkeyPresetCtrlSpace,
		TTSEnabled:             false,
		Autostart:              true,
		StopRecordingKey:       "",
		ReadClipboardKey:       "<f10>",
		ToggleTTSKey:           "<f11>",
		ToggleTranscriptionKey: "<f12>",
		Model:                  DefaultModelForLanguage("en"),
	}
}

// DefaultModelForLanguage returns the model file name the wizard
// should use when the user has not made an explicit pick. English
// gets the small.en variant (ADR-0004 — best English accuracy in the
// v1 set); every other language gets the base multilingual variant.
// Centralised here so the language step's commit handler, the
// runtime fallback in defaultConfigFor, and the test in
// state_test.go can all reference one source of truth.
func DefaultModelForLanguage(lang string) string {
	if lang == "en" {
		return "ggml-small.en.bin"
	}
	return "ggml-base.bin"
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
func (s *State) AutostartDesired() bool           { return s.Autostart }
func (s *State) StopRecordingKeyCode() string     { return s.StopRecordingKey }
func (s *State) ReadClipboardKeyCode() string     { return s.ReadClipboardKey }
func (s *State) ToggleTTSKeyCode() string         { return s.ToggleTTSKey }
func (s *State) ToggleTranscriptionKeyCode() string {
	return s.ToggleTranscriptionKey
}

// ModelFile returns the whisper model file the user has chosen (or
// the language-implied default set by NewState). Named with the
// "File" suffix to avoid colliding with the Model field.
func (s *State) ModelFile() string { return s.Model }

// TTSVoiceID returns the piper voice the user has chosen (rc1-hotpatch-29).
// Empty when the user has not reached the TTS step yet. May be a
// manifest key (e.g. "en_US-lessac-medium") or a custom-URL sentinel
// (see steps.customURLSentinel).
func (s *State) TTSVoiceID() string { return s.TTSVoice }

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
func (s *State) SetAutostart(desired bool) {
	s.Autostart = desired
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

// SetModel stores the chosen whisper model file name. Empty input
// is a no-op (the runtime falls back to DefaultModelForLanguage).
// The model step calls this from its radio-button "toggled" handler.
func (s *State) SetModel(filename string) {
	if filename != "" {
		s.Model = filename
	}
}

// SetTTSVoice stores the chosen piper voice ID. Empty input is a
// no-op so a step the user hasn't reached can't erase the State.
// rc1-hotpatch-29.
func (s *State) SetTTSVoice(id string) {
	if id != "" {
		s.TTSVoice = id
	}
}
