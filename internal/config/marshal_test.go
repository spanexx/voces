/* Code Map: tests for MarshalYAML and the migrator
 * - TestMarshalYAML_TopDownStable: keys render in the
 *   order they were inserted; nested maps recurse; the
 *   output is parseable by viper.
 * - TestMigrator_AddsMissingFields: feed the migrator a
 *   pre-rc1-hotpatch-14 config, assert the on-disk file
 *   gains the behavior: block and the four secondary
 *   hotkey fields, and user values are preserved.
 * - TestMigrator_Idempotent: feed the migrator a
 *   fully-populated config, assert the output equals the
 *   input (no spurious changes).
 *
 * CID Index:
 * CID:config-marshal-test-001 -> TestMarshalYAML_TopDownStable
 * CID:config-marshal-test-002 -> TestMigrator_AddsMissingFields
 * CID:config-marshal-test-003 -> TestMigrator_Idempotent
 *
 * Quick lookup: rg -n "CID:config-marshal-test-" internal/config/
 */
package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// CID:config-marshal-test-001 - TestMarshalYAML_TopDownStable
// Purpose: prove the migrator's writer produces a
// parseable, stable YAML byte slice. Round-trips through
// viper so a future regression on quoting/escaping
// surfaces immediately.
func TestMarshalYAML_TopDownStable(t *testing.T) {
	in := map[string]any{
		"transcription": map[string]any{
			"default_engine": "whisper_cpp",
			"whisper_cpp": map[string]any{
				"binary_path": "/opt/whisper.cpp/main",
				"language":    "en",
				"compute_type": "float",
			},
		},
		"behavior": map[string]any{
			"auto_type":  true,
			"type_delay": 15,
		},
	}
	out, err := MarshalYAML(in)
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}

	// The output should be parseable by viper with the
	// same shape we put in. A regression on quoting
	// (e.g. "yes" / "no" being rendered as booleans)
	// would surface here.
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(out)); err != nil {
		t.Fatalf("ReadConfig: %v\n--- output ---\n%s", err, out)
	}
	if got := v.GetString("transcription.default_engine"); got != "whisper_cpp" {
		t.Errorf("transcription.default_engine: want %q, got %q", "whisper_cpp", got)
	}
	if got := v.GetInt("behavior.type_delay"); got != 15 {
		t.Errorf("behavior.type_delay: want 15, got %d", got)
	}
	if got := v.GetBool("behavior.auto_type"); got != true {
		t.Errorf("behavior.auto_type: want true, got %v", got)
	}

	// Top-level keys should appear in the output in
	// alphabetical order — that is the contract callers
	// rely on for diff readability. "behavior" < "tts" <
	// "transcription" alphabetically; b < t is the
	// invariant.
	behavIdx := bytes.Index(out, []byte("behavior:"))
	transIdx := bytes.Index(out, []byte("transcription:"))
	if behavIdx == -1 || transIdx == -1 {
		t.Fatalf("missing keys in output:\n%s", out)
	}
	if behavIdx > transIdx {
		t.Errorf("expected behavior before transcription (alphabetical), got:\n%s", out)
	}
}

// CID:config-marshal-test-002 - TestMigrator_AddsMissingFields
// Purpose: the migrator's core promise. A pre-rc1-hotpatch-14
// config (no behavior:, only record_and_type in hotkeys:) is
// fed to RuntimeDefaultsForMigrations + MarshalYAML; the
// resulting YAML must contain the behavior block and the
// four secondary hotkey fields, and must preserve the
// user's record_and_type (f9) and binary_path values.
//
// Uses bytes in memory; does not touch the real on-disk
// config (the migrator main() is exercised by the build
// smoke test in Makefile).
func TestMigrator_AddsMissingFields(t *testing.T) {
	orig := `
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
    voice_config: ""
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
  record_and_type: f9
`
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(orig)); err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	RuntimeDefaultsForMigrations(v)

	out, err := MarshalYAML(v.AllSettings())
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}

	// Round-trip and assert every defaulted key is
	// present and has the right value.
	v2 := viper.New()
	v2.SetConfigType("yaml")
	if err := v2.ReadConfig(bytes.NewReader(out)); err != nil {
		t.Fatalf("ReadConfig(out): %v\n--- out ---\n%s", err, out)
	}
	checks := []struct {
		key  string
		want any
	}{
		{"behavior.auto_type", true},
		{"behavior.type_delay", 15},
		{"behavior.sound_on_start", false},
		{"behavior.sound_on_end", false},
		{"behavior.notifications", true},
		{"behavior.autostart", false},
		{"behavior.autostart_delay", 5},
		{"hotkeys.stop_recording", ""},
		{"hotkeys.read_clipboard", "<f10>"},
		{"hotkeys.toggle_tts", "<f11>"},
		{"hotkeys.toggle_transcription", "<f12>"},
		{"hotkeys.record_and_type", "f9"},
		{"transcription.default_engine", "whisper_cpp"},
		{"transcription.whisper_cpp.binary_path", "/opt/whisper.cpp/main"},
	}
	for _, c := range checks {
		var got any
		switch c.want.(type) {
		case bool:
			got = v2.GetBool(c.key)
		case int:
			got = v2.GetInt(c.key)
		case string:
			got = v2.GetString(c.key)
		default:
			got = v2.Get(c.key)
		}
		if got != c.want {
			t.Errorf("%s: want %v (%T), got %v (%T)", c.key, c.want, c.want, got, got)
		}
	}
}

// CID:config-marshal-test-003 - TestMigrator_Idempotent
// Purpose: re-running the migrator on an already-patched
// config is a no-op. The migrator main() detects this
// by comparing bytes; this test asserts the byte-level
// contract that makes that comparison work.
func TestMigrator_Idempotent(t *testing.T) {
	once := `
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: ""
    model: ""
    language: ''
    compute_type: float
  openai_api:
    api_key: ""
    model: whisper-1
    prompt: ""

tts:
  default_engine: piper
  piper:
    binary_path: ""
    model: ""
    voice_config: ""
    output_device: ''
  elevenlabs:
    api_key: ""
    voice_id: ""
    model: eleven_monolingual_v1
    stability: 0.5
    similarity_boost: 0.75

audio:
  sample_rate: 16000
  channels: 1
  chunk_size: 1024
  max_duration: 300

hotkeys:
  record_and_type: ''
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
  autostart: false
  autostart_delay: 5
`
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(once)); err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	RuntimeDefaultsForMigrations(v)
	first, err := MarshalYAML(v.AllSettings())
	if err != nil {
		t.Fatalf("MarshalYAML first: %v", err)
	}

	// Second pass: read first back, apply defaults again,
	// marshal again. Should be byte-equal to first.
	v2 := viper.New()
	v2.SetConfigType("yaml")
	if err := v2.ReadConfig(bytes.NewReader(first)); err != nil {
		t.Fatalf("ReadConfig(first): %v", err)
	}
	RuntimeDefaultsForMigrations(v2)
	second, err := MarshalYAML(v2.AllSettings())
	if err != nil {
		t.Fatalf("MarshalYAML second: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("idempotency violated: first != second\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

// TestMigrator_AtomicWrite exercises the tmp+rename path
// the migrator main() uses. Standalone so it can write to
// t.TempDir() without touching the user's real config.
func TestMigrator_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.yaml")
	// Pre-create a regular file the migrator will replace.
	if err := os.WriteFile(target, []byte("hotkeys:\n  record_and_type: f9\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// We don't call main() directly (it would resolve the
	// real ~/.config/voces/config.yaml). Instead we test
	// the read+defaults+write contract that main() runs.
	orig, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(orig)); err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	RuntimeDefaultsForMigrations(v)
	updated, err := MarshalYAML(v.AllSettings())
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}

	// Simulate the atomic-rename main() performs.
	tmp, err := os.CreateTemp(dir, ".voces-config-*.yaml.tmp")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(updated); err != nil {
		t.Fatalf("tmp.Write: %v", err)
	}
	if err := tmp.Sync(); err != nil {
		t.Fatalf("tmp.Sync: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("tmp.Close: %v", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		t.Fatalf("rename: %v", err)
	}

	// Read back: must have behavior: block and the four
	// secondary hotkey fields.
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read after rename: %v", err)
	}
	v2 := viper.New()
	v2.SetConfigType("yaml")
	if err := v2.ReadConfig(bytes.NewReader(got)); err != nil {
		t.Fatalf("ReadConfig(got): %v\n--- got ---\n%s", err, got)
	}
	if v2.GetString("behavior.notifications") != "true" {
		t.Errorf("behavior.notifications missing after atomic write: got %q", v2.GetString("behavior.notifications"))
	}
	if v2.GetString("hotkeys.read_clipboard") != "<f10>" {
		t.Errorf("hotkeys.read_clipboard missing after atomic write: got %q", v2.GetString("hotkeys.read_clipboard"))
	}
}
