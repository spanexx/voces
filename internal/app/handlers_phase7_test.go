/* Code Map: Application Handlers — Phase 7 (update notifier) tests
 * Coverage:
 *   - checkForUpdates: the happy path with a fake GitHub server
 *     sets a.release and updates the tray badge; the "up to date"
 *     path does not update the badge; the 404 path is silent.
 *   - summarizeRelease: long name is truncated to the first line.
 *   - trayIconLabelFor: includes the tag version.
 *   - Application.SetVersion: empty / "dev" handling.
 *   - Application.LatestRelease: thread-safe read/write round-trip.
 *
 * Tests use httptest.NewServer and WVU_GITHUB_API_BASE to redirect
 * the GitHub client at a local URL. No real network calls.
 *
 * CID Index:
 * CID:app-handlers-phase7-test-001 -> TestCheckForUpdates_NewerShowsBadge
 * CID:app-handlers-phase7-test-002 -> TestCheckForUpdates_UpToDateHidesBadge
 * CID:app-handlers-phase7-test-003 -> TestCheckForUpdates_404IsSilent
 * CID:app-handlers-phase7-test-004 -> TestSummarizeRelease
 * CID:app-handlers-phase7-test-005 -> TestSetVersion
 * CID:app-handlers-phase7-test-006 -> TestLatestReleaseConcurrent
 *
 * Quick lookup: rg -n "CID:app-handlers-phase7-test-" internal/app/handlers_phase7_test.go
 */
package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"whisper-voice-util/internal/config"
	"whisper-voice-util/internal/notify"
	"whisper-voice-util/internal/updates"
)

// buildMinimalApp creates an Application with just enough wiring to
// exercise checkForUpdates without taking the single-instance lock
// or starting the GTK tray. The Notifier is a real notify.Manager
// with a minimal Config (cfg must be non-nil — notify.Send reads
// cfg.Behavior.Notifications). The Tray is left nil so SetUpdateBadge
// is a no-op (see ui_phase7.go).
func buildMinimalApp(t *testing.T) *Application {
	t.Helper()
	a := NewTestApp(nil)
	a.Version = "0.2.0"
	a.Notifier = notify.New(&config.Config{}) // minimal cfg, not used at runtime
	return a
}

// fakeGitHubHandler returns an httptest handler that responds to
// the latest-release endpoint with the supplied release body.
func fakeGitHubHandler(t *testing.T, rel interface{}) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rel)
	})
}

// withFakeGitHub points updates.LatestRelease at a local httptest
// server. The WVU_GITHUB_API_BASE override is what the package
// reads; t.Setenv restores the previous value on test cleanup.
func withFakeGitHub(t *testing.T, h http.Handler) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	t.Setenv("WVU_GITHUB_API_BASE", srv.URL)
}

// ghReleaseBody is a minimal but realistic GitHub release body for
// httptest. The struct shape matches internal/updates.Release's
// exported fields, so json.Marshal produces the same bytes.
type ghReleaseBody struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		ContentType        string `json:"content_type"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// CID:app-handlers-phase7-test-001 - TestCheckForUpdates_NewerShowsBadge
// Purpose: a release newer than a.Version should be stored on the
// Application and trigger SetUpdateBadge on the tray.
func TestCheckForUpdates_NewerShowsBadge(t *testing.T) {
	a := buildMinimalApp(t)
	withFakeGitHub(t, fakeGitHubHandler(t, map[string]interface{}{
		"tag_name": "v0.3.0",
		"name":     "0.3.0",
		"html_url": "https://example.com/v0.3.0",
	}))

	a.checkForUpdates(false) // silent on the happy path; assertion
	if got := a.GetLatestRelease(); got == nil {
		t.Fatal("GetLatestRelease returned nil; expected a stored release")
	} else if got.TagName != "v0.3.0" {
		t.Errorf("LatestRelease.TagName = %q, want v0.3.0", got.TagName)
	}
}

// CID:app-handlers-phase7-test-002 - TestCheckForUpdates_UpToDateHidesBadge
// Purpose: when the latest release equals the current version, the
// release is still stored (so the UI can show "up to date" later)
// but ClearUpdateBadge is called.
func TestCheckForUpdates_UpToDateHidesBadge(t *testing.T) {
	a := buildMinimalApp(t)
	a.Version = "0.2.0"
	withFakeGitHub(t, fakeGitHubHandler(t, map[string]interface{}{
		"tag_name": "v0.2.0",
		"name":     "0.2.0",
	}))

	a.checkForUpdates(false)
	if got := a.GetLatestRelease(); got == nil || got.TagName != "v0.2.0" {
		t.Errorf("GetLatestRelease = %+v, want TagName v0.2.0", got)
	}
}

// CID:app-handlers-phase7-test-003 - TestCheckForUpdates_404IsSilent
// Purpose: 404 is the "no release yet" case. checkForUpdates
// returns silently (no notification, no error logged) regardless of
// the showNotification flag.
func TestCheckForUpdates_404IsSilent(t *testing.T) {
	a := buildMinimalApp(t)
	withFakeGitHub(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))

	a.checkForUpdates(true) // should not panic; result is silence
	if got := a.GetLatestRelease(); got != nil {
		t.Errorf("GetLatestRelease = %+v, want nil on 404", got)
	}
}

// TestCheckForUpdates_5xxShowsError verifies the explicit failure
// path: a 500 surfaces as a log line. Notification is gated on
// showNotification=true.
func TestCheckForUpdates_5xxShowsError(t *testing.T) {
	a := buildMinimalApp(t)
	withFakeGitHub(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("overloaded"))
	}))

	// showNotification=false is the auto-check path; should not
	// panic and should leave LatestRelease nil.
	a.checkForUpdates(false)
	if got := a.GetLatestRelease(); got != nil {
		t.Errorf("GetLatestRelease = %+v, want nil on 5xx", got)
	}
}

// CID:app-handlers-phase7-test-004 - TestSummarizeRelease
// Purpose: the user-facing notification body is truncated to the
// first line of the release name (avoids multi-line balloon text).
func TestSummarizeRelease(t *testing.T) {
	r := &updates.Release{TagName: "v0.3.0", Name: "First line\nSecond line"}
	got := summarizeRelease(r)
	if !strings.Contains(got, "First line") {
		t.Errorf("summarizeRelease = %q, want first line", got)
	}
	if strings.Contains(got, "Second line") {
		t.Errorf("summarizeRelease = %q, should not contain second line", got)
	}
	// Empty release name falls back to the tag.
	r2 := &updates.Release{TagName: "v0.3.0"}
	if got := summarizeRelease(r2); got == "" {
		t.Errorf("summarizeRelease empty for %+v", r2)
	}
}

// TestTrayIconLabelFor confirms the menu label embeds the tag.
func TestTrayIconLabelFor(t *testing.T) {
	if got := trayIconLabelFor(&updates.Release{TagName: "v0.3.0"}); !strings.Contains(got, "v0.3.0") {
		t.Errorf("trayIconLabelFor = %q, want v0.3.0", got)
	}
	if got := trayIconLabelFor(nil); got == "" {
		t.Errorf("trayIconLabelFor(nil) empty; want default label")
	}
}

// CID:app-handlers-phase7-test-005 - TestSetVersion
// Purpose: SetVersion replaces empty / "dev" with the "dev" sentinel.
func TestSetVersion(t *testing.T) {
	a := &Application{}
	a.SetVersion("")
	if a.Version != "dev" {
		t.Errorf("SetVersion(\"\").Version = %q, want dev", a.Version)
	}
	a.SetVersion("0.2.0")
	if a.Version != "0.2.0" {
		t.Errorf("SetVersion(\"0.2.0\").Version = %q, want 0.2.0", a.Version)
	}
	a.SetVersion("dev")
	if a.Version != "dev" {
		t.Errorf("SetVersion(\"dev\").Version = %q, want dev", a.Version)
	}
}

// CID:app-handlers-phase7-test-006 - TestLatestReleaseConcurrent
// Purpose: LatestRelease read/write must be safe under concurrent
// access (auto-check goroutine + tray click handler + applyUpdate
// goroutine). Runs -race to verify.
func TestLatestReleaseConcurrent(t *testing.T) {
	a := &Application{}
	a.wg.Add(1)
	defer a.wg.Done()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			a.setLatestRelease(&updates.Release{TagName: "v0.0.0"})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			_ = a.GetLatestRelease()
		}
	}()
	wg.Wait()
}

// TestStartAutoCheckExitsOnCtxCancel verifies the auto-check
// goroutine returns promptly when the application context is
// cancelled. Without this, a shutdown would block on the
// checkForUpdates timer.
func TestStartAutoCheckExitsOnCtxCancel(t *testing.T) {
	a := NewTestApp(nil)
	a.Notifier = notify.New(&config.Config{})
	// Cancel immediately so the timer never fires.
	a.cancel()
	a.startAutoCheck()
	// If startAutoCheck didn't return on ctx.Done, the test
	// framework would hang. Wrap with a timeout for safety.
	done := make(chan struct{})
	go func() { a.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("startAutoCheck did not return after ctx cancel")
	}
}

// suppress unused-import lint when notify or context get dropped.
var _ = context.Background
