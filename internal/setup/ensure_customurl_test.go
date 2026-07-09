/* Code Map: setup.ResolvePiperDownload tests
 * (rc1-hotpatch-29)
 *
 * - TestIsCustomURLPiperVoice: prefix detection
 * - TestResolvePiperDownload_ManifestKey: known key -> manifest entry
 * - TestResolvePiperDownload_CustomURL: sentinel -> URL pair + basename
 * - TestResolvePiperDownload_Errors: empty / unknown / malformed / path-traversal
 * - TestCustomURLBasename: query strings, fragments, traversal attempts
 * - TestEnsureModels_CustomURL: end-to-end custom URL via fake server
 *
 * CID Index:
 * CID:setup-ensure-test-004 -> TestIsCustomURLPiperVoice
 * CID:setup-ensure-test-005 -> TestResolvePiperDownload_ManifestKey
 * CID:setup-ensure-test-006 -> TestResolvePiperDownload_CustomURL
 * CID:setup-ensure-test-007 -> TestResolvePiperDownload_Errors
 * CID:setup-ensure-test-008 -> TestCustomURLBasename
 * CID:setup-ensure-test-009 -> TestEnsureModels_CustomURL
 */
package setup

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"voces/internal/download"
	"voces/internal/paths"
)

// CID:setup-ensure-test-004 - TestIsCustomURLPiperVoice
// Purpose: custom URL detection is the routing condition that
// sends EnsureModels down the custom-URL download branch. A
// false negative means the downloader tries to look up the
// sentinel in the manifest and fails. A false positive means
// a real manifest key gets parsed as a URL pair (and almost
// certainly errors). Both modes are covered.
func TestIsCustomURLPiperVoice(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"en_US-lessac-medium", false},                       // plain manifest key
		{"", false},                                          // empty is not a custom URL
		{"custom:https://huggingface.co/x.onnx|", true},      // custom URL with empty config
		{"custom:https://x|y", true},                         // custom URL with both parts
		{"Custom:https://x|y", false},                        // case-sensitive: capital C is not a sentinel
		{"custom", false},                                    // prefix without the colon is not enough
		{"prefix-custom:https://x|y", false},                 // prefix must be at the start
	}
	for _, c := range cases {
		if got := IsCustomURLPiperVoice(c.in); got != c.want {
			t.Errorf("IsCustomURLPiperVoice(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// CID:setup-ensure-test-005 - TestResolvePiperDownload_ManifestKey
// Purpose: when PiperVoice is a plain manifest key, the
// download triple is the manifest entry verbatim. The local
// filename is "<key>.onnx" so the on-disk naming matches the
// manifest's conventional key.
func TestResolvePiperDownload_ManifestKey(t *testing.T) {
	m := &Manifest{
		Piper: map[string]PiperVoiceMeta{
			"en_US-lessac-medium": {
				URL:            "https://huggingface.co/example/en_US-lessac-medium.onnx",
				VoiceConfigURL: "https://huggingface.co/example/en_US-lessac-medium.onnx.json",
				Language:       "en",
				Quality:        "medium",
			},
		},
	}
	pd, err := ResolvePiperDownload("en_US-lessac-medium", m)
	if err != nil {
		t.Fatalf("ResolvePiperDownload: %v", err)
	}
	if pd.OnnxURL != "https://huggingface.co/example/en_US-lessac-medium.onnx" {
		t.Errorf("OnnxURL = %q", pd.OnnxURL)
	}
	if pd.ConfigURL != "https://huggingface.co/example/en_US-lessac-medium.onnx.json" {
		t.Errorf("ConfigURL = %q", pd.ConfigURL)
	}
	if pd.Filename != "en_US-lessac-medium.onnx" {
		t.Errorf("Filename = %q", pd.Filename)
	}
}

// CID:setup-ensure-test-006 - TestResolvePiperDownload_CustomURL
// Purpose: the custom URL sentinel ("custom:onnxURL|configURL")
// decodes into a download triple whose Filename is derived
// from the onnx URL's last path component. The config URL
// may be empty (some piper voices ship without a sidecar).
func TestResolvePiperDownload_CustomURL(t *testing.T) {
	// Custom URL with both onnx + config.
	pd, err := ResolvePiperDownload(
		"custom:https://huggingface.co/rhasspy/voices/my-voice.onnx|https://huggingface.co/rhasspy/voices/my-voice.onnx.json",
		&Manifest{},
	)
	if err != nil {
		t.Fatalf("ResolvePiperDownload (with config): %v", err)
	}
	if pd.OnnxURL != "https://huggingface.co/rhasspy/voices/my-voice.onnx" {
		t.Errorf("OnnxURL = %q", pd.OnnxURL)
	}
	if pd.ConfigURL != "https://huggingface.co/rhasspy/voices/my-voice.onnx.json" {
		t.Errorf("ConfigURL = %q", pd.ConfigURL)
	}
	if pd.Filename != "my-voice.onnx" {
		t.Errorf("Filename = %q", pd.Filename)
	}

	// Custom URL with empty config (tail after the |).
	pd2, err := ResolvePiperDownload(
		"custom:https://huggingface.co/rhasspy/voices/solo-voice.onnx|",
		&Manifest{},
	)
	if err != nil {
		t.Fatalf("ResolvePiperDownload (no config): %v", err)
	}
	if pd2.ConfigURL != "" {
		t.Errorf("ConfigURL = %q, want empty", pd2.ConfigURL)
	}
	if pd2.Filename != "solo-voice.onnx" {
		t.Errorf("Filename = %q", pd2.Filename)
	}
}

// CID:setup-ensure-test-007 - TestResolvePiperDownload_Errors
// Purpose: error surfaces here are the safety net for the
// "trust the wizard" path. The wizard only emits valid inputs,
// but the downloader must fail loud on the edge cases so a
// stale config file (e.g. from a previous build) can't crash
// the user with a confusing "voice not found" mid-download.
func TestResolvePiperDownload_Errors(t *testing.T) {
	m := &Manifest{
		Piper: map[string]PiperVoiceMeta{
			"known": {URL: "https://x/y.onnx", VoiceConfigURL: "https://x/y.onnx.json"},
		},
	}
	cases := []struct {
		name  string
		voice string
		manif *Manifest
	}{
		{"empty voice", "", m},
		{"unknown manifest key", "unknown-voice", m},
		{"nil manifest with key", "en_US-lessac-medium", nil},
		{"custom URL with no | separator", "custom:https://x/y.onnx", m},
		{"custom URL with empty onnx", "custom:|https://x/y.onnx.json", m},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ResolvePiperDownload(c.voice, c.manif); err == nil {
				t.Errorf("expected error for %q, got nil", c.voice)
			}
		})
	}
}

// CID:setup-ensure-test-008 - TestCustomURLBasename
// Purpose: customURLBasename derives a safe on-disk filename
// from a user-pasted URL. Three classes of input:
//   - clean (no query / fragment): just the last path segment
//   - with query string or fragment: strip them first
//   - path traversal attempts: refuse
// The ".." / "/" inputs are the safety-critical cases.
func TestCustomURLBasename(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"clean", "https://huggingface.co/x/y/my-voice.onnx", "my-voice.onnx"},
		{"with query string", "https://huggingface.co/x/y/v.onnx?download=true", "v.onnx"},
		{"with fragment", "https://huggingface.co/x/y/v.onnx#section", "v.onnx"},
		{"with both", "https://x/y/v.onnx?a=1#frag", "v.onnx"},
		{"path traversal with dotdot", "https://x/../etc/passwd", "passwd"}, // filepath.Base strips leading ../
		{"trailing slash on directory", "https://x/y/z/", "z"},
		{"empty", "", ""},
		{"dot only", "https://x/y/.", ""},
		// Double trailing slash is collapsed by filepath.Clean
		// (Base calls Clean internally) so the basename
		// lands on the previous path component ("y"). This
		// is safe — the filename doesn't escape the piper
		// model dir — so we accept it rather than adding a
		// second check.
		{"double trailing slash", "https://x/y//", "y"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := customURLBasename(c.in); got != c.want {
				t.Errorf("customURLBasename(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// CID:setup-ensure-test-009 - TestEnsureModels_CustomURL
// Purpose: end-to-end happy path for the custom-URL branch.
// A state with PiperVoice = "custom:onnxURL|configURL" results
// in the downloader fetching both URLs and landing them at
// the canonical paths. The on-disk filename is the basename
// of the onnx URL (NOT "<voiceID>.onnx" — the sentinel
// already starts with "custom:").
func TestEnsureModels_CustomURL(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	whisperBytes := []byte("whisper-payload-for-custom-test")
	piperBytes := []byte("custom-piper-onnx-payload")
	piperCfgBytes := []byte(`{"lang":"de"}`)

	mux := http.NewServeMux()
	mux.HandleFunc("/whisper.bin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(whisperBytes)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(whisperBytes)
	})
	mux.HandleFunc("/custom.onnx", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(piperBytes)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(piperBytes)
	})
	mux.HandleFunc("/custom.onnx.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(piperCfgBytes)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(piperCfgBytes)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	state := &State{
		Language:     "en",
		WhisperModel: "ggml-small.en.bin",
		// Custom URL sentinel: onnx URL ends in /custom.onnx so
		// the on-disk filename is "custom.onnx".
		PiperVoice:   "custom:" + srv.URL + "/custom.onnx|" + srv.URL + "/custom.onnx.json",
		HotkeyPreset: HotkeyPresetCtrlSpace,
	}
	manifest := manifestForTest(srv) // whisper URL points at /whisper.bin

	if err := EnsureModels(context.Background(), state, manifest, download.NopProgress); err != nil {
		t.Fatalf("EnsureModels (custom URL): %v", err)
	}

	// Piper voice landed at <XDG_DATA_HOME>/voces/models/piper/custom.onnx
	piperDir, err := paths.PiperModelDir()
	if err != nil {
		t.Fatalf("paths.PiperModelDir: %v", err)
	}
	piperPath := filepath.Join(piperDir, "custom.onnx")
	if _, err := os.Stat(piperPath); err != nil {
		t.Fatalf("piper custom URL missing at %q: %v", piperPath, err)
	}
	// Config landed at <same>.json
	piperCfgPath := piperPath + ".json"
	if _, err := os.Stat(piperCfgPath); err != nil {
		t.Fatalf("piper custom config missing at %q: %v", piperCfgPath, err)
	}
}
