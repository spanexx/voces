/* Code Map: Update Notifier — Process Restart
 * Files in this package:
 *   updates.go  - Release model, LatestRelease, IsNewer, Download
 *   restart.go  - syscall.Exec helper that replaces the running process
 *
 * CID Index:
 * CID:updates-007 -> Restart
 * CID:updates-008 -> StagedPath
 *
 * Quick lookup: rg -n "CID:updates-" internal/updates/
 */
package updates

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// StagedPath returns the path of the staged update binary that lives
// next to the current binary. Mirrors NewAssetPath but takes the
// current binary's path (which may or may not be the same as the
// desired destPath) and resolves to <bindir>/<basename>.new.
func StagedPath(currentBinary string) string {
	base := filepath.Base(currentBinary)
	dir := filepath.Dir(currentBinary)
	return filepath.Join(dir, base+updateFileSuffix)
}

// CID:updates-007 - Restart
// Purpose: Swap the staged update binary into place and exec it,
// replacing the current process. The new process inherits the same
// argv (minus argv[0], which is replaced) and environment.
//
// On success, this function does not return — the process image is
// replaced. On failure, it returns an error so the tray can surface
// a "could not restart" notification.
//
// Order of operations:
//  1. Verify the staged file exists and is executable.
//  2. If a previous run left a `<bin>.new.bak`, remove it.
//  3. Rename `<bin>` → `<bin>.new.bak` (so we can roll back if the
//     new binary fails to start).
//  4. Rename `<bin>.new` → `<bin>` (the staged file lands in place).
//  5. syscall.Exec the new binary with the same argv/env as we
//     were launched with.
//
// Steps 3-4 are an atomic-ish swap: if step 4 fails after step 3,
// the .bak is left in place and we return an error before exec. The
// rollback (rename .bak → original) is the operator's job in v1;
// later phases can add automatic rollback on startup failure.
func Restart(currentBinary string, args []string) error {
	if currentBinary == "" {
		return errors.New("updates: empty binary path")
	}
	staged := StagedPath(currentBinary)
	if _, err := os.Stat(staged); err != nil {
		return fmt.Errorf("updates: staged binary missing: %w", err)
	}
	// Make the staged file executable. download.Download writes with
	// 0644 by default; we need 0755 to be runnable.
	if err := os.Chmod(staged, 0o755); err != nil {
		return fmt.Errorf("updates: chmod staged: %w", err)
	}

	bak := currentBinary + ".new.bak"
	// Best-effort cleanup of a stale .bak from a prior failed run.
	// Ignore "not exist" — that's the happy path.
	_ = os.Remove(bak)

	if err := os.Rename(currentBinary, bak); err != nil {
		return fmt.Errorf("updates: backup %s → %s: %w", currentBinary, bak, err)
	}
	if err := os.Rename(staged, currentBinary); err != nil {
		// Try to roll the backup back so the old binary stays runnable.
		_ = os.Rename(bak, currentBinary)
		return fmt.Errorf("updates: install %s → %s: %w", staged, currentBinary, err)
	}

	// exec replaces the current process image. On success this does
	// not return. If it does return, the error is non-nil and we are
	// still in the old process — roll back.
	argv := append([]string{currentBinary}, args[1:]...)
	env := os.Environ()
	if err := syscall.Exec(currentBinary, argv, env); err != nil {
		// Rollback: restore the old binary and return.
		_ = os.Remove(currentBinary)
		_ = os.Rename(bak, currentBinary)
		return fmt.Errorf("updates: exec %s: %w", currentBinary, err)
	}
	// Unreachable on success. On the rare path where syscall.Exec
	// returns nil but doesn't replace, return a sentinel so the
	// caller knows something went wrong.
	return errors.New("updates: syscall.Exec returned without replacing process")
}

// LaunchDetached spawns `path` as a brand new detached process and
// returns immediately. Used as a fallback when syscall.Exec refuses
// to run (e.g. when argv[0] doesn't match the actual binary path).
// The caller is responsible for terminating the old process afterwards.
func LaunchDetached(path string, args []string) error {
	if path == "" {
		return errors.New("updates: empty path")
	}
	cmd := exec.Command(path, args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("updates: launch %s: %w", path, err)
	}
	// Detach: do not wait. The OS reaps the child when it exits.
	go func() { _ = cmd.Wait() }()
	return nil
}
