/* Code Map: Tray UI Phase 7 (update notifier)
 * - addPhase7MenuItems: Adds the dynamic "Update available (vX.Y.Z)"
 *   menu item at the top of the menu. Hidden by default; toggled by
 *   SetUpdateBadge / ClearUpdateBadge.
 * - SetUpdateBadge: Updates the label to "Update available (vX.Y.Z)"
 *   and shows the item.
 * - ClearUpdateBadge: Hides the item. Idempotent.
 *
 * The phase 7 menu item is created once during onReady and shown /
 * hidden thereafter. systray's MenuItem.Hide() / Show() handle the
 * UI; the click goroutine is started in addPhase7MenuItems and stays
 * alive for the lifetime of the tray, discarding clicks while the
 * item is hidden (systray buffers clicks but Show()/Hide() is the
 * correct toggle for visibility).
 *
 * CID Index:
 * CID:tray-ui-phase7-001 -> addPhase7MenuItems
 * CID:tray-ui-phase7-002 -> SetUpdateBadge
 * CID:tray-ui-phase7-003 -> ClearUpdateBadge
 *
 * Quick lookup: rg -n "CID:tray-ui-phase7-" internal/tray/ui_phase7.go
 */
package tray

import (
	"log"

	"voces/internal/updates"

	"github.com/getlantern/systray"
)

// defaultUpdateLabel is the initial label rendered for the
// "Update available" menu item before any SetUpdateBadge call
// arrives. The item is hidden by default so users on the latest
// version never see it.
const defaultUpdateLabel = "Update available"

// CID:tray-ui-phase7-001 - addPhase7MenuItems
// Purpose: Creates the "Update available" menu item at the very top
// of the menu, hides it, and starts the click goroutine. Called from
// onReady before the Record item so the update entry is the first
// thing the user sees when it becomes visible.
//
// nil handlers degrade to a no-op, consistent with the Phase 6
// behaviour — tests that build the tray without wiring the handler
// still pass.
func (m *Manager) addPhase7MenuItems() {
	m.mUpdate = systray.AddMenuItem(defaultUpdateLabel, "A newer release is available — click to install")
	m.mUpdate.Hide()
	go func() {
		for range m.mUpdate.ClickedCh {
			log.Printf("Tray action: Update available")
			if m.handlers.OnApplyUpdate != nil {
				m.handlers.OnApplyUpdate()
			}
		}
	}()
}

// CID:tray-ui-phase7-002 - SetUpdateBadge
// Purpose: Updates the menu item label to "Update available (vX.Y.Z)"
// and shows the item. Safe to call from any goroutine; systray
// operations are expected to be called from the main UI thread but
// in practice (per getlantern/systray docs) Show/Hide/SetTitle are
// safe to call after Run() has started.
//
// If rel is nil, ClearUpdateBadge is called instead.
func (m *Manager) SetUpdateBadge(rel *updates.Release) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mUpdate == nil {
		// onReady has not been called yet. Defer the call by storing
		// the release on the manager — but Manager does not yet have
		// such a field, and the auto-check always runs after Run(),
		// so the only callers without mUpdate are tests. Skip silently.
		return
	}
	if rel == nil {
		m.mUpdate.Hide()
		return
	}
	m.mUpdate.SetTitle("Update available (" + rel.TagName + ")")
	m.mUpdate.Show()
}

// CID:tray-ui-phase7-003 - ClearUpdateBadge
// Purpose: Hides the "Update available" menu item. Idempotent; safe
// to call when the item is already hidden. The label is left
// untouched so re-showing it later (e.g. after a transient API
// failure) reuses the most recent version.
func (m *Manager) ClearUpdateBadge() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mUpdate == nil {
		return
	}
	m.mUpdate.Hide()
}
