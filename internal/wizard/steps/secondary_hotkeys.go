/* Code Map: Secondary Hotkeys Step
 * - Build: assemble a 3-row editor for the three remaining
 *   secondary hotkey fields (read_clipboard, toggle_tts,
 *   toggle_transcription). The stop_recording field used to live
 *   here; it moved to the main hotkey step (hotkey_stop.go) so
 *   the user can set up click-to-record (press-to-start, press-
 *   to-stop) right where they pick the record key.
 *   Each row is a label + an Entry that captures a key
 *   combination via "key-press-event" (same capture pattern as
 *   the main hotkey step, so the modifier / F-key rules stay
 *   consistent).
 *
 * The per-row builder lives in secondary_hotkey_row.go so this
 * file stays under the 250-line cap enforced by
 * scripts/check-file-size.sh.
 *
 * rc1-hotpatch-14: the wizard's generatedConfig now writes
 * defaults for the three secondary hotkeys (<f10>, <f11>, <f12>).
 * This step gives the user a chance to customize them instead
 * of hand-editing the YAML later.
 *
 * rc1-hotpatch-21: stop_recording moved to the main hotkey step.
 * The Capture closure still calls SetSecondaryHotkeys with stop=""
 * so the no-op-on-empty contract keeps the State value the user
 * entered on the main step.
 *
 * CID Index:
 * CID:wizard-secondhk-001 -> Build
 *
 * Quick lookup: rg -n "CID:wizard-secondhk-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

// secondaryHotkeyRow is the in-flight state for one row of the
// editor: the entry, the row's container, and a small status
// label for the green/orange capture feedback. The capture
// wiring lives in secondary_hotkey_row.go so this file stays
// small.
type secondaryHotkeyRow struct {
	container *gtk.Box
	entry     *gtk.Entry
	status    *gtk.Label
	// initial is the value the row was constructed with; used
	// by the Reset button to restore the StateReader value
	// instead of an empty string.
	initial string
}

// CID:wizard-secondhk-001 - BuildSecondaryHotkeys
// Purpose: build the 3-row secondary hotkey editor. Each row is
// wired identically to the main hotkey step's custom capture:
// pressing a modifier + key writes the canonical combo into the
// Entry and a green ✓ into the status label; a single printable
// key with no modifier is rejected with an orange warning. The
// "Reset" button next to each row restores the entry to the
// pre-fill value.
//
// The stop_recording field is NOT here — it lives on the main
// hotkey step (hotkey_stop.go). The Capture closure below still
// calls SetSecondaryHotkeys with stop="" so a no-op preserves
// whatever the user picked on the main step.
//
// Back is set so the user can return to the previous step.
func BuildSecondaryHotkeys(win *gtk.Window, stateReader StateReader) (*Step, error) {
	rows := []struct {
		title  string
		hint   string
		reader func(StateReader) string
	}{
		{
			title:  "Read clipboard aloud",
			hint:   "Reads the current clipboard contents through the TTS voice.",
			reader: func(r StateReader) string { return r.ReadClipboardKeyCode() },
		},
		{
			title:  "Toggle TTS on/off",
			hint:   "Enables or disables text-to-speech without leaving the focused field.",
			reader: func(r StateReader) string { return r.ToggleTTSKeyCode() },
		},
		{
			title:  "Toggle transcription on/off",
			hint:   "Pauses or resumes the hotkey without quitting Voces.",
			reader: func(r StateReader) string { return r.ToggleTranscriptionKeyCode() },
		},
	}

	built := make([]*secondaryHotkeyRow, 0, len(rows))
	content, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
	if err != nil {
		return nil, fmt.Errorf("secondaryhk: content box: %w", err)
	}

	intro, err := gtk.LabelNew(
		"Customize the three secondary hotkeys. Each row captures a key combination " +
			"the same way the main hotkey step does. Leave a row empty to use the " +
			"runtime default (<f10> / <f11> / <f12>). The stop-recording key is " +
			"configured on the main hotkey step.",
	)
	if err != nil {
		return nil, fmt.Errorf("secondaryhk: intro: %w", err)
	}
	intro.SetLineWrap(true)
	intro.SetHAlign(gtk.ALIGN_START)
	content.PackStart(intro, false, false, 0)

	for _, rs := range rows {
		row, err := buildSecondaryHotkeyRow(rs.title, rs.hint, rs.reader, stateReader)
		if err != nil {
			return nil, err
		}
		built = append(built, row)
		content.PackStart(row.container, false, false, 0)
	}

	box, back, next, err := newStepContent("Secondary hotkeys", content, "Back", "Next")
	if err != nil {
		return nil, fmt.Errorf("secondaryhk: build content: %w", err)
	}

	capture := func(setter StateSetter) error {
		read, err := built[0].entry.GetText()
		if err != nil {
			return fmt.Errorf("secondaryhk: read clipboard: %w", err)
		}
		toggleTTS, err := built[1].entry.GetText()
		if err != nil {
			return fmt.Errorf("secondaryhk: read toggle_tts: %w", err)
		}
		toggleTr, err := built[2].entry.GetText()
		if err != nil {
			return fmt.Errorf("secondaryhk: read toggle_transcription: %w", err)
		}
		// stop_recording is configured on the main hotkey step.
		// Pass "" so the no-op-on-empty contract in
		// SetSecondaryHotkeys leaves whatever the user picked
		// there untouched. The three runtime defaults
		// (<f10> / <f11> / <f12>) are filled in by
		// setup.defaultConfigFor when a row is left empty.
		setter.SetSecondaryHotkeys("", read, toggleTTS, toggleTr)
		return nil
	}

	// Note: do not win.Add(box) here. The runner is the single source
	// of truth for attaching the step box to the window (wizard.go
	// showStepAt). Double-adding raises:
	//   "Attempting to add a widget with type GtkBox to a container of
	//    type GtkWindow, but the widget is already inside a container
	//    of type GtkWindow"
	return &Step{Box: box, Next: next, Back: back, Capture: capture}, nil
}
