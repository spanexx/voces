/* Code Map: Recording Overlay
 * - Manager: owns the overlay subprocess and its stop-signal unix socket
 * - New: constructor (zero-value ready)
 * - Start: launches the overlay binary and returns a stop callback
 * - Stop: kills the overlay subprocess and cleans up the socket
 *
 * CID Index:
 * CID:overlay-001 -> Manager
 * CID:overlay-002 -> New
 * CID:overlay-003 -> Start
 * CID:overlay-004 -> Stop
 *
 * Quick lookup: rg -n "CID:overlay-" internal/overlay/manager.go
 */
package overlay

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// CID:overlay-001 - Manager
// Purpose: lifecycle owner of the cmd/voces-overlay subprocess.
// The overlay process is a tiny standalone GTK window that animates
// a "recording" bar; the manager launches it, signals STOP over a
// unix socket when the user clicks, and cleans up on shutdown.
type Manager struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	sock   string
	cancel context.CancelFunc
}

// CID:overlay-002 - New
// Purpose: zero-value constructor. The overlay subprocess is not
// spawned until Start is called.
func New() *Manager {
	return &Manager{}
}

func (m *Manager) Start(onStop func()) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil {
		return nil
	}

	sock := filepath.Join(os.TempDir(), "voces-overlay.sock")
	_ = os.Remove(sock)

	ln, err := net.Listen("unix", sock)
	if err != nil {
		return fmt.Errorf("overlay socket listen failed: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.sock = sock

	exe, err := os.Executable()
	if err != nil {
		_ = ln.Close()
		return fmt.Errorf("overlay failed to get executable path: %w", err)
	}
	overlayBin := filepath.Join(filepath.Dir(exe), "voces-overlay")
	cmd := exec.CommandContext(ctx, overlayBin, "--socket", sock)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		_ = ln.Close()
		return fmt.Errorf("overlay start failed: %w", err)
	}
	m.cmd = cmd

	go func() {
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				s := bufio.NewScanner(c)
				for s.Scan() {
					if s.Text() == "STOP" {
						if onStop != nil {
							onStop()
						}
						return
					}
				}
			}(conn)
		}
	}()

	go func() {
		_ = cmd.Wait()
	}()

	// give overlay a moment to appear
	time.Sleep(25 * time.Millisecond)
	return nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.cmd = nil
	if m.sock != "" {
		_ = os.Remove(m.sock)
		m.sock = ""
	}
}
