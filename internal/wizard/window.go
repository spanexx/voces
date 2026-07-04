/* Code Map: Wizard Window
 * - NewWindow: creates the top-level wizard window (480x420, centered)
 *
 * CID Index:
 * CID:wizard-win-001 -> NewWindow
 *
 * Quick lookup: rg -n "CID:wizard-win-" internal/wizard/
 */
package wizard

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

const (
	// windowWidth and windowHeight are the IMPL-public-setup §3 defaults.
	windowWidth  = 480
	windowHeight = 420
	// windowTitle is shown in the title bar. The em-dash is U+2014.
	windowTitle = "Whisper Voice Utility — Setup"
	// borderPadding is the inner margin around the step content.
	borderPadding = 16
)

// CID:wizard-win-001 - NewWindow
// Purpose: build the top-level wizard window. GTK init must already have
// happened (call ensureInit from wizard.go before this). The window is
// not shown; the caller calls ShowAll when the step is attached.
func NewWindow() (*gtk.Window, error) {
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return nil, fmt.Errorf("wizard: window new: %w", err)
	}
	win.SetTitle(windowTitle)
	win.SetDefaultSize(windowWidth, windowHeight)
	win.SetResizable(false)
	win.SetDecorated(true)
	win.SetPosition(gtk.WIN_POS_CENTER)
	// Closing the window with the X button emits "destroy"; the caller
	// wires that up to abort the wizard.
	return win, nil
}
