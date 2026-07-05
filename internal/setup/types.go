/* Code Map: setup YAML type definitions
 * - generatedConfig: top-level on-disk shape
 * - transcriptionBlock + whisperCPPBlock + openAIAPIBlock: ASR
 * - ttsBlock + piperBlock + elevenLabsBlock: speech synth
 * - hotkeysBlock: hotkey subsystem (rc1-11)
 * - audioBlock: recording parameters (rc1-hotpatch-13)
 * - behaviorBlock: autostart/notifications/auto_type/... (rc1-hotpatch-14)
 *
 * CID Index:
 * CID:setup-types-001 -> generatedConfig
 * CID:setup-types-002 -> hotkeysBlock
 * CID:setup-types-003 -> audioBlock
 * CID:setup-types-004 -> behaviorBlock
 *
 * Quick lookup: rg -n "CID:setup-types-" internal/setup/types.go
 */
package setup

// CID:setup-types-001 - generatedConfig
// Purpose: the on-disk YAML layout that Apply writes. Mirrors
// config.Config but only the fields Apply needs. Kept local so
// this package stays the source of truth for the post-wizard
// shape; the runtime config struct in internal/config is a
// separate, read-only contract.
// Uses: transcriptionBlock, ttsBlock, hotkeysBlock.
// Used by: Apply (via buildConfigDoc), defaultConfigFor.

type generatedConfig struct {
	Transcription transcriptionBlock `yaml:"transcription"`
	TTS           ttsBlock           `yaml:"tts"`
	Hotkeys       hotkeysBlock       `yaml:"hotkeys"`
	// Audio block (rc1-hotpatch-13) — the runtime config
	// validator (internal/config.validateConfig) requires
	// sample_rate > 0 and channels in {1, 2}. Without this
	// block, viper unmarshals Audio as the zero struct and
	// app.New() fails with "audio.sample_rate must be positive"
	// right after the wizard writes the config.
	Audio audioBlock `yaml:"audio"`
	// Behavior block (rc1-hotpatch-14) — runtime Config
	// reads autostart, notifications, auto_type, type_delay,
	// sound_on_start, sound_on_end, autostart_delay. Without
	// this block viper unmarshals Behavior as the zero struct
	// (everything false, type_delay=0) which is why logs on a
	// fresh install showed "Autostart: desired=false" and
	// "notify: system disabled in config". Keep values in sync
	// with internal/config.createDefaultConfig.
	Behavior behaviorBlock `yaml:"behavior"`
}

type transcriptionBlock struct {
	DefaultEngine string          `yaml:"default_engine"`
	WhisperCPP    whisperCPPBlock `yaml:"whisper_cpp"`
	OpenAIAPI     openAIAPIBlock  `yaml:"openai_api"`
}

type whisperCPPBlock struct {
	BinaryPath  string `yaml:"binary_path"`
	Model       string `yaml:"model"`
	Language    string `yaml:"language"`
	ComputeType string `yaml:"compute_type"`
}

type openAIAPIBlock struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
	Prompt string `yaml:"prompt"`
}

type ttsBlock struct {
	DefaultEngine string          `yaml:"default_engine"`
	Piper         piperBlock      `yaml:"piper"`
	ElevenLabs    elevenLabsBlock `yaml:"elevenlabs"`
}

type piperBlock struct {
	BinaryPath   string `yaml:"binary_path"`
	Model        string `yaml:"model"`
	VoiceConfig  string `yaml:"voice_config"`
	OutputDevice string `yaml:"output_device"`
}

type elevenLabsBlock struct {
	APIKey          string  `yaml:"api_key"`
	VoiceID         string  `yaml:"voice_id"`
	Model           string  `yaml:"model"`
	Stability       float64 `yaml:"stability"`
	SimilarityBoost float64 `yaml:"similarity_boost"`
}

// CID:setup-types-002 - hotkeysBlock
// Purpose: the on-disk hotkey layout that the wizard writes.
// Mirrors config.HotkeysConfig exactly. record_and_type is
// always written because runtime validation requires it; without
// it, voces crashes with "hotkeys.record_and_type is required"
// (rc1-hotpatch-11).
//
// rc1-hotpatch-14: the four secondary fields are also always
// written now. Previously they had omitempty so they only
// appeared when preserveHotkeys pulled a pre-existing value
// forward; on a first run they were absent and the engine
// treated them as unbound (the "read clipboard" hotkey was
// silently not registered). The defaults match
// config.createDefaultConfig (<f10>, <f11>, <f12>; stop_recording
// is intentionally empty because the hold-binding model re-uses
// the record key to stop).
// Uses: (none — leaf type).
// Used by: generatedConfig, defaultConfigFor, preserveHotkeys.

type hotkeysBlock struct {
	RecordAndType       string `yaml:"record_and_type"`
	StopRecording       string `yaml:"stop_recording"`
	ReadClipboard       string `yaml:"read_clipboard"`
	ToggleTTS           string `yaml:"toggle_tts"`
	ToggleTranscription string `yaml:"toggle_transcription"`
}

// CID:setup-types-003 - audioBlock
// Purpose: mirror the runtime config.AudioConfig shape with
// sensible defaults. SampleRate=16000 + Channels=1 (mono) is
// what whisper.cpp's ggml-small.en.bin was trained on and
// matches internal/config.createDefaultConfig.
type audioBlock struct {
	SampleRate  int `yaml:"sample_rate"`
	Channels    int `yaml:"channels"`
	ChunkSize   int `yaml:"chunk_size"`
	MaxDuration int `yaml:"max_duration"`
}

// CID:setup-types-004 - behaviorBlock
// Purpose: mirror the runtime config.BehaviorConfig shape with
// sensible defaults. The defaults match
// internal/config.createDefaultConfig:
//   - auto_type=true   (the whole point of the app — type the
//                       transcribed text into the focused field)
//   - type_delay=15    (ms between keystrokes when typing
//                       long output; small enough to feel snappy,
//                       large enough to not drop characters on
//                       slow apps)
//   - sound_on_start/end=false  (silent by default; users can
//                                 opt in via the tray menu)
//   - notifications=true        (surface transcribe start/stop
//                                + errors via libnotify)
//   - autostart=true            (rc1-hotpatch-19 flips the
//                                runtime default to match the
//                                wizard's new preselected "Yes")
//   - autostart_delay=5         (seconds to wait after login
//                                before starting the tray, so
//                                the desktop has time to settle)
type behaviorBlock struct {
	AutoType       bool `yaml:"auto_type"`
	TypeDelay      int  `yaml:"type_delay"`
	SoundOnStart   bool `yaml:"sound_on_start"`
	SoundOnEnd     bool `yaml:"sound_on_end"`
	Notifications  bool `yaml:"notifications"`
	Autostart      bool `yaml:"autostart"`
	AutostartDelay int  `yaml:"autostart_delay"`
}
