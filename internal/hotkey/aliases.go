/* Code Map: Hotkey alias tables
 * - modifierAliases: user-friendly names -> X11 key names for
 *   modifier and arrow keys
 * - fkeyMap: function-key names -> X11 keysym names
 *
 * Sibling files in this package:
 * - parse.go:        ParseKeys
 * - tracker_state.go: x11KeyTracker struct + accessors
 * - tracker_x11.go:   xmodmap + xinput tracker process
 *
 * CID Index:
 * CID:hotkey-aliases-001 -> modifierAliases
 * CID:hotkey-aliases-002 -> fkeyMap
 *
 * Quick lookup: rg -n "CID:hotkey-aliases-" internal/hotkey/
 */
package hotkey

// modifierAliases maps user-friendly modifier names to X11 key names
// used by xdotool for key-state queries.
var modifierAliases = map[string]string{
	"ctrl":      "Control_L",
	"control":   "Control_L",
	"rightctrl": "Control_R",
	"rctrl":     "Control_R",
	"leftctrl":  "Control_L",
	"lctrl":     "Control_L",
	"alt":       "Alt_L",
	"option":    "Alt_L",
	"shift":     "Shift_L",
	"super":     "Super_L",
	"win":       "Super_L",
	"cmd":       "Super_L",
	"left":      "Left",
	"right":     "Right",
	"up":        "Up",
	"down":      "Down",
	"space":     "space",
	"tab":       "Tab",
	"enter":     "Return",
	"escape":    "Escape",
}

// fkeyMap maps function key names to X11 keysym names.
var fkeyMap = map[string]string{
	"f1": "F1", "f2": "F2", "f3": "F3", "f4": "F4",
	"f5": "F5", "f6": "F6", "f7": "F7", "f8": "F8",
	"f9": "F9", "f10": "F10", "f11": "F11", "f12": "F12",
}
