/* Code Map: Hotkey Mapping
 * - ParseKeys: Translates user strings to X11 keysyms
 *
 * CID Index:
 * CID:hotkey-mapping-001 -> ParseKeys
 *
 * Quick lookup: rg -n "CID:hotkey-mapping-" internal/hotkey/mapping.go
 */
package hotkey

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// modifierAliases maps user-friendly modifier names to X11 key names
// used by xdotool for key-state queries.
var modifierAliases = map[string]string{
	"ctrl":      "Control_L",
	"control":   "Control_L",
	"rightctrl": "Control_R",
	"rctrl":     "Control_R",
	"leftctrl":  "Control_L",
	"lctrl":     "Control_L",
	"alt":       "Alt_L",
	"option":    "Alt_L",
	"shift":     "Shift_L",
	"super":     "Super_L",
	"win":       "Super_L",
	"cmd":       "Super_L",
	"left":      "Left",
	"right":     "Right",
	"up":        "Up",
	"down":      "Down",
	"space":     "space",
	"tab":       "Tab",
	"enter":     "Return",
	"escape":    "Escape",
}

// fkeyMap maps function key names to X11 keysym names.
var fkeyMap = map[string]string{
	"f1": "F1", "f2": "F2", "f3": "F3", "f4": "F4",
	"f5": "F5", "f6": "F6", "f7": "F7", "f8": "F8",
	"f9": "F9", "f10": "F10", "f11": "F11", "f12": "F12",
}

type x11KeyTracker struct {
	mu           sync.RWMutex
	keysymToCode map[string]int
	pressed      map[int]bool
}

var (
	trackerMu sync.RWMutex
	tracker   *x11KeyTracker
)

func keysymKeycode(keysym string) (int, bool) {
	trackerMu.RLock()
	t := tracker
	trackerMu.RUnlock()
	if t == nil {
		return 0, false
	}
	code, ok := t.keysymToCode[keysym]
	return code, ok
}

func logKeysymDiagnostics(bindingName, raw string) {
	keys := ParseKeys(raw)
	if len(keys) == 0 {
		return
	}
	missing := make([]string, 0)
	resolved := make([]string, 0, len(keys))
	for _, k := range keys {
		if code, ok := keysymKeycode(k); ok {
			resolved = append(resolved, fmt.Sprintf("%s=%d", k, code))
		} else {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		log.Printf("hotkey: %s has unknown keysyms: raw=%q parsed=%v missing=%v", bindingName, raw, keys, missing)
	} else {
		log.Printf("hotkey: %s keysyms resolved: raw=%q %s", bindingName, raw, strings.Join(resolved, ", "))
	}
}

// CID:hotkey-mapping-001 - ParseKeys
// Purpose: Normalizes hotkey strings (e.g., "<ctrl>+a") for use with detection logic.
func ParseKeys(hotkeyStr string) []string {
	if hotkeyStr == "" {
		return []string{}
	}

	// Handle escaped << as a literal <
	if hotkeyStr == "<<" {
		return []string{"<"}
	}

	// Strip angle brackets: <rightctrl> → rightctrl
	hotkeyStr = strings.ReplaceAll(hotkeyStr, "<", "")
	hotkeyStr = strings.ReplaceAll(hotkeyStr, ">", "")

	parts := strings.Split(hotkeyStr, "+")
	keys := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p == "" {
			continue
		}
		if fk, ok := fkeyMap[p]; ok {
			keys = append(keys, fk)
		} else if alias, ok := modifierAliases[p]; ok {
			keys = append(keys, alias)
		} else {
			// Single letter/number keys: use as-is
			keys = append(keys, p)
		}
	}
	return keys
}

func startX11KeyTracker(ctx context.Context) error {
	if os.Getenv("DISPLAY") == "" {
		return fmt.Errorf("hotkeys require X11 (DISPLAY not set)")
	}
	if _, err := exec.LookPath("xinput"); err != nil {
		return fmt.Errorf("hotkeys require xinput: %w", err)
	}
	if _, err := exec.LookPath("xmodmap"); err != nil {
		return fmt.Errorf("hotkeys require xmodmap: %w", err)
	}

	keysymToCode, err := loadX11KeysymToKeycodeMap(ctx)
	if err != nil {
		return err
	}

	t := &x11KeyTracker{
		keysymToCode: keysymToCode,
		pressed:      make(map[int]bool),
	}
	trackerMu.Lock()
	tracker = t
	trackerMu.Unlock()

	go t.run(ctx)
	return nil
}

func loadX11KeysymToKeycodeMap(ctx context.Context) (map[string]int, error) {
	cmd := exec.CommandContext(ctx, "xmodmap", "-pke")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("xmodmap -pke failed: %w", err)
	}

	keysymToCode := make(map[string]int)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// Preferred xmodmap -pke format:
		//   keycode  37 = Control_L NoSymbol Control_L
		// Some systems may emit numeric format:
		//   37 = Control_L ...
		codeField := ""
		eqIndex := -1
		if fields[0] == "keycode" {
			if len(fields) < 4 {
				continue
			}
			codeField = fields[1]
			if fields[2] != "=" {
				continue
			}
			eqIndex = 2
		} else {
			codeField = fields[0]
			if fields[1] != "=" {
				continue
			}
			eqIndex = 1
		}

		code, convErr := strconv.Atoi(codeField)
		if convErr != nil {
			continue
		}

		for _, sym := range fields[eqIndex+1:] {
			if sym == "NoSymbol" {
				continue
			}
			if _, exists := keysymToCode[sym]; !exists {
				keysymToCode[sym] = code
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("xmodmap -pk parse failed: %w", err)
	}

	return keysymToCode, nil
}

var (
	rxEventType = regexp.MustCompile(`\(RawKeyPress\)|\(RawKeyRelease\)|\(KeyPress\)|\(KeyRelease\)`)
	rxDetail    = regexp.MustCompile(`\bdetail:\s*([0-9]+)\b`)
)

var (
	rxLegacyPress   = regexp.MustCompile(`(?i)^key\s+press\s+([0-9]+)\s*$`)
	rxLegacyRelease = regexp.MustCompile(`(?i)^key\s+release\s+([0-9]+)\s*$`)
)

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

func startXinputTrackerProcess(ctx context.Context) (string, *exec.Cmd, *bufio.Reader, *bytes.Buffer) {
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

func startXinputCmd(ctx context.Context, args []string) (string, *exec.Cmd, *bufio.Reader, *bytes.Buffer) {
	cmd := exec.CommandContext(ctx, "xinput", args...)
	errBuf := &bytes.Buffer{}
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

func xinputFailedQuickly(cmd *exec.Cmd, errBuf *bytes.Buffer) bool {
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

func (t *x11KeyTracker) isPressed(keysym string) bool {
	code, ok := t.keysymToCode[keysym]
	if !ok {
		return false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.pressed[code]
}

// isKeyPressed checks if a single X11 key is currently pressed.
var isKeyPressed = func(key string) bool {
	trackerMu.RLock()
	t := tracker
	trackerMu.RUnlock()
	if t == nil {
		return false
	}
	return t.isPressed(key)
}
