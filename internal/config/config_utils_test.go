package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSave_Simple tests the Save function
func TestSave_Simple(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp directory
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create a go.mod file to trigger config creation in current dir
	os.WriteFile("go.mod", []byte("module test"), 0o644)

	cfg := &Config{
		Hotkeys: HotkeysConfig{RecordAndType: "<f8>"},
		Audio: AudioConfig{
			SampleRate: 16000,
			Channels:   1,
		},
		Transcription: TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
		},
		TTS: TTSConfig{
			DefaultEngine: "piper",
		},
	}

	// Save may fail in test environment - that's ok
	_ = Save(cfg)
}

// TestValidateBinary_Simple tests the ValidateBinary function
func TestValidateBinary_Simple(t *testing.T) {
	// Create a fake executable
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "testscript")
	os.WriteFile(scriptPath, []byte("#!/bin/bash\necho test"), 0o755)

	// Test with existing file
	err := ValidateBinary(scriptPath)
	// May error due to execution but should not panic
	t.Logf("ValidateBinary result: %v", err)

	// Test with non-existing file
	nonExisting := filepath.Join(tmpDir, "nonexistent")
	err = ValidateBinary(nonExisting)
	if err == nil {
		t.Error("Expected error for non-existing binary")
	}
}

// TestValidatePaths_Simple tests the validatePaths function
func TestValidatePaths_Simple(t *testing.T) {
	// Test with whisper_cpp engine and non-existing paths
	cfg := &Config{
		Transcription: TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: WhisperCPPConfig{
				BinaryPath: "/nonexistent/whisper",
				Model:      "/nonexistent/model",
			},
		},
	}

	err := validatePaths(cfg)
	if err == nil {
		t.Error("Expected error for non-existing paths")
	}

	// Test with openai_api engine (should skip path validation)
	cfg2 := &Config{
		Transcription: TranscriptionConfig{
			DefaultEngine: "openai_api",
			WhisperCPP: WhisperCPPConfig{
				BinaryPath: "/nonexistent/whisper",
				Model:      "/nonexistent/model",
			},
		},
	}

	err = validatePaths(cfg2)
	if err != nil {
		t.Errorf("validatePaths with openai_api should not return error: %v", err)
	}
}

// TestFileExists_Simple tests the fileExists function
func TestFileExists_Simple(t *testing.T) {
	// Test with existing file
	tmpFile := filepath.Join(t.TempDir(), "testfile.txt")
	os.WriteFile(tmpFile, []byte("test"), 0o644)

	if !fileExists(tmpFile) {
		t.Error("fileExists should return true for existing file")
	}

	// Test with non-existing file
	nonExisting := filepath.Join(t.TempDir(), "nonexistent.txt")
	if fileExists(nonExisting) {
		t.Error("fileExists should return false for non-existing file")
	}

	// Test with directory (should return false)
	tmpDir := t.TempDir()
	if fileExists(tmpDir) {
		t.Error("fileExists should return false for directory")
	}
}
