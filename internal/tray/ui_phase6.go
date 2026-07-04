/* Code Map: Tray UI Phase 6
 * - addPhase6MenuItems: Adds the 3 new menu items from IMPL §6
 *   (Run setup again..., Check for updates, Open App-managed folder)
 *   after the Settings (Open Config) item. The handlers are optional;
 *   nil handlers degrade to a no-op so a partial wiring in tests does
 *   not break the menu build.
 *
 * CID Index:
 * CID:tray-ui-phase6-001 -> addPhase6MenuItems
 *
 * Quick lookup: rg -n "CID:tray-ui-phase6-" internal/tray/ui_phase6.go
 */
package tray

import (
	"log"

	"github.com/getlantern/systray"
)

// CID:tray-ui-phase6-001 - addPhase6MenuItems
// Purpose: Adds the three new tray menu items from IMPL §6 and wires
// their click handlers. Called from onReady immediately after the
// Settings item, with a separator following.
//
// Phase 6: wizard re-run, update check, and data-dir access. All
// three handlers are optional; nil values silently drop the click
// so the menu item is still visible and clickable in tests that
// don't wire the handlers.
func (m *Manager) addPhase6MenuItems() {
	m.mRunSetup = systray.AddMenuItem("Run setup again...", "Re-open the first-run wizard")
	go func() {
		for range m.mRunSetup.ClickedCh {
			log.Printf("Tray action: Run setup again")
			if m.handlers.OnRunSetup != nil {
				m.handlers.OnRunSetup()
			}
		}
	}()

	m.mCheckUpdates = systray.AddMenuItem("Check for updates", "Check for a newer release")
	go func() {
		for range m.mCheckUpdates.ClickedCh {
			log.Printf("Tray action: Check for updates")
			if m.handlers.OnCheckUpdates != nil {
				m.handlers.OnCheckUpdates()
			}
		}
	}()

	m.mOpenDataDir = systray.AddMenuItem("Open App-managed folder", "Open the XDG data directory in your file manager")
	go func() {
		for range m.mOpenDataDir.ClickedCh {
			log.Printf("Tray action: Open App-managed folder")
			if m.handlers.OnOpenDataDir != nil {
				m.handlers.OnOpenDataDir()
			}
		}
	}()
}
