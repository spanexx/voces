package hotkey

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseKeys(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"<rightctrl>+<left>", []string{"Control_R", "Left"}},
		{"ctrl+alt+t", []string{"Control_L", "Alt_L", "t"}},
		{"<f10>", []string{"F10"}},
		{"shift+A", []string{"Shift_L", "a"}},
		{"", []string{}},
		{"<<", []string{"<"}},
	}

	for _, tc := range tests {
		actual := ParseKeys(tc.input)
		if len(actual) != len(tc.expected) {
			t.Errorf("ParseKeys(%q) expected len %d, got %d", tc.input, len(tc.expected), len(actual))
			continue
		}
		for i, v := range actual {
			if v != tc.expected[i] {
				t.Errorf("ParseKeys(%q)[%d] expected %q, got %q", tc.input, i, tc.expected[i], v)
			}
		}
	}
}

func TestHoldBinding_StateMachine(t *testing.T) {
	// 1. Setup simulated key state
	simulatedKeys := make(map[string]bool)
	var mu sync.Mutex

	// Override package variable
	oldIsKeyPressed := isKeyPressed
	defer func() { isKeyPressed = oldIsKeyPressed }()

	isKeyPressed = func(key string) bool {
		mu.Lock()
		defer mu.Unlock()
		return simulatedKeys[key]
	}

	var pressCount int32
	var releaseCount int32
	onPress := func() { atomic.AddInt32(&pressCount, 1) }
	onRelease := func() { atomic.AddInt32(&releaseCount, 1) }

	h := NewHoldBinding("ctrl+a", onPress, onRelease)
	h.pollInterval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go h.run(ctx, &wg)

	// Step 1: Initial state (nothing pressed)
	time.Sleep(20 * time.Millisecond)
	if atomic.LoadInt32(&pressCount) != 0 {
		t.Errorf("Expected 0 presses, got %d", atomic.LoadInt32(&pressCount))
	}

	// Step 2: Press the hotkey
	mu.Lock()
	simulatedKeys["Control_L"] = true
	simulatedKeys["a"] = true
	mu.Unlock()

	time.Sleep(30 * time.Millisecond)
	if atomic.LoadInt32(&pressCount) != 1 {
		t.Errorf("Expected 1 press, got %d", atomic.LoadInt32(&pressCount))
	}

	// Step 3: Release one key
	mu.Lock()
	simulatedKeys["a"] = false
	mu.Unlock()

	time.Sleep(30 * time.Millisecond)
	if atomic.LoadInt32(&releaseCount) != 1 {
		t.Errorf("Expected 1 release, got %d", atomic.LoadInt32(&releaseCount))
	}

	// Step 4: Press again
	mu.Lock()
	simulatedKeys["a"] = true
	mu.Unlock()

	time.Sleep(30 * time.Millisecond)
	if atomic.LoadInt32(&pressCount) != 2 {
		t.Errorf("Expected 2 presses, got %d", atomic.LoadInt32(&pressCount))
	}

	cancel()
	wg.Wait()
}

func TestPressBinding_StateMachine(t *testing.T) {
	simulatedKeys := make(map[string]bool)
	var mu sync.Mutex

	oldIsKeyPressed := isKeyPressed
	defer func() { isKeyPressed = oldIsKeyPressed }()

	isKeyPressed = func(key string) bool {
		mu.Lock()
		defer mu.Unlock()
		return simulatedKeys[key]
	}

	var pressCount int32
	onPress := func() { atomic.AddInt32(&pressCount, 1) }

	p := NewPressBinding("ctrl+t", onPress)
	p.pollInterval = 10 * time.Millisecond
	p.debounce = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go p.run(ctx, &wg)

	// Step 1: Rising edge
	mu.Lock()
	simulatedKeys["Control_L"] = true
	simulatedKeys["t"] = true
	mu.Unlock()

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&pressCount) != 1 {
		t.Errorf("Expected 1 press, got %d", atomic.LoadInt32(&pressCount))
	}

	// Step 2: Still held (should not re-fire)
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&pressCount) != 1 {
		t.Errorf("Expected still 1 press while held, got %d", atomic.LoadInt32(&pressCount))
	}

	// Step 3: Release
	mu.Lock()
	simulatedKeys["t"] = false
	mu.Unlock()
	time.Sleep(20 * time.Millisecond)

	// Step 4: Re-press (after debounce)
	mu.Lock()
	simulatedKeys["t"] = true
	mu.Unlock()

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&pressCount) != 2 {
		t.Errorf("Expected 2 presses after release/re-press, got %d", atomic.LoadInt32(&pressCount))
	}

	cancel()
	wg.Wait()
}
