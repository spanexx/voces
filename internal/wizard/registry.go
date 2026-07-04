/* Code Map: Wizard Step Registry
 * - stepKey: identifies a wizard step in the registry.
 * - stepRenderer: the function signature every BuildX matches.
 * - showError: inline error dialog helper used by RunFull.
 *
 * Split out from wizard.go so the main file stays under the
 * 250-line size cap enforced by scripts/check-file-size.sh.
 *
 * CID Index:
 * CID:wizard-reg-001 -> stepKey
 * CID:wizard-reg-002 -> stepRenderer
 * CID:wizard-reg-003 -> showError
 *
 * Quick lookup: rg -n "CID:wizard-reg-" internal/wizard/
 */
package wizard

import (
	"github.com/gotk3/gotk3/gtk"

	"voces/internal/wizard/steps"
)

// CID:wizard-reg-001 - stepKey
// Purpose: identifies a wizard step in the registry. The TTS key is
// included in every chain; the chain builder omits it from the
// ordered list when the language is English.
type stepKey int

const (
	stepWelcome stepKey = iota
	stepLanguage
	stepHotkey
	stepTTS
	stepFinish
)

// CID:wizard-reg-002 - stepRenderer
// Purpose: materialises a step in the window. Each step's
// BuildX function has this signature; we alias it here so the
// runner can iterate a registry.
type stepRenderer func(win *gtk.Window, state *State) (*steps.Step, error)

// CID:wizard-reg-003 - showError
// Purpose: displays a non-fatal error to the user via a GTK
// message dialog. The dialog is modal to the wizard window. Kept
// inline because it is a single-call UX helper, not a public API.
func showError(parent *gtk.Window, err error) {
	dialog := gtk.MessageDialogNew(
		parent,
		gtk.DIALOG_MODAL,
		gtk.MESSAGE_ERROR,
		gtk.BUTTONS_OK,
		"%s",
		err.Error(),
	)
	if dialog == nil {
		return
	}
	dialog.Run()
	dialog.Destroy()
}
