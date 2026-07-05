/* Code Map: GDK Keyval Table
 * - keyvalNames: maps GDK keyvals to the canonical names hotkey.ParseKeys
 *   understands. GDK does not ship a reverse name table in gotk3, so we
 *   keep the most common keys here.
 * - keyvalToString: returns a human-readable label for a GDK keyval.
 *   Falls back to the unicode character for printable keys, then to
 *   a hex form for unknown specials. Never returns the empty string.
 * - IsModifierKeyval: true for the pure modifier keyvals (Shift,
 *   Ctrl, Alt, Super, Caps Lock). The hotkey capture widget uses
 *   this to ignore modifier-only presses.
 * - IsValidAloneKeyval: true for keys that can be bound as a hotkey
 *   by themselves (F1-F12, Escape, Tab, Space, navigation keys).
 *   False for printable letters/digits/punctuation and for
 *   pure modifiers. The capture widget uses this to reject
 *   "weak" combos (e.g. "f" alone) with an inline warning.
 * - HasModifier: true if the GDK modifier state has any combo-worthy
 *   modifier set (Ctrl/Alt/Super/Shift). Ignores Caps Lock, Num
 *   Lock and button masks.
 * - BuildCombo: turn a (modifier state, base keyval) pair from a
 *   key-press event into the canonical "<mod>+<base>" string the
 *   hotkey parser understands. Pure function, exported for testing.
 *
 * The keyval→name mapping is split out from hotkey.go to keep that
 * file under the 250-line size cap.
 *
 * CID Index:
 * CID:wizard-keyval-001 -> keyvalNames
 * CID:wizard-keyval-002 -> keyvalToString
 * CID:wizard-keyval-003 -> IsModifierKeyval
 * CID:wizard-keyval-004 -> BuildCombo
 * CID:wizard-keyval-005 -> IsValidAloneKeyval
 * CID:wizard-keyval-006 -> HasModifier
 *
 * Quick lookup: rg -n "CID:wizard-keyval-" internal/wizard/steps/
 */
package steps

import (
	"fmt"
	"strings"

	"github.com/gotk3/gotk3/gdk"
)

// CID:wizard-keyval-001 - keyvalNames
// Purpose: reverse of gdk.KeyvalFromName. The F-keys are added by
// init() so the static map below stays readable.
var keyvalNames = map[uint]string{
	0x0008: "BackSpace",
	0x0009: "Tab",
	0x000a: "Return",
	0x000d: "Return",
	0x001b: "Escape",
	0x0020: "space",
	0xff08: "BackSpace",
	0xff09: "Tab",
	0xff0d: "Return",
	0xff1b: "Escape",
	0xff20: "Caps_Lock",
	0xff50: "Home",
	0xff51: "Left",
	0xff52: "Up",
	0xff53: "Right",
	0xff54: "Down",
	0xff55: "Page_Up",
	0xff56: "Page_Down",
	0xff57: "End",
	0xff63: "Insert",
	0xffe1: "Shift_L",
	0xffe2: "Shift_R",
	0xffe3: "Control_L",
	0xffe4: "Control_R",
	0xffe9: "Alt_L",
	0xffea: "Alt_R",
	0xffeb: "Super_L",
	0xffec: "Super_R",
	0xffff: "Delete",
}

// fkeyMap maps "F1".."F12" to their GDK keyvals. Used by init() to
// populate keyvalNames so the static map above stays readable.
var fkeyMap = map[string]uint{
	"F1":  0xffbe,
	"F2":  0xffbf,
	"F3":  0xffc0,
	"F4":  0xffc1,
	"F5":  0xffc2,
	"F6":  0xffc3,
	"F7":  0xffc4,
	"F8":  0xffc5,
	"F9":  0xffc6,
	"F10": 0xffc7,
	"F11": 0xffc8,
	"F12": 0xffc9,
}

func init() {
	for name, k := range fkeyMap {
		keyvalNames[k] = name
	}
}

// CID:wizard-keyval-002 - keyvalToString
// Purpose: convert a GDK keyval to the string the hotkey parser
// understands. The hotkey parser maps these names back to X11
// keysyms, so the round-trip works end-to-end.
func keyvalToString(k uint) string {
	if name, ok := keyvalNames[k]; ok {
		return name
	}
	if r := gdk.KeyvalToUnicode(k); r != 0 {
		return string(r)
	}
	return fmt.Sprintf("0x%04x", k)
}

// CID:wizard-keyval-003 - IsModifierKeyval
// Purpose: tells the hotkey capture widget whether a given keyval
// is a pure modifier (Shift, Ctrl, Alt, Super, Caps Lock). Modifier
// presses without a base key should be ignored by the capture
// handler — the user is still building a combination, not finishing
// one. Caps Lock and Shift Lock are also returned (they're toggle
// modifiers, not held keys, so we never want them in a combo).
func IsModifierKeyval(k uint) bool {
	switch k {
	case 0xffe1, 0xffe2, // Shift_L, Shift_R
		0xffe3, 0xffe4, // Control_L, Control_R
		0xffe5, 0xffe6, // Caps_Lock, Shift_Lock
		0xffe9, 0xffea, // Alt_L, Alt_R
		0xffeb, 0xffec: // Super_L, Super_R
		return true
	}
	return false
}

// CID:wizard-keyval-005 - IsValidAloneKeyval
// Purpose: tells the hotkey capture widget whether a keyval can be
// bound as a hotkey by itself (without any modifier held). The
// answer is true for the F-keys (F1-F12) and the non-printable
// special keys (Escape, Tab, BackSpace, Return, Space, navigation
// keys, Insert, Delete). False for printable keys (letters, digits,
// punctuation) because binding a single printable key as a hotkey
// would intercept that character everywhere it appears in text
// input — the user can't type "f" anymore. False for pure
// modifier keyvals (per IsModifierKeyval).
//
// The hotkey capture widget uses this to reject "weak" combos
// (letter or digit with no modifier held) with an inline warning
// so the user understands why their keypress didn't take.
func IsValidAloneKeyval(k uint) bool {
	if IsModifierKeyval(k) {
		return false
	}
	// F1-F12: contiguous keyval range 0xffbe..0xffc9.
	if k >= 0xffbe && k <= 0xffc9 {
		return true
	}
	// Special non-printable keys + space (commonly bound alone).
	switch k {
	case 0xff08, 0xff09, 0xff0d, 0xff1b, // BackSpace, Tab, Return, Escape
		0xff50, 0xff51, 0xff52, 0xff53, 0xff54, 0xff55, 0xff56, 0xff57, // Home..End
		0xff63, 0xffff, // Insert, Delete
		0x0020: // space (printable but a common alone-binding)
		return true
	}
	return false
}

// CID:wizard-keyval-006 - HasModifier
// Purpose: true if the GDK modifier-state bitmask has any of the
// four "combo-worthy" modifiers set: Control, Alt (Mod1), Super
// (Mod4), Shift. Used by the hotkey capture widget to decide
// whether a keyval needs to be valid-alone (no modifier) or any
// keyval works (with modifier). Num Lock, Caps Lock, Scroll Lock
// and button masks are intentionally ignored.
//
// Takes uint (not gdk.ModifierType) because gotk3's EventKey.State()
// already returns uint; the conversion is free.
func HasModifier(s uint) bool {
	return s&(uint(gdk.CONTROL_MASK)|uint(gdk.MOD1_MASK)|uint(gdk.MOD4_MASK)|uint(gdk.SHIFT_MASK)) != 0
}

// CID:wizard-keyval-004 - BuildCombo
// Purpose: turn a (modifier state, base keyval) pair captured from
// a key-press event into the canonical "<mod>+<base>" string the
// hotkey parser understands (e.g., "ctrl+shift+f9", "f8", "a").
//
// Behavior:
//   - Modifier-only keyvals (per IsModifierKeyval) return "" so the
//     caller can ignore them and wait for the user to press a
//     non-modifier key.
//   - Modifier order is fixed: ctrl, super, alt, shift, then the
//     base. This keeps the display stable across presses and the
//     parser is order-insensitive anyway.
//   - The base is lowercased (parser is case-insensitive on both
//     modifier and F-key tokens).
//   - Printable keys come from gdk.KeyvalToUnicode; the
//     hotkey parser passes unknown tokens through as-is so this
//     round-trips for "a", "1", "space" etc.
//
// Takes uint for modState (see HasModifier).
func BuildCombo(modState uint, keyval uint) string {
	if IsModifierKeyval(keyval) {
		return ""
	}
	base := strings.ToLower(keyvalToString(keyval))
	mods := make([]string, 0, 4)
	if modState&uint(gdk.CONTROL_MASK) != 0 {
		mods = append(mods, "ctrl")
	}
	if modState&uint(gdk.MOD4_MASK) != 0 {
		mods = append(mods, "super")
	}
	if modState&uint(gdk.MOD1_MASK) != 0 {
		mods = append(mods, "alt")
	}
	if modState&uint(gdk.SHIFT_MASK) != 0 {
		mods = append(mods, "shift")
	}
	mods = append(mods, base)
	return strings.Join(mods, "+")
}
