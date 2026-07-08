/* Code Map: Secondary Hotkey Row Builder
 * - buildSecondaryHotkeyRow: builds one row of the 3-row
 *   secondary hotkey editor. Splits out from secondary_hotkeys.go
 *   so the Build function stays readable.
 *
 * The capture wiring delegates to the shared wireHotkeyEntry
 * helper in hotkey_stop.go so the modifier + F-key validation
 * rules stay in a single place — the main hotkey step's custom
 * entry uses the same handler.
 *
 * CID Index:
 * CID:wizard-secondhk-002 -> buildSecondaryHotkeyRow
 *
 * Quick lookup: rg -n "CID:wizard-secondhk-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

// CID:wizard-secondhk-002 - buildSecondaryHotkeyRow
// Purpose: build a single label + entry + status + reset-button
// row. The capture wiring is delegated to wireHotkeyEntry (in
// hotkey_stop.go) so the validation rules stay in one place.
//
// If stateReader is non-nil, the entry is pre-filled with the
// stored value. The Reset button restores the entry to the
// pre-fill value (so the user can undo a capture they did not
// mean to commit), and the Capture closure in
// secondary_hotkeys.go reads whatever the user left in the
// entry.
func buildSecondaryHotkeyRow(title, hint string, reader func(StateReader) string, stateReader StateReader) (*secondaryHotkeyRow, error) {
	container, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	if err != nil {
		return nil, fmt.Errorf("secondaryhk: row box %q: %w", title, err)
	}
	container.SetMarginBottom(4)

	titleLabel, err := gtk.LabelNew("")
	if err != nil {
		return nil, fmt.Errorf("secondaryhk: title %q: %w", title, err)
	}
	titleLabel.SetMarkup(fmt.Sprintf("<b>%s</b>", title))
	titleLabel.SetHAlign(gtk.ALIGN_START)
	container.PackStart(titleLabel, false, false, 0)

	hintLabel, err := gtk.LabelNew(hint)
	if err != nil {
		return nil, fmt.Errorf("secondaryhk: hint %q: %w", title, err)
	}
	hintLabel.SetLineWrap(true)
	hintLabel.SetHAlign(gtk.ALIGN_START)
	hintLabel.SetMarginStart(2)
	container.PackStart(hintLabel, false, false, 0)

	entry, err := gtk.EntryNew()
	if err != nil {
		return nil, fmt.Errorf("secondaryhk: entry %q: %w", title, err)
	}
	entry.SetPlaceholderText("Press a key combination (e.g. <f10>, Ctrl+Shift+R)")
	entry.SetEditable(true)
	entry.SetMarginTop(4)

	status, err := gtk.LabelNew("")
	if err != nil {
		return nil, fmt.Errorf("secondaryhk: status %q: %w", title, err)
	}
	status.SetHAlign(gtk.ALIGN_START)
	status.SetLineWrap(true)
	status.SetMarginStart(2)
	status.SetMarginTop(2)
	status.SetMarginBottom(4)

	clearBtn, err := gtk.ButtonNewWithLabel("Reset")
	if err != nil {
		return nil, fmt.Errorf("secondaryhk: reset %q: %w", title, err)
	}
	clearBtn.SetMarginStart(8)
	clearBtn.SetMarginTop(4)

	// Initialize from the state. An empty stored value leaves the
	// entry blank so the hint text inside the entry shows.
	initial := ""
	if stateReader != nil && reader != nil {
		initial = reader(stateReader)
	}
	if initial != "" {
		entry.SetText(initial)
	}
	row := &secondaryHotkeyRow{
		container: container,
		entry:     entry,
		status:    status,
		initial:   initial,
	}

	// Shared key-press capture: same rules as the main hotkey
	// step. Lives in hotkey_stop.go so the validation (modifier
	// handling, F-key acceptance, weak-combo rejection) is in a
	// single place.
	wireHotkeyEntry(entry, status)
	// Select-all on focus so a new keypress overwrites the field.
	entry.Connect("focus-in-event", func() {
		entry.SelectRegion(0, -1)
	})
	// Reset button empties the entry and restores the pre-fill.
	clearBtn.Connect("clicked", func() {
		entry.SetText(row.initial)
		status.SetText("")
	})

	entryRow, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, fmt.Errorf("secondaryhk: entry row %q: %w", title, err)
	}
	entryRow.PackStart(entry, true, true, 0)
	entryRow.PackStart(clearBtn, false, false, 0)
	container.PackStart(entryRow, false, false, 0)
	container.PackStart(status, false, false, 0)
	return row, nil
}
