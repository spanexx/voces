/* Code Map: Phase 6 handler tests
 * - TestBuildTrayHandlers_HasPhase6: confirms the three new tray
 *   handlers are wired into buildTrayHandlers.
 * - TestOpenDataDir_ResolvesCanonicalPath: confirms the data-dir
 *   resolver returns the expected XDG path.
 * - TestNotifySetupFinished_NoNotifierNoPanic: confirms the helper
 *   does not panic when a.Notifier is nil.
 * - TestNotifySetupError_NoNotifierNoPanic: same, for the error path.
 *
 * CID Index:
 * CID:app-handlers-test-001 -> TestBuildTrayHandlers_HasPhase6
 * CID:app-handlers-test-002 -> TestOpenDataDir_ResolvesCanonicalPath
 * CID:app-handlers-test-003 -> TestNotify*NoNotifierNoPanic
 */
package app

import (
	"context"
	"io"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"whisper-voice-util/internal/paths"
)

func newTestApplication() *Application {
	ctx, cancel := context.WithCancel(context.Background())
	return &Application{
		Notifier: nil,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// TestBuildTrayHandlers_HasPhase6 confirms the three new tray
// handlers from IMPL §6 are wired into buildTrayHandlers. The IMPL
// specifies a manual smoke test for the "user clicks 'Run setup
// again...'" path; this unit test covers the wiring only.
func TestBuildTrayHandlers_HasPhase6(t *testing.T) {
	a := newTestApplication()
	defer a.cancel()

	handlers := a.buildTrayHandlers()
	if handlers.OnRunSetup == nil {
		t.Error("OnRunSetup is nil — tray 'Run setup again...' will be a no-op")
	}
	if handlers.OnCheckUpdates == nil {
		t.Error("OnCheckUpdates is nil — tray 'Check for updates' will be a no-op")
	}
	if handlers.OnOpenDataDir == nil {
		t.Error("OnOpenDataDir is nil — tray 'Open App-managed folder' will be a no-op")
	}
	// Re-confirm the legacy wiring still works.
	if handlers.OnQuit == nil {
		t.Error("OnQuit is nil — pre-Phase-6 regression")
	}
	if handlers.OnRecordStart == nil {
		t.Error("OnRecordStart is nil — pre-Phase-6 regression")
	}
}

// TestOpenDataDir_ResolvesCanonicalPath verifies the data-dir
// resolution path that openDataDir uses. We don't actually exec
// xdg-open in the test (no display) — we just confirm the path
// that would be passed to it matches the XDG data dir contract.
func TestOpenDataDir_ResolvesCanonicalPath(t *testing.T) {
	hostData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", hostData)

	got, err := paths.DataDir()
	if err != nil {
		t.Fatalf("paths.DataDir: %v", err)
	}
	wantSuffix := filepath.Join("whisper-voice-util")
	if !strings.HasSuffix(got, wantSuffix) {
		t.Errorf("paths.DataDir() = %q, want suffix %q", got, wantSuffix)
	}
	if !strings.HasPrefix(got, hostData) {
		t.Errorf("paths.DataDir() = %q, want prefix %q", got, hostData)
	}
}

// TestOpenDataDir_NilNotifierNoPanic confirms openDataDir's failure
// branch (where it would normally call a.Notifier.Error) is a
// no-op when the notifier is nil — preventing a regression where a
// partially-constructed Application crashes the tray click handler.
func TestOpenDataDir_NilNotifierNoPanic(t *testing.T) {
	hostData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", hostData)
	// Empty PATH so xdg-open cannot be resolved → openDataDir hits
	// the Start() error branch and tries to notify.
	t.Setenv("PATH", t.TempDir())

	// Silence the log output that openDataDir writes on the
	// failure path. We don't assert on the log line — only that
	// the function does not panic.
	prevOut := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(prevOut)

	a := newTestApplication()
	defer a.cancel()
	a.openDataDir()
}

// TestNotifySetupFinished_NoNotifierNoPanic confirms the helper
// tolerates a nil Notifier (used in tests and in degraded init).
func TestNotifySetupFinished_NoNotifierNoPanic(t *testing.T) {
	a := newTestApplication()
	defer a.cancel()
	a.notifySetupFinished() // must not panic
}

// TestNotifySetupError_NoNotifierNoPanic confirms the error helper
// tolerates a nil Notifier.
func TestNotifySetupError_NoNotifierNoPanic(t *testing.T) {
	a := newTestApplication()
	defer a.cancel()
	a.notifySetupError("test", &stringError{"synthetic"})
}

// stringError is a minimal error type for the nil-notifier test.
// Inline so the test does not pull in fmt / errors just for one
// sentinel.
type stringError struct{ s string }

func (e *stringError) Error() string { return e.s }
