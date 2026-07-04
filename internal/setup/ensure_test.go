/* Code Map: setup.EnsureModels tests
 * - TestEnsureModels_DownloadsWhisperAndPiper: happy path with fake server
 * - TestEnsureModels_SkipsPiperWhenNotEnabled: TTS off -> no piper download
 * - TestEnsureModels_ReportsDownloadError: fake 500 -> error surfaces
 *
 * CID Index:
 * CID:setup-ensure-test-001 -> TestEnsureModels_DownloadsWhisperAndPiper
 * CID:setup-ensure-test-002 -> TestEnsureModels_SkipsPiperWhenNotEnabled
 * CID:setup-ensure-test-003 -> TestEnsureModels_ReportsDownloadError
 */
package setup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"voces/internal/download"
)

// fakeModelServer serves a configurable payload at each path. Returns
// the same bytes for both whisper and piper files in the test, with
// matching SHA-256.
func fakeModelServer(t *testing.T, whisperBytes, piperBytes, piperCfgBytes []byte) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	whisperHash := sha256.Sum256(whisperBytes)
	piperHash := sha256.Sum256(piperBytes)
	piperCfgHash := sha256.Sum256(piperCfgBytes)

	mux.HandleFunc("/whisper.bin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(whisperBytes)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(whisperBytes)
	})
	mux.HandleFunc("/piper.onnx", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(piperBytes)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(piperBytes)
	})
	mux.HandleFunc("/piper.onnx.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(piperCfgBytes)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(piperCfgBytes)
	})
	_ = whisperHash
	_ = piperHash
	_ = piperCfgHash
	return httptest.NewServer(mux)
}

// manifestForTest builds a manifest whose URLs point at the given server.
func manifestForTest(srv *httptest.Server) *Manifest {
	return &Manifest{
		Whisper: map[string]WhisperModelMeta{
			"ggml-small.en.bin": {
				URL:         srv.URL + "/whisper.bin",
				SizeBytes:   1024,
				Language:    "en",
				Tier:        "small-en",
				DisplayName: "Small (English)",
			},
		},
		Piper: map[string]PiperVoiceMeta{
			"en_US-lessac-medium": {
				URL:            srv.URL + "/piper.onnx",
				VoiceConfigURL: srv.URL + "/piper.onnx.json",
				SizeBytes:      512,
				Language:       "en",
				Quality:        "medium",
				DisplayName:    "US English (Lessac)",
			},
		},
	}
}

// TestEnsureModels_DownloadsWhisperAndPiper is the IMPL §3 Phase 5 happy
// path: a state with TTS enabled results in both the whisper .bin and
// the piper .onnx + .json files landing at the canonical paths under
// $XDG_DATA_HOME.
func TestEnsureModels_DownloadsWhisperAndPiper(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	whisperBytes := []byte("whisper-payload-bytes-for-test")
	piperBytes := []byte("piper-onnx-payload-bytes")
	piperCfgBytes := []byte(`{"lang":"en"}`)
	srv := fakeModelServer(t, whisperBytes, piperBytes, piperCfgBytes)
	defer srv.Close()

	state := &State{
		Language:     "en",
		WhisperModel: "ggml-small.en.bin",
		PiperVoice:   "en_US-lessac-medium",
		HotkeyPreset: HotkeyPresetCtrlSpace,
	}
	manifest := manifestForTest(srv)

	if err := EnsureModels(context.Background(), state, manifest, download.NopProgress); err != nil {
		t.Fatalf("EnsureModels: %v", err)
	}

	// whisper model at canonical path
	whisperPath, _ := whisperModelPathForTest(state.WhisperModel)
	got, err := os.ReadFile(whisperPath)
	if err != nil {
		t.Fatalf("read whisper model: %v", err)
	}
	if string(got) != string(whisperBytes) {
		t.Errorf("whisper bytes mismatch: got %q want %q", got, whisperBytes)
	}

	// piper voice + config at canonical paths
	piperPath, _ := piperVoicePathForTest(state.PiperVoice)
	if _, err := os.Stat(piperPath); err != nil {
		t.Errorf("piper voice missing at %q: %v", piperPath, err)
	}
	piperCfgPath := piperPath + ".json"
	got2, err := os.ReadFile(piperCfgPath)
	if err != nil {
		t.Fatalf("read piper config: %v", err)
	}
	if string(got2) != string(piperCfgBytes) {
		t.Errorf("piper cfg bytes mismatch: got %q want %q", got2, piperCfgBytes)
	}
}

// TestEnsureModels_SkipsPiperWhenNotEnabled verifies that a state with
// PiperVoice == "" results in no piper download (and no error from
// the missing entry in the manifest).
func TestEnsureModels_SkipsPiperWhenNotEnabled(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	whisperBytes := []byte("only-whisper-bytes")
	srv := fakeModelServer(t, whisperBytes, nil, nil)
	defer srv.Close()

	state := &State{
		Language:     "en",
		WhisperModel: "ggml-small.en.bin",
		PiperVoice:   "", // TTS disabled
		HotkeyPreset: HotkeyPresetCtrlSpace,
	}
	manifest := manifestForTest(srv)

	if err := EnsureModels(context.Background(), state, manifest, download.NopProgress); err != nil {
		t.Fatalf("EnsureModels: %v", err)
	}

	// Whisper exists.
	whisperPath, _ := whisperModelPathForTest(state.WhisperModel)
	if _, err := os.Stat(whisperPath); err != nil {
		t.Errorf("whisper model missing: %v", err)
	}
	// Piper dir either absent or empty (no .onnx file).
	piperDir := filepath.Join(os.Getenv("XDG_DATA_HOME"), "voces", "models", "piper")
	if entries, err := os.ReadDir(piperDir); err == nil {
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".onnx" {
				t.Errorf("did not expect piper .onnx when TTS disabled, found %s", e.Name())
			}
		}
	}
}

// TestEnsureModels_ReportsDownloadError verifies that a 500 from the
// whisper server bubbles up as an error (not a silent skip).
func TestEnsureModels_ReportsDownloadError(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	mux := http.NewServeMux()
	mux.HandleFunc("/whisper.bin", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/piper.onnx", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	state := &State{
		Language:     "en",
		WhisperModel: "ggml-small.en.bin",
		HotkeyPreset: HotkeyPresetCtrlSpace,
	}
	manifest := manifestForTest(srv)

	err := EnsureModels(context.Background(), state, manifest, download.NopProgress)
	if err == nil {
		t.Fatal("expected error from 500 server, got nil")
	}
}

// hexSHA returns the lowercase hex SHA-256 of b. Kept here so callers
// can pass a real digest if they want. Not used in the current tests
// (the server already returns fixed bytes); included for future tests
// that want to assert digest-mismatch behaviour.
func hexSHA(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
