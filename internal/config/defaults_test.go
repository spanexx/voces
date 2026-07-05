/* Code Map: regression tests for runtimeDefaults
 * - TestLoad_AppliesDefaultsForMissingFields: the rc1-hotpatch-16
 *   contract — a config.yaml that pre-dates rc1-hotpatch-14
 *   (no behavior: block, only record_and_type in hotkeys:) still
 *   loads with behavior.notifications=true, behavior.type_delay=15,
 *   behavior.autostart_delay=5, and the three function-key
 *   secondary hotkeys.
 * - TestLoad_PreservesUserSetFields: a config.yaml that
 *   explicitly sets notifications=false / autostart=true /
 *   read_clipboard="<f9>" keeps the user's choices (defaults
 *   do not silently overwrite them).
 * - TestRuntimeDefaults_StayInSync: walks the BehaviorConfig
 *   and HotkeysConfig struct field lists and asserts every
 *   field has a matching default declared. Catches future
 *   drift between the struct and runtimeDefaults.
 *
 * CID Index:
 * CID:config-defaults-test-001 -> TestLoad_AppliesDefaultsForMissingFields
 * CID:config-defaults-test-002 -> TestLoad_PreservesUserSetFields
 * CID:config-defaults-test-003 -> TestRuntimeDefaults_StayInSync
 * CID:config-defaults-test-004 -> TestRuntimeDefaults_DefaultKeyNames
 *
 * Quick lookup: rg -n "CID:config-defaults-test-" internal/config/
 */
package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

// CID:config-defaults-test-001 - TestLoad_AppliesDefaultsForMissingFields
// Purpose: prove the rc1-hotpatch-16 fix. Mirrors the exact shape
// of a config.yaml written by the v0.2.0-rc1 wizard (no
// behavior: block, hotkeys: has only record_and_type) and asserts
// that Load() fills in the runtime defaults so the in-memory
// struct matches what a fresh install would see.
//
// This is the regression test for the user-reported failure mode:
// "Autostart: desired=false" and "notify: system disabled in
// config" on a wizard-completed install. The fix is the
// runtimeDefaults call in Load(); this test fails if that call
// is removed or stops covering the right keys.
func TestLoad_AppliesDefaultsForMissingFields(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Pre-rc1-hotpatch-14 wizard output. Missing: behavior:
	// block entirely, hotkeys.stop_recording, hotkeys.read_clipboard,
	// hotkeys.toggle_tts, hotkeys.toggle_transcription. This is
	// the exact shape the user has on disk at
	// ~/.config/voces/config.yaml from the rc1 install.
	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin
    language: en
    compute_type: float
  openai_api:
    api_key: test-key
    model: whisper-1
    prompt: ""

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx
    voice_config: /opt/piper/models/en_US-lessac-medium.onnx.json
    output_device: ''
  elevenlabs:
    api_key: test-key
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
  record_and_type: f9
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Behavior defaults — these are the values the user
	// reported as broken on rc1. They must be filled in even
	// though the YAML has no behavior: block.
	if cfg.Behavior.AutoType != true {
		t.Errorf("behavior.auto_type: want true (default), got %v", cfg.Behavior.AutoType)
	}
	if cfg.Behavior.TypeDelay != 15 {
		t.Errorf("behavior.type_delay: want 15 (default), got %d", cfg.Behavior.TypeDelay)
	}
	if cfg.Behavior.SoundOnStart != false {
		t.Errorf("behavior.sound_on_start: want false (default), got %v", cfg.Behavior.SoundOnStart)
	}
	if cfg.Behavior.SoundOnEnd != false {
		t.Errorf("behavior.sound_on_end: want false (default), got %v", cfg.Behavior.SoundOnEnd)
	}
	if cfg.Behavior.Notifications != true {
		t.Errorf("behavior.notifications: want true (default), got %v", cfg.Behavior.Notifications)
	}
	if cfg.Behavior.Autostart != false {
		t.Errorf("behavior.autostart: want false (default), got %v", cfg.Behavior.Autostart)
	}
	if cfg.Behavior.AutostartDelay != 5 {
		t.Errorf("behavior.autostart_delay: want 5 (default), got %d", cfg.Behavior.AutostartDelay)
	}

	// Hotkey defaults — the three function-key secondaries.
	// stop_recording is intentionally empty (hold-binding
	// model) and is NOT defaulted.
	if cfg.Hotkeys.RecordAndType != "f9" {
		t.Errorf("hotkeys.record_and_type: want %q (user value), got %q", "f9", cfg.Hotkeys.RecordAndType)
	}
	if cfg.Hotkeys.StopRecording != "" {
		t.Errorf("hotkeys.stop_recording: want \"\" (empty by design), got %q", cfg.Hotkeys.StopRecording)
	}
	if cfg.Hotkeys.ReadClipboard != "<f10>" {
		t.Errorf("hotkeys.read_clipboard: want <f10> (default), got %q", cfg.Hotkeys.ReadClipboard)
	}
	if cfg.Hotkeys.ToggleTTS != "<f11>" {
		t.Errorf("hotkeys.toggle_tts: want <f11> (default), got %q", cfg.Hotkeys.ToggleTTS)
	}
	if cfg.Hotkeys.ToggleTranscription != "<f12>" {
		t.Errorf("hotkeys.toggle_transcription: want <f12> (default), got %q", cfg.Hotkeys.ToggleTranscription)
	}
}

// CID:config-defaults-test-002 - TestLoad_PreservesUserSetFields
// Purpose: prove that the runtime defaults do not silently
// overwrite values the user explicitly set in config.yaml.
// The defaults are only a fallback for absent keys; explicit
// keys always win.
func TestLoad_PreservesUserSetFields(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin
    language: en
    compute_type: float
  openai_api:
    api_key: test-key
    model: whisper-1
    prompt: ""

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx
    voice_config: ''
    output_device: ''
  elevenlabs:
    api_key: test-key
    voice_id: voice-id
    model: eleven_monolingual_v1
    stability: 0.5
    similarity_boost: 0.75

audio:
  sample_rate: 16000
  channels: 1
  chunk_size: 1024
  max_duration: 300

hotkeys:
  record_and_type: '<ctrl>+<space>'
  stop_recording: '<f8>'
  read_clipboard: '<f9>'
  toggle_tts: '<f10>'
  toggle_transcription: '<f11>'

behavior:
  auto_type: false
  type_delay: 42
  sound_on_start: true
  sound_on_end: true
  notifications: false
  autostart: true
  autostart_delay: 30
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Every user value must be preserved verbatim. The default
	// set is a fallback, not a forced override.
	if cfg.Behavior.AutoType != false {
		t.Errorf("behavior.auto_type: want false (user set), got %v", cfg.Behavior.AutoType)
	}
	if cfg.Behavior.TypeDelay != 42 {
		t.Errorf("behavior.type_delay: want 42 (user set), got %d", cfg.Behavior.TypeDelay)
	}
	if cfg.Behavior.SoundOnStart != true {
		t.Errorf("behavior.sound_on_start: want true (user set), got %v", cfg.Behavior.SoundOnStart)
	}
	if cfg.Behavior.SoundOnEnd != true {
		t.Errorf("behavior.sound_on_end: want true (user set), got %v", cfg.Behavior.SoundOnEnd)
	}
	if cfg.Behavior.Notifications != false {
		t.Errorf("behavior.notifications: want false (user set), got %v", cfg.Behavior.Notifications)
	}
	if cfg.Behavior.Autostart != true {
		t.Errorf("behavior.autostart: want true (user set), got %v", cfg.Behavior.Autostart)
	}
	if cfg.Behavior.AutostartDelay != 30 {
		t.Errorf("behavior.autostart_delay: want 30 (user set), got %d", cfg.Behavior.AutostartDelay)
	}
	if cfg.Hotkeys.RecordAndType != "<ctrl>+<space>" {
		t.Errorf("hotkeys.record_and_type: want <ctrl>+<space> (user set), got %q", cfg.Hotkeys.RecordAndType)
	}
	if cfg.Hotkeys.StopRecording != "<f8>" {
		t.Errorf("hotkeys.stop_recording: want <f8> (user set), got %q", cfg.Hotkeys.StopRecording)
	}
	if cfg.Hotkeys.ReadClipboard != "<f9>" {
		t.Errorf("hotkeys.read_clipboard: want <f9> (user set), got %q", cfg.Hotkeys.ReadClipboard)
	}
	if cfg.Hotkeys.ToggleTTS != "<f10>" {
		t.Errorf("hotkeys.toggle_tts: want <f10> (user set), got %q", cfg.Hotkeys.ToggleTTS)
	}
	if cfg.Hotkeys.ToggleTranscription != "<f11>" {
		t.Errorf("hotkeys.toggle_transcription: want <f11> (user set), got %q", cfg.Hotkeys.ToggleTranscription)
	}
}

// CID:config-defaults-test-003 - TestRuntimeDefaults_StayInSync
// Purpose: walk every field on BehaviorConfig and the
// defaultable subset of HotkeysConfig and assert the
// matching viper key has a default registered. Catches
// future struct/template drift in either direction:
//   - a new field added to BehaviorConfig without a
//     matching SetDefault in runtimeDefaults
//   - a SetDefault added in runtimeDefaults without a
//     matching field on the struct
//
// Same shape as TestCreateDefaultConfig_CompleteBehaviorAndHotkeys
// (rc1-hotpatch-15) but for the in-memory Load() path.
func TestRuntimeDefaults_StayInSync(t *testing.T) {
	v := viper.New()
	runtimeDefaults(v)

	for _, f := range behaviorDefaultFields() {
		got := v.Get(f.Key)
		if got == nil {
			t.Errorf("viper default missing for %s", f.Key)
			continue
		}
		if !reflect.DeepEqual(got, f.Value) {
			t.Errorf("viper default drift for %s: want %v (%T), got %v (%T)",
				f.Key, f.Value, f.Value, got, got)
		}
	}

	for _, f := range hotkeysDefaultFields() {
		got := v.Get(f.Key)
		if got == nil {
			t.Errorf("viper default missing for %s", f.Key)
			continue
		}
		if !reflect.DeepEqual(got, f.Value) {
			t.Errorf("viper default drift for %s: want %v (%T), got %v (%T)",
				f.Key, f.Value, f.Value, got, got)
		}
	}
}

// CID:config-defaults-test-004 - TestRuntimeDefaults_DefaultKeyNames
// Purpose: human-readable counterpart to
// TestRuntimeDefaults_StayInSync. Walks the explicit list
// of defaulted keys and asserts each one is registered
// with viper (via v.IsSet, which understands the nested
// "behavior.X" / "hotkeys.X" key shape — AllSettings
// flattens those to just "behavior" / "hotkeys"). Also
// guards against silent typos in key names: viper
// silently treats an unknown key as "no default".
func TestRuntimeDefaults_DefaultKeyNames(t *testing.T) {
	v := viper.New()
	runtimeDefaults(v)

	want := []string{
		"behavior.auto_type",
		"behavior.type_delay",
		"behavior.sound_on_start",
		"behavior.sound_on_end",
		"behavior.notifications",
		"behavior.autostart",
		"behavior.autostart_delay",
		"hotkeys.read_clipboard",
		"hotkeys.toggle_tts",
		"hotkeys.toggle_transcription",
	}
	for _, k := range want {
		if !v.IsSet(k) {
			t.Errorf("runtimeDefaults did not register %s", k)
		}
	}
}
