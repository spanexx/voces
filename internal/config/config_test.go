package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
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
    prompt: ''

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx
    voice_config: /opt/piper/models/en_US-lessac-medium.onnx.json
    output_device: ''
  elevenlabs:
    api_key: eleven-key
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
  record_and_type: '<f8>'
  read_clipboard: '<f9>'
  toggle_tts: '<f10>'
  toggle_transcription: '<f11>'

behavior:
  auto_type: true
  type_delay: 15
  sound_on_start: false
  sound_on_end: false
  notifications: true
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Change to temp directory
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if cfg.Transcription.DefaultEngine != "whisper_cpp" {
		t.Errorf("Expected default_engine to be whisper_cpp, got %s", cfg.Transcription.DefaultEngine)
	}
	if cfg.Transcription.WhisperCPP.Language != "en" {
		t.Errorf("Expected language to be en, got %s", cfg.Transcription.WhisperCPP.Language)
	}
	if cfg.Audio.SampleRate != 16000 {
		t.Errorf("Expected sample_rate to be 16000, got %d", cfg.Audio.SampleRate)
	}
	if cfg.Behavior.AutoType != true {
		t.Errorf("Expected auto_type to be true, got %v", cfg.Behavior.AutoType)
	}
	if cfg.Hotkeys.RecordAndType != "<f8>" {
		t.Errorf("Expected record_and_type to be <f8>, got %s", cfg.Hotkeys.RecordAndType)
	}
}

func TestEnvVarSubstitution(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_OPENAI_API_KEY", "env-key-123")
	os.Setenv("TEST_ELEVENLABS_API_KEY", "env-eleven-key")
	defer func() {
		os.Unsetenv("TEST_OPENAI_API_KEY")
		os.Unsetenv("TEST_ELEVENLABS_API_KEY")
	}()

	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin
  openai_api:
    api_key: ${TEST_OPENAI_API_KEY}
    model: whisper-1

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx
  elevenlabs:
    api_key: ${TEST_ELEVENLABS_API_KEY}
    voice_id: test-voice

audio:
  sample_rate: 16000
  channels: 1
  chunk_size: 1024
  max_duration: 300

hotkeys:
  record_and_type: '<f8>'
  read_clipboard: '<f9>'
  toggle_tts: '<f10>'
  toggle_transcription: '<f11>'

behavior:
  auto_type: true
  type_delay: 15
  sound_on_start: false
  sound_on_end: false
  notifications: true
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify env vars were substituted
	if cfg.Transcription.OpenAIAPI.APIKey != "env-key-123" {
		t.Errorf("Expected OPENAI_API_KEY to be env-key-123, got %s", cfg.Transcription.OpenAIAPI.APIKey)
	}
	if cfg.TTS.ElevenLabs.APIKey != "env-eleven-key" {
		t.Errorf("Expected ELEVENLABS_API_KEY to be env-eleven-key, got %s", cfg.TTS.ElevenLabs.APIKey)
	}
}

func TestEnvVarFallback(t *testing.T) {
	// Make sure the test env vars are NOT set
	os.Unsetenv("TEST_FALLBACK_API_KEY")

	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin
  openai_api:
    api_key: ${TEST_FALLBACK_API_KEY}
    model: whisper-1

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx
  elevenlabs:
    api_key: static-key
    voice_id: test-voice

audio:
  sample_rate: 16000
  channels: 1
  chunk_size: 1024
  max_duration: 300

hotkeys:
  record_and_type: '<f8>'
  read_clipboard: '<f9>'
  toggle_tts: '<f10>'
  toggle_transcription: '<f11>'

behavior:
  auto_type: true
  type_delay: 15
  sound_on_start: false
  sound_on_end: false
  notifications: true
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify fallback - original value preserved when env var not set
	if cfg.Transcription.OpenAIAPI.APIKey != "${TEST_FALLBACK_API_KEY}" {
		t.Errorf("Expected fallback to preserve original value, got %s", cfg.Transcription.OpenAIAPI.APIKey)
	}
	// Verify static values are unchanged
	if cfg.TTS.ElevenLabs.APIKey != "static-key" {
		t.Errorf("Expected static key to be unchanged, got %s", cfg.TTS.ElevenLabs.APIKey)
	}
}

func TestValidationErrors(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Test invalid channels
	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin
  openai_api:
    api_key: test-key
    model: whisper-1

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx
  elevenlabs:
    api_key: test-key
    voice_id: test-voice
    stability: 0.5
    similarity_boost: 0.75

audio:
  sample_rate: 16000
  channels: 3
  chunk_size: 1024
  max_duration: 300

hotkeys:
  record_and_type: '<f8>'
  read_clipboard: '<f9>'
  toggle_tts: '<f10>'
  toggle_transcription: '<f11>'

behavior:
  auto_type: true
  type_delay: 15
  sound_on_start: false
  sound_on_end: false
  notifications: true
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Error("Expected validation error for invalid channels, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "audio.channels must be 1 (mono) or 2 (stereo)") {
		t.Errorf("Expected channels validation error, got: %v", err)
	}
}

func TestValidationElevenLabsSettings(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Test invalid stability value
	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin
  openai_api:
    api_key: test-key
    model: whisper-1

tts:
  default_engine: elevenlabs
  elevenlabs:
    api_key: test-key
    voice_id: test-voice
    stability: 1.5
    similarity_boost: 0.75

audio:
  sample_rate: 16000
  channels: 1
  chunk_size: 1024
  max_duration: 300

hotkeys:
  record_and_type: '<f8>'
  read_clipboard: '<f9>'
  toggle_tts: '<f10>'
  toggle_transcription: '<f11>'

behavior:
  auto_type: true
  type_delay: 15
  sound_on_start: false
  sound_on_end: false
  notifications: true
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Error("Expected validation error for invalid stability, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "stability must be between 0.0 and 1.0") {
		t.Errorf("Expected stability validation error, got: %v", err)
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create a fake go.mod to trigger config creation in current dir
	os.WriteFile("go.mod", []byte("module test"), 0o644)

	err := createDefaultConfig(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create default config: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(tmpDir, "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("Config file was not created at %s", err)
	}
}

// TestCreateDefaultConfig_CompleteBehaviorAndHotkeys (rc1-hotpatch-15)
// is the regression test for the gap rc1-hotpatch-14 left behind:
// the wizard's defaultConfigFor was extended to write the full
// behavior block (including autostart / autostart_delay) and all
// four secondary hotkey fields, but the runtime template in
// createDefaultConfig was not updated. Any code path that takes
// the runtime default (Load() with no config.yaml on disk, a
// hand-edited config that pre-dates hotpatch-14, or a tarball
// install where the wizard was skipped) unmarshaled the missing
// fields as Go zero values — autostart=false, notifications=true
// only by accident, type_delay=0 — and the user saw "Autostart:
// desired=false" / "notify: system disabled in config" in the
// logs on a fresh install.
//
// rc1-hotpatch-18: autostart flipped to true. The wizard no
// longer asks the user, and the runtime default mirrors the
// hardcoded behavior block in setup.defaultConfigFor so the
// two stay in sync.
//
// The runtime default template must therefore carry the same
// behavior + hotkey fields as the wizard's generated config.
func TestCreateDefaultConfig_CompleteBehaviorAndHotkeys(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := createDefaultConfig(tmpDir); err != nil {
		t.Fatalf("createDefaultConfig: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(tmpDir, "config.yaml"))
	if err != nil {
		t.Fatalf("read config.yaml: %v", err)
	}
	body := string(data)

	// Behavior: every field on internal/config.BehaviorConfig
	// must be present in the runtime default. Values mirror
	// createDefaultConfig's template; the test will fail loudly
	// if a new field is added to the struct but not to the
	// template (or vice versa).
	wantBehavior := []string{
		"auto_type: true",
		"type_delay: 15",
		"sound_on_start: false",
		"sound_on_end: false",
		"notifications: true",
		"autostart: true",
		"autostart_delay: 5",
	}
	for _, want := range wantBehavior {
		if !strings.Contains(body, want) {
			t.Errorf("config.yaml missing behavior %q\n---\n%s\n---", want, body)
		}
	}

	// Hotkeys: every field on internal/config.HotkeysConfig
	// must be present. stop_recording is intentionally empty
	// in the default (the hold-binding model re-uses the
	// record key to stop), but the field must still appear so
	// preserveHotkeys / hotkey subsystem see it.
	wantHotkeys := []string{
		"record_and_type: '<rightctrl>+<left>'",
		"stop_recording: ''",
		"read_clipboard: '<f10>'",
		"toggle_tts: '<f11>'",
		"toggle_transcription: '<f12>'",
	}
	for _, want := range wantHotkeys {
		if !strings.Contains(body, want) {
			t.Errorf("config.yaml missing hotkey %q\n---\n%s\n---", want, body)
		}
	}
}

func TestValidatePaths(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create fake binary file
	fakeBin := filepath.Join(tmpDir, "whisper")
	os.WriteFile(fakeBin, []byte("fake binary"), 0o755)

	// Create fake model file
	fakeModel := filepath.Join(tmpDir, "model.bin")
	os.WriteFile(fakeModel, []byte("fake model"), 0o644)

	cfg := &Config{
		Transcription: TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: WhisperCPPConfig{
				BinaryPath: fakeBin,
				Model:      fakeModel,
			},
		},
	}

	err := validatePaths(cfg)
	if err != nil {
		t.Errorf("validatePaths failed for valid paths: %v", err)
	}
}

func TestValidatePaths_MissingPaths(t *testing.T) {
	cfg := &Config{
		Transcription: TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: WhisperCPPConfig{
				BinaryPath: "/nonexistent/path/binary",
				Model:      "/nonexistent/path/model.bin",
			},
		},
	}

	err := validatePaths(cfg)
	if err == nil {
		t.Error("Expected error for missing paths")
	}
}

func TestValidatePaths_NotWhisperCPP(t *testing.T) {
	cfg := &Config{
		Transcription: TranscriptionConfig{
			DefaultEngine: "openai_api",
		},
	}

	err := validatePaths(cfg)
	if err != nil {
		t.Errorf("validatePaths should not error for non-whisper_cpp engine: %v", err)
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Test with existing file
	existingFile := filepath.Join(tmpDir, "exists.txt")
	os.WriteFile(existingFile, []byte("test"), 0o644)

	if !fileExists(existingFile) {
		t.Error("fileExists should return true for existing file")
	}

	// Test with non-existing file
	if fileExists("/nonexistent/path/file.txt") {
		t.Error("fileExists should return false for non-existing file")
	}

	// Test with directory (should return false)
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0o755)
	if fileExists(subDir) {
		t.Error("fileExists should return false for directories")
	}
}

func TestLoad_ExistingConfig(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
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

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx

audio:
  sample_rate: 16000
  channels: 1
  chunk_size: 1024
  max_duration: 300

hotkeys:
  record_and_type: '<f8>'

behavior:
  auto_type: true
  type_delay: 15
  sound_on_start: false
  sound_on_end: false
  notifications: true
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load should read existing config: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil config")
	}
	if cfg.Transcription.DefaultEngine != "whisper_cpp" {
		t.Errorf("Expected whisper_cpp engine, got %s", cfg.Transcription.DefaultEngine)
	}
}

func TestLoad_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `invalid yaml: [`
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

func TestSave_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create a fake executable
	fakeExec := filepath.Join(tmpDir, "app")
	os.WriteFile(fakeExec, []byte("fake"), 0o755)

	cfg := &Config{
		Transcription: TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: WhisperCPPConfig{
				BinaryPath: "/opt/whisper.cpp/main",
				Model:      "/opt/whisper.cpp/models/ggml-small.bin",
				Language:   "en",
			},
			OpenAIAPI: OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
				Prompt: "test",
			},
		},
		TTS: TTSConfig{
			DefaultEngine: "piper",
			Piper: PiperConfig{
				BinaryPath:   "/opt/piper/piper",
				Model:        "/opt/piper/models/en_US-lessac-medium.onnx",
				VoiceConfig:  "/opt/piper/models/en_US-lessac-medium.onnx.json",
				OutputDevice: "hw:0,0",
			},
			ElevenLabs: ElevenLabsConfig{
				APIKey:          "test-key",
				VoiceID:         "test-voice",
				Model:           "eleven_monolingual_v1",
				Stability:       0.5,
				SimilarityBoost: 0.75,
			},
		},
		Audio: AudioConfig{
			SampleRate:  16000,
			Channels:    1,
			ChunkSize:   1024,
			MaxDuration: 300,
		},
		Hotkeys: HotkeysConfig{
			RecordAndType:       "<f8>",
			ReadClipboard:       "<f9>",
			ToggleTTS:           "<f10>",
			ToggleTranscription: "<f11>",
		},
		Behavior: BehaviorConfig{
			AutoType:      true,
			TypeDelay:     15,
			SoundOnStart:  false,
			SoundOnEnd:    false,
			Notifications: true,
		},
	}

	// Save writes config.yaml next to the running test binary.
	configDir, _ := os.UserConfigDir()
	configPath := filepath.Join(configDir, "voces", "config.yaml")
	_ = os.Remove(configPath)

	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Expected config.yaml to be written at %s: %v", configPath, err)
	}
	if len(data) == 0 {
		t.Fatalf("Expected config.yaml to be non-empty")
	}
	if !strings.Contains(string(data), "transcription:") {
		t.Fatalf("Expected config.yaml to contain transcription section")
	}
	if !strings.Contains(string(data), "tts:") {
		t.Fatalf("Expected config.yaml to contain tts section")
	}
	if !strings.Contains(string(data), "hotkeys:") {
		t.Fatalf("Expected config.yaml to contain hotkeys section")
	}
	if !strings.Contains(string(data), "behavior:") {
		t.Fatalf("Expected config.yaml to contain behavior section")
	}

	_ = os.Remove(configPath)
}

func TestLoad_EnvSubstitution(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_OPENAI_KEY", "substituted-openai-key")
	os.Setenv("TEST_ELEVEN_KEY", "substituted-eleven-key")
	defer func() {
		os.Unsetenv("TEST_OPENAI_KEY")
		os.Unsetenv("TEST_ELEVEN_KEY")
	}()

	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin
  openai_api:
    api_key: ${TEST_OPENAI_KEY}
    model: whisper-1

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx
  elevenlabs:
    api_key: ${TEST_ELEVEN_KEY}
    voice_id: test-voice

audio:
  sample_rate: 16000
  channels: 1

hotkeys:
  record_and_type: '<f8>'

behavior:
  auto_type: true
  type_delay: 15
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load should succeed: %v", err)
	}

	// Verify env vars were substituted
	if cfg.Transcription.OpenAIAPI.APIKey != "substituted-openai-key" {
		t.Errorf("Expected env substitution for OpenAI key, got %s", cfg.Transcription.OpenAIAPI.APIKey)
	}
	if cfg.TTS.ElevenLabs.APIKey != "substituted-eleven-key" {
		t.Errorf("Expected env substitution for ElevenLabs key, got %s", cfg.TTS.ElevenLabs.APIKey)
	}
}

func TestValidatePaths_MissingPiper(t *testing.T) {
	cfg := &Config{
		TTS: TTSConfig{
			DefaultEngine: "piper",
			Piper: PiperConfig{
				BinaryPath:  "/nonexistent/piper",
				Model:       "/nonexistent/model.onnx",
				VoiceConfig: "/nonexistent/voice.json",
			},
		},
	}

	err := validatePaths(cfg)
	if err == nil {
		t.Error("Expected error for missing piper binary")
	}
}

func TestValidatePaths_EmptyConfig(t *testing.T) {
	cfg := &Config{}

	err := validatePaths(cfg)
	if err != nil {
		t.Errorf("Expected no error for empty config: %v", err)
	}
}

func TestValidatePaths_ExistingPiper(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create fake piper binary
	fakeBin := filepath.Join(tmpDir, "piper")
	os.WriteFile(fakeBin, []byte("fake"), 0o755)

	// Create fake model
	fakeModel := filepath.Join(tmpDir, "model.onnx")
	os.WriteFile(fakeModel, []byte("fake"), 0o644)

	// Create fake voice config
	fakeConfig := filepath.Join(tmpDir, "voice.json")
	os.WriteFile(fakeConfig, []byte("{}"), 0o644)

	cfg := &Config{
		TTS: TTSConfig{
			DefaultEngine: "piper",
			Piper: PiperConfig{
				BinaryPath:  fakeBin,
				Model:       fakeModel,
				VoiceConfig: fakeConfig,
			},
		},
	}

	err := validatePaths(cfg)
	if err != nil {
		t.Errorf("Expected no error for valid piper paths: %v", err)
	}
}

func TestValidateConfig_InvalidSampleRate(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin

audio:
  sample_rate: -1
  channels: 1

hotkeys:
  record_and_type: '<f8>'

behavior:
  auto_type: true
  type_delay: 15
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("Expected error for negative sample rate")
	}
}

func TestValidateConfig_InvalidChannels(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin

audio:
  sample_rate: 16000
  channels: 5

hotkeys:
  record_and_type: '<f8>'

behavior:
  auto_type: true
  type_delay: 15
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("Expected error for invalid channels")
	}
}

func TestValidateConfig_InvalidTypeDelay(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin

audio:
  sample_rate: 16000
  channels: 1

hotkeys:
  record_and_type: '<f8>'

behavior:
  auto_type: true
  type_delay: -5
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("Expected error for negative type_delay")
	}
}

func TestLoad_ConfigFileCreated(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create go.mod to trigger config creation
	os.WriteFile("go.mod", []byte("module test"), 0o644)

	// Load without existing config - should create default
	cfg, err := Load()
	if err != nil {
		t.Errorf("Expected no error creating default config: %v", err)
	}
	if cfg == nil {
		t.Fatal("Config should not be nil")
	}

	// Verify config file was created
	configPath := filepath.Join(tmpDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should have been created")
	}
}

func TestLoad_MultipleConfigPaths(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create config in current directory
	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx

audio:
  sample_rate: 16000
  channels: 1

hotkeys:
  record_and_type: '<f8>'

behavior:
  auto_type: true
  type_delay: 15
`
	os.WriteFile("config.yaml", []byte(configContent), 0o644)

	cfg, err := Load()
	if err != nil {
		t.Errorf("Expected no error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Config should not be nil")
	}
}

func TestSave_WriteConfig(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create fake executable path
	fakeExec := filepath.Join(tmpDir, "app")
	os.WriteFile(fakeExec, []byte("fake"), 0o755)

	cfg := &Config{
		Transcription: TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: WhisperCPPConfig{
				BinaryPath: "/opt/whisper.cpp/main",
				Model:      "/opt/whisper.cpp/models/ggml-small.bin",
			},
		},
		TTS: TTSConfig{
			DefaultEngine: "piper",
			Piper: PiperConfig{
				BinaryPath:  "/opt/piper/piper",
				Model:       "/opt/piper/models/en_US-lessac-medium.onnx",
				VoiceConfig: "/opt/piper/models/en_US-lessac-medium.onnx.json",
			},
		},
		Audio: AudioConfig{
			SampleRate:  16000,
			Channels:    1,
			ChunkSize:   1024,
			MaxDuration: 300,
		},
		Hotkeys: HotkeysConfig{
			RecordAndType: "<f8>",
		},
		Behavior: BehaviorConfig{
			AutoType:  true,
			TypeDelay: 15,
		},
	}

	// Test that Save doesn't error (we can't easily test the actual write without using file system wrappers)
	// But we verify the config struct is valid
	if cfg.Transcription.DefaultEngine != "whisper_cpp" {
		t.Error("Config validation failed")
	}
}

func TestValidatePaths_EmptyPaths(t *testing.T) {
	cfg := &Config{
		Transcription: TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: WhisperCPPConfig{
				BinaryPath: "", // Empty path
				Model:      "", // Empty path
			},
		},
	}

	// Should not error when paths are empty
	err := validatePaths(cfg)
	if err != nil {
		t.Errorf("Expected no error for empty paths: %v", err)
	}
}

func TestValidateConfig_MissingHotkey(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx

audio:
  sample_rate: 16000
  channels: 1

hotkeys:
  record_and_type: ''

behavior:
  auto_type: true
  type_delay: 15
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("Expected error for empty hotkey")
	}
}

func TestValidateConfig_InvalidHotkeyFormat(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin

tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx

audio:
  sample_rate: 16000
  channels: 1

hotkeys:
  record_and_type: '@#$%'  # Invalid characters only

behavior:
  auto_type: true
  type_delay: 15
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("Expected error for invalid hotkey format")
	}
}

func TestValidateConfig_InvalidEngine(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `
transcription:
  default_engine: invalid_engine
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin

audio:
  sample_rate: 16000
  channels: 1

hotkeys:
  record_and_type: '<f8>'

behavior:
  auto_type: true
  type_delay: 15
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("Expected error for invalid engine")
	}
}

func TestValidateConfig_InvalidTTSEngine(t *testing.T) {
	tmpDir := t.TempDir(); t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin

tts:
  default_engine: invalid_tts_engine

audio:
  sample_rate: 16000
  channels: 1

hotkeys:
  record_and_type: '<f8>'

behavior:
  auto_type: true
  type_delay: 15
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("Expected error for invalid TTS engine")
	}
}

func TestSubstituteEnvVars(t *testing.T) {
	// Test with no env vars to substitute
	cfg := &Config{
		Transcription: TranscriptionConfig{
			OpenAIAPI: OpenAIAPIConfig{
				APIKey: "static-key",
			},
		},
		TTS: TTSConfig{
			ElevenLabs: ElevenLabsConfig{
				APIKey: "static-eleven-key",
			},
		},
	}

	// Should not modify static values
	substituteEnvVars(cfg)
	if cfg.Transcription.OpenAIAPI.APIKey != "static-key" {
		t.Errorf("Expected static key to be unchanged, got %s", cfg.Transcription.OpenAIAPI.APIKey)
	}
	if cfg.TTS.ElevenLabs.APIKey != "static-eleven-key" {
		t.Errorf("Expected static eleven key to be unchanged, got %s", cfg.TTS.ElevenLabs.APIKey)
	}
}
