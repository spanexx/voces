// Package input provides keyboard input handling and simulation.
//
// Code Map:
// CID: input-004 - Clipboard integration for backup/restore and paste fallback
package input

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// Clipboard provides clipboard read/write operations using xclip.
// CID: input-004
type Clipboard struct {
	mu     sync.Mutex
	getter func() (string, error)
	setter func(string) error
	avail  func() bool
}

// NewClipboard creates a new Clipboard instance.
func NewClipboard() *Clipboard {
	return &Clipboard{
		getter: func() (string, error) {
			cmd := exec.Command("xclip", "-selection", "clipboard", "-o")
			out, err := cmd.Output()
			if err != nil {
				if _, ok := err.(*exec.ExitError); ok {
					return "", nil
				}
				return "", err
			}
			return string(out), nil
		},
		setter: func(text string) error {
			cmd := exec.Command("xclip", "-selection", "clipboard")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		},
		avail: func() bool {
			_, err := exec.LookPath("xclip")
			return err == nil
		},
	}
}

// Get returns the current clipboard contents.
func (c *Clipboard) Get() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	out, err := c.getter()
	if err != nil {
		return "", fmt.Errorf("failed to read clipboard: %w", err)
	}
	return out, nil
}

// Set writes text to the clipboard.
func (c *Clipboard) Set(text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.setter(text); err != nil {
		return fmt.Errorf("failed to write clipboard: %w", err)
	}
	return nil
}

// Backup saves the current clipboard contents and returns a restore function.
// Call the returned function to restore the original clipboard contents.
func (c *Clipboard) Backup() (restore func() error, err error) {
	original, err := c.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to backup clipboard: %w", err)
	}

	return func() error {
		return c.Set(original)
	}, nil
}

// Available returns true if xclip is installed and usable.
func (c *Clipboard) Available() bool {
	return c.avail()
}
