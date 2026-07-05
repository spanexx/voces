/* Code Map: Secondary Hotkeys Step
 * - Build: assemble a 4-row editor for the four secondary hotkey
 *   fields (stop_recording, read_clipboard, toggle_tts,
 *   toggle_transcription). Each row is a label + an Entry that
 *   captures a key combination via "key-press-event" (same capture
 *   pattern as the main hotkey step, so the modifier / F-key rules
 *   stay consistent).
 *
 * The per-row builder lives in secondary_hotkey_row.go so this
 * file stays under the 250-line cap enforced by
 * scripts/check-file-size.sh.
 *
 * rc1-hotpatch-14: the wizard's generatedConfig now writes
 * defaults for the four secondary hotkeys (<f10>, <f11>, <f12>,
 * '' for stop_recording). This step gives the user a chance to
 * customize them instead of hand-editing the YAML later.
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
// Purpose: build the 4-row secondary hotkey editor. Each row is
// wired identically to the main hotkey step's custom capture:
// pressing a modifier + key writes the canonical combo into the
// Entry and a green ✓ into the status label; a single printable
// key with no modifier is rejected with an orange warning. The
// "Reset" button next to each row restores the entry to the
// pre-fill value.
//
// Back is set so the user can return to the previous step.
func BuildSecondaryHotkeys(win *gtk.Window, stateReader StateReader) (*Step, error) {
	rows := []struct {
		title  string
		hint   string
		reader func(StateReader) string
	}{
		{
			title: "Stop recording",
			hint:  "Optional. Hold the record key to stop, or pick a separate key.",
			reader: func(r StateReader) string { return r.StopRecordingKeyCode() },
		},
		{
			title: "Read clipboard aloud",
			hint:  "Reads the current clipboard contents through the TTS voice.",
			reader: func(r StateReader) string { return r.ReadClipboardKeyCode() },
		},
		{
			title: "Toggle TTS on/off",
			hint:  "Enables or disables text-to-speech without leaving the focused field.",
			reader: func(r StateReader) string { return r.ToggleTTSKeyCode() },
		},
		{
			title: "Toggle transcription on/off",
			hint:  "Pauses or resumes the hotkey without quitting Voces.",
			reader: func(r StateReader) string { return r.ToggleTranscriptionKeyCode() },
		},
	}

	built := make([]*secondaryHotkeyRow, 0, len(rows))
	content, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
	if err != nil {
		return nil, fmt.Errorf("secondaryhk: content box: %w", err)
	}

	intro, err := gtk.LabelNew(
		"Customize the four secondary hotkeys. Each row captures a key combination " +
			"the same way the main hotkey step does. Leave a row empty to use the " +
			"runtime default (<f10> / <f11> / <f12>).",
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
		stop, err := built[0].entry.GetText()
		if err != nil {
			return fmt.Errorf("secondaryhk: read stop: %w", err)
		}
		read, err := built[1].entry.GetText()
		if err != nil {
			return fmt.Errorf("secondaryhk: read clipboard: %w", err)
		}
		toggleTTS, err := built[2].entry.GetText()
		if err != nil {
			return fmt.Errorf("secondaryhk: read toggle_tts: %w", err)
		}
		toggleTr, err := built[3].entry.GetText()
		if err != nil {
			return fmt.Errorf("secondaryhk: read toggle_transcription: %w", err)
		}
		// An empty field means "use the runtime default" — the
		// setup.defaultConfigFor already fills in <f10>/<f11>/<f12>;
		// stop_recording stays empty (intentional, the hold-binding
		// model has no separate stop key).
		setter.SetSecondaryHotkeys(stop, read, toggleTTS, toggleTr)
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
