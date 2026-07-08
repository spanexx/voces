/* Code Map: Hotkey Step - Stop Recording Row
 * - buildStopRecordingRow: thin wrapper around buildSecondaryHotkeyRow
 *   that returns the *gtk.Entry so the main hotkey step's Capture
 *   closure can read whatever the user picked. The container is
 *   returned separately so BuildHotkey can pack it into its content
 *   box without knowing the row's internal layout.
 * - wireHotkeyEntry: shared key-press-event wiring used by both the
 *   main hotkey step's custom entry AND the stop-recording row. The
 *   inline copy that used to live in hotkey.go was duplicated almost
 *   verbatim in secondary_hotkey_row.go; extracting it here keeps
 *   hotkey.go under the 250-line cap and keeps the validation rules
 *   in a single place.
 *
 * Lives in its own file so internal/wizard/steps/hotkey.go stays
 * under the 250-line file-size cap enforced by
 * scripts/check-file-size.sh.
 *
 * CID Index:
 * CID:wizard-hotstep-003 -> buildStopRecordingRow
 * CID:wizard-hotstep-004 -> wireHotkeyEntry
 *
 * Quick lookup: rg -n "CID:wizard-hotstep-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

// CID:wizard-hotstep-004 - wireHotkeyEntry
// Purpose: attach the shared key-press handler to an Entry so a key
// combination typed while the Entry has focus is captured into the
// Entry as a canonical combo string ("ctrl+shift+f9", "f8",
// "space"). The status label receives green/orange feedback:
// green ✓ on a valid combo, orange ⚠ on a rejected weak combo
// (single printable key with no modifier).
//
// Modifier-only keypresses are ignored (the in-progress combo is
// preserved and the user is hinted to add a key). Single printable
// keys with no modifier are rejected because binding one would
// intercept that character everywhere.
//
// Returns true from the handler to swallow the keypress so the
// captured combo never reaches the Entry's text buffer mid-edit.
func wireHotkeyEntry(entry *gtk.Entry, status *gtk.Label) {
	entry.Connect("key-press-event", func(_ *gtk.Entry, ev *gdk.Event) bool {
		ek := gdk.EventKeyNewFromEvent(ev)
		if ek == nil {
			return false
		}
		state := ek.State()
		keyval := ek.KeyVal()

		if IsModifierKeyval(keyval) {
			status.SetMarkup(`<i>hold the modifier and press a key (F1-F12, letter, etc.)</i>`)
			return true
		}
		if !HasModifier(state) && !IsValidAloneKeyval(keyval) {
			weak := BuildCombo(0, keyval)
			status.SetMarkup(fmt.Sprintf(
				`<span foreground="orange">⚠ "%s" is a single printable key and would conflict with typing. `+
					`Hold a modifier (Ctrl/Alt/Super) or use an F-key (F1-F12).</span>`,
				weak,
			))
			return true
		}
		combo := BuildCombo(state, keyval)
		if combo == "" {
			return true
		}
		entry.SetText(combo)
		status.SetMarkup(fmt.Sprintf(`<span foreground="green">✓ %s</span>`, combo))
		return true
	})
}

// CID:wizard-hotstep-003 - buildStopRecordingRow
// Purpose: build the optional "Stop recording" row that lives on the
// main hotkey step. The row is a labelled Entry + Reset button +
// capture feedback status (delegated to buildSecondaryHotkeyRow). If
// the user enters a key combination it is committed to
// State.StopRecordingKey via SetSecondaryHotkeys; an empty entry
// means the record key stays as a hold-to-talk binding.
//
// Returns the Entry so the caller can read it in the Capture closure
// and the container so the caller can pack it.
func buildStopRecordingRow(stateReader StateReader) (*gtk.Entry, *gtk.Box, error) {
	row, err := buildSecondaryHotkeyRow(
		"Stop recording (optional)",
		"Pick a separate key to stop recording. "+
			"Leave empty to keep the hold-to-talk binding (release the record key to stop).",
		func(r StateReader) string { return r.StopRecordingKeyCode() },
		stateReader,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("hotkey stop row: %w", err)
	}
	return row.entry, row.container, nil
}
