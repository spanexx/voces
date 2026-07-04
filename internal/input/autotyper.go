// Package input provides keyboard input handling and simulation.
//
// Code Map:
// CID: input-005 - AutoTyper orchestrates typing with clipboard fallback
package input

import (
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"voces/internal/config"
)

// AutoTyper orchestrates keyboard typing with clipboard-paste fallback.
// It uses the KeyboardSimulator for character-by-character input and falls
// back to Clipboard paste when encountering unsupported characters or errors.
// CID: input-005
type AutoTyper struct {
	keyboard  *KeyboardSimulator
	clipboard *Clipboard
	cfg       *config.Config
}

// NewAutoTyper creates a new AutoTyper from the application config.
func NewAutoTyper(cfg *config.Config) *AutoTyper {
	return &AutoTyper{
		keyboard:  NewKeyboardSimulator(cfg.Behavior.TypeDelay),
		clipboard: NewClipboard(),
		cfg:       cfg,
	}
}

// SetRunner overrides the command runner for testing.
func (a *AutoTyper) SetRunner(runner func(args ...string) error) {
	a.keyboard.SetRunner(runner)
}

// Type types the given text into the currently focused window. It uses
// character-by-character simulation as the primary method and falls back to
// clipboard paste for segments that contain special characters.
func (a *AutoTyper) Type(text string) error {
	if !a.cfg.Behavior.AutoType {
		return fmt.Errorf("auto-type is disabled in configuration")
	}

	if text == "" {
		return nil
	}

	// Backup clipboard before we use it
	var restoreClipboard func() error
	if a.clipboard.Available() {
		var err error
		restoreClipboard, err = a.clipboard.Backup()
		if err != nil {
			log.Printf("clipboard backup failed (non-fatal): %v", err)
		}
	}

	// Ensure clipboard is restored when done
	defer func() {
		if restoreClipboard != nil {
			if err := restoreClipboard(); err != nil {
				log.Printf("clipboard restore failed (non-fatal): %v", err)
			}
		}
	}()

	// Split text into segments: typeable (ASCII) vs special (needs clipboard)
	segments := splitByTypeability(text)

	for _, seg := range segments {
		if seg.special {
			// Use clipboard paste for special characters
			if err := a.pasteText(seg.text); err != nil {
				return fmt.Errorf("failed to paste special text: %w", err)
			}
		} else {
			// Use keyboard simulation for regular text
			if err := a.keyboard.TypeText(seg.text); err != nil {
				// Fallback to clipboard paste on keyboard error
				log.Printf("keyboard simulation failed, falling back to clipboard: %v", err)
				if pasteErr := a.pasteText(seg.text); pasteErr != nil {
					return fmt.Errorf("both keyboard and clipboard failed: keyboard=%v, clipboard=%w", err, pasteErr)
				}
			}
		}
	}

	return nil
}

// Abort stops any in-progress typing operation.
func (a *AutoTyper) Abort() {
	a.keyboard.Abort()
}

// IsTyping returns whether a typing operation is currently in progress.
func (a *AutoTyper) IsTyping() bool {
	return a.keyboard.IsTyping()
}

// pasteText copies text to clipboard and simulates Ctrl+V.
func (a *AutoTyper) pasteText(text string) error {
	if !a.clipboard.Available() {
		return fmt.Errorf("clipboard not available (xclip not installed)")
	}

	if err := a.clipboard.Set(text); err != nil {
		return fmt.Errorf("failed to set clipboard: %w", err)
	}

	// Small delay to ensure clipboard is ready
	time.Sleep(50 * time.Millisecond)

	// Simulate Ctrl+V using xdotool
	if err := a.keyboard.runner("key", "control+v"); err != nil {
		return fmt.Errorf("xdotool paste failed: %w", err)
	}

	// Small delay to ensure paste completes
	time.Sleep(50 * time.Millisecond)

	return nil
}

// textSegment represents a chunk of text that is either typeable via keyboard
// simulation or requires clipboard paste.
type textSegment struct {
	text    string
	special bool // true if this segment needs clipboard paste
}

// splitByTypeability splits text into alternating segments of directly-typeable
// characters and special characters that require clipboard paste.
func splitByTypeability(text string) []textSegment {
	if text == "" {
		return nil
	}

	var segments []textSegment
	var current strings.Builder
	currentSpecial := isSpecialChar(rune(text[0]))

	for _, ch := range text {
		charSpecial := isSpecialChar(ch)
		if charSpecial != currentSpecial {
			// Boundary: flush current segment
			if current.Len() > 0 {
				segments = append(segments, textSegment{
					text:    current.String(),
					special: currentSpecial,
				})
				current.Reset()
			}
			currentSpecial = charSpecial
		}
		current.WriteRune(ch)
	}

	// Flush final segment
	if current.Len() > 0 {
		segments = append(segments, textSegment{
			text:    current.String(),
			special: currentSpecial,
		})
	}

	return segments
}

// isSpecialChar returns true if the character cannot be reliably typed using
// keyboard simulation and should use clipboard paste instead.
func isSpecialChar(ch rune) bool {
	// Control characters (except newline and tab) are special
	if unicode.IsControl(ch) && ch != '\n' && ch != '\t' {
		return true
	}
	// Non-ASCII characters (unicode, emoji, accented chars) are special
	if ch > 127 {
		return true
	}
	return false
}
