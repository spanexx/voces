/* Code Map: setup.EnsureModels
 * - EnsureModels: download the whisper model + (optionally) the piper
 *   voice + voice config that the wizard chose. Resolves canonical
 *   paths via internal/paths. Lives in internal/setup so the cmd
 *   layer can call setup.Apply + setup.EnsureModels back-to-back
 *   after wizard.RunFull returns.
 * - ResolvePiperDownload: turns a PiperVoice field into the
 *   (URL, config URL, local filename) triple the downloader
 *   needs. Manifest keys look up the manifest; custom URL
 *   sentinels (rc1-hotpatch-29) parse the encoded URL pair
 *   and derive a local filename from the onnx URL's last
 *   path component.
 *
 * rc1-hotpatch-29: the wizard's "Custom URL..." picker stores
 * the user's onnx + json URLs in a single string in the
 * PiperVoice field (see steps.customURLSentinel). EnsureModels
 * detects the `custom:` prefix, falls through to
 * ResolvePiperDownload's custom-URL branch, and downloads
 * directly to ~/.local/share/voces/models/piper/<basename>.
 * The basename is sanitised (no slashes) so a user-pasted
 * URL can't escape the piper model dir.
 *
 * CID Index:
 * CID:setup-ensure-001 -> EnsureModels
 * CID:setup-ensure-002 -> ResolvePiperDownload
 * CID:setup-ensure-003 -> IsCustomURLPiperVoice
 *
 * Quick lookup: rg -n "CID:setup-ensure-" internal/setup/
 */
package setup

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"voces/internal/download"
	"voces/internal/paths"
)

// customURLPrefix is the sentinel marker steps.customURLSentinel
// prepends to mark a PiperVoice as a custom URL pair. We
// redeclare it here (instead of importing steps) to keep the
// setup package free of wizard dependencies — wizard already
// imports setup for HotkeyPreset* constants.
const customURLPrefix = "custom:"

// IsCustomURLPiperVoice reports whether the given PiperVoice
// string is a custom-URL sentinel (rc1-hotpatch-29) rather
// than a manifest key. Used by EnsureModels and Apply to
// route to the custom-URL download branch.
func IsCustomURLPiperVoice(voice string) bool {
	return strings.HasPrefix(voice, customURLPrefix)
}

// PiperDownload describes one piper voice's on-disk location
// and the source URL(s). Used by EnsureModels and (later) by
// Validate so Apply can pre-check that a custom-URL is
// reachable before kicking off a long download.
type PiperDownload struct {
	// OnnxURL is the source URL for the .onnx file.
	OnnxURL string
	// ConfigURL is the source URL for the .onnx.json file
	// (may be empty for voices that ship without a config).
	ConfigURL string
	// Filename is the on-disk filename, e.g.
	// "en_US-lessac-medium.onnx" for a manifest key, or
	// "my-custom-voice.onnx" for a custom URL. The
	// downloader writes to ~/.local/share/voces/models/piper/<filename>.
	Filename string
}

// CID:setup-ensure-002 - ResolvePiperDownload
// Purpose: turn a PiperVoice field into the (URL, config URL,
// local filename) triple the downloader needs. Manifest keys
// look up the manifest; custom URL sentinels (rc1-hotpatch-29)
// parse the encoded URL pair and derive a local filename from
// the onnx URL's last path component (sanitised — no slashes,
// no parent-dir traversal).
//
// Returns an error when:
//   - the manifest key is unknown
//   - the custom URL is malformed (no `|` separator)
//   - the onnx URL has no usable basename
func ResolvePiperDownload(piperVoice string, m *Manifest) (*PiperDownload, error) {
	if piperVoice == "" {
		return nil, fmt.Errorf("ResolvePiperDownload: empty piper voice")
	}
	if IsCustomURLPiperVoice(piperVoice) {
		// Strip the prefix, split on the first "|".
		body := strings.TrimPrefix(piperVoice, customURLPrefix)
		idx := strings.Index(body, "|")
		if idx < 0 {
			return nil, fmt.Errorf("ResolvePiperDownload: custom URL %q has no `|` separator", piperVoice)
		}
		onnx := body[:idx]
		cfg := body[idx+1:]
		base := customURLBasename(onnx)
		if base == "" {
			return nil, fmt.Errorf("ResolvePiperDownload: custom URL %q has no usable basename", onnx)
		}
		return &PiperDownload{
			OnnxURL:   onnx,
			ConfigURL: cfg,
			Filename:  base,
		}, nil
	}
	// Manifest-key path. Look up the entry directly; errors
	// out on unknown keys (which the wizard should prevent
	// by only listing known voices, but defence in depth is
	// cheap).
	if m == nil {
		return nil, fmt.Errorf("ResolvePiperDownload: manifest is nil for voice %q", piperVoice)
	}
	meta, ok := m.Piper[piperVoice]
	if !ok {
		return nil, fmt.Errorf("ResolvePiperDownload: piper voice %q not in manifest", piperVoice)
	}
	return &PiperDownload{
		OnnxURL:   meta.URL,
		ConfigURL: meta.VoiceConfigURL,
		Filename:  piperVoice + ".onnx",
	}, nil
}

// customURLBasename derives a safe on-disk filename from a
// user-pasted URL. Returns the last path component with
// slashes stripped — or empty if the URL has no useful tail
// (e.g. the trailing slash on a directory listing). Sanitised
// against path traversal: any leading ".." or "/" in the
// derived basename is replaced.
func customURLBasename(onnxURL string) string {
	// strip query string + fragment (piper URLs sometimes
	// have ?download=true; we want the .onnx filename)
	clean := strings.SplitN(onnxURL, "?", 2)[0]
	clean = strings.SplitN(clean, "#", 2)[0]
	base := filepath.Base(clean)
	if base == "." || base == "/" || base == "" {
		return ""
	}
	// Defensive: the user can't actually reach the piper
	// model dir's parent via ".." because filepath.Base
	// already strips them, but we re-check so future
	// refactors that change filepath.Base behaviour don't
	// introduce a path-traversal bug.
	if base == ".." || strings.ContainsAny(base, "/\\") {
		return ""
	}
	return base
}

// CID:setup-ensure-001 - EnsureModels
// Purpose: download the whisper .bin (always) and the piper .onnx +
// .json (when the wizard enabled TTS, signalled by PiperVoice != "").
// Resolves canonical paths via internal/paths and delegates to
// internal/download.Download for the actual fetch (retry, progress,
// SHA-256 verify, .partial handling).
//
// rc1-hotpatch-29: PiperVoice may be a manifest key
// (e.g. "en_US-lessac-medium") OR a custom URL sentinel
// ("custom:https://...voice.onnx|https://...voice.onnx.json").
// ResolvePiperDownload handles both; the download branch is
// the same either way.
//
// progress may be nil; EnsureModels substitutes download.NopProgress
// in that case so callers that don't care about progress don't have
// to thread the default through.
//
// Returns the first error encountered. Downloads are sequential, not
// concurrent, so the user sees a single progress bar reading and the
// failure mode is "whisper first, piper second" — easier to debug
// than a race for which one landed.
//
// Empty SHA-256 strings in the manifest are honoured: the downloader
// skips verification when sha256Hex == "". This is the "dev mode"
// path for the built-in DefaultManifest.
func EnsureModels(ctx context.Context, s *State, m *Manifest, progress download.ProgressFunc) error {
	if s == nil {
		return fmt.Errorf("EnsureModels: state is nil")
	}
	if m == nil {
		return fmt.Errorf("EnsureModels: manifest is nil")
	}
	if progress == nil {
		progress = download.NopProgress
	}

	whisperMeta, ok := m.Whisper[s.WhisperModel]
	if !ok {
		return fmt.Errorf("EnsureModels: whisper model %q not in manifest", s.WhisperModel)
	}
	whisperPath, err := paths.WhisperModelPath(s.WhisperModel)
	if err != nil {
		return fmt.Errorf("EnsureModels: resolve whisper path: %w", err)
	}
	if err := download.Download(ctx, whisperMeta.URL, whisperPath, whisperMeta.SHA256, progress); err != nil {
		return fmt.Errorf("EnsureModels: download whisper model: %w", err)
	}

	if s.PiperVoice == "" {
		return nil
	}
	pd, err := ResolvePiperDownload(s.PiperVoice, m)
	if err != nil {
		return fmt.Errorf("EnsureModels: resolve piper download: %w", err)
	}
	piperPath, err := paths.PiperModelDir()
	if err != nil {
		return fmt.Errorf("EnsureModels: resolve piper dir: %w", err)
	}
	piperOnnxPath := filepath.Join(piperPath, pd.Filename)
	if err := download.Download(ctx, pd.OnnxURL, piperOnnxPath, "", progress); err != nil {
		return fmt.Errorf("EnsureModels: download piper voice: %w", err)
	}
	if pd.ConfigURL != "" {
		piperCfgPath := piperOnnxPath + ".json"
		if err := download.Download(ctx, pd.ConfigURL, piperCfgPath, "", progress); err != nil {
			return fmt.Errorf("EnsureModels: download piper voice config: %w", err)
		}
	}
	return nil
}
