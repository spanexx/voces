/*
 * Code Map: setup "user wins" preservation
 * - preserveBinaryPath: keep user-set engine binary paths
 * - preserveHotkeys: keep user-set secondary hotkey fields
 * - loadConfigRaw: read pre-existing config.yaml into a map
 *
 * CID Index:
 * CID:setup-preserve-001 -> preserveBinaryPath
 * CID:setup-preserve-002 -> preserveHotkeys
 * CID:setup-preserve-003 -> loadConfigRaw
 *
 * Quick lookup: rg -n "CID:setup-preserve-" internal/setup/preserve.go
 */
package setup

import (
	"os"

	"go.yaml.in/yaml/v3"
)

// CID:setup-preserve-001 - preserveBinaryPath
// Purpose: the "user wins" rule for engine binary paths. If the
// pre-existing config had a non-empty binary_path for either
// engine, keep that value; the wizard's defaults only fill in
// empty fields. The wizard owns the model path (it is the source
// of truth for which model the user picked).
// Uses: (none — pure data).
// Used by: buildConfigDoc.

// preserveBinaryPath: the "user wins" rule. Pre-existing non-empty
// binary_path values survive Apply; the wizard's defaults only fill
// in empty fields.
func preserveBinaryPath(cfg *generatedConfig, existing map[string]any) {
	if t, ok := existing["transcription"].(map[string]any); ok {
		if w, ok := t["whisper_cpp"].(map[string]any); ok {
			if v, ok := w["binary_path"].(string); ok && v != "" {
				cfg.Transcription.WhisperCPP.BinaryPath = v
			}
		}
	}
	if t, ok := existing["tts"].(map[string]any); ok {
		if p, ok := t["piper"].(map[string]any); ok {
			if v, ok := p["binary_path"].(string); ok && v != "" {
				cfg.TTS.Piper.BinaryPath = v
			}
		}
	}
}

// CID:setup-preserve-002 - preserveHotkeys
// Purpose: the "user wins" rule for the four secondary hotkey
// fields (stop_recording, read_clipboard, toggle_tts,
// toggle_transcription). The wizard owns record_and_type and is
// NOT preserved — the wizard's choice always wins for that.
// rc1-hotpatch-11: previously Apply did not write hotkeys at all,
// causing voces to crash with "hotkeys.record_and_type is required".
// Uses: (none — pure data).
// Used by: buildConfigDoc.

// preserveHotkeys: same "user wins" rule for the four secondary
// hotkey fields. The wizard owns record_and_type; the other four
// (stop_recording, read_clipboard, toggle_tts,
// toggle_transcription) survive a re-run when the user had set
// them previously. We intentionally do NOT preserve
// record_and_type — the wizard's choice always wins for that.
func preserveHotkeys(cfg *generatedConfig, existing map[string]any) {
	h, ok := existing["hotkeys"].(map[string]any)
	if !ok {
		return
	}
	if v, ok := h["stop_recording"].(string); ok && v != "" {
		cfg.Hotkeys.StopRecording = v
	}
	if v, ok := h["read_clipboard"].(string); ok && v != "" {
		cfg.Hotkeys.ReadClipboard = v
	}
	if v, ok := h["toggle_tts"].(string); ok && v != "" {
		cfg.Hotkeys.ToggleTTS = v
	}
	if v, ok := h["toggle_transcription"].(string); ok && v != "" {
		cfg.Hotkeys.ToggleTranscription = v
	}
}

// CID:setup-preserve-003 - loadConfigRaw
// Purpose: read the pre-existing config.yaml into a generic map
// for the preserve* helpers. Returns an empty map (NOT an error)
// when the file is missing — that is the "first run" case. Real
// parse errors propagate so the user can fix a broken config.
// Uses: yaml.Unmarshal.
// Used by: buildConfigDoc.

// loadConfigRaw reads existing config.yaml into a generic map.
// Returns an empty map (not an error) when the file is missing —
// that is the "first run" case. Real parse errors propagate.
func loadConfigRaw(cfgPath string) (map[string]any, error) {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}
