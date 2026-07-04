/* Code Map: x11KeyTracker load (X11 deps + keysym map)
 * - startX11KeyTracker: public entry point; checks X11 deps, builds
 *   the keysym map, spawns the reader goroutine.
 * - loadX11KeysymToKeycodeMap: parses `xmodmap -pke` output.
 *
 * Sibling files in this package:
 * - aliases.go:        modifierAliases + fkeyMap
 * - parse.go:          ParseKeys
 * - tracker_state.go:  x11KeyTracker struct + accessors
 * - tracker_x11_run.go: xinput reader + bridge
 *
 * CID Index:
 * CID:hotkey-tracker-x11-load-001 -> startX11KeyTracker
 * CID:hotkey-tracker-x11-load-002 -> loadX11KeysymToKeycodeMap
 *
 * Quick lookup: rg -n "CID:hotkey-tracker-x11-load-" internal/hotkey/
 */
package hotkey

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// CID:hotkey-tracker-x11-load-001 - startX11KeyTracker
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

// CID:hotkey-tracker-x11-load-002 - loadX11KeysymToKeycodeMap
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
