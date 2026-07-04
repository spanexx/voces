/* Code Map: Config Persistence
 * - Save: Persists config struct to YAML
 * - createDefaultConfig: Generates initial template
 *
 * CID Index:
 * CID:config-save-001 -> Save
 * CID:config-save-002 -> createDefaultConfig
 *
 * Quick lookup: rg -n "CID:config-save-" internal/config/save.go
 */
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// CID:config-save-001 - Save
// Purpose: Writes the current configuration to the config.yaml file.
// Used by: internal/tray handlers for settings changes.
func Save(cfg *Config) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get user config dir: %w", err)
	}
	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf("refusing to save invalid config: %w", err)
	}
	dir := filepath.Join(configDir, "voces")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	configPath := filepath.Join(dir, "config.yaml")

	v := viper.New()
	v.Set("transcription", cfg.Transcription)
	v.Set("tts", cfg.TTS)
	v.Set("audio", cfg.Audio)
	v.Set("hotkeys", cfg.Hotkeys)
	v.Set("behavior", cfg.Behavior)

	v.SetConfigType("yaml")
	if err := v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func isRunningUnderGoTest() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}

// CID:config-save-002 - createDefaultConfig
// Purpose: Generates a default config.yaml with template values if none exists.
// Used by: Load()
func createDefaultConfig(dir string) error {
	// If dir is passed, use it, otherwise fall back to user config dir
	configPath := ""
	if dir != "" {
		configPath = filepath.Join(dir, "config.yaml")
	} else {
		userDir, err := os.UserConfigDir()
		if err == nil {
			dr := filepath.Join(userDir, "voces")
			os.MkdirAll(dr, 0755)
			configPath = filepath.Join(dr, "config.yaml")
			if isRunningUnderGoTest() {
				if _, statErr := os.Stat("go.mod"); statErr == nil {
					configPath = filepath.Join(userDir, "config.yaml")
				}
			}
		} else {
			configPath = "config.yaml"
		}
	}

	if st, err := os.Stat(configPath); err == nil {
		if st.Size() > 0 {
			return nil
		}
	}

	defaultConfig := `# Voces Configuration
# ===================================
# Auto-generated default configuration.
# Path placeholders below are empty — the first-run setup wizard
# (or "Run setup again..." from the tray menu) fills them in.
# If the wizard was skipped, the App will error out at first use
# until these paths point to real whisper.cpp / piper binaries
# and downloaded model files.

transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: ""
    model: ""
    language: ''
    compute_type: float
  openai_api:
    api_key: ${OPENAI_API_KEY}
    model: whisper-1
    prompt: ''

tts:
  default_engine: piper
  piper:
    binary_path: ""
    model: ""
    voice_config: ""
    output_device: ''
  elevenlabs:
    api_key: ${ELEVENLABS_API_KEY}
    voice_id: 21m00Tcm4TlvDq8ikWAM
    model: eleven_monolingual_v1
    stability: 0.5
    similarity_boost: 0.75

audio:
  sample_rate: 16000
  channels: 1
  chunk_size: 1024
  max_duration: 300

hotkeys:
  record_and_type: '<rightctrl>+<left>'
  stop_recording: ''
  read_clipboard: '<f10>'
  toggle_tts: '<f11>'
  toggle_transcription: '<f12>'

behavior:
  auto_type: true
  type_delay: 15
  sound_on_start: false
  sound_on_end: false
  notifications: true
`

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
		return fmt.Errorf("failed to write default config: %w", err)
	}

	return nil
}
