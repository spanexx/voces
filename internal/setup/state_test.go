package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestState_RoundTrip verifies a State value can be saved and loaded
// without losing any field.
func TestState_RoundTrip(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	original := &State{
		SchemaVersion: "1",
		CompletedAt:   time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
		AppVersion:    "v1.0.0",
		Language:      "en",
		WhisperModel:  "ggml-small.en.bin",
		PiperVoice:    "en_US-lessac-medium",
		HotkeyPreset:  HotkeyPresetCtrlSpace,
		CustomHotkey:  "",
	}
	if err := Save(original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.AppVersion != original.AppVersion {
		t.Errorf("AppVersion round-trip: got %q want %q", loaded.AppVersion, original.AppVersion)
	}
	if loaded.Language != original.Language {
		t.Errorf("Language round-trip: got %q want %q", loaded.Language, original.Language)
	}
	if loaded.WhisperModel != original.WhisperModel {
		t.Errorf("WhisperModel round-trip: got %q want %q", loaded.WhisperModel, original.WhisperModel)
	}
	if loaded.HotkeyPreset != original.HotkeyPreset {
		t.Errorf("HotkeyPreset round-trip: got %q want %q", loaded.HotkeyPreset, original.HotkeyPreset)
	}
	if !loaded.CompletedAt.Equal(original.CompletedAt) {
		t.Errorf("CompletedAt round-trip: got %v want %v", loaded.CompletedAt, original.CompletedAt)
	}
}

// TestSave_AtomicOnExistingFile verifies Save replaces the existing file
// atomically and the on-disk JSON is parseable.
func TestSave_AtomicOnExistingFile(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	first := &State{AppVersion: "v1.0.0", Language: "en", HotkeyPreset: HotkeyPresetCtrlSpace}
	second := &State{AppVersion: "v1.0.1", Language: "es", HotkeyPreset: HotkeyPresetRCtrlLeft}

	if err := Save(first); err != nil {
		t.Fatal(err)
	}
	if err := Save(second); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AppVersion != "v1.0.1" || loaded.Language != "es" {
		t.Errorf("second save did not win: %+v", loaded)
	}

	// The file should not be left as .tmp on disk.
	p, _ := pathForState()
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("expected .tmp file to be gone, stat err=%v", err)
	}
}

// TestLoad_MissingFile returns os.ErrNotExist (which the wizard checks
// via os.IsNotExist).
func TestLoad_MissingFile(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail when state.json does not exist")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.IsNotExist, got %v", err)
	}
}

// TestShouldRun_TruthTable covers every case in the spec.
func TestShouldRun_TruthTable(t *testing.T) {
	type tc struct {
		name              string
		seed              *State // nil = no state.json
		seedConfig        string // "" = no config.yaml
		currentAppVersion string
		want              bool
	}
	cases := []tc{
		{
			name:              "no state -> run",
			seed:              nil,
			seedConfig:        "",
			currentAppVersion: "v1.0.0",
			want:              true,
		},
		{
			name:              "same version -> skip",
			seed:              &State{AppVersion: "v1.0.0"},
			seedConfig:        "transcription:\n  whisper_cpp:\n    model: ggml-small.en.bin\n",
			currentAppVersion: "v1.0.0",
			want:              false,
		},
		{
			name:              "version upgrade -> run",
			seed:              &State{AppVersion: "v1.0.0"},
			seedConfig:        "transcription:\n  whisper_cpp:\n    model: ggml-small.en.bin\n",
			currentAppVersion: "v1.0.1",
			want:              true,
		},
		{
			name:              "version downgrade -> run",
			seed:              &State{AppVersion: "v1.0.1"},
			seedConfig:        "transcription:\n  whisper_cpp:\n    model: ggml-small.en.bin\n",
			currentAppVersion: "v1.0.0",
			want:              true,
		},
		{
			// Stale-state regression (rc1-hotpatch-12): the user
			// removed ~/.config/voces but kept
			// ~/.local/share/voces (where state.json lives). The
			// old ShouldRun saw state.AppVersion == current and
			// skipped the wizard — but config.yaml was missing
			// too, so the app loaded a default config with empty
			// model/binary paths and was effectively unusable.
			// The wizard must run to regenerate config.yaml.
			name:              "state present, config missing -> run",
			seed:              &State{AppVersion: "v1.0.0"},
			seedConfig:        "",
			currentAppVersion: "v1.0.0",
			want:              true,
		},
		{
			// Config present but model field is empty: the
			// wizard was never finished (e.g. user killed it
			// after language selection, before model step).
			// Re-run the wizard so the model is downloaded and
			// the field is filled in.
			name:              "state present, config model empty -> run",
			seed:              &State{AppVersion: "v1.0.0"},
			seedConfig:        "transcription:\n  whisper_cpp:\n    model: ''\n",
			currentAppVersion: "v1.0.0",
			want:              true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			if c.seed != nil {
				if err := Save(c.seed); err != nil {
					t.Fatal(err)
				}
			}
			if c.seedConfig != "" {
				cfgPath, err := configPath()
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(cfgPath, []byte(c.seedConfig), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := ShouldRun(c.currentAppVersion)
			if err != nil {
				t.Fatalf("ShouldRun: %v", err)
			}
			if got != c.want {
				t.Errorf("ShouldRun(%q) = %v, want %v", c.currentAppVersion, got, c.want)
			}
		})
	}
}

// TestLoadManifest_ParsesFixture writes a minimal models.json and verifies
// LoadManifest returns the expected fields.
func TestLoadManifest_ParsesFixture(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, manifestFileName)
	body := map[string]any{
		"whisper": map[string]any{
			"ggml-small.en.bin": map[string]any{
				"url":          "https://example.invalid/ggml-small.en.bin",
				"size_bytes":   float64(488479232),
				"language":     "en",
				"tier":         "small-en",
				"display_name": "Small (English)",
			},
		},
		"piper": map[string]any{
			"en_US-lessac-medium": map[string]any{
				"url":              "https://example.invalid/voice.onnx",
				"voice_config_url": "https://example.invalid/voice.onnx.json",
				"size_bytes":       float64(63123456),
				"language":         "en",
				"quality":          "medium",
				"display_name":     "US English (Lessac)",
			},
		},
	}
	data, _ := json.Marshal(body)
	if err := os.WriteFile(fixture, data, 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(fixture)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if _, ok := m.Whisper["ggml-small.en.bin"]; !ok {
		t.Error("expected whisper entry 'ggml-small.en.bin'")
	}
	if _, ok := m.Piper["en_US-lessac-medium"]; !ok {
		t.Error("expected piper entry 'en_US-lessac-medium'")
	}
	if m.Whisper["ggml-small.en.bin"].SizeBytes != 488479232 {
		t.Errorf("SizeBytes round-trip failed: got %d", m.Whisper["ggml-small.en.bin"].SizeBytes)
	}
}

// TestLoadManifest_InitialisesNilMaps verifies that an empty manifest
// (no whisper, no piper keys) returns maps that are non-nil so callers
// can range over them safely.
func TestLoadManifest_InitialisesNilMaps(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, manifestFileName)
	if err := os.WriteFile(fixture, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if m.Whisper == nil || m.Piper == nil {
		t.Errorf("expected non-nil empty maps, got whisper=%v piper=%v", m.Whisper, m.Piper)
	}
}

// TestDefaultManifest_HasBothRoutes verifies the fallback manifest includes
// the small.en and base entries that the wizard routes between.
func TestDefaultManifest_HasBothRoutes(t *testing.T) {
	m := DefaultManifest()
	if _, ok := m.Whisper["ggml-small.en.bin"]; !ok {
		t.Error("default manifest missing ggml-small.en.bin")
	}
	if _, ok := m.Whisper["ggml-base.bin"]; !ok {
		t.Error("default manifest missing ggml-base.bin")
	}
	if _, ok := m.Piper["en_US-lessac-medium"]; !ok {
		t.Error("default manifest missing en_US-lessac-medium")
	}
}
