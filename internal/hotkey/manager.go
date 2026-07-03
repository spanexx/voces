/* Code Map: Hotkey Manager
 * - Manager: Orchestrates global keyboard listeners
 * - NewManager: Factory for creating the manager
 * - Start: Spawns listener routines for configured keys
 *
 * CID Index:
 * CID:hotkey-manager-001 -> Manager
 * CID:hotkey-manager-002 -> NewManager
 * CID:hotkey-manager-003 -> Start
 *
 * Quick lookup: rg -n "CID:hotkey-manager-" internal/hotkey/manager.go
 */
package hotkey

import (
	"context"
	"fmt"
	"log"
	"sync"

	"whisper-voice-util/internal/config"
)

// CID:hotkey-manager-001 - Manager
// Purpose: Central controller for starting and stopping background key polling/listeners.
type Manager struct {
	cfg      *config.Config
	handlers ActionHandlers
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// CID:hotkey-manager-002 - NewManager
// Purpose: Initializes the manager with user-defined handlers.
func NewManager(cfg *config.Config, handlers ActionHandlers) *Manager {
	return &Manager{
		cfg:      cfg,
		handlers: handlers,
	}
}

// CID:hotkey-manager-003 - Start
// Purpose: Registers and spawns goroutines for each configured hotkey binding.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("hotkey manager already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.running = true

	if err := startX11KeyTracker(ctx); err != nil {
		m.running = false
		m.cancel()
		return err
	}

	logKeysymDiagnostics("record_and_type", m.cfg.Hotkeys.RecordAndType)
	logKeysymDiagnostics("stop_recording", m.cfg.Hotkeys.StopRecording)
	logKeysymDiagnostics("read_clipboard", m.cfg.Hotkeys.ReadClipboard)
	logKeysymDiagnostics("toggle_tts", m.cfg.Hotkeys.ToggleTTS)
	logKeysymDiagnostics("toggle_transcription", m.cfg.Hotkeys.ToggleTranscription)

	// Hold-mode: Record & Type
	if m.cfg.Hotkeys.StopRecording != "" {
		if m.cfg.Hotkeys.RecordAndType != "" {
			pressStart := NewPressBinding(m.cfg.Hotkeys.RecordAndType, m.handlers.OnRecordStart)
			m.wg.Add(1)
			go pressStart.run(ctx, &m.wg)
			log.Printf("hotkey: registered press binding: %s", m.cfg.Hotkeys.RecordAndType)
		}
		pressStop := NewPressBinding(m.cfg.Hotkeys.StopRecording, m.handlers.OnRecordStop)
		m.wg.Add(1)
		go pressStop.run(ctx, &m.wg)
		log.Printf("hotkey: registered press binding: %s", m.cfg.Hotkeys.StopRecording)
	} else if m.cfg.Hotkeys.RecordAndType != "" {
		hold := NewHoldBinding(
			m.cfg.Hotkeys.RecordAndType,
			m.handlers.OnRecordStart,
			m.handlers.OnRecordStop,
		)
		m.wg.Add(1)
		go hold.run(ctx, &m.wg)
		log.Printf("hotkey: registered hold binding: %s", m.cfg.Hotkeys.RecordAndType)
	}

	// Press-mode: Read Clipboard (F10)
	if m.cfg.Hotkeys.ReadClipboard != "" {
		press := NewPressBinding(m.cfg.Hotkeys.ReadClipboard, m.handlers.OnReadClipboard)
		m.wg.Add(1)
		go press.run(ctx, &m.wg)
		log.Printf("hotkey: registered press binding: %s", m.cfg.Hotkeys.ReadClipboard)
	}

	// Press-mode: Toggle TTS (F11)
	if m.cfg.Hotkeys.ToggleTTS != "" {
		press := NewPressBinding(m.cfg.Hotkeys.ToggleTTS, m.handlers.OnToggleTTS)
		m.wg.Add(1)
		go press.run(ctx, &m.wg)
		log.Printf("hotkey: registered press binding: %s", m.cfg.Hotkeys.ToggleTTS)
	}

	// Press-mode: Toggle Transcription (F12)
	if m.cfg.Hotkeys.ToggleTranscription != "" {
		press := NewPressBinding(m.cfg.Hotkeys.ToggleTranscription, m.handlers.OnToggleTranscription)
		m.wg.Add(1)
		go press.run(ctx, &m.wg)
		log.Printf("hotkey: registered press binding: %s", m.cfg.Hotkeys.ToggleTranscription)
	}

	return nil
}

// Stop unregisters all hotkeys and stops listening.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.cancel()
	m.wg.Wait()
	m.running = false
	log.Println("hotkey: all bindings stopped")
}

// IsRunning returns whether the manager is actively listening.
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}
