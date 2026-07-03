package input

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"whisper-voice-util/internal/config"
)

func TestSplitByTypeability(t *testing.T) {
	tests := []struct {
		input    string
		expected []textSegment
	}{
		{
			input: "Hello World",
			expected: []textSegment{
				{text: "Hello World", special: false},
			},
		},
		{
			input: "Hello 👋 World",
			expected: []textSegment{
				{text: "Hello ", special: false},
				{text: "👋", special: true},
				{text: " World", special: false},
			},
		},
		{
			input: "日本語",
			expected: []textSegment{
				{text: "日本語", special: true},
			},
		},
	}

	for _, tc := range tests {
		actual := splitByTypeability(tc.input)
		if len(actual) != len(tc.expected) {
			t.Errorf("splitByTypeability(%q) expected %d segments, got %d", tc.input, len(tc.expected), len(actual))
			continue
		}
		for i, seg := range actual {
			if seg.text != tc.expected[i].text || seg.special != tc.expected[i].special {
				t.Errorf("splitByTypeability(%q) segment %d expected %+v, got %+v", tc.input, i, tc.expected[i], seg)
			}
		}
	}
}

func TestKeyboardSimulator_Logic(t *testing.T) {
	k := NewKeyboardSimulator(1)

	var mu sync.Mutex
	var commands []string
	k.runner = func(args ...string) error {
		mu.Lock()
		defer mu.Unlock()
		commands = append(commands, strings.Join(args, " "))
		return nil
	}

	err := k.TypeText("A b")
	if err != nil {
		t.Fatalf("TypeText failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	// "A" (shift+a) -> type A or key a with shift logic in typeChar
	// " " -> key space
	// "b" -> key b

	// Verification of sequence
	if len(commands) < 3 {
		t.Errorf("Expected at least 3 commands, got %d", len(commands))
	}
}

func TestAutoTyper_Fallback(t *testing.T) {
	cfg := &config.Config{}
	cfg.Behavior.AutoType = true

	at := NewAutoTyper(cfg)

	// Configure Keyboard to fail on typing, but succeed on pasting
	at.keyboard.runner = func(args ...string) error {
		if len(args) >= 2 && args[0] == "key" && args[1] == "control+v" {
			return nil
		}
		// Fail everything else related to typing/keying
		if args[0] == "key" || args[0] == "type" {
			return errors.New("simulated keyboard failure")
		}
		return nil
	}

	// Configure Clipboard
	var mu sync.Mutex
	var setHistory []string
	at.clipboard.setter = func(text string) error {
		mu.Lock()
		defer mu.Unlock()
		t.Logf("DEBUG-TEST: setter called with %q", text)
		setHistory = append(setHistory, text)
		return nil
	}
	at.clipboard.avail = func() bool { return true }
	at.clipboard.getter = func() (string, error) { return "OriginalValue", nil }

	// Trigger type
	err := at.Type("Simple")
	if err != nil {
		t.Fatalf("AutoTyper.Type failed: %v", err)
	}

	// Verify fallback to clipboard happened, and then restore happened
	mu.Lock()
	defer mu.Unlock()

	foundSimple := false
	for _, val := range setHistory {
		if val == "Simple" {
			foundSimple = true
		}
	}

	if !foundSimple {
		t.Errorf("Expected fallback to clipboard with content 'Simple' in history, got %v", setHistory)
	}

	// Final value should be restored to "OriginalValue"
	if len(setHistory) > 0 && setHistory[len(setHistory)-1] != "OriginalValue" {
		t.Errorf("Expected final restore to 'OriginalValue', got %q", setHistory[len(setHistory)-1])
	}
}

func TestClipboard_BackupRestore(t *testing.T) {
	c := NewClipboard()

	val := "Original"
	c.getter = func() (string, error) { return val, nil }
	c.setter = func(text string) error {
		val = text
		return nil
	}

	restore, err := c.Backup()
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	c.Set("New Value")
	if val != "New Value" {
		t.Error("Set failed to update value")
	}

	err = restore()
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	if val != "Original" {
		t.Errorf("Restore failed, expected 'Original', got %q", val)
	}
}

func TestClipboard_Operations(t *testing.T) {
	c := NewClipboard()
	c.avail = func() bool { return true }

	// Test Get
	c.getter = func() (string, error) { return "hello", nil }
	val, err := c.Get()
	if val != "hello" || err != nil {
		t.Errorf("Get failed")
	}

	// Test Set fails
	c.setter = func(string) error { return fmt.Errorf("fail") }
	err = c.Set("boom")
	if err == nil {
		t.Error("Expected error because setter failed")
	}

	// Test available
	if !c.Available() {
		t.Error("Expected available to be true")
	}

	// Backup fails if get fails
	c.getter = func() (string, error) { return "", fmt.Errorf("fail") }
	_, err = c.Backup()
	if err == nil {
		t.Error("Expected backup failure")
	}
}

func TestKeyboard_UtilityFunctions(t *testing.T) {
	// Test needsShift
	shifts := []rune{'A', '!', '@', '{', '>', '?'}
	for _, r := range shifts {
		if !needsShift(r) {
			t.Errorf("Expected %c to need shift", r)
		}
	}
	noShifts := []rune{'a', '1', ';', '.', ' '}
	for _, r := range noShifts {
		if needsShift(r) {
			t.Errorf("Expected %c to NOT need shift", r)
		}
	}

	// Test isSpecialKey
	specials := []rune{'🚀', '世', rune(1), '\v'}
	for _, r := range specials {
		if !isSpecialKey(r) {
			t.Errorf("Expected %c to be special", r)
		}
	}
	notSpecials := []rune{'a', '\n', '\t', '1', '!'}
	for _, r := range notSpecials {
		if isSpecialKey(r) {
			t.Errorf("Expected %c to NOT be special", r)
		}
	}
}
