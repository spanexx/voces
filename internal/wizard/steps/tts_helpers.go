/* Code Map: TTS step helpers (rc1-hotpatch-29)
 *
 * Pure functions over the piper voice manifest. Split out of
 * tts.go so the file stays under the 250-line size cap enforced
 * by scripts/check-file-size.sh. These functions have no GTK
 * dependency and are unit-testable in isolation.
 *
 *   filterVoicesForLanguage — flatten the manifest's Piper map
 *     into a voiceView slice filtered by language. Insertion
 *     order is preserved (the manifest is a map but we walk it
 *     via sortedKeys to give the dropdown a stable order).
 *   sortedKeys — return the manifest's Piper keys in sorted
 *     order. Used by filterVoicesForLanguage so the dropdown
 *     is the same on every build (no map-iteration surprises).
 *   defaultVoiceForLanguage — pick the voice to pre-select
 *     when the user has no prior choice. Returns the first
 *     voice in the filtered list.
 *
 * CID Index:
 * CID:wizard-ttsstep-004 -> filterVoicesForLanguage
 * CID:wizard-ttsstep-005 -> defaultVoiceForLanguage
 * CID:wizard-ttsstep-010 -> sortedKeys
 *
 * Quick lookup: rg -n "CID:wizard-ttsstep-" internal/wizard/steps/
 */
package steps

import (
	"sort"

	"voces/internal/setup"
)

// piperVoicesDocURL is the public Piper voices catalogue. The
// hint label in the TTS step links here so the user can browse
// the full library and copy a .onnx URL when the curated list
// doesn't have what they want.
const piperVoicesDocURL = "https://github.com/rhasspy/piper/blob/master/VOICES.md"

// CID:wizard-ttsstep-010 - sortedKeys
// Purpose: returns the manifest's Piper voice keys in
// lexicographic order. Used by filterVoicesForLanguage to keep
// the dropdown's order stable across builds. Go map iteration
// is intentionally randomised; a stable order is a UX
// requirement (users notice when "the English voices jump
// around" between sessions).
func sortedKeys(m map[string]setup.PiperVoiceMeta) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// CID:wizard-ttsstep-004 - filterVoicesForLanguage
// Purpose: returns the manifest voices for the chosen language,
// in manifest insertion order (so the dropdown is stable
// across builds). The two-tier fallback chain is:
//   1. Voices whose Language matches the pick.
//   2. English voices (so a user who picks a language we don't
//      ship a curated voice for still sees something useful).
//   3. The first available voice in the manifest (last-ditch
//      fallback when even English is missing).
//
// Returns an empty slice if the manifest has no voices at all
// (caller handles the error).
func filterVoicesForLanguage(m *setup.Manifest, lang string) []voiceView {
	if m == nil || len(m.Piper) == 0 {
		return nil
	}
	keys := sortedKeys(m.Piper)

	// Tier 1: exact language match.
	exact := make([]voiceView, 0, len(keys))
	for _, k := range keys {
		v := m.Piper[k]
		if v.Language == lang {
			exact = append(exact, voiceView{ID: k, DisplayName: v.DisplayName, Language: v.Language})
		}
	}
	if len(exact) > 0 {
		return exact
	}

	// Tier 2: English fallback (handles "de" / "ja" picks when
	// we don't ship a curated voice for that language yet).
	english := make([]voiceView, 0, len(keys))
	for _, k := range keys {
		v := m.Piper[k]
		if v.Language == "en" {
			english = append(english, voiceView{ID: k, DisplayName: v.DisplayName, Language: v.Language})
		}
	}
	if len(english) > 0 {
		return english
	}

	// Tier 3: first available voice (last-ditch when even
	// English is missing). The first sorted key gives us a
	// deterministic choice.
	for _, k := range keys {
		v := m.Piper[k]
		return []voiceView{{ID: k, DisplayName: v.DisplayName, Language: v.Language}}
	}
	return nil
}

// CID:wizard-ttsstep-005 - defaultVoiceForLanguage
// Purpose: returns the voice ID to pre-select when the user
// has no prior pick. Just the first element of
// filterVoicesForLanguage — the tier chain is what makes this
// robust to unknown languages.
func defaultVoiceForLanguage(m *setup.Manifest, lang string) string {
	voices := filterVoicesForLanguage(m, lang)
	if len(voices) == 0 {
		return ""
	}
	return voices[0].ID
}
