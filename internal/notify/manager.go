/* Code Map: Notification Manager
 * - Manager: Thread-safe queue for D-Bus alerts
 * - New: Factory for creating the manager
 * - Start: Connects to D-Bus and runs the worker
 * - Stop: Graceful shutdown of the queue
 *
 * CID Index:
 * CID:notify-manager-001 -> Manager
 * CID:notify-manager-002 -> New
 * CID:notify-manager-003 -> Start
 *
 * Quick lookup: rg -n "CID:notify-manager-" internal/notify/manager.go
 */
package notify

import (
	"log"
	"sync"
	"time"

	"whisper-voice-util/internal/config"
	"whisper-voice-util/internal/input"

	"github.com/esiqveland/notify"
	"github.com/godbus/dbus/v5"
)

// CID:notify-manager-001 - Manager
// Purpose: Orchestrates the delivery of notifications via the XDG Notification spec.
type Manager struct {
	cfg       *config.Config
	dbusConn  *dbus.Conn
	notifier  notify.Notifier
	queue     chan Message
	cancel    chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	running   bool
	clipboard *input.Clipboard

	// Debouncing to prevent spam
	lastNotify time.Time
}

// CID:notify-manager-002 - New
// Purpose: Initializes the manager with a buffered delivery channel.
func New(cfg *config.Config) *Manager {
	return &Manager{
		cfg:       cfg,
		queue:     make(chan Message, 10), // buffer up to 10 notifications
		cancel:    make(chan struct{}),
		clipboard: input.NewClipboard(),
	}
}

// CID:notify-manager-003 - Start
// Purpose: Establishes a connection to the system session bus and spawns the listener.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}
	if !m.cfg.Behavior.Notifications {
		log.Println("notify: system disabled in config")
	}

	if m.notifier == nil {
		conn, err := dbus.SessionBus()
		if err != nil {
			return err
		}
		m.dbusConn = conn

		notif, err := notify.New(conn)
		if err != nil {
			return err
		}
		m.notifier = notif
	}

	m.running = true
	m.cancel = make(chan struct{})

	m.wg.Add(1)
	go m.processQueue()

	log.Println("notify: manager started")
	return nil
}

// Stop shuts down the notification queue and closes connections.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.cancel)
	m.wg.Wait()

	if m.notifier != nil {
		m.notifier.Close()
	}

	m.running = false
	log.Println("notify: manager stopped")
}

// Send queues a notification. It's safe to call concurrently.
func (m *Manager) Send(msg Message) {
	if !m.cfg.Behavior.Notifications {
		return // Silently drop if globally disabled
	}

	select {
	case m.queue <- msg:
	default:
		log.Println("notify: queue full, dropping notification")
	}
}

// processQueue reads from the channel and sends via D-Bus.
func (m *Manager) processQueue() {
	defer m.wg.Done()

	for {
		select {
		case <-m.cancel:
			return
		case msg := <-m.queue:
			m.dispatch(msg)
		}
	}
}

// dispatch sends a single notification.
func (m *Manager) dispatch(msg Message) {
	m.mu.Lock()
	if msg.Urgency != notify.UrgencyCritical && time.Since(m.lastNotify) < time.Second {
		m.mu.Unlock()
		return
	}
	m.lastNotify = time.Now()
	m.mu.Unlock()

	iconName := "dialog-information"
	switch msg.Type {
	case TypeSuccess:
		iconName = "emblem-default"
	case TypeWarning:
		iconName = "dialog-warning"
	case TypeError:
		iconName = "dialog-error"
	}

	n := notify.Notification{
		AppName:       "Whisper Voice Utility",
		Summary:       msg.Title,
		Body:          msg.Body,
		AppIcon:       iconName,
		ExpireTimeout: 5 * time.Second, // 5 seconds default
	}
	// Fixing the Summary typo I almost made: original code used msg.Title
	n.Summary = msg.Title

	n.SetUrgency(msg.Urgency)

	if msg.Urgency == notify.UrgencyCritical {
		n.ExpireTimeout = 10 * time.Second // 10 seconds for critical
	}

	if m.notifier == nil {
		return
	}

	_, err := m.notifier.SendNotification(n)
	if err != nil {
		log.Printf("notify: failed to send: %v", err)
	}
}
