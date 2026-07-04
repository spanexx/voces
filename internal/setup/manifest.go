/* Code Map: Model Manifest
 * - WhisperModelMeta: one whisper .bin entry from models.json
 * - PiperVoiceMeta: one piper .onnx voice entry
 * - Manifest: full manifest of all available models and voices
 * - LoadManifest: read models.json from disk
 * - DefaultManifest: built-in fallback when models.json is missing
 *   (dev mode only)
 *
 * CID Index:
 * CID:setup-manifest-001 -> WhisperModelMeta
 * CID:setup-manifest-002 -> PiperVoiceMeta
 * CID:setup-manifest-003 -> Manifest
 * CID:setup-manifest-004 -> LoadManifest
 * CID:setup-manifest-005 -> DefaultManifest
 *
 * Quick lookup: rg -n "CID:setup-manifest-" internal/setup/
 */
package setup

import (
	"encoding/json"
	"fmt"
	"os"
)

// manifestFileName is the leaf name of the model manifest inside engines/.
const manifestFileName = "models.json"

// CID:setup-manifest-001 - WhisperModelMeta
// Purpose: Metadata for one whisper .bin file.
type WhisperModelMeta struct {
	URL         string `json:"url"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256,omitempty"`
	Language    string `json:"language"`     // "en" or "multilingual"
	Tier        string `json:"tier"`         // "tiny-en", "base", "small-en", etc.
	DisplayName string `json:"display_name"` // human label for the UI
}

// CID:setup-manifest-002 - PiperVoiceMeta
// Purpose: Metadata for one piper .onnx voice.
type PiperVoiceMeta struct {
	URL            string `json:"url"`
	VoiceConfigURL string `json:"voice_config_url"`
	SizeBytes      int64  `json:"size_bytes"`
	Language       string `json:"language"`    // ISO 639-1
	Quality        string `json:"quality"`     // "low", "medium", "high"
	DisplayName    string `json:"display_name"`
}

// CID:setup-manifest-003 - Manifest
// Purpose: Full list of models and voices the wizard can offer.
type Manifest struct {
	Whisper map[string]WhisperModelMeta `json:"whisper"` // keyed by file name, e.g. "ggml-small.en.bin"
	Piper   map[string]PiperVoiceMeta   `json:"piper"`   // keyed by voice base name, e.g. "en_US-lessac-medium"
}

// CID:setup-manifest-004 - LoadManifest
// Purpose: Reads models.json from disk. Returns an error if the file is
// missing or malformed. Callers may fall back to DefaultManifest in dev.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %q: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest %q: %w", path, err)
	}
	if m.Whisper == nil {
		m.Whisper = map[string]WhisperModelMeta{}
	}
	if m.Piper == nil {
		m.Piper = map[string]PiperVoiceMeta{}
	}
	return &m, nil
}

// CID:setup-manifest-005 - DefaultManifest
// Purpose: Returns a built-in manifest pointing at the canonical HuggingFace
// URLs. Used when models.json is missing (dev mode) and as a sanity check in
// tests. The real tarball ships models.json with the URLs frozen at build
// time; this is a fallback only.
func DefaultManifest() *Manifest {
	const baseURL = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main"
	return &Manifest{
		Whisper: map[string]WhisperModelMeta{
			"ggml-small.en.bin": {
				URL:         baseURL + "/ggml-small.en.bin",
				SizeBytes:   488479232, // ~466 MB; pinned at IMPL time
				Language:    "en",
				Tier:        "small-en",
				DisplayName: "Small (English, 466 MB)",
			},
			"ggml-base.bin": {
				URL:         baseURL + "/ggml-base.bin",
				SizeBytes:   147964480, // ~141 MB; pinned at IMPL time
				Language:    "multilingual",
				Tier:        "base",
				DisplayName: "Base (multilingual, 141 MB)",
			},
		},
		Piper: map[string]PiperVoiceMeta{
			"en_US-lessac-medium": {
				URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx",
				VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json",
				SizeBytes:      63123456, // ~60 MB; pinned at IMPL time
				Language:       "en",
				Quality:        "medium",
				DisplayName:    "US English (Lessac, medium quality)",
			},
		},
	}
}
