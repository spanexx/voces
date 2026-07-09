/* Code Map: Custom URL sentinel codec (rc1-hotpatch-29)
 *
 * The TTS step's "Custom URL..." dropdown entry stashes the
 * user's onnx + json URLs in a single string (the State.TTSVoice
 * field). The codec here defines the sentinel format and
 * provides a parser that round-trips the values.
 *
 * Split out of tts.go so the file stays under the 250-line
 * size cap enforced by scripts/check-file-size.sh. The codec
 * is pure (no GTK, no I/O) and is unit-tested in
 * TestCustomURLSentinel + TestIsCustomURLVoice.
 *
 *   customURLSentinel(onnx, config) -> "custom:<onnx>|<config>"
 *   parseCustomURLSentinel(s)       -> (onnx, config, ok)
 *   isCustomURLVoice(voiceID)       -> true when s starts with "custom:"
 *
 * The sentinel format:
 *   "custom:" prefix + <onnx URL> + "|" + <config URL or "">
 *   Example: "custom:https://huggingface.co/.../v.onnx|https://huggingface.co/.../v.onnx.json"
 *
 * Why the vertical bar:
 *   HF URLs use "/" + "?" + "=" + "&" — all of which appear
 *   in piper voice URLs. "|" never appears in a URL, so it's
 *   a safe single-character separator. The parser splits on
 *   the FIRST "|" so the config URL can contain anything.
 *
 * Why case-sensitive "custom:":
 *   Piper voice IDs are lowercase by convention (e.g.
 *   "en_US-lessac-medium"). A case-sensitive prefix keeps
 *   the check cheap and makes the codec foolproof — a voice
 *   ID like "Custom:..." is impossible to generate by
 *   accident.
 *
 * CID Index:
 * CID:wizard-ttsstep-006 -> customURLSentinel
 * CID:wizard-ttsstep-007 -> parseCustomURLSentinel
 * CID:wizard-ttsstep-008 -> isCustomURLVoice
 *
 * Quick lookup: rg -n "CID:wizard-ttsstep-" internal/wizard/steps/
 */
package steps

import "strings"

// customURLSentinelPrefix marks a TTSVoice string as a custom
// URL pair instead of a manifest key. The colon is part of
// the prefix; everything after is the onnx URL up to the
// first "|".
const customURLSentinelPrefix = "custom:"

// CID:wizard-ttsstep-006 - customURLSentinel
// Purpose: builds the "custom:<onnx>|<config>" sentinel from a
// pair of URLs. The config URL may be empty (some piper voices
// ship without a .json sidecar); the empty tail is preserved
// so the parser can still round-trip it.
//
// The format is identical to parseCustomURLSentinel's input.
func customURLSentinel(onnxURL, configURL string) string {
	return customURLSentinelPrefix + onnxURL + "|" + configURL
}

// CID:wizard-ttsstep-007 - parseCustomURLSentinel
// Purpose: reverses customURLSentinel. Returns ok=false for
// any non-sentinel string so plain manifest keys (e.g.
// "en_US-lessac-medium") never misclassify as custom URLs.
//
// Splits on the FIRST "|" so the config URL can contain
// "|" — improbable for a URL but the parser is
// future-proof against custom encodings.
func parseCustomURLSentinel(s string) (onnxURL, configURL string, ok bool) {
	if !strings.HasPrefix(s, customURLSentinelPrefix) {
		return "", "", false
	}
	body := s[len(customURLSentinelPrefix):]
	bar := strings.IndexByte(body, '|')
	if bar < 0 {
		// No "|" in the body — the sentinel must be malformed.
		// Treat as non-sentinel so the caller doesn't try to
		// download an empty onnx URL.
		return "", "", false
	}
	return body[:bar], body[bar+1:], true
}

// CID:wizard-ttsstep-008 - isCustomURLVoice
// Purpose: cheap boolean check for "is this TTSVoice a custom
// URL?". Used by the downloader to decide between "fetch the
// manifest entry" and "fetch the user's two URLs". Case-
// sensitive by design (see file header for rationale).
func isCustomURLVoice(voiceID string) bool {
	return strings.HasPrefix(voiceID, customURLSentinelPrefix)
}
