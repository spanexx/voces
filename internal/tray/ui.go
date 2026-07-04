/* Code Map: Tray UI
 * - UpdateTranscriptionEngine: Syncs radio buttons for transcription
 * - UpdateTTSEngine: Syncs radio buttons for TTS
 * - onReady: Initializes menu items and submenus
 * - openEditor: Helper to launch files in xdg-open
 *
 * CID Index:
 * CID:tray-ui-001 -> UpdateTranscriptionEngine
 * CID:tray-ui-002 -> UpdateTTSEngine
 * CID:tray-ui-003 -> onReady
 *
 * Quick lookup: rg -n "CID:tray-ui-" internal/tray/ui.go
 */
package tray

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/getlantern/systray"
)

// CID:tray-ui-001 - UpdateTranscriptionEngine
// Purpose: Synchronizes the checked state of transcription engine submenus.
func (m *Manager) UpdateTranscriptionEngine(engine string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, item := range m.mEnginesTrans {
		if name == engine {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
}

// CID:tray-ui-002 - UpdateTTSEngine
// Purpose: Synchronizes the checked state of TTS engine submenus.
func (m *Manager) UpdateTTSEngine(engine string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, item := range m.mEnginesTTS {
		if name == engine {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
}

// CID:tray-ui-003 - onReady
// Purpose: Entry point for systray initialization. Builds all menu items and attaches event loops.
func (m *Manager) onReady() {
	systray.SetIcon(IconIdle)
	systray.SetTitle("Whisper Voice Utility")
	systray.SetTooltip("Whisper Voice Utility - Ready")

	// Phase 7 — dynamic "Update available" item at the top of the
	// menu. Hidden until SetUpdateBadge is called with a newer
	// release. See ui_phase7.go.
	m.addPhase7MenuItems()

	m.mRecord = systray.AddMenuItem("Record Now", "Trigger manual recording")
	go func() {
		for range m.mRecord.ClickedCh {
			if m.handlers.OnRecordStart != nil {
				m.handlers.OnRecordStart()
			}
		}
	}()

	m.mRead = systray.AddMenuItem("Read Clipboard", "Read clipboard content aloud")
	go func() {
		for range m.mRead.ClickedCh {
			if m.handlers.OnReadClipboard != nil {
				m.handlers.OnReadClipboard()
			}
		}
	}()

	systray.AddSeparator()

	transMenu := systray.AddMenuItem("Transcription Engine", "Select transcription engine")
	m.mEnginesTrans["whisper_cpp"] = transMenu.AddSubMenuItemCheckbox("whisper.cpp (local)", "Use local whisper.cpp", m.cfg.Transcription.DefaultEngine == "whisper_cpp")
	m.mEnginesTrans["openai_api"] = transMenu.AddSubMenuItemCheckbox("OpenAI API (cloud)", "Use OpenAI cloud API", m.cfg.Transcription.DefaultEngine == "openai_api")

	for engineName, item := range m.mEnginesTrans {
		name := engineName
		menuItem := item
		go func() {
			for range menuItem.ClickedCh {
				if m.handlers.OnSetTranscriptionEngine != nil {
					m.handlers.OnSetTranscriptionEngine(name)
					m.UpdateTranscriptionEngine(name)
				}
			}
		}()
	}

	// TTS Engine
	ttsMenu := systray.AddMenuItem("TTS Engine", "Select text-to-speech engine")
	m.mEnginesTTS["piper"] = ttsMenu.AddSubMenuItemCheckbox("Piper (local)", "Use local Piper TTS", m.cfg.TTS.DefaultEngine == "piper")
	m.mEnginesTTS["elevenlabs"] = ttsMenu.AddSubMenuItemCheckbox("ElevenLabs (cloud)", "Use ElevenLabs cloud TTS", m.cfg.TTS.DefaultEngine == "elevenlabs")

	for engineName, item := range m.mEnginesTTS {
		name := engineName
		menuItem := item
		go func() {
			for range menuItem.ClickedCh {
				if m.handlers.OnSetTTSEngine != nil {
					m.handlers.OnSetTTSEngine(name)
					m.UpdateTTSEngine(name)
				}
			}
		}()
	}

	systray.AddSeparator()

	m.mSettings = systray.AddMenuItem("Settings", "Open configuration file")
	go func() {
		for range m.mSettings.ClickedCh {
			log.Printf("Tray action: Settings")
			openEditor(resolveConfigPath())
		}
	}()

	// Phase 6 — wizard / updates / data-dir access. See ui_phase6.go.
	m.addPhase6MenuItems()

	systray.AddSeparator()

	m.mLogs = systray.AddMenuItem("View Logs", "Open application logs")
	go func() {
		for range m.mLogs.ClickedCh {
			log.Printf("Tray action: View Logs")
			openEditor(resolveLogPath())
		}
	}()

	systray.AddSeparator()

	m.mQuit = systray.AddMenuItem("Quit", "Quit application")
	go func() {
		for range m.mQuit.ClickedCh {
			if m.handlers.OnQuit != nil {
				m.handlers.OnQuit()
			} else {
				systray.Quit()
			}
		}
	}()
}

func (m *Manager) onExit() {
	log.Println("Tray shutting down")
}

func openEditor(path string) {
	if path == "" {
		return
	}

	if strings.HasSuffix(path, ".log") {
		_ = os.MkdirAll(filepath.Dir(path), 0755)
		if _, err := os.Stat(path); err != nil {
			if f, createErr := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); createErr == nil {
				_ = f.Close()
			}
		}
	}

	openers := [][]string{}
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".log") || strings.HasSuffix(path, ".txt") {
		openers = append(openers,
			[]string{"code", "--reuse-window", "-g", path},
			[]string{"code", "-g", path},
		)
	}
	openers = append(openers,
		[]string{"xdg-open", path},
		[]string{"gio", "open", path},
		[]string{"sensible-editor", path},
	)

	for _, args := range openers {
		cmd := exec.Command(args[0], args[1:]...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Start(); err != nil {
			if s := strings.TrimSpace(out.String()); s != "" {
				log.Printf("Failed to open %s with %s: %v (%s)", path, args[0], err, s)
			} else {
				log.Printf("Failed to open %s with %s: %v", path, args[0], err)
			}
			continue
		}

		// For GUI launchers, the process often exits quickly with 0 after handing off.
		// But if it exits quickly with a non-zero status, try the next opener.
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()

		select {
		case err := <-done:
			if err != nil {
				if s := strings.TrimSpace(out.String()); s != "" {
					log.Printf("Failed to open %s with %s: %v (%s)", path, args[0], err, s)
				} else {
					log.Printf("Failed to open %s with %s: %v", path, args[0], err)
				}
				continue
			}
			return
		case <-time.After(300 * time.Millisecond):
			// Assume it launched successfully and don't block the tray UI.
			return
		}
	}

	log.Printf("Failed to open %s: no opener available", path)
}

func resolveConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(configDir, "whisper-voice-util", "config.yaml")
}

func resolveLogPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join("logs", "whisper-voice-util.log")
	}
	return filepath.Join(configDir, "whisper-voice-util", "logs", "whisper-voice-util.log")
}
