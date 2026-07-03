// Package input provides keyboard input handling and simulation.
//
// Code Map:
// CID: input-003 - KeyboardSimulator for typing text via xdotool
package input

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode"
)

// KeyboardSimulator simulates keyboard input to type text into the active window.
// CID: input-003
type KeyboardSimulator struct {
	mu         sync.Mutex
	charDelay  time.Duration // delay between individual keystrokes
	wordDelay  time.Duration // delay after space/word boundaries
	burstSize  int           // number of chars before a micro-pause
	burstPause time.Duration // pause after each burst
	typing     bool          // whether a typing operation is in progress
	abortCh    chan struct{} // signal channel to abort mid-type
	runner     func(args ...string) error
}

// NewKeyboardSimulator creates a KeyboardSimulator with the given per-character
// delay (in milliseconds). A delay of 0 types as fast as possible.
func NewKeyboardSimulator(charDelayMs int) *KeyboardSimulator {
	if charDelayMs < 0 {
		charDelayMs = 0
	}
	return &KeyboardSimulator{
		charDelay:  time.Duration(charDelayMs) * time.Millisecond,
		wordDelay:  time.Duration(charDelayMs*3) * time.Millisecond,
		burstSize:  5,
		burstPause: time.Duration(charDelayMs) * time.Millisecond,
		abortCh:    make(chan struct{}),
		runner: func(args ...string) error {
			return exec.Command("xdotool", args...).Run()
		},
	}
}

// SetRunner overrides the command runner for testing.
func (k *KeyboardSimulator) SetRunner(runner func(args ...string) error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.runner = runner
}

// TypeText types the given text into the currently focused window character by
// character using xdotool key simulation. Returns an error if a typing session
// is already active.
func (k *KeyboardSimulator) TypeText(text string) error {
	k.mu.Lock()
	if k.typing {
		k.mu.Unlock()
		return fmt.Errorf("typing already in progress")
	}
	k.typing = true
	k.abortCh = make(chan struct{})
	k.mu.Unlock()

	defer func() {
		k.mu.Lock()
		k.typing = false
		k.mu.Unlock()
	}()

	charCount := 0
	for _, ch := range text {
		// Check for abort signal
		select {
		case <-k.abortCh:
			return fmt.Errorf("typing aborted after %d characters", charCount)
		default:
		}

		if err := k.typeChar(ch); err != nil {
			return fmt.Errorf("failed to type character at position %d: %w", charCount, err)
		}
		charCount++

		// Apply delays
		if ch == ' ' || ch == '\t' {
			k.sleep(k.wordDelay)
		} else {
			k.sleep(k.charDelay)
		}

		// Burst pause
		if k.burstSize > 0 && charCount%k.burstSize == 0 {
			k.sleep(k.burstPause)
		}
	}

	return nil
}

// Abort stops the current typing operation. Safe to call even when not typing.
func (k *KeyboardSimulator) Abort() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.typing {
		close(k.abortCh)
	}
}

// IsTyping returns whether a typing operation is currently in progress.
func (k *KeyboardSimulator) IsTyping() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.typing
}

// typeChar sends a single character via xdotool.
func (k *KeyboardSimulator) typeChar(ch rune) error {
	var cmdArgs []string
	switch ch {
	case '\n':
		cmdArgs = []string{"key", "Return"}
	case '\t':
		cmdArgs = []string{"key", "Tab"}
	case ' ':
		cmdArgs = []string{"key", "space"}
	default:
		s := string(ch)
		// xdotool's `key` is fragile for punctuation (e.g. '-' can lead to exit status 2).
		// Use `type` for anything that isn't a simple ASCII letter/digit.
		if isSpecialKey(ch) || needsShift(ch) || (!unicode.IsLetter(ch) && !unicode.IsDigit(ch)) {
			cmdArgs = []string{"type", s}
		} else {
			cmdArgs = []string{"key", s}
		}
	}
	if cmdArgs != nil {
		return k.runner(cmdArgs...)
	}
	return nil
}

// sleep pauses for the given duration while remaining responsive to abort.
func (k *KeyboardSimulator) sleep(d time.Duration) {
	if d <= 0 {
		return
	}
	select {
	case <-k.abortCh:
	case <-time.After(d):
	}
}

// needsShift returns true if the character requires the Shift modifier.
func needsShift(ch rune) bool {
	if unicode.IsUpper(ch) {
		return true
	}
	// Common shifted punctuation on US keyboard layout
	shifted := "~!@#$%^&*()_+{}|:\"<>?"
	return strings.ContainsRune(shifted, ch)
}

// isSpecialKey returns true for characters that cannot be typed with a simple
// KeyTap call (unicode, emoji, non-Latin scripts, etc.).
func isSpecialKey(ch rune) bool {
	// Non-ASCII runes, control chars (except the ones we handle), and rare
	// symbols are best handled via TypeStr.
	if ch > 127 {
		return true
	}
	if unicode.IsControl(ch) && ch != '\n' && ch != '\t' {
		return true
	}
	return false
}
