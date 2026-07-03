/* Code Map: Notification Methods
 * - Info: Sends informational alerts
 * - Error: Sends critical error alerts
 * - SuccessTranscriptionComplete: Specialized alert for transcription results
 *
 * CID Index:
 * CID:notify-methods-001 -> Info
 * CID:notify-methods-002 -> Error
 * CID:notify-methods-003 -> SuccessTranscriptionComplete
 *
 * Quick lookup: rg -n "CID:notify-methods-" internal/notify/notifications.go
 */
package notify

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/esiqveland/notify"
)

// CID:notify-methods-001 - Info
// Purpose: Dispatches a low-urgency informational notification.
func (m *Manager) Info(title, body string) {
	m.Send(Message{
		Title:   title,
		Body:    body,
		Type:    TypeInfo,
		Urgency: notify.UrgencyLow,
	})
}

// CID:notify-methods-002 - Error
// Purpose: Dispatches a critical error notification.
func (m *Manager) Error(title, body string) {
	m.Send(Message{
		Title:   title,
		Body:    body,
		Type:    TypeError,
		Urgency: notify.UrgencyCritical,
	})
}

// CID:notify-methods-003 - SuccessTranscriptionComplete
// Purpose: Formats the transcribed text and copies it to the system clipboard.
func (m *Manager) SuccessTranscriptionComplete(text string) {
	if len(text) > 100 {
		text = text[:97] + "..."
	}

	// For simplicity, we removed action buttons and just copy it straight away
	if m.clipboard.Available() {
		m.clipboard.Set(text)
	}

	m.Send(Message{
		Title:   "Transcription Complete",
		Body:    text + "\n\n(Copied to Clipboard)",
		Type:    TypeSuccess,
		Urgency: notify.UrgencyNormal,
	})
}

// ErrorBinaryNotFound notifications.
func (m *Manager) ErrorBinaryNotFound(engineName string) {
	m.Send(Message{
		Title:   "Setup Required",
		Body:    fmt.Sprintf("%s not found. Please review config.yaml.", engineName),
		Type:    TypeError,
		Urgency: notify.UrgencyCritical,
	})
}

// ErrorModelMissing notifications.
func (m *Manager) ErrorModelMissing(engineName string) {
	m.Send(Message{
		Title:   "Model Missing",
		Body:    fmt.Sprintf("%s model not found. Download required.", engineName),
		Type:    TypeError,
		Urgency: notify.UrgencyCritical,
	})
}

// ErrorAPIKey notifications.
func (m *Manager) ErrorAPIKey(provider string) {
	m.Send(Message{
		Title:   "API Key Required",
		Body:    fmt.Sprintf("%s API key not configured.", provider),
		Type:    TypeError,
		Urgency: notify.UrgencyCritical,
	})
}

// ViewLogs sends a notification with a button to view logs.
func (m *Manager) ViewLogs(title, body string) {
	m.Send(Message{
		Title:   title,
		Body:    body,
		Type:    TypeInfo,
		Urgency: notify.UrgencyNormal,
	})
	// Automatically open the logs without requiring a click action
	exec.Command("xdg-open", filepath.Join("logs", "whisper-voice-util.log")).Start()
}
