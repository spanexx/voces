/*
 * Code Map: setup wizard-derived defaults
 * - defaultConfigFor: builds the wizard's recommended config
 * - hotkeyFromState: maps (preset, custom) -> record_and_type
 *
 * CID Index:
 * CID:setup-defaults-001 -> defaultConfigFor
 * CID:setup-defaults-002 -> hotkeyFromState
 *
 * Quick lookup: rg -n "CID:setup-defaults-" internal/setup/defaults.go
 */
package setup

import (
	"path/filepath"

	"voces/internal/paths"
)

// CID:setup-defaults-001 - defaultConfigFor
// Purpose: populate the wizard-derived values for a fresh
// config.yaml. Engine binary paths come from paths.EnginesDir();
// model paths come from paths.WhisperModelPath /
// paths.PiperVoicePath. The hotkey is the only field the wizard
// actually asks the user about (rc1-11).
// Uses: paths.EnginesDir, paths.WhisperModelPath, paths.PiperVoicePath,
// hotkeyFromState.
// Used by: buildConfigDoc.

// defaultConfigFor populates the wizard-derived values: model paths
// from state, engine binary paths from paths.EnginesDir(), and the
// hotkey from state.HotkeyPreset+state.CustomHotkey.
func defaultConfigFor(s *State) generatedConfig {
	engines, _ := paths.EnginesDir()
	whisperModel, _ := paths.WhisperModelPath(s.WhisperModel)
	piperModel, _ := paths.PiperVoicePath(s.PiperVoice)
	piperVoiceCfg := ""
	if s.PiperVoice != "" {
		piperVoiceCfg = piperModel + ".json"
	}
	return generatedConfig{
		Transcription: transcriptionBlock{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: whisperCPPBlock{
				BinaryPath:  filepath.Join(engines, "whisper-cli"),
				Model:       whisperModel,
				Language:    s.Language,
				ComputeType: "float",
			},
			OpenAIAPI: openAIAPIBlock{
				APIKey: "${OPENAI_API_KEY}",
				Model:  "whisper-1",
				Prompt: "",
			},
		},
		TTS: ttsBlock{
			DefaultEngine: "piper",
			Piper: piperBlock{
				BinaryPath:   filepath.Join(engines, "piper"),
				Model:        piperModel,
				VoiceConfig:  piperVoiceCfg,
				OutputDevice: "",
			},
			ElevenLabs: elevenLabsBlock{
				APIKey:          "${ELEVENLABS_API_KEY}",
				VoiceID:         "21m00Tcm4TlvDq8ikWAM",
				Model:           "eleven_monolingual_v1",
				Stability:       0.5,
				SimilarityBoost: 0.75,
			},
		},
		// Wizard owns record_and_type. The four secondary fields
		// start empty; preserveHotkeys pulls any pre-existing
		// user-set values forward so a re-run does not stomp them.
		Hotkeys: hotkeysBlock{
			RecordAndType: hotkeyFromState(s.HotkeyPreset, s.CustomHotkey),
		},
	}
}

// CID:setup-defaults-002 - hotkeyFromState
// Purpose: convert the wizard's (HotkeyPreset, CustomHotkey) pair
// into the single string the hotkey subsystem expects in
// config.yaml's hotkeys.record_and_type. The custom case passes
// the user's text through unchanged because the wizard's capture
// box already produces parseable strings and ParseKeys handles
// both the bracketed and unbracketed forms.
// Uses: HotkeyPreset* constants.
// Used by: defaultConfigFor.

// hotkeyFromState converts the wizard's (HotkeyPreset, CustomHotkey)
// pair into the single string the hotkey subsystem expects in
// config.yaml's hotkeys.record_and_type. Returns "<f8>" for any
// unknown preset so runtime validation never fails on a hotkey we
// don't know about; the wizard's state is the source of truth, so
// an unknown value here is a code bug.
func hotkeyFromState(preset, custom string) string {
	switch preset {
	case HotkeyPresetCtrlSpace:
		return "ctrl+space"
	case HotkeyPresetRCtrlLeft:
		return "<rightctrl>+<left>"
	case HotkeyPresetF8:
		return "<f8>"
	case HotkeyPresetCustom:
		return custom
	}
	return "<f8>"
}
