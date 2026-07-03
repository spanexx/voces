/* Code Map: Hotkey Bindings
 * - NewHoldBinding: Detects sustained key presses
 * - NewPressBinding: Detects single key taps
 *
 * CID Index:
 * CID:hotkey-binding-001 -> NewHoldBinding
 * CID:hotkey-binding-002 -> NewPressBinding
 *
 * Quick lookup: rg -n "CID:hotkey-binding-" internal/hotkey/binding.go
 */
package hotkey

import (
	"context"
	"sync"
	"time"
)

// CID:hotkey-binding-001 - NewHoldBinding
// Purpose: Initializes a watcher for sustained key combinations (push-to-talk).
func NewHoldBinding(hotkeyStr string, onPress, onRelease func()) *HoldBinding {
	return &HoldBinding{
		keys:         ParseKeys(hotkeyStr),
		rawHotkey:    hotkeyStr,
		onPress:      onPress,
		onRelease:    onRelease,
		pollInterval: 20 * time.Millisecond,
	}
}

// run polls the key state until ctx is cancelled.
func (h *HoldBinding) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(h.pollInterval)
	defer ticker.Stop()

	held := false

	for {
		select {
		case <-ctx.Done():
			if held && h.onRelease != nil {
				h.onRelease()
			}
			return
		case <-ticker.C:
			nowHeld := h.allKeysDown()
			if nowHeld && !held {
				held = true
				if h.onPress != nil {
					h.onPress()
				}
			} else if !nowHeld && held {
				held = false
				if h.onRelease != nil {
					h.onRelease()
				}
			}
		}
	}
}

// allKeysDown checks if all keys in the combination are pressed.
func (h *HoldBinding) allKeysDown() bool {
	for _, k := range h.keys {
		if !isKeyPressed(k) {
			return false
		}
	}
	return true
}

// CID:hotkey-binding-002 - NewPressBinding
// Purpose: Initializes a watcher for quick key combinations (toggle).
func NewPressBinding(hotkeyStr string, onPress func()) *PressBinding {
	return &PressBinding{
		keys:         ParseKeys(hotkeyStr),
		rawHotkey:    hotkeyStr,
		onPress:      onPress,
		pollInterval: 20 * time.Millisecond,
		debounce:     200 * time.Millisecond,
	}
}

// run polls until ctx is cancelled, fires onPress on rising edge with debounce.
func (p *PressBinding) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	wasDown := false
	var lastFire time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			down := p.allKeysDown()
			if down && !wasDown && time.Since(lastFire) > p.debounce {
				lastFire = time.Now()
				if p.onPress != nil {
					go p.onPress() // non-blocking
				}
			}
			wasDown = down
		}
	}
}

// allKeysDown checks if all keys in the combination are pressed.
func (p *PressBinding) allKeysDown() bool {
	for _, k := range p.keys {
		if !isKeyPressed(k) {
			return false
		}
	}
	return true
}
