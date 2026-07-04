package notify

import (
	"sync"
	"testing"
	"time"

	"voces/internal/config"

	"github.com/esiqveland/notify"
)

// LocalNotifier implements notify.Notifier for logic testing without a desktop bus.
type LocalNotifier struct {
	mu            sync.Mutex
	notifications []notify.Notification
	closedIDs     []uint32
}

func (t *LocalNotifier) SendNotification(n notify.Notification) (uint32, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.notifications = append(t.notifications, n)
	return uint32(len(t.notifications)), nil
}

func (t *LocalNotifier) GetCapabilities() ([]string, error) { return nil, nil }
func (t *LocalNotifier) GetServerInformation() (notify.ServerInformation, error) {
	return notify.ServerInformation{}, nil
}
func (t *LocalNotifier) CloseNotification(id uint32) (bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closedIDs = append(t.closedIDs, id)
	return true, nil
}
func (t *LocalNotifier) Close() error { return nil }

func TestManager_QueueAndDebounce(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.Notifications = true

	m := New(cfg)
	tn := &LocalNotifier{}
	m.notifier = tn

	if err := m.Start(); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer m.Stop()

	// 1. Test basic sending
	m.Info("Test Title", "Test Body")

	// Wait for queue processing (very small buffer/delay)
	time.Sleep(50 * time.Millisecond)

	tn.mu.Lock()
	if len(tn.notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(tn.notifications))
	} else {
		if tn.notifications[0].Summary != "Test Title" {
			t.Errorf("Expected summary 'Test Title', got %q", tn.notifications[0].Summary)
		}
	}
	tn.mu.Unlock()

	// 2. Test debouncing (these should be dropped)
	m.Info("Spam 1", "Body")
	m.Info("Spam 2", "Body")

	time.Sleep(50 * time.Millisecond)

	tn.mu.Lock()
	// Only 1 notification total (still only the first one, spams dropped by 1s debounce)
	if len(tn.notifications) != 1 {
		t.Errorf("Expected 1 notification total (spams dropped by debounce), got %d", len(tn.notifications))
	}
	tn.mu.Unlock()

	// 3. Test Critical urgency (should bypass debounce)
	m.Error("Critical Error", "Immediate")

	time.Sleep(50 * time.Millisecond)

	tn.mu.Lock()
	if len(tn.notifications) != 2 {
		t.Errorf("Expected 2 notifications total (Critical bypassed debounce), got %d", len(tn.notifications))
	}
	tn.mu.Unlock()
}

func TestManager_Disabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.Notifications = false

	m := New(cfg)
	tn := &LocalNotifier{}
	m.notifier = tn

	m.Info("Ignored", "Body")

	time.Sleep(50 * time.Millisecond)

	tn.mu.Lock()
	if len(tn.notifications) != 0 {
		t.Error("Expected 0 notifications when disabled in config")
	}
	tn.mu.Unlock()
}

func TestManager_SuccessTranscription(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.Notifications = true

	m := New(cfg)
	tn := &LocalNotifier{}
	m.notifier = tn

	if err := m.Start(); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer m.Stop()

	m.SuccessTranscriptionComplete("Hello World")

	time.Sleep(50 * time.Millisecond)

	tn.mu.Lock()
	if len(tn.notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(tn.notifications))
	}
	if tn.notifications[0].Summary != "Transcription Complete" {
		t.Errorf("Expected 'Transcription Complete', got %q", tn.notifications[0].Summary)
	}
	tn.mu.Unlock()
}
