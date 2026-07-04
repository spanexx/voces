/* Code Map: Hotkey string parser
 * - ParseKeys: translates "<ctrl>+a" style strings into the canonical
 *   keysym list used by the tracker.
 *
 * Sibling files in this package:
 * - aliases.go:      modifierAliases + fkeyMap
 * - tracker_state.go: x11KeyTracker struct + accessors
 * - tracker_x11.go:   xmodmap + xinput tracker process
 *
 * CID Index:
 * CID:hotkey-parse-001 -> ParseKeys
 *
 * Quick lookup: rg -n "CID:hotkey-parse-" internal/hotkey/
 */
package hotkey

import "strings"

// CID:hotkey-parse-001 - ParseKeys
// Purpose: normalizes hotkey strings (e.g., "<ctrl>+a") for use with
// the detection logic. Strips angle brackets, lowercases, splits on
// '+', and looks up each token in fkeyMap / modifierAliases.
func ParseKeys(hotkeyStr string) []string {
	if hotkeyStr == "" {
		return []string{}
	}

	// Handle escaped << as a literal <
	if hotkeyStr == "<<" {
		return []string{"<"}
	}

	// Strip angle brackets: <rightctrl> → rightctrl
	hotkeyStr = strings.ReplaceAll(hotkeyStr, "<", "")
	hotkeyStr = strings.ReplaceAll(hotkeyStr, ">", "")

	parts := strings.Split(hotkeyStr, "+")
	keys := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p == "" {
			continue
		}
		if fk, ok := fkeyMap[p]; ok {
			keys = append(keys, fk)
		} else if alias, ok := modifierAliases[p]; ok {
			keys = append(keys, alias)
		} else {
			// Single letter/number keys: use as-is
			keys = append(keys, p)
		}
	}
	return keys
}
