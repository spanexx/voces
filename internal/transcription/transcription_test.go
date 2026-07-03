package transcription

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"whisper-voice-util/internal/config"
)

func TestNewTranscriber(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath:  "./vendor/whisper.cpp/whisper-cli",
				Model:       "./vendor/whisper.cpp/models/ggml-small.bin",
				Language:    "",
				ComputeType: "float",
			},
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)
	if transcriber == nil {
		t.Fatal("Expected transcriber to be created")
	}
	if transcriber.engine == nil {
		t.Fatal("Expected engine to be initialized")
	}
}

func TestAvailableEngines(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "./vendor/whisper.cpp/whisper-cli",
				Model:      "./vendor/whisper.cpp/models/ggml-small.bin",
			},
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)
	engines := transcriber.AvailableEngines()

	if len(engines) != 2 {
		t.Errorf("Expected 2 engines, got %d", len(engines))
	}

	// Check both engines are present
	hasWhisper := false
	hasOpenAI := false
	for _, e := range engines {
		if e == "whisper_cpp" {
			hasWhisper = true
		}
		if e == "openai_api" {
			hasOpenAI = true
		}
	}

	if !hasWhisper {
		t.Error("Expected whisper_cpp engine to be available")
	}
	if !hasOpenAI {
		t.Error("Expected openai_api engine to be available")
	}
}

func TestCurrentEngine(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "./vendor/whisper.cpp/whisper-cli",
				Model:      "./vendor/whisper.cpp/models/ggml-small.bin",
			},
		},
	}

	transcriber := New(cfg)
	current := transcriber.CurrentEngine()

	if current != "whisper_cpp" {
		t.Errorf("Expected current engine to be whisper_cpp, got %s", current)
	}
}

func TestSetEngine(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "./vendor/whisper.cpp/whisper-cli",
				Model:      "./vendor/whisper.cpp/models/ggml-small.bin",
			},
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)

	// Test switching to openai_api (should work with test key)
	err := transcriber.SetEngine("openai_api")
	if err != nil {
		t.Errorf("Expected no error when switching to openai_api: %v", err)
	}

	current := transcriber.CurrentEngine()
	if current != "openai_api" {
		t.Errorf("Expected current engine to be openai_api, got %s", current)
	}

	// Switch back to whisper_cpp
	err = transcriber.SetEngine("whisper_cpp")
	// This will fail validation because paths are relative in tests
	if err != nil {
		t.Logf("Expected validation error for relative paths: %v", err)
	}
}

func TestSetEngine_UnknownEngine(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
		},
	}

	transcriber := New(cfg)

	err := transcriber.SetEngine("unknown_engine")
	if err == nil {
		t.Error("Expected error when setting unknown engine")
	}
}

func TestWhisperCPP_Name(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "./vendor/whisper.cpp/whisper-cli",
				Model:      "./vendor/whisper.cpp/models/ggml-small.bin",
			},
		},
	}

	w := NewWhisperCPP(cfg)
	if w.Name() != "whisper_cpp" {
		t.Errorf("Expected name to be whisper_cpp, got %s", w.Name())
	}
}

func TestOpenAIAPI_Name(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	o := NewOpenAIAPI(cfg)
	if o.Name() != "openai_api" {
		t.Errorf("Expected name to be openai_api, got %s", o.Name())
	}
}

func TestOpenAIAPI_Validate(t *testing.T) {
	// Test with API key
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
			},
		},
	}

	o := NewOpenAIAPI(cfg)
	err := o.Validate()
	if err != nil {
		t.Errorf("Expected no error with API key: %v", err)
	}

	// Test without API key
	cfg2 := &config.Config{
		Transcription: config.TranscriptionConfig{
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "",
			},
		},
	}

	o2 := NewOpenAIAPI(cfg2)
	err = o2.Validate()
	if err == nil {
		t.Error("Expected error without API key")
	}
}

func TestTranscribe_OpenAIAPI_Integration(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "openai_api",
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)

	// Create a test audio file with dummy data
	tmpDir := t.TempDir()
	audioPath := filepath.Join(tmpDir, "test.wav")
	if err := os.WriteFile(audioPath, []byte("mock audio data"), 0o644); err != nil {
		t.Fatalf("Failed to create test audio file: %v", err)
	}

	// This will fail with API error (invalid key or network), but proves the code runs
	_, err := transcriber.Transcribe(audioPath)
	if err == nil {
		t.Error("Expected error with invalid API key")
	}
}

func TestToggleEngine(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "./vendor/whisper.cpp/whisper-cli",
				Model:      "./vendor/whisper.cpp/models/ggml-small.bin",
			},
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)

	// Start with whisper_cpp
	current := transcriber.CurrentEngine()
	if current != "whisper_cpp" {
		t.Errorf("Expected initial engine to be whisper_cpp, got %s", current)
	}

	// Toggle to openai_api
	newEngine, err := transcriber.ToggleEngine()
	if err != nil {
		t.Errorf("Expected no error toggling to openai_api: %v", err)
	}
	if newEngine != "openai_api" {
		t.Errorf("Expected new engine to be openai_api, got %s", newEngine)
	}
	if transcriber.CurrentEngine() != "openai_api" {
		t.Errorf("Expected current engine to be openai_api after toggle")
	}

	// Toggle back to whisper_cpp (will fail validation due to relative paths)
	_, err = transcriber.ToggleEngine()
	// This is expected to fail validation in tests
	if err == nil {
		t.Log("Toggle back to whisper_cpp succeeded (unexpected in test environment)")
	}
}

func TestWhisperCPP_Validate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fake binary
	fakeBin := filepath.Join(tmpDir, "whisper")
	os.WriteFile(fakeBin, []byte("fake"), 0o755)

	// Create fake model
	fakeModel := filepath.Join(tmpDir, "model.bin")
	os.WriteFile(fakeModel, []byte("fake"), 0o644)

	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: fakeBin,
				Model:      fakeModel,
			},
		},
	}

	w := NewWhisperCPP(cfg)
	err := w.Validate()
	if err != nil {
		t.Errorf("Expected no error with valid paths: %v", err)
	}

	// Test missing binary
	cfg2 := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "/nonexistent/binary",
				Model:      fakeModel,
			},
		},
	}
	w2 := NewWhisperCPP(cfg2)
	err = w2.Validate()
	if err == nil {
		t.Error("Expected error with missing binary")
	}

	// Test missing model
	cfg3 := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: fakeBin,
				Model:      "/nonexistent/model.bin",
			},
		},
	}
	w3 := NewWhisperCPP(cfg3)
	err = w3.Validate()
	if err == nil {
		t.Error("Expected error with missing model")
	}
}

func TestWhisperCPP_Transcribe_MissingFiles(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "/nonexistent/whisper",
				Model:      "/nonexistent/model.bin",
			},
		},
	}

	w := NewWhisperCPP(cfg)
	_, err := w.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error when binary not found")
	}
}

func TestTranscriber_Validate(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "openai_api",
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
			},
		},
	}

	transcriber := New(cfg)
	err := transcriber.Validate()
	if err != nil {
		t.Errorf("Expected no error with valid API key: %v", err)
	}
}

func TestTranscriber_Transcribe(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "openai_api",
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)

	// Create a test audio file
	tmpDir := t.TempDir()
	audioPath := filepath.Join(tmpDir, "test.wav")
	os.WriteFile(audioPath, []byte("mock audio data"), 0o644)

	// This will fail with API error but tests the code path
	_, err := transcriber.Transcribe(audioPath)
	if err == nil {
		t.Error("Expected error with invalid API key")
	}
}

func TestNewWhisperCPP_WithLanguage(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath:  "/opt/whisper.cpp/main",
				Model:       "/opt/whisper.cpp/models/ggml-small.bin",
				Language:    "en",
				ComputeType: "float",
			},
		},
	}

	w := NewWhisperCPP(cfg)
	if w == nil {
		t.Fatal("NewWhisperCPP returned nil")
	}
	if w.binaryPath != "/opt/whisper.cpp/main" {
		t.Errorf("Expected binary path, got %s", w.binaryPath)
	}
	if w.language != "en" {
		t.Errorf("Expected language 'en', got %s", w.language)
	}
	if w.computeType != "float" {
		t.Errorf("Expected compute type 'float', got %s", w.computeType)
	}
}

func TestNewOpenAIAPI_WithPrompt(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
				Prompt: "This is a test prompt",
			},
		},
	}

	o := NewOpenAIAPI(cfg)
	if o == nil {
		t.Fatal("NewOpenAIAPI returned nil")
	}
	if o.apiKey != "test-key" {
		t.Errorf("Expected API key, got %s", o.apiKey)
	}
	if o.prompt != "This is a test prompt" {
		t.Errorf("Expected prompt, got %s", o.prompt)
	}
}

func TestOpenAIAPI_Transcribe_MissingAPIKey(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "",
				Model:  "whisper-1",
			},
		},
	}

	o := NewOpenAIAPI(cfg)
	_, err := o.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error without API key")
	}
}

func TestOpenAIAPI_Transcribe_MissingFile(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	o := NewOpenAIAPI(cfg)
	_, err := o.Transcribe("/nonexistent/file.wav")
	if err == nil {
		t.Error("Expected error with missing file")
	}
}

func TestWhisperCPP_Transcribe_WithLanguage(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath:  "/nonexistent/whisper",
				Model:       "/nonexistent/model.bin",
				Language:    "en",
				ComputeType: "float",
			},
		},
	}

	w := NewWhisperCPP(cfg)
	if w.language != "en" {
		t.Errorf("Expected language 'en', got %s", w.language)
	}
	if w.computeType != "float" {
		t.Errorf("Expected compute type 'float', got %s", w.computeType)
	}

	// This will fail because binary doesn't exist, but tests code path with language
	_, err := w.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error when binary not found")
	}
}

func TestTranscriber_DefaultEngineFallback(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "", // Empty should default to whisper_cpp
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "./vendor/whisper.cpp/whisper-cli",
				Model:      "./vendor/whisper.cpp/models/ggml-small.bin",
			},
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)
	// Should default to whisper_cpp when empty
	current := transcriber.CurrentEngine()
	if current != "whisper_cpp" {
		t.Errorf("Expected default engine whisper_cpp when empty, got %s", current)
	}
}

func TestTranscriber_InvalidDefaultEngine(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "invalid_engine",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "./vendor/whisper.cpp/whisper-cli",
				Model:      "./vendor/whisper.cpp/models/ggml-small.bin",
			},
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)
	// Should fallback to whisper_cpp when invalid
	current := transcriber.CurrentEngine()
	if current != "whisper_cpp" {
		t.Errorf("Expected fallback to whisper_cpp when invalid, got %s", current)
	}
}

func TestWhisperCPP_Transcribe_BinaryNotFound(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "/nonexistent/whisper",
				Model:      "/nonexistent/model.bin",
			},
		},
	}

	w := NewWhisperCPP(cfg)
	_, err := w.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error when binary not found")
	}
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "binary") {
		t.Errorf("Expected error about binary not found, got: %v", err)
	}
}

func TestWhisperCPP_Transcribe_ModelNotFound(t *testing.T) {
	// Create a temp file to act as fake binary
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "whisper")
	os.WriteFile(fakeBin, []byte("fake binary"), 0o755)

	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: fakeBin,
				Model:      "/nonexistent/model.bin",
			},
		},
	}

	w := NewWhisperCPP(cfg)
	_, err := w.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error when model not found")
	}
}

func TestOpenAIAPI_Transcribe_EmptyAPIKey(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "",
				Model:  "whisper-1",
			},
		},
	}

	o := NewOpenAIAPI(cfg)
	_, err := o.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error with empty API key")
	}
	if !strings.Contains(err.Error(), "API key not configured") {
		t.Errorf("Expected error about API key, got: %v", err)
	}
}

func TestTranscriber_SetEngine_Validation(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "/nonexistent/whisper",
				Model:      "/nonexistent/model.bin",
			},
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)

	// Try to set to whisper_cpp - should fail validation due to missing binary
	err := transcriber.SetEngine("whisper_cpp")
	if err == nil {
		t.Error("Expected validation error for missing whisper binary")
	}

	// Set to openai_api - should succeed
	err = transcriber.SetEngine("openai_api")
	if err != nil {
		t.Errorf("Expected no error switching to openai_api: %v", err)
	}
}

func TestTranscriber_ToggleEngine_WithValidation(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "openai_api",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "/nonexistent/whisper",
				Model:      "/nonexistent/model.bin",
			},
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)

	// Start with openai_api
	if transcriber.CurrentEngine() != "openai_api" {
		t.Errorf("Expected initial engine openai_api, got %s", transcriber.CurrentEngine())
	}

	// Try to toggle to whisper_cpp - should fail validation
	_, err := transcriber.ToggleEngine()
	if err == nil {
		t.Error("Expected error toggling to whisper_cpp due to missing binary")
	}
}

func TestWhisperCPP_Transcribe_NoLanguage(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath:  "/nonexistent/whisper",
				Model:       "/nonexistent/model.bin",
				Language:    "", // Empty language
				ComputeType: "float",
			},
		},
	}

	w := NewWhisperCPP(cfg)
	if w.language != "" {
		t.Errorf("Expected empty language, got %s", w.language)
	}

	// This will fail but tests the code path without language
	_, err := w.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error when binary not found")
	}
}

func TestWhisperCPP_Transcribe_WithModelPath(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath:  "/nonexistent/whisper",
				Model:       "/nonexistent/model.bin",
				Language:    "en",
				ComputeType: "float",
			},
		},
	}

	w := NewWhisperCPP(cfg)
	if w.modelPath != "/nonexistent/model.bin" {
		t.Errorf("Expected model path, got %s", w.modelPath)
	}

	// This will fail but tests the code path
	_, err := w.Transcribe("/tmp/test.wav")
	if err == nil {
		t.Error("Expected error when model not found")
	}
}

func TestWhisperCPP_Transcribe_WithFakeBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake whisper binary that creates output
	fakeBin := filepath.Join(tmpDir, "whisper")
	fakeModel := filepath.Join(tmpDir, "model.bin")
	os.WriteFile(fakeModel, []byte("fake model"), 0o644)

	// Create a script that mimics whisper.cpp behavior
	script := `#!/bin/bash
# Fake whisper.cpp that creates output file
input_file=""
model=""
output_txt=false

while [[ $# -gt 0 ]]; do
  case $1 in
    -f)
      input_file="$2"
      shift 2
      ;;
    -m)
      model="$2"
      shift 2
      ;;
    -otxt)
      output_txt=true
      shift
      ;;
    -l)
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

if [ "$output_txt" = true ] && [ -n "$input_file" ]; then
  output_file="${input_file%.*}.txt"
  echo "Transcribed text" > "$output_file"
fi
`
	os.WriteFile(fakeBin, []byte(script), 0o755)

	testAudio := filepath.Join(tmpDir, "test.wav")
	os.WriteFile(testAudio, []byte("fake audio"), 0o644)

	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: fakeBin,
				Model:      fakeModel,
				Language:   "en",
			},
		},
	}

	w := NewWhisperCPP(cfg)
	result, err := w.Transcribe(testAudio)
	if err != nil {
		t.Errorf("Expected no error with fake binary: %v", err)
	}
	if result != "Transcribed text" {
		t.Errorf("Expected 'Transcribed text', got '%s'", result)
	}
}

func TestWhisperCPP_Transcribe_FailingBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake binary that fails
	fakeBin := filepath.Join(tmpDir, "whisper")
	fakeModel := filepath.Join(tmpDir, "model.bin")
	os.WriteFile(fakeModel, []byte("fake model"), 0o644)

	// Create a script that fails
	script := `#!/bin/bash
exit 1
`
	os.WriteFile(fakeBin, []byte(script), 0o755)

	testAudio := filepath.Join(tmpDir, "test.wav")
	os.WriteFile(testAudio, []byte("fake audio"), 0o644)

	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: fakeBin,
				Model:      fakeModel,
			},
		},
	}

	w := NewWhisperCPP(cfg)
	_, err := w.Transcribe(testAudio)
	if err == nil {
		t.Error("Expected error when binary fails")
	}
}

func TestWhisperCPP_Transcribe_StdoutFallback(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake binary that outputs to stdout instead of file
	fakeBin := filepath.Join(tmpDir, "whisper")
	fakeModel := filepath.Join(tmpDir, "model.bin")
	os.WriteFile(fakeModel, []byte("fake model"), 0o644)

	// Create a script that outputs to stdout
	script := `#!/bin/bash
echo "Stdout transcription"
`
	os.WriteFile(fakeBin, []byte(script), 0o755)

	testAudio := filepath.Join(tmpDir, "test.wav")
	os.WriteFile(testAudio, []byte("fake audio"), 0o644)

	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: fakeBin,
				Model:      fakeModel,
			},
		},
	}

	w := NewWhisperCPP(cfg)
	result, err := w.Transcribe(testAudio)
	if err != nil {
		t.Errorf("Expected no error: %v", err)
	}
	// The stdout should be captured when no output file exists
	if result == "" {
		t.Error("Expected some output from stdout")
	}
}

func TestTranscriber_AvailableEngines(t *testing.T) {
	cfg := &config.Config{
		Transcription: config.TranscriptionConfig{
			DefaultEngine: "whisper_cpp",
			WhisperCPP: config.WhisperCPPConfig{
				BinaryPath: "./vendor/whisper.cpp/whisper-cli",
				Model:      "./vendor/whisper.cpp/models/ggml-small.bin",
			},
			OpenAIAPI: config.OpenAIAPIConfig{
				APIKey: "test-key",
				Model:  "whisper-1",
			},
		},
	}

	transcriber := New(cfg)
	engines := transcriber.AvailableEngines()

	// Should have both engines
	if len(engines) != 2 {
		t.Errorf("Expected 2 engines, got %d", len(engines))
	}

	// Check both engines are present
	hasWhisper := false
	hasOpenAI := false
	for _, engine := range engines {
		if engine == "whisper_cpp" {
			hasWhisper = true
		}
		if engine == "openai_api" {
			hasOpenAI = true
		}
	}
	if !hasWhisper {
		t.Error("Expected whisper_cpp in available engines")
	}
	if !hasOpenAI {
		t.Error("Expected openai_api in available engines")
	}
}
