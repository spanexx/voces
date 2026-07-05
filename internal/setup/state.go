/* Code Map: Setup State Persistence
 * - State: user-completed setup snapshot (language, models, hotkey)
 * - Load/Save: round-trip State to disk at $XDG_DATA_HOME/voces/state.json
 * - ShouldRun: decides if the wizard should auto-launch
 * - pathForState: resolves the canonical state file path
 *
 * CID Index:
 * CID:setup-state-001 -> State
 * CID:setup-state-002 -> Load
 * CID:setup-state-003 -> Save
 * CID:setup-state-004 -> ShouldRun
 * CID:setup-state-005 -> pathForState
 *
 * Quick lookup: rg -n "CID:setup-state-" internal/setup/
 */
package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"voces/internal/paths"
)

// stateFileName is the leaf filename under the data dir.
const stateFileName = "state.json"

// stateSchemaVersion is bumped when the State struct changes incompatibly.
const stateSchemaVersion = "1"

// Hotkey presets recognised by the wizard.
const (
	HotkeyPresetCtrlSpace = "ctrl-space"
	HotkeyPresetRCtrlLeft = "rctrl-left"
	HotkeyPresetF8        = "f8"
	HotkeyPresetCustom    = "custom"
)

// CID:setup-state-001 - State
// Purpose: Persisted snapshot of what the user picked during the wizard.
// Written by: Save. Read by: Load. Decision: shouldRun.
type State struct {
	// SchemaVersion is "1" for the current layout. Bump on breaking changes.
	SchemaVersion string `json:"schema_version"`
	// CompletedAt is when the user finished the wizard.
	CompletedAt time.Time `json:"completed_at"`
	// AppVersion is the App's Version constant at the time of save.
	AppVersion string `json:"app_version"`
	// Language is the user's chosen language code (ISO 639-1, e.g. "en", "es").
	Language string `json:"language"`
	// WhisperModel is the file name of the downloaded whisper model.
	WhisperModel string `json:"whisper_model"`
	// PiperVoice is the base name of the downloaded piper voice (empty if TTS skipped).
	PiperVoice string `json:"piper_voice,omitempty"`
	// HotkeyPreset is one of the HotkeyPreset* constants or "custom".
	HotkeyPreset string `json:"hotkey_preset"`
	// CustomHotkey is the captured hotkey string when HotkeyPreset == "custom".
	CustomHotkey string `json:"custom_hotkey,omitempty"`
	// Secondary hotkey fields (rc1-hotpatch-14). These come
	// from the wizard's SecondaryHotkeys step and are written
	// to hotkeys.{stop_recording, read_clipboard, toggle_tts,
	// toggle_transcription} by defaultConfigFor. An empty
	// string means "use the runtime default" (<f10>/<f11>/<f12>,
	// '' for stop_recording). Older state.json files parse
	// with empty strings, which defaultConfigFor leaves alone
	// (preserves the runtime default).
	StopRecordingKey       string `json:"stop_recording_key,omitempty"`
	ReadClipboardKey       string `json:"read_clipboard_key,omitempty"`
	ToggleTTSKey           string `json:"toggle_tts_key,omitempty"`
	ToggleTranscriptionKey string `json:"toggle_transcription_key,omitempty"`
}

// CID:setup-state-005 - pathForState
// Purpose: Returns the canonical absolute path to state.json.
func pathForState() (string, error) {
	data, err := paths.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(data, stateFileName), nil
}

// CID:setup-state-002 - Load
// Purpose: Reads state.json from disk. Returns os.ErrNotExist when no state
// has been written (i.e. first run). Returns an error on parse failure.
func Load() (*State, error) {
	p, err := pathForState()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %q: %w", p, err)
	}
	return &s, nil
}

// CID:setup-state-003 - Save
// Purpose: Writes state.json atomically (write to .tmp, then rename).
// Creates the data dir if missing.
func Save(s *State) error {
	if s.SchemaVersion == "" {
		s.SchemaVersion = stateSchemaVersion
	}
	if s.CompletedAt.IsZero() {
		s.CompletedAt = time.Now().UTC()
	}
	p, err := pathForState()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("ensure data dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, p); err != nil {
		return fmt.Errorf("rename %q -> %q: %w", tmp, p, err)
	}
	return nil
}

// CID:setup-state-004 - ShouldRun
// Purpose: Returns true if the wizard should auto-launch.
// Triggers (rc1-hotpatch-12):
//   1. state.json missing             -> first install
//   2. state.AppVersion != current    -> upgrade or downgrade
//   3. config.yaml missing            -> stale state (user
//      removed ~/.config/voces but kept
//      ~/.local/share/voces; the prior
//      wizard's state survives but the
//      config it produced is gone)
//   4. config.transcription.whisper_cpp.model empty
//      -> wizard was killed mid-step or
//         the model download failed;
//         the runtime cannot start
//         transcription without a model
//   5. otherwise                      -> skip (setup is complete)
// Returns the error from Load only when it's NOT os.ErrNotExist.
func ShouldRun(currentAppVersion string) (bool, error) {
	s, err := Load()
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	if s.AppVersion != currentAppVersion {
		return true, nil
	}
	// (3) + (4): config-side checks. Cheaper state check
	// already passed, so most users never touch the config
	// file. These branches only fire for users who removed
	// their config but kept their state (or who never
	// finished the wizard).
	cfgPath, err := configPath()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return true, nil
	}
	if configModelEmpty(cfgPath) {
		return true, nil
	}
	return false, nil
}

// configModelEmpty reports whether the loaded config has an
// empty model field under transcription.whisper_cpp.model —
// the path the wizard fills in. Returns false on any parse
// error so we don't force a re-run on a config the user
// hand-edited; the wizard will simply see what the user wrote.
func configModelEmpty(cfgPath string) bool {
	raw, err := loadConfigRaw(cfgPath)
	if err != nil {
		return false
	}
	t, ok := raw["transcription"].(map[string]any)
	if !ok {
		return false
	}
	w, ok := t["whisper_cpp"].(map[string]any)
	if !ok {
		return false
	}
	v, ok := w["model"].(string)
	if !ok {
		return false
	}
	return v == ""
}
