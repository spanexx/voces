/* Code Map: Hotkey Step
 * - Build: assemble the hotkey step (3 preset radios + custom entry)
 *   and return a *Step with a Capture that commits (preset, custom)
 *   to the wizard State.
 * - presetLabels: human-readable labels for the 3 setup.HotkeyPreset
 *   constants. English first; rest matches the IMPL-public-setup §3
 *   ordering.
 *
 * The custom Entry captures a key combination via the
// "key-press-event" signal. The captured string is stored verbatim
// in the Entry; on Next the Capture copies the active radio + the
// Entry text into the State.
 *
 * CID Index:
 * CID:wizard-hotstep-001 -> Build
 * CID:wizard-hotstep-002 -> presetLabels
 *
 * Quick lookup: rg -n "CID:wizard-hotstep-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"

	"voces/internal/setup"
)

// CID:wizard-hotstep-002 - presetLabels
// Purpose: user-facing labels for the 3 setup.HotkeyPreset* constants
// the step shows as radio buttons. The order matches the IMPL list
// (most familiar first). Kept here (not in setup) so the wizard owns
// its own presentation strings.
var presetLabels = []struct {
	preset string
	label  string
}{
	{setup.HotkeyPresetCtrlSpace, "Ctrl + Space (recommended)"},
	{setup.HotkeyPresetRCtrlLeft, "Right Ctrl + Left"},
	{setup.HotkeyPresetF8, "F8"},
	{setup.HotkeyPresetCustom, "Press your own combination"},
}

// CID:wizard-hotstep-001 - BuildHotkey
// Purpose: build the hotkey step. The first three radios are the
// fixed presets; the fourth toggles a custom key Entry. The custom
// Entry is wired to "key-press-event" so any key the user presses
// while focused gets written into the Entry as a string. The Entry
// is also editable so the user can paste a combination if they
// prefer.
//
// The Capture closure reads which radio is active and copies
// (preset, custom) into the State. When the custom radio is active
// but the Entry is empty, Capture returns an error so the runner
// can surface it before the user advances.
//
// Back is set so the user can return to the language step.
func BuildHotkey(win *gtk.Window, stateReader StateReader) (*Step, error) {
	presets := make([]*gtk.RadioButton, 0, len(presetLabels))
	for i, pl := range presetLabels {
		var (
			rb  *gtk.RadioButton
			err error
		)
		if i == 0 {
			rb, err = gtk.RadioButtonNewWithLabel(nil, pl.label)
		} else {
			rb, err = gtk.RadioButtonNewWithLabelFromWidget(presets[0], pl.label)
		}
		if err != nil {
			return nil, fmt.Errorf("hotkey: radio %d: %w", i, err)
		}
		presets = append(presets, rb)
	}

	// Custom Entry — initially hidden, shown when the custom radio
	// becomes active. The key-press-event handler writes the
	// pressed key's name (Escape, Tab, F8, printable runes) into
	// the Entry so the user can see what they pressed.
	customEntry, err := gtk.EntryNew()
	if err != nil {
		return nil, fmt.Errorf("hotkey: custom entry: %w", err)
	}
	customEntry.SetPlaceholderText("Press a key combination...")
	customEntry.SetEditable(true)
	customEntry.SetNoShowAll(true)
	customEntry.Hide()

	// Toggle visibility based on the custom radio.
	customRadio := presets[len(presets)-1]
	refresh := func() {
		if customRadio.GetActive() {
			customEntry.Show()
		} else {
			customEntry.Hide()
		}
	}
	for _, rb := range presets {
		rb.Connect("toggled", func() { refresh() })
	}
	refresh()

	// Capture key presses into the Entry as a string. We use
	// "key-press-event" so the Entry shows the live combination
	// the user is pressing. The returned bool is ignored; the
	// event is allowed to propagate.
	customEntry.Connect("key-press-event", func(_ *gtk.Entry, ev *gdk.Event) bool {
		ek := gdk.EventKeyNewFromEvent(ev)
		if ek == nil {
			return false
		}
		customEntry.SetText(keyvalToString(ek.KeyVal()))
		return true
	})

	// Preselect the current state. Default to ctrl-space if no match.
	initialPreset := setup.HotkeyPresetCtrlSpace
	initialCustom := ""
	if stateReader != nil {
		initialPreset = stateReader.Hotkey()
		initialCustom = stateReader.Custom()
	}
	for i, pl := range presetLabels {
		if pl.preset == initialPreset {
			presets[i].SetActive(true)
		}
	}
	if initialCustom != "" {
		customEntry.SetText(initialCustom)
	}

	hint, err := gtk.LabelNew(
		"This is the key combination that triggers transcription. " +
			"Pick a preset or press your own combination in the field below.",
	)
	if err != nil {
		return nil, fmt.Errorf("hotkey: hint label: %w", err)
	}
	hint.SetLineWrap(true)
	hint.SetHAlign(gtk.ALIGN_START)

	radioBox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	if err != nil {
		return nil, fmt.Errorf("hotkey: radio box: %w", err)
	}
	for _, rb := range presets {
		radioBox.PackStart(rb, false, false, 0)
	}

	content, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	if err != nil {
		return nil, fmt.Errorf("hotkey: content box: %w", err)
	}
	content.PackStart(hint, false, false, 0)
	content.PackStart(radioBox, false, false, 0)
	content.PackStart(customEntry, false, false, 0)

	box, back, next, err := newStepContent("Choose your hotkey", content, "Back", "Next")
	if err != nil {
		return nil, fmt.Errorf("hotkey: build content: %w", err)
	}

	capture := func(setter StateSetter) error {
		var (
			chosenPreset = setup.HotkeyPresetCtrlSpace
			chosenCustom = ""
		)
		for i, rb := range presets {
			if rb.GetActive() {
				chosenPreset = presetLabels[i].preset
				break
			}
		}
		if chosenPreset == setup.HotkeyPresetCustom {
			text, err := customEntry.GetText()
			if err != nil {
				return fmt.Errorf("hotkey: read custom entry: %w", err)
			}
			if text == "" {
				return fmt.Errorf("hotkey: custom combination is empty; press a key or pick a preset")
			}
			chosenCustom = text
		}
		setter.SetHotkey(chosenPreset, chosenCustom)
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
