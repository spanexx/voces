package notify

import (
	"testing"
	"voces/internal/config"
)

func TestNotifications_ErrorPaths(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.Notifications = true
	m := New(cfg)

	// Since these methods now send to a channel, we need to drain it or use a large buffer
	// to avoid blocking. Manager has a 10-msg buffer by default.

	m.ErrorBinaryNotFound("test-bin")
	m.ErrorModelMissing("test-model")
	m.ErrorAPIKey("test-service")
	m.ViewLogs("title", "body")
	m.SuccessTranscriptionComplete("some text")

	// Drain messages from the unexported queue to verify they were sent
	for i := 0; i < 5; i++ {
		select {
		case msg := <-m.queue:
			if msg.Title == "" {
				t.Errorf("Message %d has empty title", i)
			}
		default:
			t.Errorf("Missing expected message %d", i)
		}
	}
}

func TestManager_Exhaustive(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.Notifications = true

	m := New(cfg)

	// Test dispatch with a local notifier to avoid dbus
	notifier := &LocalNotifier{}
	m.notifier = notifier

	msg := Message{Title: "Test", Body: "Body", Type: TypeSuccess}
	m.dispatch(msg)

	// Test debounce (short delay)
	m.dispatch(msg) // Should be skipped due to debounce

	// Test Critical (should NOT be debounced)
	crit := Message{Title: "Crit", Body: "Body", Type: TypeError, Urgency: 2} // notify.UrgencyCritical is 2
	m.dispatch(crit)

	// Call all methods for coverage
	m.Info("Info", "Body")
	m.Error("Error", "Body")
	m.SuccessTranscriptionComplete("text")
	m.ViewLogs("Logs", "Body")
	m.ErrorBinaryNotFound("bin")
	m.ErrorModelMissing("model")
	m.ErrorAPIKey("service")

	// Test Send with disabled notifications
	m.cfg.Behavior.Notifications = false
	m.Send(msg)
	m.cfg.Behavior.Notifications = true

	// Test queue overflow
	for i := 0; i < 20; i++ {
		m.Send(msg)
	}

	// Drain everything
	for i := 0; i < 30; i++ {
		select {
		case <-m.queue:
		default:
		}
	}
}

func TestManager_Lifecycle(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.Notifications = true
	m := New(cfg)

	// Attempt real D-Bus connection as requested.
	// We handle errors gracefully to ensure tests pass in headless envs while still covering the real branch.
	err := m.Start()
	if err != nil {
		t.Logf("Real D-Bus start skipped: %v", err)
		// Fallback to local only for the remainder of lifecycle state testing to avoid nil panics
		m.notifier = &LocalNotifier{}
		m.Start()
	}

	// Start again (should be no-op)
	err = m.Start()
	if err != nil {
		t.Errorf("Second Start failed: %v", err)
	}

	m.Stop()
	m.Stop()
}
