/* Code Map: Piper paths resolution (rc30)
 *
 * piperPathsForState translates setup.State.PiperVoice into
 * the (model, voice_config) pair that goes into the
 * generated config.yaml's tts.piper block. Handles two
 * shapes of PiperVoice:
 *
 *   1. Manifest key (e.g. "en_US-lessac-medium"): the model
 *      lives at <models>/piper/<key>.onnx, the config at
 *      <models>/piper/<key>.onnx.json. The pre-rc30 code
 *      used paths.PiperVoicePath(s.PiperVoice) which
 *      appends ".onnx" to the basename — correct for
 *      manifest keys.
 *
 *   2. Custom URL sentinel (e.g.
 *      "custom:https://x/v.onnx|https://x/v.onnx.json"):
 *      the basename is derived from the onnx URL's last
 *      path component, sanitised. The voice config basename
 *      is derived from the config URL's last path
 *      component (empty when the user omits the config
 *      URL — some piper voices ship without a sidecar).
 *      The pre-rc30 code produced an invalid path
 *      containing ":" and "|" from the sentinel — Piper
 *      .Validate() then failed the model stat and the
 *      hotkey handler showed "TTS Unavailable" despite
 *      the model having downloaded successfully.
 *
 * CID Index:
 * CID:setup-defaults-003 -> piperPathsForState
 *
 * Quick lookup: rg -n "CID:setup-defaults-" internal/setup/
 */
package setup

import (
	"path/filepath"

	"voces/internal/paths"
)

// CID:setup-defaults-003 - piperPathsForState
// Purpose: turn setup.State.PiperVoice into the
// (model_path, voice_config_path) pair for config.yaml.
// Manifest keys use paths.PiperVoicePath; custom URL
// sentinels use ResolvePiperDownload to derive the
// basename from the onnx URL. Empty PiperVoice returns
// ("", "") so the rest of the config picks it up as
// "TTS disabled".
//
// Returns ("", "") on any error (missing manifest, malformed
// sentinel). The previous behaviour returned ("", error)
// which the caller already discarded with `_`; the
// equivalent here is to surface the failure as a non-fatal
// empty value. Apply will then write an empty model path
// and the runtime will treat TTS as disabled, which is the
// safer answer than panicking on a corrupt state file.
func piperPathsForState(s *State) (model, voiceConfig string) {
	if s == nil || s.PiperVoice == "" {
		return "", ""
	}
	// Custom URL branch. ResolvePiperDownload does the
	// sanitisation; we just join the basename onto the
	// canonical piper model dir. Empty ConfigURL yields
	// an empty voice config (the runtime treats that as
	// "no sidecar").
	if IsCustomURLPiperVoice(s.PiperVoice) {
		pd, err := ResolvePiperDownload(s.PiperVoice, nil)
		if err != nil {
			return "", ""
		}
		piperDir, err := paths.PiperModelDir()
		if err != nil {
			return "", ""
		}
		model = filepath.Join(piperDir, pd.Filename)
		if pd.ConfigURL != "" {
			voiceConfig = model + ".json"
		}
		return model, voiceConfig
	}
	// Manifest key branch. Existing behaviour, kept
	// identical to the pre-rc30 code so the manifest-key
	// config path doesn't shift.
	model, err := paths.PiperVoicePath(s.PiperVoice)
	if err != nil {
		return "", ""
	}
	voiceConfig = model + ".json"
	return model, voiceConfig
}
