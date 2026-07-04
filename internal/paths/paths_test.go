package paths

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDataDir_XDGOverride verifies DataDir honors $XDG_DATA_HOME.
func TestDataDir_XDGOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	got, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	want := filepath.Join(tmp, appDataDirName)
	if got != want {
		t.Errorf("DataDir() = %q, want %q", got, want)
	}
	if info, err := os.Stat(got); err != nil || !info.IsDir() {
		t.Errorf("DataDir should exist as a directory, got stat err=%v", err)
	}
}

// TestDataDir_FallsBackToHome verifies the $HOME/.local/share fallback.
func TestDataDir_FallsBackToHome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", "") // force fallback
	t.Setenv("HOME", tmp)

	got, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	want := filepath.Join(tmp, ".local", "share", appDataDirName)
	if got != want {
		t.Errorf("DataDir() = %q, want %q", got, want)
	}
}

// TestModelsDir_IsUnderDataDir verifies models live under the data dir.
func TestModelsDir_IsUnderDataDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	models, err := ModelsDir()
	if err != nil {
		t.Fatalf("ModelsDir: %v", err)
	}
	data, _ := DataDir()
	if filepath.Dir(models) != data {
		t.Errorf("ModelsDir (%q) should be directly under DataDir (%q)", models, data)
	}
}

// TestWhisperModelPath_Prefix verifies the canonical path layout.
func TestWhisperModelPath_Prefix(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	got, err := WhisperModelPath("ggml-small.en.bin")
	if err != nil {
		t.Fatalf("WhisperModelPath: %v", err)
	}
	models, _ := ModelsDir()
	want := filepath.Join(models, "whisper", "ggml-small.en.bin")
	if got != want {
		t.Errorf("WhisperModelPath = %q, want %q", got, want)
	}
}

// TestPiperVoicePath_AppendsOnnx verifies .onnx is appended to the base name.
func TestPiperVoicePath_AppendsOnnx(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	got, err := PiperVoicePath("en_US-lessac-medium")
	if err != nil {
		t.Fatalf("PiperVoicePath: %v", err)
	}
	models, _ := ModelsDir()
	want := filepath.Join(models, "piper", "en_US-lessac-medium.onnx")
	if got != want {
		t.Errorf("PiperVoicePath = %q, want %q", got, want)
	}
}

// TestEnginesDir_EnvOverride verifies $WVU_ENGINES_DIR wins when set.
func TestEnginesDir_EnvOverride(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(enginesEnvVar, tmp)
	got, err := EnginesDir()
	if err != nil {
		t.Fatalf("EnginesDir: %v", err)
	}
	if got != tmp {
		t.Errorf("EnginesDir = %q, want %q", got, tmp)
	}
}

// TestEnginesDir_BinSiblingLayout verifies <root>/bin/whisper-voice-util
// resolves to <root>/engines.
func TestEnginesDir_BinSiblingLayout(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	engines := filepath.Join(root, enginesSubdir)
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(engines, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(enginesEnvVar, "") // disable override

	// Simulate the binary living at <root>/bin/whisper-voice-util by
	// passing that fake path to the testable helper.
	fakeBin := filepath.Join(bin, "whisper-voice-util")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := enginesDirFrom(fakeBin)
	if !ok {
		t.Fatalf("enginesDirFrom(%q) returned miss", fakeBin)
	}
	if got != engines {
		t.Errorf("enginesDirFrom = %q, want %q", got, engines)
	}
}

// TestEnginesDir_MissingReturnsError verifies a clear error when nothing resolves.
func TestEnginesDir_MissingReturnsError(t *testing.T) {
	t.Setenv(enginesEnvVar, "/nonexistent/path/that/does/not/exist")
	if _, err := EnginesDir(); err == nil {
		t.Error("EnginesDir should fail when no candidate exists")
	}
}
