/* Code Map: Configuration Types
 * - Config: Root configuration structure
 * - TranscriptionConfig: Settings for Whisper/OpenAI
 * - TTSConfig: Settings for Piper/ElevenLabs
 * - AudioConfig: Hardware/format parameters
 * - HotkeysConfig: Global shortcut definitions
 * - BehaviorConfig: Application behavior flags
 *
 * CID Index:
 * CID:config-types-001 -> Config
 * CID:config-types-002 -> TranscriptionConfig
 * CID:config-types-003 -> TTSConfig
 * CID:config-types-004 -> AudioConfig
 * CID:config-types-005 -> HotkeysConfig
 * CID:config-types-006 -> BehaviorConfig
 *
 * Quick lookup: rg -n "CID:config-types-" internal/config/types.go
 */
package config

// CID:config-types-001 - Config
// Purpose: Root structure for application configuration.
// Uses: TranscriptionConfig, TTSConfig, AudioConfig, HotkeysConfig, BehaviorConfig
// Used by: internal/app, internal/tray, internal/hotkey
type Config struct {
	Transcription TranscriptionConfig `mapstructure:"transcription" yaml:"transcription"`
	TTS           TTSConfig           `mapstructure:"tts" yaml:"tts"`
	Audio         AudioConfig         `mapstructure:"audio" yaml:"audio"`
	Hotkeys       HotkeysConfig       `mapstructure:"hotkeys" yaml:"hotkeys"`
	Behavior      BehaviorConfig      `mapstructure:"behavior" yaml:"behavior"`
}

// CID:config-types-002 - TranscriptionConfig
// Purpose: Holds settings for audio-to-text engines.
// Uses: WhisperCPPConfig, OpenAIAPIConfig
type TranscriptionConfig struct {
	DefaultEngine string           `mapstructure:"default_engine" yaml:"default_engine"`
	WhisperCPP    WhisperCPPConfig `mapstructure:"whisper_cpp" yaml:"whisper_cpp"`
	OpenAIAPI     OpenAIAPIConfig  `mapstructure:"openai_api" yaml:"openai_api"`
}

// WhisperCPPConfig holds Whisper CPP settings.
type WhisperCPPConfig struct {
	BinaryPath  string `mapstructure:"binary_path" yaml:"binary_path"`
	Model       string `mapstructure:"model" yaml:"model"`
	Language    string `mapstructure:"language" yaml:"language"`
	ComputeType string `mapstructure:"compute_type" yaml:"compute_type"`
}

// OpenAIAPIConfig holds OpenAI API settings.
type OpenAIAPIConfig struct {
	APIKey string `mapstructure:"api_key" yaml:"api_key"`
	Model  string `mapstructure:"model" yaml:"model"`
	Prompt string `mapstructure:"prompt" yaml:"prompt"`
}

// CID:config-types-003 - TTSConfig
// Purpose: Holds settings for text-to-speech engines.
// Uses: PiperConfig, ElevenLabsConfig
type TTSConfig struct {
	DefaultEngine string           `mapstructure:"default_engine" yaml:"default_engine"`
	Piper         PiperConfig      `mapstructure:"piper" yaml:"piper"`
	ElevenLabs    ElevenLabsConfig `mapstructure:"elevenlabs" yaml:"elevenlabs"`
}

// PiperConfig holds Piper TTS settings.
type PiperConfig struct {
	BinaryPath   string `mapstructure:"binary_path" yaml:"binary_path"`
	Model        string `mapstructure:"model" yaml:"model"`
	VoiceConfig  string `mapstructure:"voice_config" yaml:"voice_config"`
	OutputDevice string `mapstructure:"output_device" yaml:"output_device"`
}

// ElevenLabsConfig holds ElevenLabs API settings.
type ElevenLabsConfig struct {
	APIKey          string  `mapstructure:"api_key" yaml:"api_key"`
	VoiceID         string  `mapstructure:"voice_id" yaml:"voice_id"`
	Model           string  `mapstructure:"model" yaml:"model"`
	Stability       float64 `mapstructure:"stability" yaml:"stability"`
	SimilarityBoost float64 `mapstructure:"similarity_boost" yaml:"similarity_boost"`
}

// CID:config-types-004 - AudioConfig
// Purpose: Defines recording parameters and hardware settings.
type AudioConfig struct {
	SampleRate  int `mapstructure:"sample_rate" yaml:"sample_rate"`
	Channels    int `mapstructure:"channels" yaml:"channels"`
	ChunkSize   int `mapstructure:"chunk_size" yaml:"chunk_size"`
	MaxDuration int `mapstructure:"max_duration" yaml:"max_duration"`
}

// CID:config-types-005 - HotkeysConfig
// Purpose: Maps user actions to global key combinations.
type HotkeysConfig struct {
	RecordAndType       string `mapstructure:"record_and_type" yaml:"record_and_type"`
	StopRecording       string `mapstructure:"stop_recording" yaml:"stop_recording"`
	ReadClipboard       string `mapstructure:"read_clipboard" yaml:"read_clipboard"`
	ToggleTTS           string `mapstructure:"toggle_tts" yaml:"toggle_tts"`
	ToggleTranscription string `mapstructure:"toggle_transcription" yaml:"toggle_transcription"`
}

// CID:config-types-006 - BehaviorConfig
// Purpose: Controls application-wide operational flags.
type BehaviorConfig struct {
	AutoType       bool `mapstructure:"auto_type" yaml:"auto_type"`
	TypeDelay      int  `mapstructure:"type_delay" yaml:"type_delay"`
	SoundOnStart   bool `mapstructure:"sound_on_start" yaml:"sound_on_start"`
	SoundOnEnd     bool `mapstructure:"sound_on_end" yaml:"sound_on_end"`
	Notifications  bool `mapstructure:"notifications" yaml:"notifications"`
	Autostart      bool `mapstructure:"autostart" yaml:"autostart"`
	AutostartDelay int  `mapstructure:"autostart_delay" yaml:"autostart_delay"`
}
