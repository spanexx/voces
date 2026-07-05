package steps

import (
	"testing"

	"github.com/gotk3/gotk3/gdk"
)

// TestIsModifierKeyval covers the modifier keyvals BuildCombo
// needs to special-case. The map below mirrors the magic numbers
// in IsModifierKeyval — if a keyval is added, add it here too.
func TestIsModifierKeyval(t *testing.T) {
	modifiers := map[uint]string{
		0xffe1: "Shift_L", 0xffe2: "Shift_R",
		0xffe3: "Control_L", 0xffe4: "Control_R",
		0xffe5: "Caps_Lock", 0xffe6: "Shift_Lock",
		0xffe9: "Alt_L", 0xffea: "Alt_R",
		0xffeb: "Super_L", 0xffec: "Super_R",
	}
	for k, name := range modifiers {
		if !IsModifierKeyval(k) {
			t.Errorf("IsModifierKeyval(0x%04x = %s) = false, want true", k, name)
		}
	}
	// Sanity: common non-modifier keyvals should return false.
	for k, name := range map[uint]string{
		0xff08: "BackSpace", 0xff09: "Tab", 0xff0d: "Return",
		0xff1b: "Escape", 0xffc6: "F9", 0x0061: "a", 0x0020: "space",
	} {
		if IsModifierKeyval(k) {
			t.Errorf("IsModifierKeyval(0x%04x = %s) = true, want false", k, name)
		}
	}
}

// TestBuildCombo covers the keypress → canonical-combo-string
// conversion used by the hotkey step's capture widget. The
// canonical form is what hotkey.ParseKeys understands, so a
// round-trip should produce the same keysym list for every case.
//
// Keyvals (decimal/hex) reference:
//   F8=0xffc5, F9=0xffc6, F10=0xffc7
//   BackSpace=0xff08, Tab=0xff09, Escape=0xff1b
//   a=0x0061, A=0x0041 (Shift+a), space=0x0020
func TestBuildCombo(t *testing.T) {
	tests := []struct {
		name    string
		state   gdk.ModifierType
		keyval  uint
		want    string
	}{
		// Plain F-key (no modifiers).
		{"F8 alone", 0, 0xffc5, "f8"},
		{"F9 alone", 0, 0xffc6, "f9"},

		// Single modifier combinations.
		{"Ctrl+Space", gdk.CONTROL_MASK, 0x0020, "ctrl+space"},
		{"Alt+F4", gdk.MOD1_MASK, 0xffc1, "alt+f4"},
		{"Super+a", gdk.MOD4_MASK, 0x0061, "super+a"},

		// Multiple modifiers — order is fixed (ctrl, super, alt, shift).
		{"Ctrl+Shift+F9 (user's reported case)", gdk.CONTROL_MASK | gdk.SHIFT_MASK, 0xffc6, "ctrl+shift+f9"},
		{"Ctrl+Alt+F10", gdk.CONTROL_MASK | gdk.MOD1_MASK, 0xffc7, "ctrl+alt+f10"},
		{"Ctrl+Super+Alt+Shift+a (all four)", gdk.CONTROL_MASK | gdk.MOD4_MASK | gdk.MOD1_MASK | gdk.SHIFT_MASK, 0x0041, "ctrl+super+alt+shift+a"},

		// Capital letter with Shift only — should produce "shift+a" (lowercased).
		{"Shift+a (capital A pressed)", gdk.SHIFT_MASK, 0x0041, "shift+a"},

		// Modifier-only presses return "" so the caller can ignore them.
		{"Shift_L alone (pure modifier)", gdk.SHIFT_MASK, 0xffe1, ""},
		{"Control_L alone (pure modifier)", gdk.CONTROL_MASK, 0xffe3, ""},
		{"Alt_L alone (pure modifier)", gdk.MOD1_MASK, 0xffe9, ""},

		// Special named keys (Escape, BackSpace, Tab).
		{"Escape alone", 0, 0xff1b, "escape"},
		{"Ctrl+Escape", gdk.CONTROL_MASK, 0xff1b, "ctrl+escape"},
		{"Tab alone", 0, 0xff09, "tab"},

		// Non-modifier state bits (Caps Lock, Num Lock, button masks)
		// must be ignored. Bit 1 is LOCK_MASK, bit 4 is MOD3 (NumLk).
		{"Caps Lock + a (should be plain a)", gdk.ModifierType(1 << 1), 0x0061, "a"},
		{"Num Lock + F9 (should be plain f9)", gdk.ModifierType(1 << 4), 0xffc6, "f9"},
		{"Button1 + a (should be plain a)", gdk.ModifierType(1 << 8), 0x0061, "a"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildCombo(tc.state, tc.keyval)
			if got != tc.want {
				t.Errorf("BuildCombo(state=0x%04x, keyval=0x%04x) = %q, want %q",
					uint(tc.state), tc.keyval, got, tc.want)
			}
		})
	}
}
