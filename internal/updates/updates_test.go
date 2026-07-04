/* Code Map: Update Notifier — Tests
 * Coverage:
 *   - LatestRelease: parses a real GitHub response, 404 → ErrNoRelease,
 *     500 → typed error, network error → wrapped error.
 *   - IsNewer: table-driven over the semver comparison matrix,
 *     including the "dev" sentinel and missing components.
 *   - PickAsset: prefers linux-amd64 suffix; returns nil when no
 *     compatible asset is present.
 *   - Download: httptest-backed end-to-end fetch, verifies the staged
 *     file lands at the right path and matches the asset size.
 *
 * No real network calls — every test uses httptest.NewServer and
 * WVU_GITHUB_API_BASE to redirect the client to a local URL.
 *
 * CID Index:
 * CID:updates-test-001 -> TestLatestRelease_ParsesGitHubResponse
 * CID:updates-test-002 -> TestLatestRelease_404ReturnsErrNoRelease
 * CID:updates-test-003 -> TestLatestRelease_5xxReturnsTypedError
 * CID:updates-test-004 -> TestIsNewer_SemverCompare
 * CID:updates-test-005 -> TestPickAsset_LinuxAmd64
 * CID:updates-test-006 -> TestDownload_WritesToStagedPath
 * CID:updates-test-007 -> TestDownload_NoAssetReturnsError
 *
 * Quick lookup: rg -n "CID:updates-test-" internal/updates/
 */
package updates

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// withTestServer points the updates package at a local httptest server
// and returns the server URL. All tests that exercise LatestRelease
// must call this first (or set WVU_GITHUB_API_BASE themselves).
// Cleanup is handled via t.Cleanup.
func withTestServer(t *testing.T, h http.Handler) string {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	t.Setenv(envBaseURL, srv.URL)
	return srv.URL
}

// ghRelease is a minimal GitHub release JSON body, just enough for
// the parser. Tests can mutate fields before serializing.
type ghRelease struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	HTMLURL     string  `json:"html_url"`
	PublishedAt string  `json:"published_at"`
	Body        string  `json:"body"`
	Assets      []Asset `json:"assets"`
}

// CID:updates-test-001 - TestLatestRelease_ParsesGitHubResponse
// Purpose: a realistic GitHub release response round-trips through
// LatestRelease without loss of key fields.
func TestLatestRelease_ParsesGitHubResponse(t *testing.T) {
	want := ghRelease{
		TagName:     "v0.3.1",
		Name:        "Whisper Voice Utility 0.3.1",
		HTMLURL:     "https://github.com/spanexx/whisper-voice-util/releases/tag/v0.3.1",
		PublishedAt: "2026-07-04T10:00:00Z",
		Body:        "## Notes\n- bug fix",
		Assets: []Asset{{
			Name:               "whisper-voice-util-v0.3.1-linux-amd64.tar.gz",
			BrowserDownloadURL: "https://example.com/v0.3.1.tar.gz",
			ContentType:        "application/gzip",
			Size:               1234567,
		}},
	}
	withTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent = %q, want %q", r.Header.Get("User-Agent"), userAgent)
		}
		if r.Header.Get("Accept") == "" {
			t.Error("Accept header missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))

	got, err := LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if got.TagName != want.TagName {
		t.Errorf("TagName = %q, want %q", got.TagName, want.TagName)
	}
	if got.HTMLURL != want.HTMLURL {
		t.Errorf("HTMLURL = %q, want %q", got.HTMLURL, want.HTMLURL)
	}
	if len(got.Assets) != 1 {
		t.Fatalf("len(Assets) = %d, want 1", len(got.Assets))
	}
	if got.Assets[0].Name != want.Assets[0].Name {
		t.Errorf("Assets[0].Name = %q, want %q", got.Assets[0].Name, want.Assets[0].Name)
	}
	if got.Body != want.Body {
		t.Errorf("Body = %q, want %q", got.Body, want.Body)
	}
}

// CID:updates-test-002 - TestLatestRelease_404ReturnsErrNoRelease
// Purpose: GitHub returns 404 when the repo has no published releases.
// The caller treats this as "nothing to update to" and stays silent.
func TestLatestRelease_404ReturnsErrNoRelease(t *testing.T) {
	withTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))

	got, err := LatestRelease(context.Background())
	if err == nil {
		t.Fatalf("expected error, got release = %+v", got)
	}
	if got != nil {
		t.Errorf("expected nil release on 404, got %+v", got)
	}
	if err != ErrNoRelease {
		t.Errorf("err = %v, want ErrNoRelease", err)
	}
}

// CID:updates-test-003 - TestLatestRelease_5xxReturnsTypedError
// Purpose: 5xx and other non-2xx / non-404 responses surface as a
// typed error that includes the status. The caller (tray click
// handler) shows a notification like "Update check failed: 500".
func TestLatestRelease_5xxReturnsTypedError(t *testing.T) {
	withTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server overloaded"))
	}))

	got, err := LatestRelease(context.Background())
	if err == nil {
		t.Fatalf("expected error, got release = %+v", got)
	}
	if got != nil {
		t.Errorf("expected nil release on 5xx, got %+v", got)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("err = %q, want substring '500'", err.Error())
	}
}

// CID:updates-test-004 - TestIsNewer_SemverCompare
// Purpose: the semver comparison is the single most security-relevant
// piece of the updater — a wrong answer means the user is told to
// "downgrade" or to "stay" when an update is available. Table-driven
// coverage of the matrix below.
func TestIsNewer_SemverCompare(t *testing.T) {
	cases := []struct {
		name           string
		tag            string
		currentVersion string
		want           bool
	}{
		{"exact match not newer", "v0.2.0", "v0.2.0", false},
		{"patch bump", "v0.2.1", "v0.2.0", true},
		{"minor bump", "v0.3.0", "v0.2.9", true},
		{"major bump", "v1.0.0", "v0.99.99", true},
		{"older patch", "v0.2.0", "v0.2.1", false},
		{"older minor", "v0.1.9", "v0.2.0", false},
		{"dev build is never newer", "v99.0.0", "dev", false},
		{"empty current is never newer", "v1.0.0", "", false},
		{"empty tag is never newer", "", "v0.2.0", false},
		{"tag without v prefix", "0.3.0", "0.2.0", true},
		{"pre-release suffix ignored", "v0.3.0-rc1", "v0.2.0", true},
		{"missing patch treated as 0", "v0.3", "v0.2.5", true},
		{"two-digit version compared", "v0.3", "v0.2", true},
		{"garbage tag is not newer", "not-a-version", "v0.2.0", false},
		{"garbage current is not newer", "v0.3.0", "garbage", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &Release{TagName: c.tag}
			if got := r.IsNewer(c.currentVersion); got != c.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", c.tag, c.currentVersion, got, c.want)
			}
		})
	}
}

// CID:updates-test-005 - TestPickAsset_LinuxAmd64
// Purpose: PickAsset must return the linux/amd64 tarball when one is
// present, even if other assets (mac, win, source) come first.
func TestPickAsset_LinuxAmd64(t *testing.T) {
	r := &Release{
		Assets: []Asset{
			{Name: "whisper-voice-util-v0.2.0-darwin-amd64.tar.gz"},
			{Name: "whisper-voice-util-v0.2.0-windows-amd64.zip"},
			{Name: "whisper-voice-util-v0.2.0-linux-amd64.tar.gz"},
			{Name: "source.tar.gz"},
		},
	}
	got := r.PickAsset("linux", "amd64")
	if got == nil {
		t.Fatal("PickAsset returned nil; expected the linux/amd64 tarball")
	}
	if got.Name != "whisper-voice-util-v0.2.0-linux-amd64.tar.gz" {
		t.Errorf("got %q, want linux-amd64 tarball", got.Name)
	}
}

// Purpose: PickAsset returns nil when no matching asset is present.
// (Companion to the happy-path test above.)
func TestPickAsset_NoMatch(t *testing.T) {
	r := &Release{
		Assets: []Asset{
			{Name: "whisper-voice-util-v0.2.0-darwin-amd64.tar.gz"},
		},
	}
	if got := r.PickAsset("linux", "amd64"); got != nil {
		t.Errorf("PickAsset = %+v, want nil", got)
	}
}

// CID:updates-test-006 - TestDownload_WritesToStagedPath
// Purpose: end-to-end download from a fake GitHub-style server to the
// staged path. Verifies the file lands at <dest>.new with the right
// bytes and the right size.
func TestDownload_WritesToStagedPath(t *testing.T) {
	const payload = "fake-tarball-contents-1234567890"
	asset := Asset{
		Name:               "whisper-voice-util-v0.2.0-linux-amd64.tar.gz",
		BrowserDownloadURL: "", // set after server start
		ContentType:        "application/gzip",
		Size:               int64(len(payload)),
	}
	withTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", itoa(len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))
	// The fake server URL is the only URL we have; point the asset at it.
	// We re-derive the URL by reading the env we just set.
	asset.BrowserDownloadURL = os.Getenv(envBaseURL) + "/v0.2.0.tar.gz"
	r := &Release{TagName: "v0.2.0", Assets: []Asset{asset}}

	dest := filepath.Join(t.TempDir(), "whisper-voice-util")
	staged, err := r.Download(context.Background(), dest)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	wantStaged := dest + updateFileSuffix
	if staged != wantStaged {
		t.Errorf("staged = %q, want %q", staged, wantStaged)
	}
	got, err := os.ReadFile(staged)
	if err != nil {
		t.Fatalf("read staged: %v", err)
	}
	if string(got) != payload {
		t.Errorf("staged contents = %q, want %q", string(got), payload)
	}
}

// CID:updates-test-007 - TestDownload_NoAssetReturnsError
// Purpose: a release with no linux/amd64 asset must fail Download
// with a clear error mentioning the tag.
func TestDownload_NoAssetReturnsError(t *testing.T) {
	r := &Release{
		TagName: "v0.2.0",
		Assets: []Asset{
			{Name: "source.tar.gz"},
		},
	}
	_, err := r.Download(context.Background(), filepath.Join(t.TempDir(), "bin"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "v0.2.0") {
		t.Errorf("err = %q, want substring 'v0.2.0'", err.Error())
	}
	if !strings.Contains(err.Error(), "linux/amd64") {
		t.Errorf("err = %q, want substring 'linux/amd64'", err.Error())
	}
}

// TestIsNewer_NilReceiver confirms a nil Release is a safe no-op
// (used by the tray flow that may not have a release yet).
func TestIsNewer_NilReceiver(t *testing.T) {
	var r *Release
	if got := r.IsNewer("v0.2.0"); got {
		t.Errorf("nil.IsNewer(v0.2.0) = true, want false")
	}
}

// TestLatestRelease_ContextCancelled confirms the context reaches the
// HTTP client — cancels during the request return ctx.Err.
func TestLatestRelease_ContextCancelled(t *testing.T) {
	withTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := LatestRelease(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "context") &&
		!strings.Contains(err.Error(), "deadline") {
		t.Errorf("err = %q, want context/deadline mention", err.Error())
	}
}

// itoa is a tiny stdlib-free int→string helper used in the download
// test. We avoid strconv here to keep the test imports tight.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
