/* Code Map: Language Step
 * - Build: assemble the language picker (ComboBoxText, English first)
 *   and return a *Step with a Capture that commits the chosen code
 *   to the wizard State.
 * - ComboBoxForTest: build a stand-alone ComboBoxText populated from
 *   defaultLanguages. Exposed so tests can verify the default row
 *   without spinning up the full wizard.
 *
 * CID Index:
 * CID:wizard-langstep-001 -> Build
 * CID:wizard-langstep-002 -> ComboBoxForTest
 *
 * Quick lookup: rg -n "CID:wizard-langstep-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

// CID:wizard-langstep-002 - ComboBoxForTest
// Purpose: build a stand-alone ComboBoxText populated from
// defaultLanguages and preselect the row matching selected. Returns
// the combo so tests can call GetActiveText() and GetActive(). The
// combo is not packed into a parent box; the caller is responsible
// for the widget lifecycle in tests.
func ComboBoxForTest(selected string) (*gtk.ComboBoxText, error) {
	combo, err := gtk.ComboBoxTextNew()
	if err != nil {
		return nil, fmt.Errorf("language: combo box: %w", err)
	}
	initial := 0
	for i, lang := range defaultLanguages {
		combo.AppendText(lang.name)
		if lang.code == selected {
			initial = i
		}
	}
	combo.SetActive(initial)
	return combo, nil
}

// CID:wizard-langstep-001 - BuildLanguage
// Purpose: build the language picker step. The ComboBoxText is
// populated from defaultLanguages (English first, rest alphabetical).
// The row whose code matches stateReader.LanguageCode() is preselected
// (defaulting to row 0 / English if no match).
//
// The Capture closure reads the combo's active row and writes the
// matching code into the state via the StateSetter.
//
// Back is set so the user can return to the welcome step.
func BuildLanguage(win *gtk.Window, stateReader StateReader) (*Step, error) {
	combo, err := gtk.ComboBoxTextNew()
	if err != nil {
		return nil, fmt.Errorf("language: combo box: %w", err)
	}

	initialIndex := 0
	selected := ""
	if stateReader != nil {
		selected = stateReader.LanguageCode()
	}
	for i, lang := range defaultLanguages {
		combo.AppendText(lang.name)
		if lang.code == selected {
			initialIndex = i
		}
	}
	combo.SetActive(initialIndex)

	hint, err := gtk.LabelNew(
		"This determines which speech model the app downloads. " +
			"English gets a higher-quality model; other languages use the multilingual model.",
	)
	if err != nil {
		return nil, fmt.Errorf("language: hint label: %w", err)
	}
	hint.SetLineWrap(true)
	hint.SetHAlign(gtk.ALIGN_START)

	content, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
	if err != nil {
		return nil, fmt.Errorf("language: content box: %w", err)
	}
	content.PackStart(hint, false, false, 0)
	content.PackStart(combo, false, false, 0)

	box, back, next, err := newStepContent("Choose your language", content, "Back", "Next")
	if err != nil {
		return nil, fmt.Errorf("language: build content: %w", err)
	}

	capture := func(setter StateSetter) error {
		idx := combo.GetActive()
		if idx < 0 || idx >= len(defaultLanguages) {
			return fmt.Errorf("language: invalid active index %d", idx)
		}
		setter.SetLanguageCode(defaultLanguages[idx].code)
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
