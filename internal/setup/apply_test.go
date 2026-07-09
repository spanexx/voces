/* Code Map: setup.Apply tests
 * - TestApply_WritesStateAndConfig: happy path, both files exist with right fields
 * - TestApply_PreservesExistingBinaryPaths: pre-existing user paths survive
 *
 * CID Index:
 * CID:setup-apply-test-001 -> TestApply_WritesStateAndConfig
 * CID:setup-apply-test-002 -> TestApply_PreservesExistingBinaryPaths
 */
package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestApply_WritesRecordAndTypeFromWizardChoice is the regression test
// for the rc1-hotpatch-11 bug: the user ran the wizard, picked
// "custom" hotkey = "f9", the wizard wrote state.json correctly,
// but config.yaml had no hotkeys.record_and_type field at all, so
// the runtime validation
//   hotkeys.record_and_type is required
// crashed voces with "Failed to initialize application".
//
// The wizard's HotkeyPreset + CustomHotkey must land in
// config.yaml's hotkeys.record_and_type field, otherwise the
// hotkey subsystem has nothing to bind to.
//
// Each preset is tested in its own subtest because the conversion
// is per-preset (ctrl-space → "ctrl+space", rctrl-left →
// "<rightctrl>+<left>", f8 → "<f8>", custom → verbatim).
func TestApply_WritesRecordAndTypeFromWizardChoice(t *testing.T) {
	cases := []struct {
		name     string
		preset   string
		custom   string
		wantSub  string // substring expected in config.yaml's record_and_type
	}{
		{"ctrl-space", HotkeyPresetCtrlSpace, "", "ctrl+space"},
		{"rctrl-left", HotkeyPresetRCtrlLeft, "", "<rightctrl>+<left>"},
		{"f8", HotkeyPresetF8, "", "<f8>"},
		{"custom-f9", HotkeyPresetCustom, "f9", "f9"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())

			state := &State{
				SchemaVersion: "1",
				AppVersion:    "v0.1.0",
				Language:      "en",
				WhisperModel:  "ggml-small.en.bin",
				HotkeyPreset:  tc.preset,
				CustomHotkey:  tc.custom,
			}
			if err := Apply(state, DefaultManifest()); err != nil {
				t.Fatalf("Apply: %v", err)
			}
			cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "voces", "config.yaml")
			data, err := os.ReadFile(cfgPath)
			if err != nil {
				t.Fatalf("read config.yaml: %v", err)
			}
			if !contains(data, []byte("record_and_type:")) {
				t.Errorf("config.yaml has no record_and_type field at all:\n%s", data)
			}
			if !contains(data, []byte(tc.wantSub)) {
				t.Errorf("config.yaml record_and_type missing %q\nfull config:\n%s", tc.wantSub, data)
			}
		})
	}
}

// TestApply_PreservesPreExistingHotkeys: if a user re-runs the
// wizard but had customized the read_clipboard / toggle_tts / etc.
// hotkeys in a previous config, the wizard must not stomp them.
// The wizard owns record_and_type (it's the one being picked), but
// the other four fields follow the "user wins" rule that already
// applies to binary paths.
func TestApply_PreservesPreExistingHotkeys(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	appCfgDir := filepath.Join(cfgDir, "voces")
	if err := os.MkdirAll(appCfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-existing config with a user-customized read_clipboard.
	preExisting := `hotkeys:
  record_and_type: '<rightctrl>+<left>'
  read_clipboard: '<f5>'
  toggle_tts: '<f6>'
`
	if err := os.WriteFile(filepath.Join(appCfgDir, "config.yaml"), []byte(preExisting), 0o644); err != nil {
		t.Fatal(err)
	}

	state := &State{
		AppVersion:   "v0.1.0",
		Language:     "en",
		WhisperModel: "ggml-small.en.bin",
		HotkeyPreset: HotkeyPresetCtrlSpace, // wizard's new choice
	}
	if err := Apply(state, DefaultManifest()); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(appCfgDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(data, []byte("<f5>")) {
		t.Errorf("user-customized read_clipboard <f5> was stomped:\n%s", data)
	}
	if !contains(data, []byte("<f6>")) {
		t.Errorf("user-customized toggle_tts <f6> was stomped:\n%s", data)
	}
}

// TestApply_WritesStateAndConfig verifies that Apply writes both state.json
// and config.yaml, and that config.yaml carries the model + engine paths
// derived from the State. This is the IMPL §3 Phase 5 happy-path contract.
func TestApply_WritesStateAndConfig(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	state := &State{
		SchemaVersion: "1",
		AppVersion:    "v0.1.0",
		CompletedAt:   time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC),
		Language:      "en",
		WhisperModel:  "ggml-small.en.bin",
		PiperVoice:    "",
		HotkeyPreset:  HotkeyPresetCtrlSpace,
		CustomHotkey:  "",
	}
	manifest := DefaultManifest()

	if err := Apply(state, manifest); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// state.json must exist at the canonical path and round-trip.
	statePath, err := pathForState()
	if err != nil {
		t.Fatalf("pathForState: %v", err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load after Apply: %v", err)
	}
	if loaded.AppVersion != state.AppVersion {
		t.Errorf("state.json AppVersion: got %q want %q", loaded.AppVersion, state.AppVersion)
	}
	if loaded.WhisperModel != state.WhisperModel {
		t.Errorf("state.json WhisperModel: got %q want %q", loaded.WhisperModel, state.WhisperModel)
	}
	_ = statePath

	// config.yaml must exist at $XDG_CONFIG_HOME/voces/config.yaml
	configPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "voces", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config.yaml missing at %q: %v", configPath, err)
	}

	// The config must point at the canonical model path for the chosen model.
	modelPath, err := whisperModelPathForTest(state.WhisperModel)
	if err != nil {
		t.Fatalf("whisperModelPathForTest: %v", err)
	}
	wantModel := modelPath
	if !contains(data, []byte(wantModel)) {
		t.Errorf("config.yaml missing whisper model path %q\n---\n%s\n---", wantModel, data)
	}
}

// TestApply_PreservesExistingBinaryPaths verifies that when a user already
// has a config.yaml with custom binary paths (from a pre-wizard install),
// Apply does not stomp on those paths. This is the IMPL §3 Phase 5
// "preserves user paths" regression contract.
//
// rc30: preserveBinaryPath now also validates that the pre-existing
// value points at a real, executable file (rc30's TTS Unavailable
// fix). To keep this end-to-end test honest, we drop real executable
// files in t.TempDir() and point the pre-existing config at them —
// a real file is the only honest way to make the test pass under
// the new validation rule.
func TestApply_PreservesExistingBinaryPaths(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	// Real, executable binaries in t.TempDir() so the rc30
	// preserveBinaryPath validation passes (os.Stat + exec bit).
	binDir := t.TempDir()
	whisperBin := filepath.Join(binDir, "whisper-cli")
	piperBin := filepath.Join(binDir, "piper")
	if err := os.WriteFile(whisperBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(piperBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-write a config.yaml with the user's chosen binary paths.
	appCfgDir := filepath.Join(cfgDir, "voces")
	if err := os.MkdirAll(appCfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	preExisting := `transcription:
  whisper_cpp:
    binary_path: ` + whisperBin + `
    model: ""
tts:
  piper:
    binary_path: ` + piperBin + `
    model: ""
    voice_config: ""
`
	if err := os.WriteFile(filepath.Join(appCfgDir, "config.yaml"), []byte(preExisting), 0o644); err != nil {
		t.Fatal(err)
	}

	state := &State{
		AppVersion:   "v0.1.0",
		Language:     "en",
		WhisperModel: "ggml-small.en.bin",
		HotkeyPreset: HotkeyPresetCtrlSpace,
	}
	manifest := DefaultManifest()

	if err := Apply(state, manifest); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(appCfgDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(data, []byte(whisperBin)) {
		t.Errorf("user whisper binary path not preserved:\n%s", data)
	}
	if !contains(data, []byte(piperBin)) {
		t.Errorf("user piper binary path not preserved:\n%s", data)
	}
	// And the model path the wizard chose must now be filled in.
	modelPath, err := whisperModelPathForTest(state.WhisperModel)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(data, []byte(modelPath)) {
		t.Errorf("wizard model path not filled in: %q\n%s", modelPath, data)
	}
}

// TestApply_WritesAudioBlock is the regression test for the
// rc1-hotpatch-13 bug: after the wizard ran, app.New() crashed
// with "audio.sample_rate must be positive" because the wizard's
// generatedConfig struct did not include the audio block. Without
// it, viper unmarshals Audio as the zero struct (sample_rate=0,
// channels=0) and the runtime validator rejects it.
//
// The wizard must write a complete audio block with sane defaults
// (16000 Hz, 1 channel — what ggml-small.en.bin was trained on).
func TestApply_WritesAudioBlock(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	state := &State{
		AppVersion:   "v0.1.0",
		Language:     "en",
		WhisperModel: "ggml-small.en.bin",
		HotkeyPreset: HotkeyPresetCtrlSpace,
	}
	if err := Apply(state, DefaultManifest()); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "voces", "config.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	// Required-by-validator fields
	wantPairs := []string{
		"sample_rate: 16000",
		"channels: 1",
		"chunk_size: 1024",
		"max_duration: 300",
	}
	for _, want := range wantPairs {
		if !contains(data, []byte(want)) {
			t.Errorf("config.yaml missing %q\n---\n%s\n---", want, data)
		}
	}
}

// TestApply_WritesCompleteConfig is the regression test for the
// rc1-hotpatch-14 bug: the wizard's generatedConfig was missing
// the behavior: block AND the four secondary hotkey fields. On
// first run, the user's config.yaml had no autostart, no
// notifications flag, no auto_type flag, no read_clipboard key
// (so the "read clipboard" hotkey feature was silently unbound),
// etc. The runtime Config struct read these as Go zero values
// (autostart=false, notifications=false, ...) which is why logs
// showed "Autostart: desired=false" and "notify: system
// disabled in config" on a fresh install.
//
// The wizard must write a complete behavior block matching
// config.createDefaultConfig's defaults AND the four secondary
// hotkey fields with their runtime defaults (<f10>, <f11>,
// <f12>; stop_recording is intentionally empty — the hold-
// binding model has no separate stop key).
func TestApply_WritesCompleteConfig(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	state := &State{
		AppVersion:   "v0.1.0",
		Language:     "en",
		WhisperModel: "ggml-small.en.bin",
		HotkeyPreset: HotkeyPresetCtrlSpace,
	}
	if err := Apply(state, DefaultManifest()); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "voces", "config.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	// behavior: block (matches config.BehaviorConfig)
	wantBehavior := []string{
		"auto_type: true",
		"type_delay: 15",
		"sound_on_start: false",
		"sound_on_end: false",
		"notifications: true",
		"autostart: false",
		"autostart_delay: 5",
	}
	for _, want := range wantBehavior {
		if !contains(data, []byte(want)) {
			t.Errorf("config.yaml missing behavior %q\n---\n%s\n---", want, data)
		}
	}
	// Four secondary hotkey fields, with the runtime defaults.
	// stop_recording is intentionally empty (the hold-binding
	// model re-uses the record key to stop) but the field must
	// still appear so preserveHotkeys can pick up user changes.
	// Note: YAML encoder emits "<f10>" without quotes (it is a
	// valid unquoted string) and stop_recording as a quoted "".
	wantHotkeys := []string{
		"stop_recording: \"\"",
		"read_clipboard: <f10>",
		"toggle_tts: <f11>",
		"toggle_transcription: <f12>",
	}
	for _, want := range wantHotkeys {
		if !contains(data, []byte(want)) {
			t.Errorf("config.yaml missing hotkey %q\n---\n%s\n---", want, data)
		}
	}
}

// TestApply_PreservesUserChangedSecondaryHotkeys: when a user
// re-runs the wizard but had previously customized one of the
// four secondary hotkey fields, the new defaults (rc1-hotpatch-14)
// must not stomp that value. The pre-existing test
// TestApply_PreservesPreExistingHotkeys only covered the
// "wizard writes nothing" case; after hotpatch-14 the wizard
// writes defaults, so the preserve path must keep overriding.
func TestApply_PreservesUserChangedSecondaryHotkeys(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	appCfgDir := filepath.Join(cfgDir, "voces")
	if err := os.MkdirAll(appCfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-existing config with a user-customized read_clipboard.
	preExisting := `hotkeys:
  record_and_type: '<rightctrl>+<left>'
  read_clipboard: '<f5>'
  toggle_tts: '<f6>'
  toggle_transcription: '<f7>'
  stop_recording: '<esc>'
`
	if err := os.WriteFile(filepath.Join(appCfgDir, "config.yaml"), []byte(preExisting), 0o644); err != nil {
		t.Fatal(err)
	}

	state := &State{
		AppVersion:   "v0.1.0",
		Language:     "en",
		WhisperModel: "ggml-small.en.bin",
		HotkeyPreset: HotkeyPresetCtrlSpace,
	}
	if err := Apply(state, DefaultManifest()); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(appCfgDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// User-customized values must survive.
	mustContain := []string{"<f5>", "<f6>", "<f7>", "<esc>"}
	for _, want := range mustContain {
		if !contains(data, []byte(want)) {
			t.Errorf("user-customized %q was stomped:\n%s", want, data)
		}
	}
	// Wizard's default <f10> must NOT appear — the user picked
	// something different, preserveHotkeys must win.
	if contains(data, []byte("<f10>")) {
		t.Errorf("user-customized read_clipboard was replaced by wizard default <f10>:\n%s", data)
	}
	// record_and_type is wizard-owned and must reflect the new
	// choice (ctrl+space), not the old <rightctrl>+<left>.
	if contains(data, []byte("record_and_type: '<rightctrl>+<left>'")) {
		t.Errorf("record_and_type was preserved; wizard's new choice must win:\n%s", data)
	}
}

// TestApply_HonorsWizardAutostart (rc1-hotpatch-14) verifies
// that when the wizard's State has Autostart=true, the
// generated config.yaml has behavior.autostart: true. The
// pre-existing TestApply_WritesCompleteConfig covers the
// default (false) case; this covers the user-yes path.
func TestApply_HonorsWizardAutostart(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	state := &State{
		AppVersion:   "v0.1.0",
		Language:     "en",
		WhisperModel: "ggml-small.en.bin",
		HotkeyPreset: HotkeyPresetCtrlSpace,
		Autostart:    true,
	}
	if err := Apply(state, DefaultManifest()); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "voces", "config.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(data, []byte("autostart: true")) {
		t.Errorf("config.yaml missing autostart: true after wizard said yes:\n%s", data)
	}
}

// TestApply_HonorsWizardSecondaryHotkey (rc1-hotpatch-14)
// verifies that a user-customized read_clipboard hotkey from
// the wizard's SecondaryHotkeys step is written verbatim. The
// pre-existing TestApply_WritesCompleteConfig covers the
// "user did not customize" case; this covers the user-picked-
// a-different-key path.
func TestApply_HonorsWizardSecondaryHotkey(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	state := &State{
		AppVersion:        "v0.1.0",
		Language:          "en",
		WhisperModel:      "ggml-small.en.bin",
		HotkeyPreset:      HotkeyPresetCtrlSpace,
		ReadClipboardKey:  "ctrl+shift+c",
		ToggleTTSKey:      "ctrl+shift+t",
		ToggleTranscriptionKey: "ctrl+shift+y",
	}
	if err := Apply(state, DefaultManifest()); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "voces", "config.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	mustContain := []string{
		"read_clipboard: ctrl+shift+c",
		"toggle_tts: ctrl+shift+t",
		"toggle_transcription: ctrl+shift+y",
	}
	for _, want := range mustContain {
		if !contains(data, []byte(want)) {
			t.Errorf("config.yaml missing %q:\n%s", want, data)
		}
	}
	if contains(data, []byte("<f10>")) {
		t.Errorf("user-customized read_clipboard was replaced by default <f10>:\n%s", data)
	}
}

// TestApply_HonorsWizardModel (rc1-hotpatch-24): the wizard's
// chosen model (setup.State.WhisperModel) flows into
// config.yaml's transcription.whisper_cpp.model field. After
// Phase 5 the model step is the source of truth; no ADR-0004
// routing applies in defaultConfigFor. Picks a non-default
// model (base.en instead of the small.en default) to prove
// the field actually follows the State.
func TestApply_HonorsWizardModel(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	state := &State{
		AppVersion:   "v0.1.0",
		Language:     "en",
		WhisperModel: "ggml-base.en.bin", // user picked base.en on the picker
		HotkeyPreset: HotkeyPresetCtrlSpace,
	}
	if err := Apply(state, DefaultManifest()); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "voces", "config.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	// config.yaml must reference the chosen model (its basename
	// appears in the canonical model path).
	if !contains(data, []byte("ggml-base.en.bin")) {
		t.Errorf("config.yaml missing chosen model %q:\n%s", "ggml-base.en.bin", data)
	}
	// The default small.en must NOT appear — the user explicitly
	// chose base.en on the picker. This is the regression guard
	// for the rc1-hotpatch-24 contract.
	if contains(data, []byte("ggml-small.en.bin")) {
		t.Errorf("config.yaml still has the small.en default after user picked base.en:\n%s", data)
	}
}

// TestApply_PreservesUserModel (rc1-hotpatch-24): the "user
// wins" rule for the model field. If a user re-runs the wizard
// but had previously picked a non-default model (or hand-edited
// config.yaml to point at a different valid model in the
// manifest), the pre-existing value survives. This is analogous
// to preserveHotkeys / preserveBinaryPath.
func TestApply_PreservesUserModel(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	appCfgDir := filepath.Join(cfgDir, "voces")
	if err := os.MkdirAll(appCfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-existing config with the user's previous pick
	// (ggml-tiny.en.bin — a valid manifest entry that isn't the
	// default). Use the basename; defaultConfigFor writes the
	// canonical path which the preserve helper keeps intact.
	preExisting := `transcription:
  whisper_cpp:
    model: ` + whisperModelPathForTestForPreserve("ggml-tiny.en.bin") + `
`
	if err := os.WriteFile(filepath.Join(appCfgDir, "config.yaml"), []byte(preExisting), 0o644); err != nil {
		t.Fatal(err)
	}

	state := &State{
		AppVersion:   "v0.1.0",
		Language:     "en",
		WhisperModel: "ggml-small.en.bin", // wizard's new default
		HotkeyPreset: HotkeyPresetCtrlSpace,
	}
	if err := Apply(state, DefaultManifest()); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(appCfgDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// User's previous pick must survive.
	if !contains(data, []byte("ggml-tiny.en.bin")) {
		t.Errorf("user's previous model %q was stomped by the wizard's default:\n%s", "ggml-tiny.en.bin", data)
	}
	// The wizard's default must NOT have replaced the user's pick.
	if contains(data, []byte("ggml-small.en.bin")) {
		t.Errorf("preserveModel failed: config.yaml has the wizard's default %q, expected only %q:\n%s",
			"ggml-small.en.bin", "ggml-tiny.en.bin", data)
	}
}

// TestApply_DropsPhantomModel (rc1-hotpatch-24): if a user
// hand-edited config.yaml to point at a model not in the
// manifest (e.g. a typo, a deleted file, a fine-tuned model
// we don't ship), the preserve rule is conservative — it
// only keeps the previous value if it's a known manifest
// entry. Otherwise the wizard's pick wins, dropping the
// phantom so the runtime can start.
func TestApply_DropsPhantomModel(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	appCfgDir := filepath.Join(cfgDir, "voces")
	if err := os.MkdirAll(appCfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-existing config with a phantom model.
	preExisting := `transcription:
  whisper_cpp:
    model: /home/user/.local/share/voces/models/ggml-fake.bin
`
	if err := os.WriteFile(filepath.Join(appCfgDir, "config.yaml"), []byte(preExisting), 0o644); err != nil {
		t.Fatal(err)
	}

	state := &State{
		AppVersion:   "v0.1.0",
		Language:     "en",
		WhisperModel: "ggml-small.en.bin",
		HotkeyPreset: HotkeyPresetCtrlSpace,
	}
	if err := Apply(state, DefaultManifest()); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(appCfgDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// Phantom must be gone.
	if contains(data, []byte("ggml-fake.bin")) {
		t.Errorf("phantom model should be dropped, but it's still in config.yaml:\n%s", data)
	}
	// Wizard's pick must be present.
	if !contains(data, []byte("ggml-small.en.bin")) {
		t.Errorf("wizard's pick %q missing after phantom was dropped:\n%s", "ggml-small.en.bin", data)
	}
}

// whisperModelPathForTestForPreserve mirrors
// whisperModelPathForTest (defined later in this file) but is
// referenced from TestApply_PreservesUserModel above. We keep
// the canonical-path construction here so the pre-existing
// config string matches what defaultConfigFor would have
// written, which is the scenario preserveModel is designed
// to protect.
func whisperModelPathForTestForPreserve(name string) string {
	return filepath.Join(os.Getenv("XDG_DATA_HOME"), "voces", "models", "whisper", name)
}

// contains is a tiny helper kept local to this test file to avoid pulling
// in strings.Contains from the test binary twice. Returns true if needle
// appears anywhere in haystack.
func contains(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if string(haystack[i:i+len(needle)]) == string(needle) {
			return true
		}
	}
	return false
}

// jsonMarshal is a tiny shim so the test compiles even before the real
// manifest marshaller is wired. Kept here so removing it later is a
// one-line delete. Body of Apply will not use this.
var _ = json.Marshal

// whisperModelPathForTest returns the canonical path a downloaded model
// would land at. Lives in the test file so we can import paths without
// wiring internal/paths imports into the production package during
// the red phase.
func whisperModelPathForTest(name string) (string, error) {
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		return "", os.ErrNotExist
	}
	return filepath.Join(data, "voces", "models", "whisper", name), nil
}

// piperVoicePathForTest returns the canonical path a downloaded piper
// .onnx voice would land at. Mirrors paths.PiperVoicePath without
// importing it (so the test compiles during the red phase before
// EnsureModels wires the real import).
func piperVoicePathForTest(base string) (string, error) {
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		return "", os.ErrNotExist
	}
	return filepath.Join(data, "voces", "models", "piper", base+".onnx"), nil
}
