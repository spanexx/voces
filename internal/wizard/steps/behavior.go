/* Code Map: Behavior Step
 * - Build: assemble the autostart yes/no radios and return a *Step
 *   with a Capture that commits the boolean to the wizard State.
 *   Wired into config.Behavior.Autostart by defaultConfigFor.
 *
 * CID Index:
 * CID:wizard-behavstep-001 -> Build
 *
 * Quick lookup: rg -n "CID:wizard-behavstep-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

// CID:wizard-behavstep-001 - BuildBehavior
// Purpose: build the behavior step. Two radio buttons: "Yes, start
// Voces when I log in" and "No, only when I launch it". Default is
// "No" so users who do not want autostart never get it. The
// "Yes" radio marks Autostart=true on the State; defaultConfigFor
// reads that flag and writes behavior.autostart: true.
//
// Back is set so the user can return to the previous step.
func BuildBehavior(win *gtk.Window, stateReader StateReader) (*Step, error) {
	yesBtn, err := gtk.RadioButtonNewWithLabel(nil, "Yes, start Voces when I log in")
	if err != nil {
		return nil, fmt.Errorf("behavior: yes radio: %w", err)
	}
	noBtn, err := gtk.RadioButtonNewWithLabelFromWidget(yesBtn, "No, only when I launch it")
	if err != nil {
		return nil, fmt.Errorf("behavior: no radio: %w", err)
	}
	// Default: No. (Matches the createDefaultConfig default of
	// autostart=false, so the wizard never surprises a user with
	// a daemon they did not opt into.)
	noBtn.SetActive(true)
	if stateReader != nil && stateReader.AutostartDesired() {
		yesBtn.SetActive(true)
	}

	hint, err := gtk.LabelNew(
		"Autostart adds a small entry to your login items so the tray icon " +
			"and hotkey are ready when you reach the desktop. You can change " +
			"this later from the tray menu or by re-running the setup.",
	)
	if err != nil {
		return nil, fmt.Errorf("behavior: hint label: %w", err)
	}
	hint.SetLineWrap(true)
	hint.SetHAlign(gtk.ALIGN_START)

	radioBox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	if err != nil {
		return nil, fmt.Errorf("behavior: radio box: %w", err)
	}
	radioBox.PackStart(yesBtn, false, false, 0)
	radioBox.PackStart(noBtn, false, false, 0)

	content, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	if err != nil {
		return nil, fmt.Errorf("behavior: content box: %w", err)
	}
	content.PackStart(hint, false, false, 0)
	content.PackStart(radioBox, false, false, 0)

	box, back, next, err := newStepContent("Start Voces on login?", content, "Back", "Next")
	if err != nil {
		return nil, fmt.Errorf("behavior: build content: %w", err)
	}

	capture := func(setter StateSetter) error {
		setter.SetAutostart(yesBtn.GetActive())
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
