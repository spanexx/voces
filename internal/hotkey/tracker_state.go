/* Code Map: x11KeyTracker struct + state accessors
 * - x11KeyTracker: holds the keysym→keycode map and the live
 *   pressed-state for each keycode.
 * - keysymKeycode: read accessor used by logKeysymDiagnostics
 * - logKeysymDiagnostics: debug log when binding is registered
 * - isPressed: package-private predicate for binding.go
 * - isKeyPressed: package-level indirection used by binding.go
 *
 * Sibling files in this package:
 * - aliases.go:    modifierAliases + fkeyMap
 * - parse.go:      ParseKeys
 * - tracker_x11.go: xmodmap + xinput tracker process
 *
 * CID Index:
 * CID:hotkey-tracker-state-001 -> x11KeyTracker
 * CID:hotkey-tracker-state-002 -> keysymKeycode
 * CID:hotkey-tracker-state-003 -> logKeysymDiagnostics
 * CID:hotkey-tracker-state-004 -> isPressed
 * CID:hotkey-tracker-state-005 -> isKeyPressed
 *
 * Quick lookup: rg -n "CID:hotkey-tracker-state-" internal/hotkey/
 */
package hotkey

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

// x11KeyTracker holds the keysym→keycode map and the pressed-state
// for each keycode. Mutated only from the xinput reader goroutine
// (write side via t.mu) and from the binding predicate (read side
// via t.mu.RLock).
type x11KeyTracker struct {
	mu           sync.RWMutex
	keysymToCode map[string]int
	pressed      map[int]bool
}

var (
	trackerMu sync.RWMutex
	tracker   *x11KeyTracker
)

func keysymKeycode(keysym string) (int, bool) {
	trackerMu.RLock()
	t := tracker
	trackerMu.RUnlock()
	if t == nil {
		return 0, false
	}
	code, ok := t.keysymToCode[keysym]
	return code, ok
}

func logKeysymDiagnostics(bindingName, raw string) {
	keys := ParseKeys(raw)
	if len(keys) == 0 {
		return
	}
	missing := make([]string, 0)
	resolved := make([]string, 0, len(keys))
	for _, k := range keys {
		if code, ok := keysymKeycode(k); ok {
			resolved = append(resolved, fmt.Sprintf("%s=%d", k, code))
		} else {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		log.Printf("hotkey: %s has unknown keysyms: raw=%q parsed=%v missing=%v", bindingName, raw, keys, missing)
	} else {
		log.Printf("hotkey: %s keysyms resolved: raw=%q %s", bindingName, raw, strings.Join(resolved, ", "))
	}
}

func (t *x11KeyTracker) isPressed(keysym string) bool {
	code, ok := t.keysymToCode[keysym]
	if !ok {
		return false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.pressed[code]
}

// isKeyPressed checks if a single X11 key is currently pressed.
var isKeyPressed = func(key string) bool {
	trackerMu.RLock()
	t := tracker
	trackerMu.RUnlock()
	if t == nil {
		return false
	}
	return t.isPressed(key)
}
