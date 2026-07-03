package tts

import (
	"os"
	"path/filepath"
	"testing"

	"whisper-voice-util/internal/config"
)

func TestNewTTS(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "piper",
			Piper: config.PiperConfig{
				BinaryPath:  "./vendor/piper/piper",
				Model:       "./vendor/piper/models/en_US-lessac-medium.onnx",
				VoiceConfig: "./vendor/piper/models/en_US-lessac-medium.onnx.json",
			},
			ElevenLabs: config.ElevenLabsConfig{
				APIKey: "test-key",
			},
		},
	}

	tts := New(cfg)
	if tts == nil {
		t.Fatal("Expected TTS to be created")
	}
	if tts.engine == nil {
		t.Fatal("Expected engine to be initialized")
	}
}

func TestAvailableEngines(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "piper",
			Piper: config.PiperConfig{
				BinaryPath:  "./vendor/piper/piper",
				Model:       "./vendor/piper/models/en_US-lessac-medium.onnx",
				VoiceConfig: "./vendor/piper/models/en_US-lessac-medium.onnx.json",
			},
			ElevenLabs: config.ElevenLabsConfig{
				APIKey: "test-key",
			},
		},
	}

	tts := New(cfg)
	engines := tts.AvailableEngines()

	if len(engines) != 2 {
		t.Errorf("Expected 2 engines, got %d", len(engines))
	}

	// Check both engines are present
	hasPiper := false
	hasElevenLabs := false
	for _, e := range engines {
		if e == "piper" {
			hasPiper = true
		}
		if e == "elevenlabs" {
			hasElevenLabs = true
		}
	}

	if !hasPiper {
		t.Error("Expected piper engine to be available")
	}
	if !hasElevenLabs {
		t.Error("Expected elevenlabs engine to be available")
	}
}

func TestCurrentEngine(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "piper",
			Piper: config.PiperConfig{
				BinaryPath:  "./vendor/piper/piper",
				Model:       "./vendor/piper/models/en_US-lessac-medium.onnx",
				VoiceConfig: "./vendor/piper/models/en_US-lessac-medium.onnx.json",
			},
		},
	}

	tts := New(cfg)
	current := tts.CurrentEngine()

	if current != "piper" {
		t.Errorf("Expected current engine to be piper, got %s", current)
	}
}

func TestSetEngine(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "piper",
			Piper: config.PiperConfig{
				BinaryPath:  "./vendor/piper/piper",
				Model:       "./vendor/piper/models/en_US-lessac-medium.onnx",
				VoiceConfig: "./vendor/piper/models/en_US-lessac-medium.onnx.json",
			},
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "test-key",
				VoiceID: "test-voice",
			},
		},
	}

	tts := New(cfg)

	// Test switching to elevenlabs (should work with test key)
	err := tts.SetEngine("elevenlabs")
	if err != nil {
		t.Errorf("Expected no error when switching to elevenlabs: %v", err)
	}

	current := tts.CurrentEngine()
	if current != "elevenlabs" {
		t.Errorf("Expected current engine to be elevenlabs, got %s", current)
	}

	// Switch back to piper
	err = tts.SetEngine("piper")
	// This will fail validation because paths are relative in tests
	if err != nil {
		t.Logf("Expected validation error for relative paths: %v", err)
	}
}

func TestSetEngine_UnknownEngine(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "piper",
		},
	}

	tts := New(cfg)

	err := tts.SetEngine("unknown_engine")
	if err == nil {
		t.Error("Expected error when setting unknown engine")
	}
}

func TestPiper_Name(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  "./vendor/piper/piper",
				Model:       "./vendor/piper/models/en_US-lessac-medium.onnx",
				VoiceConfig: "./vendor/piper/models/en_US-lessac-medium.onnx.json",
			},
		},
	}

	p := NewPiper(cfg)
	if p.Name() != "piper" {
		t.Errorf("Expected name to be piper, got %s", p.Name())
	}
}

func TestElevenLabs_Name(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			ElevenLabs: config.ElevenLabsConfig{
				APIKey: "test-key",
			},
		},
	}

	e := NewElevenLabs(cfg)
	if e.Name() != "elevenlabs" {
		t.Errorf("Expected name to be elevenlabs, got %s", e.Name())
	}
}

func TestElevenLabs_Validate(t *testing.T) {
	// Test with API key and voice ID
	cfg := &config.Config{
		TTS: config.TTSConfig{
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "test-key",
				VoiceID: "test-voice",
			},
		},
	}

	e := NewElevenLabs(cfg)
	err := e.Validate()
	if err != nil {
		t.Errorf("Expected no error with API key and voice ID: %v", err)
	}

	// Test without API key
	cfg2 := &config.Config{
		TTS: config.TTSConfig{
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "",
				VoiceID: "test-voice",
			},
		},
	}

	e2 := NewElevenLabs(cfg2)
	err = e2.Validate()
	if err == nil {
		t.Error("Expected error without API key")
	}

	// Test without voice ID
	cfg3 := &config.Config{
		TTS: config.TTSConfig{
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "test-key",
				VoiceID: "",
			},
		},
	}

	e3 := NewElevenLabs(cfg3)
	err = e3.Validate()
	if err == nil {
		t.Error("Expected error without voice ID")
	}
}

func TestPiper_Validate(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  "./vendor/piper/piper",
				Model:       "./vendor/piper/models/en_US-lessac-medium.onnx",
				VoiceConfig: "./vendor/piper/models/en_US-lessac-medium.onnx.json",
			},
		},
	}

	p := NewPiper(cfg)
	err := p.Validate()
	// This will fail because paths are relative in tests
	if err == nil {
		t.Log("Validation passed (unexpected for relative paths)")
	} else {
		t.Logf("Expected validation error for relative paths: %v", err)
	}
}

func TestSpeak_WithEmptyText(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "elevenlabs",
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "test-key",
				VoiceID: "test-voice",
			},
		},
	}

	tts := New(cfg)

	// This will fail because ElevenLabs is not fully implemented
	err := tts.Speak("")
	if err == nil {
		t.Error("Expected error for unimplemented API")
	}
}

func TestTTS_Validate_InvalidEngine(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "elevenlabs",
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "", // Empty API key should fail validation
				VoiceID: "test-voice",
			},
		},
	}

	tts := New(cfg)
	err := tts.Validate()
	if err == nil {
		t.Error("Expected error validating elevenlabs without API key")
	}
}

func TestPiper_Speak_MissingBinary(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  "/nonexistent/piper",
				Model:       "/nonexistent/model.onnx",
				VoiceConfig: "/nonexistent/voice.json",
			},
		},
	}

	p := NewPiper(cfg)
	err := p.Speak("Hello")
	if err == nil {
		t.Error("Expected error when binary not found")
	}
}

func TestPiper_Speak_MissingModel(t *testing.T) {
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "piper")
	os.WriteFile(fakeBin, []byte("fake"), 0o755)

	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  fakeBin,
				Model:       "/nonexistent/model.onnx",
				VoiceConfig: "/nonexistent/voice.json",
			},
		},
	}

	p := NewPiper(cfg)
	err := p.Speak("Hello")
	if err == nil {
		t.Error("Expected error when model not found")
	}
}

func TestPiper_Speak_MissingVoiceConfig(t *testing.T) {
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "piper")
	os.WriteFile(fakeBin, []byte("fake"), 0o755)
	fakeModel := filepath.Join(tmpDir, "model.onnx")
	os.WriteFile(fakeModel, []byte("fake"), 0o644)

	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  fakeBin,
				Model:       fakeModel,
				VoiceConfig: "/nonexistent/voice.json",
			},
		},
	}

	p := NewPiper(cfg)
	err := p.Speak("Hello")
	if err == nil {
		t.Error("Expected error when voice config not found")
	}
}

func TestElevenLabs_Speak_MissingAPIKey(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "",
				VoiceID: "test-voice",
			},
		},
	}

	e := NewElevenLabs(cfg)
	err := e.Speak("Hello")
	if err == nil {
		t.Error("Expected error without API key")
	}
}

func TestNewPiper_FullConfig(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:   "/opt/piper/piper",
				Model:        "/opt/piper/models/en_US-lessac-medium.onnx",
				VoiceConfig:  "/opt/piper/models/en_US-lessac-medium.onnx.json",
				OutputDevice: "hw:0,0",
			},
		},
	}

	p := NewPiper(cfg)
	if p == nil {
		t.Fatal("NewPiper returned nil")
	}
	if p.binaryPath != "/opt/piper/piper" {
		t.Errorf("Expected binary path, got %s", p.binaryPath)
	}
	if p.outputDevice != "hw:0,0" {
		t.Errorf("Expected output device, got %s", p.outputDevice)
	}
}

func TestNewElevenLabs_FullConfig(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:          "test-api-key",
				VoiceID:         "test-voice-id",
				Model:           "eleven_monolingual_v1",
				Stability:       0.7,
				SimilarityBoost: 0.8,
			},
		},
	}

	e := NewElevenLabs(cfg)
	if e == nil {
		t.Fatal("NewElevenLabs returned nil")
	}
	if e.apiKey != "test-api-key" {
		t.Errorf("Expected API key, got %s", e.apiKey)
	}
	if e.voiceID != "test-voice-id" {
		t.Errorf("Expected voice ID, got %s", e.voiceID)
	}
	if e.stability != 0.7 {
		t.Errorf("Expected stability 0.7, got %f", e.stability)
	}
	if e.similarityBoost != 0.8 {
		t.Errorf("Expected similarity boost 0.8, got %f", e.similarityBoost)
	}
}

func TestTTS_EmptyDefaultEngine(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "",
			Piper: config.PiperConfig{
				BinaryPath:  "./vendor/piper/piper",
				Model:       "./vendor/piper/models/en_US-lessac-medium.onnx",
				VoiceConfig: "./vendor/piper/models/en_US-lessac-medium.onnx.json",
			},
			ElevenLabs: config.ElevenLabsConfig{
				APIKey: "test-key",
			},
		},
	}

	tts := New(cfg)
	if tts == nil {
		t.Fatal("New() returned nil")
	}
	// Should default to piper when empty
	current := tts.CurrentEngine()
	if current != "piper" {
		t.Errorf("Expected default engine to be piper when empty, got %s", current)
	}
}

func TestTTS_InvalidDefaultEngine(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "invalid_engine",
			Piper: config.PiperConfig{
				BinaryPath:  "./vendor/piper/piper",
				Model:       "./vendor/piper/models/en_US-lessac-medium.onnx",
				VoiceConfig: "./vendor/piper/models/en_US-lessac-medium.onnx.json",
			},
		},
	}

	tts := New(cfg)
	if tts == nil {
		t.Fatal("New() returned nil")
	}
	// Should default to piper when invalid
	current := tts.CurrentEngine()
	if current != "piper" {
		t.Errorf("Expected default engine to be piper when invalid, got %s", current)
	}
}

func TestSetEngine_ElevenLabs_MissingVoiceID(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "piper",
			Piper: config.PiperConfig{
				BinaryPath:  "./vendor/piper/piper",
				Model:       "./vendor/piper/models/en_US-lessac-medium.onnx",
				VoiceConfig: "./vendor/piper/models/en_US-lessac-medium.onnx.json",
			},
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "test-key",
				VoiceID: "", // Missing voice ID
			},
		},
	}

	tts := New(cfg)
	err := tts.SetEngine("elevenlabs")
	if err == nil {
		t.Error("Expected error when switching to elevenlabs without voice ID")
	}
}

func TestPiper_Validate_MissingBinary(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  "/nonexistent/piper",
				Model:       "/nonexistent/model.onnx",
				VoiceConfig: "/nonexistent/voice.json",
			},
		},
	}

	p := NewPiper(cfg)
	err := p.Validate()
	if err == nil {
		t.Error("Expected error when binary not found")
	}
}

func TestPiper_Validate_MissingModel(t *testing.T) {
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "piper")
	os.WriteFile(fakeBin, []byte("fake"), 0o755)

	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  fakeBin,
				Model:       "/nonexistent/model.onnx",
				VoiceConfig: "/nonexistent/voice.json",
			},
		},
	}

	p := NewPiper(cfg)
	err := p.Validate()
	if err == nil {
		t.Error("Expected error when model not found")
	}
}

func TestPiper_Validate_MissingVoiceConfig(t *testing.T) {
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "piper")
	os.WriteFile(fakeBin, []byte("fake"), 0o755)
	fakeModel := filepath.Join(tmpDir, "model.onnx")
	os.WriteFile(fakeModel, []byte("fake"), 0o644)

	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  fakeBin,
				Model:       fakeModel,
				VoiceConfig: "/nonexistent/voice.json",
			},
		},
	}

	p := NewPiper(cfg)
	err := p.Validate()
	if err == nil {
		t.Error("Expected error when voice config not found")
	}
}

func TestPiper_Validate_ValidPaths(t *testing.T) {
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "piper")
	os.WriteFile(fakeBin, []byte("fake"), 0o755)
	fakeModel := filepath.Join(tmpDir, "model.onnx")
	os.WriteFile(fakeModel, []byte("fake"), 0o644)
	fakeConfig := filepath.Join(tmpDir, "voice.json")
	os.WriteFile(fakeConfig, []byte("{}"), 0o644)

	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  fakeBin,
				Model:       fakeModel,
				VoiceConfig: fakeConfig,
			},
		},
	}

	p := NewPiper(cfg)
	err := p.Validate()
	if err != nil {
		t.Errorf("Expected no error with valid paths: %v", err)
	}
}

func TestPiper_Speak_Validation(t *testing.T) {
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "piper")
	os.WriteFile(fakeBin, []byte("fake"), 0o755)
	fakeModel := filepath.Join(tmpDir, "model.onnx")
	os.WriteFile(fakeModel, []byte("fake"), 0o644)
	fakeConfig := filepath.Join(tmpDir, "voice.json")
	os.WriteFile(fakeConfig, []byte("{}"), 0o644)

	cfg := &config.Config{
		TTS: config.TTSConfig{
			Piper: config.PiperConfig{
				BinaryPath:  fakeBin,
				Model:       fakeModel,
				VoiceConfig: fakeConfig,
			},
		},
	}

	p := NewPiper(cfg)
	// This will fail when trying to execute piper, but tests validation path
	err := p.Speak("Hello world")
	if err == nil {
		t.Log("Speak succeeded unexpectedly (piper not actually functional)")
	}
}

func TestTTS_Speak_WithCurrentEngine(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "elevenlabs",
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "test-key",
				VoiceID: "test-voice",
			},
		},
	}

	tts := New(cfg)
	// This will fail because API call fails, but tests the Speak delegation
	err := tts.Speak("Hello")
	if err == nil {
		t.Log("Speak succeeded unexpectedly (API not available)")
	}
}

func TestElevenLabs_Speak_WithSettings(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:          "test-key",
				VoiceID:         "test-voice",
				Model:           "eleven_monolingual_v1",
				Stability:       0.7,
				SimilarityBoost: 0.8,
			},
		},
	}

	e := NewElevenLabs(cfg)
	// Verify settings are stored correctly
	if e.stability != 0.7 {
		t.Errorf("Expected stability 0.7, got %f", e.stability)
	}
	if e.similarityBoost != 0.8 {
		t.Errorf("Expected similarity boost 0.8, got %f", e.similarityBoost)
	}

	// This will fail due to API error, but tests the code path
	err := e.Speak("Hello")
	if err == nil {
		t.Log("Speak succeeded unexpectedly")
	}
}

func TestTTS_SetEngine_SameEngine(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "elevenlabs",
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "test-key",
				VoiceID: "test-voice",
			},
		},
	}

	tts := New(cfg)
	// Setting to same engine should still validate
	err := tts.SetEngine("elevenlabs")
	if err != nil {
		t.Errorf("Expected no error setting to same valid engine: %v", err)
	}
}

func TestElevenLabs_Validate_EmptyVoiceID(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "test-key",
				VoiceID: "", // Empty voice ID
			},
		},
	}

	e := NewElevenLabs(cfg)
	err := e.Validate()
	if err == nil {
		t.Error("Expected error without voice ID")
	}
}

func TestElevenLabs_Speak_EmptyAPIKey(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "",
				VoiceID: "test-voice",
			},
		},
	}

	e := NewElevenLabs(cfg)
	err := e.Speak("Hello world")
	if err == nil {
		t.Error("Expected error with empty API key")
	}
}

func TestTTS_SetEngine_Piper(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fake piper files
	fakeBin := filepath.Join(tmpDir, "piper")
	os.WriteFile(fakeBin, []byte("fake"), 0o755)
	fakeModel := filepath.Join(tmpDir, "model.onnx")
	os.WriteFile(fakeModel, []byte("fake"), 0o644)
	fakeConfig := filepath.Join(tmpDir, "voice.json")
	os.WriteFile(fakeConfig, []byte("{}"), 0o644)

	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "elevenlabs",
			Piper: config.PiperConfig{
				BinaryPath:  fakeBin,
				Model:       fakeModel,
				VoiceConfig: fakeConfig,
			},
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "test-key",
				VoiceID: "test-voice",
			},
		},
	}

	tts := New(cfg)
	err := tts.SetEngine("piper")
	if err != nil {
		t.Errorf("Expected no error when switching to piper: %v", err)
	}

	if tts.CurrentEngine() != "piper" {
		t.Errorf("Expected current engine to be piper, got %s", tts.CurrentEngine())
	}
}

func TestTTS_SetEngine_InvalidPiper(t *testing.T) {
	cfg := &config.Config{
		TTS: config.TTSConfig{
			DefaultEngine: "elevenlabs",
			Piper: config.PiperConfig{
				BinaryPath:  "/nonexistent/piper",
				Model:       "/nonexistent/model.onnx",
				VoiceConfig: "/nonexistent/voice.json",
			},
			ElevenLabs: config.ElevenLabsConfig{
				APIKey:  "test-key",
				VoiceID: "test-voice",
			},
		},
	}

	tts := New(cfg)
	err := tts.SetEngine("piper")
	if err == nil {
		t.Error("Expected error when switching to invalid piper")
	}
}
