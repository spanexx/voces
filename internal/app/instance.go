/* Code Map: Single Instance Logic
 * - CheckAndLockSingleInstance: Prevents multiple app processes
 * - processExists: Unix process validation
 *
 * CID Index:
 * CID:app-instance-001 -> CheckAndLockSingleInstance
 *
 * Quick lookup: rg -n "CID:app-instance-" internal/app/instance.go
 */
package app

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"
)

var lockFilePath = "/tmp/whisper-voice-util.lock"

// CID:app-instance-001 - CheckAndLockSingleInstance
// Purpose: Employs a lock file to ensure only one instance of the utility is active.
func CheckAndLockSingleInstance() (func(), error) {
	if _, err := os.Stat(lockFilePath); err == nil {
		// File exists, check if process is running
		file, err := os.Open(lockFilePath)
		if err == nil {
			pidBytes, _ := io.ReadAll(file)
			file.Close()
			if pid, err := strconv.Atoi(string(pidBytes)); err == nil {
				if processExists(pid) {
					return nil, fmt.Errorf("app is already running (PID: %d)", pid)
				}
			}
		}
	}

	// Create/overwrite lock file with our PID
	err := os.WriteFile(lockFilePath, []byte(strconv.Itoa(os.Getpid())), 0644)
	if err != nil {
		return nil, fmt.Errorf("could not create lock file: %w", err)
	}

	cleanup := func() {
		os.Remove(lockFilePath)
	}

	return cleanup, nil
}

// processExists checks if a process with a given PID is currently running.
// On Unix systems, sending signal 0 checks for process existence.
func processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false // FindProcess only returns error on non-Unix systems usually
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
