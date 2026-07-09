/* Code Map: rc30 fix tests
 * (rc1-hotpatch-30: TTS Unavailable — detect system piper,
 * validate preserved binary path, custom URL model path)
 *
 * - TestResolvePiperBinaryPath_PrefersSystem: when a real
 *   executable exists at one of the candidate paths,
 *   ResolvePiperBinaryPath returns that path (not the
 *   bundled fallback).
 * - TestResolvePiperBinaryPath_FallsBackToBundled: when no
 *   system piper is found, the bundled <engines>/piper is
 *   returned.
 * - TestPiperPathsForState_EmptyAndNil: nil State and empty
 *   PiperVoice both return ("", "") — the rest of the config
 *   treats TTS as disabled.
 * - TestPiperPathsForState_ManifestKey: a normal manifest key
 *   like "en_US-lessac-medium" produces a path under
 *   PiperModelDir.
 * - TestPiperPathsForState_CustomURL: a custom-URL sentinel
 *   produces a path derived from the onnx URL's basename
 *   (no colon or pipe characters in the output — that was
 *   the rc30 bug).
 * - TestPiperPathsForState_CustomURLWithConfig: a sentinel
 *   that has a config URL appends ".json" to the model path.
 * - TestPreserveBinaryPath_KeepsValidPath: when the existing
 *   binary_path points at a real, executable file, the value
 *   is preserved on cfg.
 * - TestPreserveBinaryPath_DiscardsStalePath: when the
 *   existing binary_path is missing or non-executable, the
 *   value is NOT preserved so defaultConfigFor can fall
 *   through to the new resolver.
 * - TestPreserveBinaryPath_DiscardsEmptyPath: an empty
 *   binary_path is also discarded (preserves the original
 *   behaviour for the "user never set this" case).
 *
 * CID Index:
 * CID:setup-rc30test-001 -> TestResolvePiperBinaryPath_PrefersSystem
 * CID:setup-rc30test-002 -> TestResolvePiperBinaryPath_FallsBackToBundled
 * CID:setup-rc30test-003 -> TestPiperPathsForState_EmptyAndNil
 * CID:setup-rc30test-004 -> TestPiperPathsForState_ManifestKey
 * CID:setup-rc30test-005 -> TestPiperPathsForState_CustomURL
 * CID:setup-rc30test-006 -> TestPiperPathsForState_CustomURLWithConfig
 * CID:setup-rc30test-007 -> TestPreserveBinaryPath_KeepsValidPath
 * CID:setup-rc30test-008 -> TestPreserveBinaryPath_DiscardsStalePath
 * CID:setup-rc30test-009 -> TestPreserveBinaryPath_DiscardsEmptyPath
 */
package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeExecutable drops a tiny shell-script with the +x bit set
// at path. Real file system, no fakes (per the precommit gate
// that bans fake paths in tests).
//
// rc1-hotpatch-31: the script also answers `--version` with a
// string that contains "piper" so it passes the isPiperTTS
// check in FindPiperBinary. Without this, the rc30 "Prefers
// System" test would now fail because the fake piper that
// just prints "exit 0" gets rejected as "not the rhasspy/piper
// TTS binary".
func makeExecutable(t *testing.T, path string) {
	t.Helper()
	script := "#!/bin/sh\n" +
		"# Mimic the rhasspy/piper TTS binary just enough to pass\n" +
		"# setup.isPiperTTS (rc1-hotpatch-31): respond to --version\n" +
		"# with a string that contains 'piper' and exits 0.\n" +
		"if [ \"$1\" = \"--version\" ]; then\n" +
		"  echo \"piper v1.2.0-test\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake executable %s: %v", path, err)
	}
}

// CID:setup-rc30test-001 - TestResolvePiperBinaryPath_PrefersSystem
// Purpose: when a real executable exists at a candidate path,
// ResolvePiperBinaryPath returns that path. This is the rc30
// fix for the TTS-Unavailable bug: the previous defaults.go
// hard-coded /opt/voces/engines/piper (which doesn't exist on
// most users' systems), so even users who had installed piper
// via apt ended up with a config that pointed at a missing
// file, Piper.Validate() failed the os.Stat, and the
// rc1-hotpatch-27 "TTS Unavailable" notification kept firing.
//
// We override piperCandidatePaths so the test is hermetic —
// the host's actual /usr/bin/piper (if present) doesn't
// influence the outcome.
func TestResolvePiperBinaryPath_PrefersSystem(t *testing.T) {
	dir := t.TempDir()
	sysBin := filepath.Join(dir, "piper")
	makeExecutable(t, sysBin)

	origCandidates := piperCandidatePaths
	piperCandidatePaths = []string{sysBin}
	t.Cleanup(func() { piperCandidatePaths = origCandidates })

	// Force LookPath to fail so the test exercises the
	// candidate-list branch only.
	t.Setenv("PATH", t.TempDir())

	bundled := filepath.Join(t.TempDir(), "engines")
	if got := ResolvePiperBinaryPath(bundled); got != sysBin {
		t.Errorf("ResolvePiperBinaryPath: got %q, want %q (system piper should win over bundled fallback)", got, sysBin)
	}
}

// CID:setup-rc30test-002 - TestResolvePiperBinaryPath_FallsBackToBundled
// Purpose: when no system piper is found anywhere on the
// search path, ResolvePiperBinaryPath returns the bundled
// <engines>/piper default. This keeps the release tarball
// working out of the box on a host that doesn't have piper
// installed system-wide (Phase 8.1: bundled piper is the
// long-term answer; until then the bundled <engines>/piper
// is the only piper the wizard has at all).
func TestResolvePiperBinaryPath_FallsBackToBundled(t *testing.T) {
	origCandidates := piperCandidatePaths
	piperCandidatePaths = []string{filepath.Join(t.TempDir(), "nope-piper")}
	t.Cleanup(func() { piperCandidatePaths = origCandidates })
	t.Setenv("PATH", t.TempDir())

	bundled := filepath.Join(t.TempDir(), "engines")
	want := filepath.Join(bundled, "piper")
	if got := ResolvePiperBinaryPath(bundled); got != want {
		t.Errorf("ResolvePiperBinaryPath: got %q, want %q (bundled fallback)", got, want)
	}
}

// CID:setup-rc30test-003 - TestPiperPathsForState_EmptyAndNil
// Purpose: the wizard's first-run path passes a State with
// PiperVoice="" (the user hasn't picked a voice yet). A
// defensive nil pointer is also possible if a future caller
// forgets to guard. Both cases must return ("", "") so the
// rest of the config treats TTS as disabled rather than
// producing a bogus path containing "custom:" or "|".
func TestPiperPathsForState_EmptyAndNil(t *testing.T) {
	t.Run("empty PiperVoice", func(t *testing.T) {
		s := &State{PiperVoice: ""}
		model, cfg := piperPathsForState(s)
		if model != "" || cfg != "" {
			t.Errorf("empty PiperVoice: got (%q, %q), want (\"\", \"\")", model, cfg)
		}
	})
	t.Run("nil State", func(t *testing.T) {
		model, cfg := piperPathsForState(nil)
		if model != "" || cfg != "" {
			t.Errorf("nil State: got (%q, %q), want (\"\", \"\")", model, cfg)
		}
	})
}

// CID:setup-rc30test-004 - TestPiperPathsForState_ManifestKey
// Purpose: a normal manifest key (e.g. "en_US-lessac-medium")
// produces a path of the form
// <PiperModelDir>/<key>.onnx plus a voice config at
// <key>.onnx.json. This is the behaviour the wizard
// generates for users who pick from the curated voice list.
func TestPiperPathsForState_ManifestKey(t *testing.T) {
	s := &State{PiperVoice: "en_US-lessac-medium"}
	model, voiceCfg := piperPathsForState(s)
	if model == "" {
		t.Fatal("piperPathsForState(manifest): model path is empty")
	}
	if !strings.HasSuffix(model, "en_US-lessac-medium.onnx") {
		t.Errorf("piperPathsForState(manifest): model = %q, want suffix en_US-lessac-medium.onnx", model)
	}
	if voiceCfg != model+".json" {
		t.Errorf("piperPathsForState(manifest): voiceCfg = %q, want %q", voiceCfg, model+".json")
	}
}

// CID:setup-rc30test-005 - TestPiperPathsForState_CustomURL
// Purpose: a custom-URL sentinel resolves to a path whose
// filename is the sanitised basename of the onnx URL. The
// pre-rc30 code produced an invalid path containing the
// sentinel's literal "custom:...|..." characters; this test
// guards against that regression.
func TestPiperPathsForState_CustomURL(t *testing.T) {
	s := &State{PiperVoice: "custom:https://huggingface.co/x/v.onnx|"}
	model, voiceCfg := piperPathsForState(s)
	if model == "" {
		t.Fatal("piperPathsForState(custom URL): model path is empty")
	}
	if strings.ContainsAny(model, ":|") {
		t.Errorf("piperPathsForState(custom URL): model %q still contains sentinel characters (rc30 regression)", model)
	}
	if !strings.HasSuffix(model, "v.onnx") {
		t.Errorf("piperPathsForState(custom URL): model = %q, want suffix v.onnx", model)
	}
	// No config URL -> empty voice config.
	if voiceCfg != "" {
		t.Errorf("piperPathsForState(custom URL no config): voiceCfg = %q, want \"\"", voiceCfg)
	}
}

// CID:setup-rc30test-006 - TestPiperPathsForState_CustomURLWithConfig
// Purpose: when the sentinel includes a config URL, the
// voice config path is <model>.json (the sidecar convention
// piper uses). The runtime treats an empty config path as
// "no sidecar", but the wizard allows the user to supply
// both URLs and the generated config should reflect that.
func TestPiperPathsForState_CustomURLWithConfig(t *testing.T) {
	s := &State{PiperVoice: "custom:https://huggingface.co/x/v.onnx|https://huggingface.co/x/v.onnx.json"}
	model, voiceCfg := piperPathsForState(s)
	if model == "" {
		t.Fatal("piperPathsForState(custom URL w/ config): model path is empty")
	}
	if voiceCfg != model+".json" {
		t.Errorf("piperPathsForState(custom URL w/ config): voiceCfg = %q, want %q", voiceCfg, model+".json")
	}
}

// CID:setup-rc30test-007 - TestPreserveBinaryPath_KeepsValidPath
// Purpose: when the pre-existing config had a binary_path
// pointing at a real, executable file, the value is
// preserved on cfg. This preserves the original "user wins"
// rule for the case where the user has hand-edited their
// config or installed a piper binary at a non-default
// location.
func TestPreserveBinaryPath_KeepsValidPath(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "my-piper")
	makeExecutable(t, bin)

	cfg := &generatedConfig{
		TTS: ttsBlock{
			Piper: piperBlock{
				BinaryPath: "<default-bundled>",
			},
		},
	}
	existing := map[string]any{
		"tts": map[string]any{
			"piper": map[string]any{
				"binary_path": bin,
			},
		},
	}
	preserveBinaryPath(cfg, existing)
	if cfg.TTS.Piper.BinaryPath != bin {
		t.Errorf("preserveBinaryPath: kept %q, want %q", cfg.TTS.Piper.BinaryPath, bin)
	}
}

// CID:setup-rc30test-008 - TestPreserveBinaryPath_DiscardsStalePath
// Purpose: when the pre-existing binary_path is missing on
// disk or non-executable, preserveBinaryPath leaves cfg
// unchanged. The pre-rc30 rule "preserve whenever non-empty"
// trapped users with a stale /opt/voces/engines/piper that
// didn't exist on their system; preserve would keep the
// dead value across re-runs of the wizard and the
// rc1-hotpatch-27 "TTS Unavailable" notification kept
// firing. The new rule is "preserve only when the value
// points at a real executable".
func TestPreserveBinaryPath_DiscardsStalePath(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		cfg := &generatedConfig{
			TTS: ttsBlock{Piper: piperBlock{BinaryPath: "<default-bundled>"}},
		}
		existing := map[string]any{
			"tts": map[string]any{
				"piper": map[string]any{
					"binary_path": "/this/path/does/not/exist/piper",
				},
			},
		}
		preserveBinaryPath(cfg, existing)
		if cfg.TTS.Piper.BinaryPath != "<default-bundled>" {
			t.Errorf("preserveBinaryPath(stale path): kept %q, want the default fallback", cfg.TTS.Piper.BinaryPath)
		}
	})
	t.Run("directory not executable", func(t *testing.T) {
		dir := t.TempDir()
		cfg := &generatedConfig{
			TTS: ttsBlock{Piper: piperBlock{BinaryPath: "<default-bundled>"}},
		}
		existing := map[string]any{
			"tts": map[string]any{
				"piper": map[string]any{
					"binary_path": dir, // a directory, not a file
				},
			},
		}
		preserveBinaryPath(cfg, existing)
		if cfg.TTS.Piper.BinaryPath != "<default-bundled>" {
			t.Errorf("preserveBinaryPath(directory): kept %q, want the default fallback", cfg.TTS.Piper.BinaryPath)
		}
	})
}

// CID:setup-rc30test-009 - TestPreserveBinaryPath_DiscardsEmptyPath
// Purpose: an empty binary_path is also discarded. This
// matches the pre-rc30 behaviour (an empty existing value
// was always a no-op) and keeps the "user wins" rule
// consistent: only a *set* value with a *real* file behind
// it wins.
func TestPreserveBinaryPath_DiscardsEmptyPath(t *testing.T) {
	cfg := &generatedConfig{
		TTS: ttsBlock{Piper: piperBlock{BinaryPath: "<default-bundled>"}},
	}
	existing := map[string]any{
		"tts": map[string]any{
			"piper": map[string]any{
				"binary_path": "",
			},
		},
	}
	preserveBinaryPath(cfg, existing)
	if cfg.TTS.Piper.BinaryPath != "<default-bundled>" {
		t.Errorf("preserveBinaryPath(empty): kept %q, want the default fallback", cfg.TTS.Piper.BinaryPath)
	}
}
