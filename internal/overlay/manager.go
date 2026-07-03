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

type Manager struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	sock   string
	cancel context.CancelFunc
}

func New() *Manager {
	return &Manager{}
}

func (m *Manager) Start(onStop func()) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil {
		return nil
	}

	sock := filepath.Join(os.TempDir(), "whisper-voice-util-overlay.sock")
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
	overlayBin := filepath.Join(filepath.Dir(exe), "whisper-voice-overlay")
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
