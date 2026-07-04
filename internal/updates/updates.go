/* Code Map: Update Notifier — GitHub Releases Client
 * Files in this package:
 *   updates.go   - Release model, LatestRelease, asset pick/download
 *   semver.go    - IsNewer, parseSemver
 *   download.go  - Download (HTTP asset fetch to staged path)
 *   restart.go   - syscall.Exec helper that replaces the running process
 *
 * CID Index:
 * CID:updates-001 -> Release
 * CID:updates-002 -> Asset
 * CID:updates-003 -> ErrNoRelease
 * CID:updates-004 -> LatestRelease
 * CID:updates-005 -> IsNewer             (semver.go)
 * CID:updates-006 -> Download            (download.go)
 * CID:updates-007 -> Restart             (restart.go)
 * CID:updates-008 -> StagedPath          (restart.go)
 *
 * Quick lookup: rg -n "CID:updates-" internal/updates/
 */
package updates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// defaultRequestTimeout is the per-request HTTP timeout for the GitHub API.
// Per IMPL §7: "5-second timeout."
const defaultRequestTimeout = 5 * time.Second

// Env vars that override the default GitHub owner/repo. The defaults
// point at the upstream public repo; CI or local builds can override
// without recompiling. Phase 8 will freeze these via -ldflags.
const (
	defaultOwner = "spanexx"
	defaultRepo  = "whisper-voice-util"
	envOwner     = "WVU_GITHUB_OWNER"
	envRepo      = "WVU_GITHUB_REPO"
	envBaseURL   = "WVU_GITHUB_API_BASE" // override for tests / GitHub Enterprise
)

// CID:updates-001 - Release
// Purpose: A subset of the GitHub Releases API `release` object — only
// the fields we actually consume. Parsed from the JSON body in
// LatestRelease. The struct is exported so callers can inspect
// TagName / HTMLURL after a check.
type Release struct {
	TagName     string  `json:"tag_name"`     // e.g. "v0.2.0"
	Name        string  `json:"name"`         // human label, e.g. "Whisper Voice Utility 0.2.0"
	HTMLURL     string  `json:"html_url"`     // link to the release page
	PublishedAt string  `json:"published_at"` // ISO 8601; kept as string to avoid TZ surprises
	Assets      []Asset `json:"assets"`
	// Body is the release notes (markdown). Truncated in tray notifications.
	Body string `json:"body"`
}

// CID:updates-002 - Asset
// Purpose: One downloadable file attached to a release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
	Size               int64  `json:"size"`
}

// CID:updates-003 - ErrNoRelease
// Purpose: Sentinel returned by LatestRelease when GitHub has no releases
// yet (HTTP 404). Callers can ignore it silently; other errors are
// unexpected and should be surfaced.
var ErrNoRelease = errors.New("updates: no release published yet")

// apiBase returns the GitHub API base URL. Default is the public
// api.github.com; WVU_GITHUB_API_BASE overrides for tests / GHES.
func apiBase() string {
	if v := os.Getenv(envBaseURL); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://api.github.com"
}

// OwnerRepo returns the configured GitHub owner/repo. Reads
// WVU_GITHUB_OWNER / WVU_GITHUB_REPO; falls back to the upstream
// defaults. Exported so tests and the build pipeline can read the
// same values the check uses.
func OwnerRepo() (owner, repo string) {
	owner = os.Getenv(envOwner)
	if owner == "" {
		owner = defaultOwner
	}
	repo = os.Getenv(envRepo)
	if repo == "" {
		repo = defaultRepo
	}
	return owner, repo
}

// SetOwnerRepo overrides the configured owner/repo for the current
// process. Used by tests; process-wide side-effect, not safe to call
// from goroutines. Production code should rely on env vars.
func SetOwnerRepo(owner, repo string) {
	_ = os.Setenv(envOwner, owner)
	_ = os.Setenv(envRepo, repo)
}

// releaseURL returns the absolute GitHub API URL for the latest
// release of the configured owner/repo.
func releaseURL() string {
	owner, repo := OwnerRepo()
	return fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBase(), owner, repo)
}

// userAgent is the UA string the App sends. GitHub requires a UA for
// API calls (per their docs); "go-github" is a conventional value.
const userAgent = "whisper-voice-util"

// CID:updates-004 - LatestRelease
// Purpose: GET GitHub's "latest release" endpoint, parse the JSON
// response into a Release, and return it. 5 s timeout per IMPL §7.
//
// Special cases:
//   - HTTP 404 → return (nil, ErrNoRelease). The repo exists but has
//     no published releases yet. Callers ignore this.
//   - any other non-2xx → return a typed error mentioning the status.
//   - network / parse error → return as-is; caller logs and moves on.
//
// baseURL override is provided for tests via WVU_GITHUB_API_BASE;
// releaseURL() reads it.
func LatestRelease(ctx context.Context) (*Release, error) {
	url := releaseURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("updates: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: defaultRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("updates: GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through
	case http.StatusNotFound:
		return nil, ErrNoRelease
	default:
		// Drain a small prefix of the body for the error message so the
		// log line tells the user what GitHub said (rate-limit hint,
		// permissions, etc.).
		prefix, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("updates: GET %s: %s: %s",
			url, resp.Status, strings.TrimSpace(string(prefix)))
	}

	var r Release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("updates: parse %s: %w", url, err)
	}
	return &r, nil
}
