/* Code Map: setup wizard-derived defaults
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
		// Wizard owns record_and_type. The four secondary
		// fields are populated from the State when the user
		// customized them in the SecondaryHotkeys step
		// (rc1-hotpatch-14); an empty field means "use the
		// runtime default" (<f10>/<f11>/<f12>; stop_recording
		// stays empty by design). preserveHotkeys still pulls
		// any pre-existing user-set values forward.
		Hotkeys: hotkeysBlock{
			RecordAndType:       hotkeyFromState(s.HotkeyPreset, s.CustomHotkey),
			StopRecording:       s.StopRecordingKey,
			ReadClipboard:       secondaryOrDefault(s.ReadClipboardKey, "<f10>"),
			ToggleTTS:           secondaryOrDefault(s.ToggleTTSKey, "<f11>"),
			ToggleTranscription: secondaryOrDefault(s.ToggleTranscriptionKey, "<f12>"),
		},
		// Audio defaults (rc1-hotpatch-13). The runtime
		// validator requires sample_rate > 0 and channels in
		// {1, 2}; without this block viper unmarshals Audio
		// as the zero struct and app.New() fails. Keep these
		// values in sync with internal/config.createDefaultConfig.
		Audio: audioBlock{
			SampleRate:  16000,
			Channels:    1,
			ChunkSize:   1024,
			MaxDuration: 300,
		},
		// Behavior defaults (rc1-hotpatch-14). Mirror
		// config.createDefaultConfig; the wizard's new
		// Behavior step (Part B) may overwrite Autostart
		// with the user's choice. Without this block viper
		// unmarshals Behavior as the zero struct (autostart
		// =false, notifications=false, type_delay=0) which
		// is why a fresh install showed "Autostart:
		// desired=false" and "notify: system disabled in
		// config" in the logs.
		Behavior: behaviorBlock{
			AutoType:       true,
			TypeDelay:      15,
			SoundOnStart:   false,
			SoundOnEnd:     false,
			Notifications:  true,
			Autostart:      s.Autostart,
			AutostartDelay: 5,
		},
	}
}

// secondaryOrDefault returns user if non-empty, otherwise def.
// Centralised here so the four secondary hotkey fields stay in
// lock-step with the runtime defaults; if the runtime default
// for read_clipboard ever changes, only this helper and
// config.createDefaultConfig need an update.
func secondaryOrDefault(user, def string) string {
	if user != "" {
		return user
	}
	return def
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
