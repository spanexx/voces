/* Code Map: Secondary Hotkey Row Builder
 * - buildSecondaryHotkeyRow: builds one row of the 4-row
 *   secondary hotkey editor. Splits out from secondary_hotkeys.go
 *   so the Build function stays readable.
 *
 * The capture wiring matches the main hotkey step's custom
 * entry: modifier+key writes the combo, single printable key
 * is rejected with an orange warning, F1-F12 alone is
 * accepted. The Reset button restores the entry to the
 * StateReader pre-fill value.
 *
 * CID Index:
 * CID:wizard-secondhk-002 -> buildSecondaryHotkeyRow
 *
 * Quick lookup: rg -n "CID:wizard-secondhk-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

// CID:wizard-secondhk-002 - buildSecondaryHotkeyRow
// Purpose: build a single label + entry + status + reset-button
// row. The capture wiring matches the main hotkey step's
// custom entry.
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

	// Wire the capture: same rules as the main hotkey step.
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
				`<span foreground="orange">⚠ "%s" is a single printable key. `+
					`Hold a modifier or use an F-key (F1-F12).</span>`,
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
