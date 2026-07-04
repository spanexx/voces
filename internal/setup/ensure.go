/* Code Map: setup.EnsureModels
 * - EnsureModels: download the whisper model + (optionally) the piper
 *   voice + voice config that the wizard chose. Resolves canonical
 *   paths via internal/paths. Lives in internal/setup so the cmd
 *   layer can call setup.Apply + setup.EnsureModels back-to-back
 *   after wizard.RunFull returns.
 *
 * CID Index:
 * CID:setup-ensure-001 -> EnsureModels
 *
 * Quick lookup: rg -n "CID:setup-ensure-" internal/setup/
 */
package setup

import (
	"context"
	"fmt"

	"whisper-voice-util/internal/download"
	"whisper-voice-util/internal/paths"
)

// CID:setup-ensure-001 - EnsureModels
// Purpose: download the whisper .bin (always) and the piper .onnx +
// .json (when the wizard enabled TTS, signalled by PiperVoice != "").
// Resolves canonical paths via internal/paths and delegates to
// internal/download.Download for the actual fetch (retry, progress,
// SHA-256 verify, .partial handling).
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
	piperMeta, ok := m.Piper[s.PiperVoice]
	if !ok {
		return fmt.Errorf("EnsureModels: piper voice %q not in manifest", s.PiperVoice)
	}
	piperPath, err := paths.PiperVoicePath(s.PiperVoice)
	if err != nil {
		return fmt.Errorf("EnsureModels: resolve piper path: %w", err)
	}
	if err := download.Download(ctx, piperMeta.URL, piperPath, "", progress); err != nil {
		return fmt.Errorf("EnsureModels: download piper voice: %w", err)
	}
	if piperMeta.VoiceConfigURL != "" {
		if err := download.Download(ctx, piperMeta.VoiceConfigURL, piperPath+".json", "", progress); err != nil {
			return fmt.Errorf("EnsureModels: download piper voice config: %w", err)
		}
	}
	return nil
}
