/* Code Map: Tray Manager
 * - Manager: Orchestrates the system tray lifecycle
 * - New: Factory for creating the manager
 * - Run/Quit: Control methods for the tray process
 * - SetState: Updates icons and tooltips dynamically
 *
 * CID Index:
 * CID:tray-manager-001 -> Manager
 * CID:tray-manager-002 -> New
 * CID:tray-manager-003 -> SetState
 *
 * Quick lookup: rg -n "CID:tray-manager-" internal/tray/manager.go
 */
package tray

import (
	"sync"

	"whisper-voice-util/internal/config"

	"github.com/getlantern/systray"
)

// CID:tray-manager-001 - Manager
// Purpose: Central controller for the system tray UI and menu items.
// Uses: ActionHandlers, config.Config, State
type Manager struct {
	cfg      *config.Config
	handlers ActionHandlers
	mu       sync.Mutex
	state    State

	// Menu items
	mRecord       *systray.MenuItem
	mRead         *systray.MenuItem
	mEnginesTrans map[string]*systray.MenuItem
	mEnginesTTS   map[string]*systray.MenuItem
	mQuit         *systray.MenuItem
	mSettings     *systray.MenuItem
	mLogs         *systray.MenuItem
	mRunSetup     *systray.MenuItem
	mCheckUpdates *systray.MenuItem
	mOpenDataDir  *systray.MenuItem
	mUpdate       *systray.MenuItem // Phase 7: hidden by default; shown when an update is available
}

// CID:tray-manager-002 - New
// Purpose: Initializes a new tray manager with provided handlers.
func New(cfg *config.Config, handlers ActionHandlers) *Manager {
	return &Manager{
		cfg:           cfg,
		handlers:      handlers,
		state:         StateIdle,
		mEnginesTrans: make(map[string]*systray.MenuItem),
		mEnginesTTS:   make(map[string]*systray.MenuItem),
	}
}

// GetState returns the current tray state.
func (m *Manager) GetState() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// Run starts the system tray execution. This function blocks until Quit is called.
func (m *Manager) Run() {
	systray.Run(m.onReady, m.onExit)
}

// Quit requests the application to exit and the tray to shut down.
func (m *Manager) Quit() {
	systray.Quit()
}

// CID:tray-manager-003 - SetState
// Purpose: Dynamically updates the tray icon and tooltips to reflect app status.
func (m *Manager) SetState(state State, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state = state
	tooltip := "Whisper Voice Utility - " + state.String()
	if message != "" {
		tooltip += ": " + message
	}

	systray.SetTooltip(tooltip)

	switch state {
	case StateIdle:
		systray.SetIcon(IconIdle)
	case StateRecording:
		systray.SetIcon(IconRecording)
	case StateProcessing:
		systray.SetIcon(IconProcessing)
	case StateError:
		systray.SetIcon(IconError)
	case StateDisabled:
		systray.SetIcon(IconDisabled)
	}

	if m.mRecord != nil {
		if state == StateRecording || state == StateProcessing {
			m.mRecord.Disable()
		} else {
			m.mRecord.Enable()
		}
	}
}
