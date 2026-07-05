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

// TestIsValidAloneKeyval covers the keyvals the hotkey capture
// widget accepts as a binding by themselves. F1-F12 (0xffbe..0xffc9)
// and the non-printable specials (Escape, Tab, BackSpace, Return,
// space, navigation keys, Insert, Delete) return true. Printable
// letters/digits/punctuation and pure modifiers return false.
func TestIsValidAloneKeyval(t *testing.T) {
	// All 12 F-keys should be valid alone.
	for i := uint(0xffbe); i <= 0xffc9; i++ {
		if !IsValidAloneKeyval(i) {
			t.Errorf("IsValidAloneKeyval(0x%04x = F%d) = false, want true", i, i-0xffbd)
		}
	}
	// Special non-printable keys.
	for _, k := range []uint{
		0xff08, 0xff09, 0xff0d, 0xff1b, // BackSpace, Tab, Return, Escape
		0xff50, 0xff51, 0xff52, 0xff53, 0xff54, 0xff55, 0xff56, 0xff57, // Home..End
		0xff63, 0xffff, // Insert, Delete
		0x0020, // space
	} {
		if !IsValidAloneKeyval(k) {
			t.Errorf("IsValidAloneKeyval(0x%04x) = false, want true", k)
		}
	}
	// Printable keys should NOT be valid alone.
	for _, k := range []uint{0x0061, 0x0041, 0x0030, 0x0031, 0x002c, 0x002e} {
		if IsValidAloneKeyval(k) {
			t.Errorf("IsValidAloneKeyval(0x%04x = printable) = true, want false", k)
		}
	}
	// Pure modifier keyvals should NOT be valid alone.
	for _, k := range []uint{0xffe1, 0xffe3, 0xffe9, 0xffeb, 0xffe5} {
		if IsValidAloneKeyval(k) {
			t.Errorf("IsValidAloneKeyval(0x%04x = modifier) = true, want false", k)
		}
	}
}

// TestHasModifier covers the GDK modifier-state bitmask helper.
// True for any of Ctrl/Alt/Super/Shift. False for the empty
// mask and for non-combo-worthy bits (Caps Lock, Num Lock,
// button masks).
func TestHasModifier(t *testing.T) {
	cases := []struct {
		name  string
		state uint
		want  bool
	}{
		{"empty state", 0, false},
		{"Shift only", uint(gdk.SHIFT_MASK), true},
		{"Ctrl only", uint(gdk.CONTROL_MASK), true},
		{"Alt only", uint(gdk.MOD1_MASK), true},
		{"Super only", uint(gdk.MOD4_MASK), true},
		{"Ctrl+Shift", uint(gdk.CONTROL_MASK | gdk.SHIFT_MASK), true},
		{"Ctrl+Alt+Super+Shift (all four)", uint(gdk.CONTROL_MASK | gdk.MOD1_MASK | gdk.MOD4_MASK | gdk.SHIFT_MASK), true},
		// Non-combo-worthy bits: Caps Lock (LOCK_MASK=bit1), Num Lock
		// (MOD2_MASK=bit5), button masks (bits 8-11).
		{"Caps Lock only", 1 << 1, false},
		{"Num Lock only", 1 << 5, false},
		{"Button1 only", 1 << 8, false},
		{"Ctrl + Caps Lock (should still be true — Ctrl is set)", uint(gdk.CONTROL_MASK) | (1 << 1), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := HasModifier(tc.state)
			if got != tc.want {
				t.Errorf("HasModifier(0x%04x) = %v, want %v", tc.state, got, tc.want)
			}
		})
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
		state   uint
		keyval  uint
		want    string
	}{
		// Plain F-key (no modifiers).
		{"F8 alone", 0, 0xffc5, "f8"},
		{"F9 alone", 0, 0xffc6, "f9"},

		// Single modifier combinations.
		{"Ctrl+Space", uint(gdk.CONTROL_MASK), 0x0020, "ctrl+space"},
		{"Alt+F4", uint(gdk.MOD1_MASK), 0xffc1, "alt+f4"},
		{"Super+a", uint(gdk.MOD4_MASK), 0x0061, "super+a"},

		// Multiple modifiers — order is fixed (ctrl, super, alt, shift).
		{"Ctrl+Shift+F9 (user's reported case)", uint(gdk.CONTROL_MASK | gdk.SHIFT_MASK), 0xffc6, "ctrl+shift+f9"},
		{"Ctrl+Alt+F10", uint(gdk.CONTROL_MASK | gdk.MOD1_MASK), 0xffc7, "ctrl+alt+f10"},
		{"Ctrl+Super+Alt+Shift+a (all four)", uint(gdk.CONTROL_MASK | gdk.MOD4_MASK | gdk.MOD1_MASK | gdk.SHIFT_MASK), 0x0041, "ctrl+super+alt+shift+a"},

		// Capital letter with Shift only — should produce "shift+a" (lowercased).
		{"Shift+a (capital A pressed)", uint(gdk.SHIFT_MASK), 0x0041, "shift+a"},

		// Modifier-only presses return "" so the caller can ignore them.
		{"Shift_L alone (pure modifier)", uint(gdk.SHIFT_MASK), 0xffe1, ""},
		{"Control_L alone (pure modifier)", uint(gdk.CONTROL_MASK), 0xffe3, ""},
		{"Alt_L alone (pure modifier)", uint(gdk.MOD1_MASK), 0xffe9, ""},

		// Special named keys (Escape, BackSpace, Tab).
		{"Escape alone", 0, 0xff1b, "escape"},
		{"Ctrl+Escape", uint(gdk.CONTROL_MASK), 0xff1b, "ctrl+escape"},
		{"Tab alone", 0, 0xff09, "tab"},

		// Non-modifier state bits (Caps Lock, Num Lock, button masks)
		// must be ignored. Bit 1 is LOCK_MASK, bit 4 is MOD3 (NumLk).
		{"Caps Lock + a (should be plain a)", 1 << 1, 0x0061, "a"},
		{"Num Lock + F9 (should be plain f9)", 1 << 4, 0xffc6, "f9"},
		{"Button1 + a (should be plain a)", 1 << 8, 0x0061, "a"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildCombo(tc.state, tc.keyval)
			if got != tc.want {
				t.Errorf("BuildCombo(state=0x%04x, keyval=0x%04x) = %q, want %q",
					tc.state, tc.keyval, got, tc.want)
			}
		})
	}
}
