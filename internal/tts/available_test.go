/* Code Map: TTS.Available() tests
 * - TestAvailable_PiperMissingBinary: the rc1-hotpatch-27 case —
 *   the release tarball doesn't bundle piper, so the user
 *   installs v0.2.0-rc12+ and hits Ctrl+U. Available() must
 *   return false so the read_clipboard handler can show a
 *   friendly info notification instead of the raw "piper
 *   binary not found" error.
 * - TestAvailable_PiperMissingModel: same shape, but the
 *   binary exists and the model doesn't. Still unavailable.
 * - TestAvailable_PiperAllPresent: end-to-end happy path —
 *   binary + model + voice config all exist; Available()
 *   returns true.
 *
 * The Available() method is the only thing standing between
 * a happy user (no error popup) and a confused user (raw
 * piper error in a desktop notification). Pure stdlib, no
 * GTK, no config files.
 *
 * CID Index:
 * CID:tts-manager-test-001 -> TestAvailable_PiperMissingBinary
 * CID:tts-manager-test-002 -> TestAvailable_PiperMissingModel
 * CID:tts-manager-test-003 -> TestAvailable_PiperAllPresent
 */
package tts

import (
	"os"
	"path/filepath"
	"testing"

	"voces/internal/config"
)

// CID:tts-manager-test-001 - TestAvailable_PiperMissingBinary
// Purpose: rc1-hotpatch-27. The release tarball doesn't bundle
// piper (Makefile piper-build is a no-op because piper needs
// ONNX runtime + espeak-ng to build). A fresh install
// has no /opt/voces/engines/piper. When the user hits Ctrl+U
// (read_clipboard), the hotkey handler asks Available() before
// invoking Piper.Speak. With the binary missing, Available()
// MUST return false so the handler can show a friendly
// "TTS Unavailable" notification instead of the raw
// "piper binary not found" error.
//
// Note: we point BinaryPath at a real but non-existent file
// inside t.TempDir(); the test exercises the os.Stat failure
// path the same way a real /opt/voces/engines/piper with no
// piper file would.
func TestAvailable_PiperMissingBinary(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "piper",
			Piper: config.PiperConfig{
				BinaryPath:  filepath.Join(tmpDir, "piper-missing"),
				Model:       "/nonexistent/model.onnx",
				VoiceConfig: "/nonexistent/config.json",
			},
		},
	}
	mgr := New(cfg)
	if mgr.Available() {
		t.Errorf("Available() = true with no piper binary; want false (rc27 contract)")
	}
}

// CID:tts-manager-test-002 - TestAvailable_PiperMissingModel
// Purpose: pin the contract for the "binary exists but model
// is missing" case. Some users might have piper installed
// system-wide (e.g. via the install-deps.sh apt fallback) but
// the model download step never ran (e.g. a hand-rolled
// config). Available() must still return false — the engine
// can be invoked, but Speak() would fail.
func TestAvailable_PiperMissingModel(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "piper")
	// Write a small file at the binary path so the binary
	// stat succeeds. We do not mark it executable: Piper's
	// Validate() only does os.Stat, and we want this test
	// to exercise the "binary found, model missing" branch.
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\nexit 0\n"), 0o644); err != nil {
		t.Fatalf("piper file write: %v", err)
	}
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "piper",
			Piper: config.PiperConfig{
				BinaryPath:  binPath,
				Model:       "/nonexistent/model.onnx",
				VoiceConfig: "/nonexistent/config.json",
			},
		},
	}
	mgr := New(cfg)
	if mgr.Available() {
		t.Errorf("Available() = true with no piper model; want false (rc27 contract)")
	}
}

// CID:tts-manager-test-003 - TestAvailable_PiperAllPresent
// Purpose: the happy path. Write a real file at each of the
// three piper paths (binary, model, voice config); Available()
// must return true. This pins the contract that the three
// os.Stat calls inside Piper.Validate() are the only gate,
// so a future "let's also check the binary is executable"
// tweak does not accidentally regress this. (We do not
// assert the binary is executable here — Piper.Speak exec's
// it via exec.CommandContext, which will surface a real
// error if the file is +x. The manager's job is the
// stat-based availability check, not permission validation.)
func TestAvailable_PiperAllPresent(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "piper")
	modelPath := filepath.Join(tmpDir, "model.onnx")
	cfgPath := filepath.Join(tmpDir, "model.onnx.json")
	for _, p := range []string{binPath, modelPath, cfgPath} {
		if err := os.WriteFile(p, []byte("sample"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "piper",
			Piper: config.PiperConfig{
				BinaryPath:  binPath,
				Model:       modelPath,
				VoiceConfig: cfgPath,
			},
		},
	}
	mgr := New(cfg)
	if !mgr.Available() {
		t.Errorf("Available() = false with all three piper files present; want true (rc27 contract)")
	}
}
