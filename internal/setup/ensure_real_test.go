//go:build smoke_real

// Code Map: Real-download smoke test (opt-in, build-tag gated)
// - TestEnsureModels_RealDownload_HappyPath: pulls the smallest real
//   whisper model from HuggingFace via DefaultManifest() and verifies
//   it lands at the canonical path with the right size. Catches URL
//   drift, hash check regressions, and path bugs that fake servers
//   miss.
//
// CID Index:
// CID:setup-ensure-test-004 -> TestEnsureModels_RealDownload_HappyPath
//
// Run:
//   go test -mod=vendor -tags=smoke_real \
//     -run TestEnsureModels_RealDownload -v ./internal/setup/...
//
// Skip behaviour: the file is build-tagged `smoke_real` so it is
// invisible to `go test ./...` (CI / pre-commit stay fast and
// offline). The fake-server tests in ensure_test.go cover CI.

package setup

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"voces/internal/download"
)

// TestEnsureModels_RealDownload_HappyPath pulls ggml-base.bin (the
// smallest model in DefaultManifest, ~141 MB) over the public
// HuggingFace URL, verifies it lands at the canonical path, and
// runs Apply to confirm state.json + config.yaml persist with
// the wizard-derived values.
//
// This file is build-tagged `smoke_real` so it is excluded from
// `go test ./...` by default. Build with `-tags=smoke_real` to run.
// Why: slow (network bound), requires outbound HTTPS, depends on
// HuggingFace availability. CI and pre-commit must not depend on
// it. Run it manually before cutting a release that touches the
// downloader or the manifest defaults.
func TestEnsureModels_RealDownload_HappyPath(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	// Smallest model in DefaultManifest — keeps the test under ~30s
	// on a normal link while still exercising the full pipeline.
	const modelKey = "ggml-base.bin"
	manifest := DefaultManifest()
	meta, ok := manifest.Whisper[modelKey]
	if !ok {
		t.Fatalf("DefaultManifest missing %q", modelKey)
	}
	if meta.SizeBytes <= 0 {
		t.Fatalf("DefaultManifest %q has no SizeBytes", modelKey)
	}

	// Bound the request so a hung server doesn't stall the test.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	state := &State{
		SchemaVersion: "1",
		AppVersion:    "vSmoke",
		CompletedAt:   time.Now().UTC(),
		Language:      "en",
		WhisperModel:  modelKey,
		PiperVoice:    "", // skip piper; faster, separate test if needed
		HotkeyPreset:  HotkeyPresetCtrlSpace,
	}

	start := time.Now()
	if err := EnsureModels(ctx, state, manifest, download.NopProgress); err != nil {
		t.Fatalf("EnsureModels (real): %v", err)
	}
	took := time.Since(start)

	// File lands at the canonical path with the expected size (±1% for
	// size_bytes drift; the manifest value is pinned at IMPL time and
	// may not match the file byte-for-byte if HuggingFace is rebuilt).
	whisperPath, err := whisperModelPathForTest(modelKey)
	if err != nil {
		t.Fatalf("whisperModelPathForTest: %v", err)
	}
	fi, err := os.Stat(whisperPath)
	if err != nil {
		t.Fatalf("stat downloaded model at %q: %v", whisperPath, err)
	}
	delta := fi.Size() - meta.SizeBytes
	if delta < 0 {
		delta = -delta
	}
	tolerance := meta.SizeBytes / 100 // 1%
	if delta > tolerance {
		t.Errorf("model size drift: got %d, manifest says %d (delta %d, tolerance %d)",
			fi.Size(), meta.SizeBytes, delta, tolerance)
	}
	t.Logf("downloaded %s: %d bytes in %s (manifest %d)", modelKey, fi.Size(), took, meta.SizeBytes)

	// Apply writes state.json + config.yaml. State must round-trip.
	if err := Apply(state, manifest); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.WhisperModel != modelKey {
		t.Errorf("state.WhisperModel = %q, want %q", loaded.WhisperModel, modelKey)
	}
	if loaded.HotkeyPreset != HotkeyPresetCtrlSpace {
		t.Errorf("state.HotkeyPreset = %q, want %q", loaded.HotkeyPreset, HotkeyPresetCtrlSpace)
	}
	if loaded.Language != "en" {
		t.Errorf("state.Language = %q, want %q", loaded.Language, "en")
	}

	// config.yaml at the canonical XDG_CONFIG_HOME path.
	cfgPath := fmt.Sprintf("%s/voces/config.yaml", cfgDir)
	if _, err := os.Stat(cfgPath); err != nil {
		t.Errorf("config.yaml missing at %q: %v", cfgPath, err)
	}
}
