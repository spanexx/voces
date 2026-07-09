/* Code Map: TTS Step (rc1-hotpatch-29)
 *
 * The TTS step replaces the previous yes/no radio step with a
 * dropdown of curated voices filtered by the chosen language,
 * plus a "Custom URL..." entry that reveals two text inputs
 * for the .onnx and .onnx.json URLs. The hint label links to
 * the piper-voices manifest (VOICES.md) and explains how to
 * copy a voice link.
 *
 * Split into multiple files (this one + tts_helpers.go +
 * tts_sentinel.go + tts_build.go) so each file stays under the
 * 250-line size cap enforced by scripts/check-file-size.sh.
 * The split is by concern:
 *
 *   tts.go            — entry point: ShouldShow + voiceView +
 *                       ttsVoiceFromStateReader (this file)
 *   tts_helpers.go    — filterVoicesForLanguage / sortedKeys /
 *                       defaultVoiceForLanguage (pure functions
 *                       over the voice manifest)
 *   tts_sentinel.go   — customURLSentinel / parseCustomURLSentinel
 *                       / isCustomURLVoice (the sentinel codec
 *                       that lets one TTSVoice field hold either
 *                       a manifest key or a custom URL pair)
 *   tts_build.go      — BuildTTS (the GTK step factory)
 *
 * ShouldShow is now always true. The TTS step is shown for
 * every language (rc1-hotpatch-29 supersedes the rc1-hotpatch-19
 * "skip for English" rule) so the user can pick a non-default
 * voice or paste a custom URL. The setup.State.PiperVoice
 * defaults to en_US-lessac-medium for English when the user
 * keeps the default, preserving the rc1-hotpatch-19
 * "read_clipboard can speak" behavior (see wizardcli/translate.go).
 *
 * CID Index:
 * CID:wizard-ttsstep-001 -> ShouldShow
 * CID:wizard-ttsstep-002 -> BuildTTS          (tts_build.go)
 * CID:wizard-ttsstep-003 -> voiceView
 * CID:wizard-ttsstep-004 -> filterVoicesForLanguage  (tts_helpers.go)
 * CID:wizard-ttsstep-005 -> defaultVoiceForLanguage  (tts_helpers.go)
 * CID:wizard-ttsstep-006 -> customURLSentinel        (tts_sentinel.go)
 * CID:wizard-ttsstep-007 -> parseCustomURLSentinel   (tts_sentinel.go)
 * CID:wizard-ttsstep-008 -> isCustomURLVoice         (tts_sentinel.go)
 * CID:wizard-ttsstep-009 -> ttsVoiceFromStateReader
 *
 * Quick lookup: rg -n "CID:wizard-ttsstep-" internal/wizard/steps/
 */
package steps

// CID:wizard-ttsstep-001 - ShouldShow
// Purpose: rc1-hotpatch-29. The TTS step is shown for every
// language so the user can either pick a curated voice, change
// the default English voice, or paste a custom voice URL. The
// previous rc1-hotpatch-19 rule (English -> skip) is preserved
// downstream by StateFromWizard defaulting PiperVoice to
// en_US-lessac-medium when the user has not picked
// (see wizardcli/translate.go).
func ShouldShow(_ string) bool { return true }

// CID:wizard-ttsstep-003 - voiceView
// Purpose: flattened view of a single piper manifest entry. The
// dropdown only needs the ID and the display name; passing the
// full PiperVoiceMeta would couple the step to the manifest type
// and pull URL/size through transitively.
type voiceView struct {
	ID          string
	DisplayName string
	Language    string
}

// CID:wizard-ttsstep-009 - ttsVoiceFromStateReader
// Purpose: pulls the prior TTSVoice pick out of the StateReader
// via a type assertion to avoid forcing every StateReader
// implementation to add a TTSVoice method (only the wizard's
// *State provides one today). The assertion fails in tests that
// pass a minimal reader; the empty-string fallback keeps the
// picker working with the default first-row preselect.
func ttsVoiceFromStateReader(s StateReader) string {
	type ttvSource interface{ TTSVoiceID() string }
	if src, ok := s.(ttvSource); ok {
		return src.TTSVoiceID()
	}
	return ""
}
