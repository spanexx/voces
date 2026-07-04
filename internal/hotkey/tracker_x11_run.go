/* Code Map: x11KeyTracker reader + xinput bridge
 * - (t *x11KeyTracker).run: reader loop; picks xi2 vs legacy parser.
 * - startXinputTrackerProcess: tries xi2 on root, falls back to master
 *   keyboard, then to legacy test mode.
 * - startXinputCmd: xinput subprocess scaffolding.
 * - xinputFailedQuickly: detects BadAccess / X Error within 150ms.
 * - xinputMasterKeyboardID: discovers the master keyboard device id.
 *
 * Sibling files in this package:
 * - aliases.go:           modifierAliases + fkeyMap
 * - parse.go:             ParseKeys
 * - tracker_state.go:     x11KeyTracker struct + accessors
 * - tracker_x11_load.go:  startX11KeyTracker + loadX11KeysymToKeycodeMap
 *
 * CID Index:
 * CID:hotkey-tracker-x11-run-001 -> (t *x11KeyTracker).run
 * CID:hotkey-tracker-x11-run-002 -> startXinputTrackerProcess
 * CID:hotkey-tracker-x11-run-003 -> startXinputCmd
 * CID:hotkey-tracker-x11-run-004 -> xinputFailedQuickly
 * CID:hotkey-tracker-x11-run-005 -> xinputMasterKeyboardID
 *
 * Quick lookup: rg -n "CID:hotkey-tracker-x11-run-" internal/hotkey/
 */
package hotkey

import (
	"bufio"
	"bytes"
	"context"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// safeBuffer is a goroutine-safe wrapper around bytes.Buffer. The
// xinput subprocess writes to it via cmd.Stderr while the parent
// goroutine reads from it via xinputFailedQuickly, so the underlying
// buffer needs its own mutex to keep `go test -race` clean.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

var (
	rxEventType = regexp.MustCompile(`\(RawKeyPress\)|\(RawKeyRelease\)|\(KeyPress\)|\(KeyRelease\)`)
	rxDetail    = regexp.MustCompile(`\bdetail:\s*([0-9]+)\b`)
)

var (
	rxLegacyPress   = regexp.MustCompile(`(?i)^key\s+press\s+([0-9]+)\s*$`)
	rxLegacyRelease = regexp.MustCompile(`(?i)^key\s+release\s+([0-9]+)\s*$`)
)

// CID:hotkey-tracker-x11-run-001 - (t *x11KeyTracker).run
func (t *x11KeyTracker) run(ctx context.Context) {
	mode, cmd, stdout, _ := startXinputTrackerProcess(ctx)
	if cmd == nil || stdout == nil {
		return
	}
	log.Printf("hotkey: xinput tracker mode=%s", mode)

	scanner := bufio.NewScanner(stdout)
	lastEvent := ""
	for scanner.Scan() {
		line := scanner.Text()
		switch mode {
		case "xi2":
			if m := rxEventType.FindString(line); m != "" {
				lastEvent = m
				continue
			}
			m := rxDetail.FindStringSubmatch(line)
			if len(m) != 2 {
				continue
			}
			code, convErr := strconv.Atoi(m[1])
			if convErr != nil {
				continue
			}
			t.mu.Lock()
			if strings.Contains(lastEvent, "Press") {
				t.pressed[code] = true
			} else if strings.Contains(lastEvent, "Release") {
				delete(t.pressed, code)
			}
			t.mu.Unlock()
		case "legacy":
			if m := rxLegacyPress.FindStringSubmatch(strings.TrimSpace(line)); len(m) == 2 {
				code, convErr := strconv.Atoi(m[1])
				if convErr != nil {
					continue
				}
				t.mu.Lock()
				t.pressed[code] = true
				t.mu.Unlock()
				continue
			}
			if m := rxLegacyRelease.FindStringSubmatch(strings.TrimSpace(line)); len(m) == 2 {
				code, convErr := strconv.Atoi(m[1])
				if convErr != nil {
					continue
				}
				t.mu.Lock()
				delete(t.pressed, code)
				t.mu.Unlock()
				continue
			}
		}
	}
	_ = cmd.Wait()
}

// CID:hotkey-tracker-x11-run-002 - startXinputTrackerProcess
func startXinputTrackerProcess(ctx context.Context) (string, *exec.Cmd, *bufio.Reader, *safeBuffer) {
	// Try XI2 on root first.
	mode, cmd, out, errBuf := startXinputCmd(ctx, []string{"test-xi2", "--root"})
	if cmd != nil {
		if !xinputFailedQuickly(cmd, errBuf) {
			return mode, cmd, out, errBuf
		}
		if strings.Contains(errBuf.String(), "BadAccess") {
			log.Printf("hotkey: xinput test-xi2 --root denied (BadAccess), falling back")
		}
	}

	// Fallback: XI2 on master keyboard device.
	if id := xinputMasterKeyboardID(ctx); id != "" {
		mode, cmd, out, errBuf = startXinputCmd(ctx, []string{"test-xi2", id})
		if cmd != nil && !xinputFailedQuickly(cmd, errBuf) {
			return mode, cmd, out, errBuf
		}
		mode, cmd, out, errBuf = startXinputCmd(ctx, []string{"test", id})
		if cmd != nil && !xinputFailedQuickly(cmd, errBuf) {
			return mode, cmd, out, errBuf
		}
	}

	return "", nil, nil, nil
}

// CID:hotkey-tracker-x11-run-003 - startXinputCmd
func startXinputCmd(ctx context.Context, args []string) (string, *exec.Cmd, *bufio.Reader, *safeBuffer) {
	cmd := exec.CommandContext(ctx, "xinput", args...)
	errBuf := &safeBuffer{}
	cmd.Stderr = errBuf
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", nil, nil, nil
	}
	if err := cmd.Start(); err != nil {
		return "", nil, nil, nil
	}
	mode := "xi2"
	if len(args) > 0 && args[0] == "test" {
		mode = "legacy"
	}
	return mode, cmd, bufio.NewReader(stdout), errBuf
}

// CID:hotkey-tracker-x11-run-004 - xinputFailedQuickly
func xinputFailedQuickly(cmd *exec.Cmd, errBuf *safeBuffer) bool {
	if cmd == nil {
		return true
	}
	// Give xinput a moment to error out (BadAccess typically happens immediately).
	t := time.NewTimer(150 * time.Millisecond)
	defer t.Stop()
	<-t.C
	if errBuf == nil {
		return false
	}
	s := errBuf.String()
	return strings.Contains(s, "BadAccess") || strings.Contains(s, "X Error")
}

// CID:hotkey-tracker-x11-run-005 - xinputMasterKeyboardID
func xinputMasterKeyboardID(ctx context.Context) string {
	// Prefer master keyboard. This works on most desktops.
	cmd := exec.CommandContext(ctx, "xinput", "list", "--id-only", "Virtual core keyboard")
	out, err := cmd.Output()
	if err == nil {
		id := strings.TrimSpace(string(out))
		if id != "" {
			return id
		}
	}
	return ""
}
