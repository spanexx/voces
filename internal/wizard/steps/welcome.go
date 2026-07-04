/* Code Map: Welcome Step
 * - Build: assemble the welcome step (title + body + Next button)
 *   and return a *Step the wizard runner can attach + wire up.
 *
 * CID Index:
 * CID:wizard-welcome-001 -> Build
 *
 * Quick lookup: rg -n "CID:wizard-welcome-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

// welcomeText is the body copy shown under the title. Plain prose; no
// markup so it survives copy/paste without surprises.
const welcomeText = "This setup will download the speech recognition model and verify " +
	"that all required system components (clipboard, auto-typer, tray icon, " +
	"hotkey) are installed.\n\n" +
	"You can cancel at any time by closing this window. Re-run the setup " +
	"any time with: voces --setup."

// CID:wizard-welcome-001 - BuildWelcome
// Purpose: assemble the welcome step (title + body + Next button) and
// return a *Step the wizard runner attaches to the window. Back is
// nil because the welcome step is always first; Capture is nil because
// the welcome step does not write to wizard.State.
//
// Layout:
//   - Title: "Welcome to Voces"
//   - Body: welcomeText (wrapped)
//   - Footer: "Voces v<version>"
//   - Buttons: Next only ("Get started")
func BuildWelcome(win *gtk.Window, version string) (*Step, error) {
	body, err := gtk.LabelNew(welcomeText)
	if err != nil {
		return nil, fmt.Errorf("welcome: body label: %w", err)
	}
	body.SetLineWrap(true)
	body.SetHAlign(gtk.ALIGN_START)

	content, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	if err != nil {
		return nil, fmt.Errorf("welcome: content box: %w", err)
	}
	content.PackStart(body, true, true, 0)

	footer, err := gtk.LabelNew(fmt.Sprintf("Voces v%s", version))
	if err != nil {
		return nil, fmt.Errorf("welcome: footer label: %w", err)
	}
	footer.SetHAlign(gtk.ALIGN_END)
	content.PackStart(footer, false, false, 0)

	box, _, next, err := newStepContent("Welcome to Voces", content, "", "Get started")
	if err != nil {
		return nil, fmt.Errorf("welcome: build content: %w", err)
	}

	// Note: do not win.Add(box) here. The runner is the single source
	// of truth for attaching the step box to the window (wizard.go
	// showStepAt). Double-adding raises:
	//   "Attempting to add a widget with type GtkBox to a container of
	//    type GtkWindow, but the widget is already inside a container
	//    of type GtkWindow"
	return &Step{Box: box, Next: next, Back: nil, Capture: nil}, nil
}
