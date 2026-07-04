/* Code Map: Autostart Configuration
 * - SyncAutostartState: Updates XDG autostart entry
 * - Enable/DisableAutostart: File-level operations
 *
 * CID Index:
 * CID:app-autostart-001 -> SyncAutostartState
 *
 * Quick lookup: rg -n "CID:app-autostart-" internal/app/autostart.go
 */
package app

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const desktopEntryContent = `[Desktop Entry]
Type=Application
Name=Voces
Comment=Voice transcription and TTS assistant
Exec=%s
Icon=audio-input-microphone
Terminal=false
Categories=Utility;Accessibility;
X-GNOME-Autostart-enabled=true
X-GNOME-Autostart-Delay=2
`

// desktopEntryPath returns the path to the autostart .desktop file.
func desktopEntryPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	autoStartDir := filepath.Join(configDir, "autostart")
	if err := os.MkdirAll(autoStartDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(autoStartDir, "voces.desktop"), nil
}

// EnableAutostart creates the XDG autostart desktop entry.
func EnableAutostart() error {
	path, err := desktopEntryPath()
	if err != nil {
		return err
	}

	execPath, err := exec.LookPath("voces")
	if err != nil {
		execPath, err = os.Executable()
		if err != nil {
			return err
		}
		// We resolve symlinks to get the real binary if needed
		if realPath, evalErr := filepath.EvalSymlinks(execPath); evalErr == nil {
			execPath = realPath
		}
	}

	content := fmt.Sprintf(desktopEntryContent, execPath)
	log.Printf("Autostart: enabling (path=%s, exec=%s)", path, execPath)
	return os.WriteFile(path, []byte(content), 0644)
}

// DisableAutostart removes the XDG autostart desktop entry.
func DisableAutostart() error {
	path, err := desktopEntryPath()
	if err != nil {
		return err
	}
	log.Printf("Autostart: disabling (path=%s)", path)

	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// CID:app-autostart-001 - SyncAutostartState
// Purpose: Ensures the system-level autostart configuration matches the app setting.
func SyncAutostartState(enabled bool) error {
	if enabled {
		return EnableAutostart()
	}
	return DisableAutostart()
}
