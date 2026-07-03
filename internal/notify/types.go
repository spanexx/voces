/* Code Map: Notification Types
 * - Message: Structured data for desktop alerts
 *
 * CID Index:
 * CID:notify-types-001 -> Message
 *
 * Quick lookup: rg -n "CID:notify-types-" internal/notify/types.go
 */
package notify

import "github.com/esiqveland/notify"

// Type represents the kind of notification.
type Type int

const (
	TypeInfo Type = iota
	TypeSuccess
	TypeWarning
	TypeError
)

// CID:notify-types-001 - Message
// Purpose: Encapsulates a notification request before it is queued for delivery.
type Message struct {
	Title   string
	Body    string
	Type    Type
	Urgency notify.Urgency
}
