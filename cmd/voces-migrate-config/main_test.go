/* Code Map: tests for cmd/voces-migrate-config
 * - TestMigrate_AddsMissingFields: pre-rc1-hotpatch-14
 *   config gets the behavior: block and the four secondary
 *   hotkey fields on disk; user values are preserved.
 * - TestMigrate_Idempotent: re-running the migrator on an
 *   already-patched config is a no-op (no diff, no rewrite).
 * - TestMigrate_DryRun: dry-run returns a "would update"
 *   status and does not write the file.
 * - TestMigrate_BadPath: missing config path returns an
 *   error rather than panicking.
 * - TestParseArgs: flag parsing covers the supported
 *   short / long / kebab-case forms and unknown-flag
 *   rejection.
 *
 * CID Index:
 * CID:voces-migrate-test-001 -> TestMigrate_AddsMissingFields
 * CID:voces-migrate-test-002 -> TestMigrate_Idempotent
 * CID:voces-migrate-test-003 -> TestMigrate_DryRun
 * CID:voces-migrate-test-004 -> TestMigrate_BadPath
 * CID:voces-migrate-test-005 -> TestParseArgs
 *
 * Quick lookup: rg -n "CID:voces-migrate-test-" cmd/voces-migrate-config/
 */
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CID:voces-migrate-test-001 - TestMigrate_AddsMissingFields
// Purpose: end-to-end migrator contract on a temp config.
// Seeds a pre-rc1-hotpatch-14 file, calls migrate(),
// and asserts every defaulted field landed on disk and
// every user value survived untouched.
func TestMigrate_AddsMissingFields(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte(preHotpatchConfig), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	status, err := migrate(cfg, false)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !strings.Contains(status, "migrated") {
		t.Errorf("status: want to mention migration, got %q", status)
	}

	got, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read after migrate: %v", err)
	}

	// All defaulted fields must now be present.
	for _, want := range []string{
		"behavior:",
		"auto_type: true",
		"type_delay: 15",
		"notifications: true",
		"autostart: false",
		"autostart_delay: 5",
		"read_clipboard: <f10>",
		"toggle_tts: <f11>",
		"toggle_transcription: <f12>",
	} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("migrated config missing %q\n--- got ---\n%s", want, got)
		}
	}

	// User values must survive untouched.
	for _, want := range []string{
		"record_and_type: f9",
		"binary_path: /opt/whisper.cpp/main",
		"model: /opt/whisper.cpp/models/ggml-small.bin",
		"language: en",
		"compute_type: float",
	} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("migrated config lost user value %q\n--- got ---\n%s", want, got)
		}
	}
}

// CID:voces-migrate-test-002 - TestMigrate_Idempotent
// Purpose: re-running the migrator on an already-patched
// config is a no-op. The status string reflects this and
// the file content is byte-equal to the first run.
func TestMigrate_Idempotent(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte(preHotpatchConfig), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// First run.
	if _, err := migrate(cfg, false); err != nil {
		t.Fatalf("migrate first: %v", err)
	}
	first, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read first: %v", err)
	}

	// Second run: must be a no-op.
	status, err := migrate(cfg, false)
	if err != nil {
		t.Fatalf("migrate second: %v", err)
	}
	if !strings.Contains(status, "already has") {
		t.Errorf("second-run status: want 'already has', got %q", status)
	}
	second, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read second: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("idempotency violated: file changed between runs\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

// CID:voces-migrate-test-003 - TestMigrate_DryRun
// Purpose: dry-run reports what would happen and does
// not write the file. The pre-rc1-hotpatch-14 seed
// must be byte-equal after a dry-run call.
func TestMigrate_DryRun(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte(preHotpatchConfig), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	before, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}

	status, err := migrate(cfg, true)
	if err != nil {
		t.Fatalf("migrate dry: %v", err)
	}
	if !strings.Contains(status, "would update") {
		t.Errorf("dry-run status: want 'would update', got %q", status)
	}

	after, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("dry-run must not write; before != after\n--- before ---\n%s\n--- after ---\n%s", before, after)
	}
}

// CID:voces-migrate-test-004 - TestMigrate_BadPath
// Purpose: missing config path returns an error rather
// than panicking, so the migrator can be safely scripted.
func TestMigrate_BadPath(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist.yaml")
	_, err := migrate(missing, false)
	if err == nil {
		t.Errorf("migrate on missing file: want error, got nil")
	}
	if !strings.Contains(err.Error(), "read") {
		t.Errorf("migrate on missing file: error should mention 'read', got %v", err)
	}
}

// CID:voces-migrate-test-005 - TestParseArgs
// Purpose: flag parsing covers the documented forms and
// rejects unknown flags with exit code 2.
func TestParseArgs(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantOpts  cliOptions
		wantErr   bool
	}{
		{name: "empty", args: nil, wantOpts: cliOptions{}},
		{name: "dry short", args: []string{"-dry"}, wantOpts: cliOptions{dry: true}},
		{name: "dry long", args: []string{"--dry"}, wantOpts: cliOptions{dry: true}},
		{name: "dry kebab", args: []string{"--dry-run"}, wantOpts: cliOptions{dry: true}},
		{name: "path short", args: []string{"-path=/x/y"}, wantOpts: cliOptions{pathOverride: "/x/y"}},
		{name: "path long", args: []string{"--config=/x/y"}, wantOpts: cliOptions{pathOverride: "/x/y"}},
		{name: "help short", args: []string{"-h"}, wantOpts: cliOptions{showUsage: true}},
		{name: "help long", args: []string{"--help"}, wantOpts: cliOptions{showUsage: true}},
		{name: "version", args: []string{"-v"}, wantOpts: cliOptions{showVersion: true}},
		{name: "all flags", args: []string{"-dry", "-path=/x", "--help"}, wantOpts: cliOptions{dry: true, pathOverride: "/x", showUsage: true}},
		{name: "unknown", args: []string{"-wat"}, wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts, code, err := parseArgs(c.args)
			if c.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (code=%d, opts=%+v)", code, opts)
				}
				if code != 2 {
					t.Errorf("expected exit code 2 on usage error, got %d", code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if opts != c.wantOpts {
				t.Errorf("opts: want %+v, got %+v", c.wantOpts, opts)
			}
		})
	}
}

// preHotpatchConfig is the canonical pre-rc1-hotpatch-14
// wizard output: no behavior: block, only record_and_type
// in hotkeys:. Shared between the smoke test and the
// marshal_test.go migration test.
const preHotpatchConfig = `
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
