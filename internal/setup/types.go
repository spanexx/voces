/* Code Map: setup YAML type definitions
 * - generatedConfig: top-level on-disk shape
 * - transcriptionBlock + whisperCPPBlock + openAIAPIBlock: ASR
 * - ttsBlock + piperBlock + elevenLabsBlock: speech synth
 * - hotkeysBlock: hotkey subsystem (rc1-11)
 * - audioBlock: recording parameters (rc1-hotpatch-13)
 *
 * CID Index:
 * CID:setup-types-001 -> generatedConfig
 * CID:setup-types-002 -> hotkeysBlock
 * CID:setup-types-003 -> audioBlock
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
// Mirrors config.HotkeysConfig exactly. omitempty on the four
// secondary fields keeps the first-run output tidy. The primary
// field record_and_type is always written because runtime
// validation requires it; without it, voces crashes with
// "hotkeys.record_and_type is required" (rc1-hotpatch-11).
// Uses: (none — leaf type).
// Used by: generatedConfig, defaultConfigFor, preserveHotkeys.

type hotkeysBlock struct {
	RecordAndType       string `yaml:"record_and_type"`
	StopRecording       string `yaml:"stop_recording,omitempty"`
	ReadClipboard       string `yaml:"read_clipboard,omitempty"`
	ToggleTTS           string `yaml:"toggle_tts,omitempty"`
	ToggleTranscription string `yaml:"toggle_transcription,omitempty"`
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
