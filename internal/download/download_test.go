package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// CID:download-test-001 - test helpers
// makePayload returns n bytes of deterministic, repeatable content
// (0,1,2,...,255,0,1,...). Tests use the SHA-256 of this payload as
// the expected hash.
func makePayload(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i % 256)
	}
	return b
}

func hashHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// freshServer spins up an httptest.Server that responds to every request
// with status + body. Sets Content-Length explicitly so the downloader
// can report a meaningful final progress tick.
func freshServer(t *testing.T, status int, body []byte) (string, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(body) > 0 {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		}
		w.WriteHeader(status)
		if len(body) > 0 {
			_, _ = w.Write(body)
		}
	}))
	return srv.URL, srv.Close
}

// flakyServer returns a server that responds with `failStatus` for the
// first `failCount` Download attempts (each attempt = 1 HEAD + 1 GET),
// then with `successStatus` and `body` thereafter. The internal counter
// is the current attempt number, incremented only when a GET completes
// the attempt; HEAD requests read but do not increment, so HEAD and GET
// of the same attempt see the same number.
func flakyServer(t *testing.T, failCount int, failStatus int, successStatus int, body []byte) (string, func()) {
	t.Helper()
	var attemptNum int32 // 0-indexed: current attempt
	var getCount int32    // total GETs received (for tests that count)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.LoadInt32(&attemptNum)
		if r.Method == http.MethodGet {
			atomic.AddInt32(&getCount, 1)
			// Increment the attempt counter only on the GET that
			// completes the attempt, so HEAD + GET of the same
			// attempt share n.
			defer atomic.AddInt32(&attemptNum, 1)
		}
		if n < int32(failCount) {
			w.WriteHeader(failStatus)
			return
		}
		if len(body) > 0 {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		}
		w.WriteHeader(successStatus)
		_, _ = w.Write(body)
	}))
	return srv.URL, srv.Close
}

// progressRecorder counts progress events and remembers the last one.
type progressRecorder struct {
	calls int
	lastF float64
	lastD int64
	lastT int64
}

func (p *progressRecorder) fn(fraction float64, done, total int64) {
	p.calls++
	p.lastF = fraction
	p.lastD = done
	p.lastT = total
}

// CID:download-test-002 - TestDownload_OneMegabytePayload
// Purpose: a 1 MB payload downloads successfully, the resulting file
// matches the payload bytes, and the final progress event reports
// fraction=1.0 with the correct totals.
func TestDownload_OneMegabytePayload(t *testing.T) {
	const size = 1 << 20 // 1 MiB
	payload := makePayload(size)
	url, stop := freshServer(t, http.StatusOK, payload)
	defer stop()

	dest := filepath.Join(t.TempDir(), "out.bin")
	rec := &progressRecorder{}

	if err := Download(context.Background(), url, dest, hashHex(payload), rec.fn); err != nil {
		t.Fatalf("Download: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if len(got) != size {
		t.Errorf("file size = %d, want %d", len(got), size)
	}
	for i := range got {
		if got[i] != payload[i] {
			t.Errorf("byte %d = %d, want %d", i, got[i], payload[i])
			break
		}
	}

	if rec.calls == 0 {
		t.Fatalf("progress was never called")
	}
	if rec.lastF != 1.0 {
		t.Errorf("final fraction = %v, want 1.0", rec.lastF)
	}
	if rec.lastD != int64(size) || rec.lastT != int64(size) {
		t.Errorf("final bytes done=%d total=%d, want %d/%d", rec.lastD, rec.lastT, size, size)
	}
}

// CID:download-test-003 - TestDownload_ProgressFrequencyOnOneMB
// Purpose: 1 MB / 256 KB = 4 interval events, plus the final 1.0 tick
// from doAttempt, plus the initial 0.0 tick. So at least 4 events.
func TestDownload_ProgressFrequencyOnOneMB(t *testing.T) {
	const size = 1 << 20
	payload := makePayload(size)
	url, stop := freshServer(t, http.StatusOK, payload)
	defer stop()

	dest := filepath.Join(t.TempDir(), "out.bin")
	rec := &progressRecorder{}

	if err := Download(context.Background(), url, dest, "", rec.fn); err != nil {
		t.Fatalf("Download: %v", err)
	}

	if rec.calls < 4 {
		t.Errorf("progress calls = %d, want >= 4", rec.calls)
	}
}

// CID:download-test-004 - TestDownload_404NoRetry
// Purpose: a 404 must fail with an error that mentions the HTTP status
// and must not be retried. We count GETs (one per Download attempt);
// HEADs are a separate request but do not count as retries.
func TestDownload_404NoRetry(t *testing.T) {
	var getCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			atomic.AddInt32(&getCount, 1)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	err := Download(context.Background(), srv.URL, dest, "", NopProgress)
	if err == nil {
		t.Fatalf("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error %q should mention 404", err.Error())
	}
	if got := atomic.LoadInt32(&getCount); got != 1 {
		t.Errorf("GET attempts = %d, want 1 (no retry on 4xx)", got)
	}
}

// CID:download-test-005 - TestDownload_500Then200SucceedsAfterRetry
// Purpose: two 500s followed by a 200 must succeed. The test runs in
// ~3 s (1 s + 2 s of backoff). Uses a short payload to keep the file
// copy trivial.
func TestDownload_500Then200SucceedsAfterRetry(t *testing.T) {
	payload := makePayload(1024)
	url, stop := flakyServer(t, 2 /*failCount*/, http.StatusInternalServerError, http.StatusOK, payload)
	defer stop()

	dest := filepath.Join(t.TempDir(), "out.bin")
	rec := &progressRecorder{}

	start := time.Now()
	if err := Download(context.Background(), url, dest, hashHex(payload), rec.fn); err != nil {
		t.Fatalf("Download: %v", err)
	}
	elapsed := time.Since(start)

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if len(got) != len(payload) {
		t.Errorf("size = %d, want %d", len(got), len(payload))
	}

	// Sanity: at least 1s+2s of backoff elapsed (a tiny bit more for
	// jitter, but we don't add any).
	if elapsed < 3*time.Second {
		t.Errorf("elapsed = %v, want >= 3s for two retries", elapsed)
	}
	if rec.lastF != 1.0 {
		t.Errorf("final fraction = %v, want 1.0", rec.lastF)
	}
}

// CID:download-test-006 - TestDownload_GivesUpAfterThreeAttempts
// Purpose: a permanently-500 endpoint exhausts retries and returns
// the underlying error. HEAD requests do not count as attempts.
func TestDownload_GivesUpAfterThreeAttempts(t *testing.T) {
	var getCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			atomic.AddInt32(&getCount, 1)
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	err := Download(context.Background(), srv.URL, dest, "", NopProgress)
	if err == nil {
		t.Fatalf("expected error after exhausting retries, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error %q should mention 500", err.Error())
	}
	if got := atomic.LoadInt32(&getCount); got != int32(maxAttempts) {
		t.Errorf("GET attempts = %d, want %d", got, maxAttempts)
	}
}

// CID:download-test-007 - TestDownload_SHA256Mismatch
// Purpose: when the expected hash does not match the payload, the
// download is treated as a permanent failure and the .partial file
// is removed (corrupt data, no point keeping it for resume).
func TestDownload_SHA256Mismatch(t *testing.T) {
	payload := makePayload(2048)
	wrongHash := strings.Repeat("0", 64) // valid hex, wrong value
	url, stop := freshServer(t, http.StatusOK, payload)
	defer stop()

	dest := filepath.Join(t.TempDir(), "out.bin")
	err := Download(context.Background(), url, dest, wrongHash, NopProgress)
	if err == nil {
		t.Fatalf("expected sha256 mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "sha256") {
		t.Errorf("error %q should mention sha256", err.Error())
	}

	// dest must not exist (rename never happened).
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("dest should not exist after hash mismatch, stat err = %v", err)
	}
	// .partial must be removed for the same reason.
	partial := dest + partialSuffix
	if _, err := os.Stat(partial); !os.IsNotExist(err) {
		t.Errorf(".partial should be removed after hash mismatch, stat err = %v", err)
	}
}

// CID:download-test-008 - TestDownload_EmptyArgsRejected
// Purpose: missing url or destPath is a usage error, not a retryable
// network failure.
func TestDownload_EmptyArgsRejected(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		dest    string
		wantSub string
	}{
		{"empty url", "", "/tmp/x", "url"},
		{"empty dest", "http://example.invalid/", "", "destPath"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Download(context.Background(), c.url, c.dest, "", NopProgress)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), c.wantSub) {
				t.Errorf("error %q should mention %q", err.Error(), c.wantSub)
			}
		})
	}
}

// CID:download-test-009 - TestDownload_NopProgressIsSafe
// Purpose: callers can pass a nil ProgressFunc and get a no-op.
// (Download assigns NopProgress internally; this test guards that
// behavior so a future refactor does not regress it.)
func TestDownload_NopProgressIsSafe(t *testing.T) {
	payload := makePayload(512)
	url, stop := freshServer(t, http.StatusOK, payload)
	defer stop()

	dest := filepath.Join(t.TempDir(), "out.bin")
	if err := Download(context.Background(), url, dest, "", nil); err != nil {
		t.Fatalf("Download: %v", err)
	}
}

// CID:download-test-010 - TestDownload_ContextCancelledDuringBackoff
// Purpose: cancelling ctx during the inter-attempt sleep should abort
// the retry loop promptly. We trigger this by configuring a server
// that always returns 500 and cancelling the context right after the
// first attempt completes.
func TestDownload_ContextCancelledDuringBackoff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Cancel after the first attempt should have started its backoff.
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	dest := filepath.Join(t.TempDir(), "out.bin")
	start := time.Now()
	err := Download(ctx, srv.URL, dest, "", NopProgress)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errorsIsCancelled(err) {
		t.Errorf("error %q should be ctx cancellation", err.Error())
	}
	if elapsed > 1500*time.Millisecond {
		t.Errorf("elapsed = %v, want < 1.5s (cancel during first backoff)", elapsed)
	}
}

// CID:download-test-011 - TestDownload_PartialFileLeftOnPermanentFailure
// Purpose: a 404 leaves no .partial behind (we never wrote one) and
// no dest behind (rename never ran). This documents the contract
// even though it's also covered by the 404 test.
func TestDownload_PartialFileLeftOnPermanentFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	_ = Download(context.Background(), srv.URL, dest, "", NopProgress)

	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("dest should not exist on 4xx")
	}
	partial := dest + partialSuffix
	if _, err := os.Stat(partial); !os.IsNotExist(err) {
		t.Errorf(".partial should not exist on 4xx (never created)")
	}
}

// CID:download-test-012 - TestDownload_WritesUnderDestDir
// Purpose: a deeply-nested dest path is created via MkdirAll so the
// downloader works for paths the caller has not pre-created.
func TestDownload_WritesUnderDestDir(t *testing.T) {
	payload := makePayload(64)
	url, stop := freshServer(t, http.StatusOK, payload)
	defer stop()

	deep := filepath.Join(t.TempDir(), "a", "b", "c", "model.bin")
	if err := Download(context.Background(), url, deep, "", NopProgress); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if _, err := os.Stat(deep); err != nil {
		t.Errorf("stat deep dest: %v", err)
	}
}

// CID:download-test-013 - TestDownload_HTTPStatusErrorMessage
// Purpose: the typed httpStatusError produces a message that contains
// both the numeric code and the status text, so logs and the wizard
// can show something useful.
func TestDownload_HTTPStatusErrorMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	err := Download(context.Background(), srv.URL, dest, "", NopProgress)
	if err == nil {
		t.Fatalf("expected error for 403")
	}
	msg := err.Error()
	if !strings.Contains(msg, "403") {
		t.Errorf("error %q should contain 403", msg)
	}
	if !strings.Contains(msg, "Forbidden") {
		t.Errorf("error %q should contain status text", msg)
	}
}

// errorsIsCancelled is a local helper so we don't import errors just
// for one call. The wrapping in Download means we need errors.Is.
func errorsIsCancelled(err error) bool {
	if err == nil {
		return false
	}
	return err == context.Canceled || strings.Contains(err.Error(), context.Canceled.Error()) ||
		strings.Contains(err.Error(), "context canceled")
}

// CID:download-test-014 - TestDownload_FinalProgressTickOnSmallFile
// Purpose: a payload smaller than the progress interval still gets
// the 1.0 final tick from doAttempt. This guards the manual
// `progress(1.0, total, total)` call at the end of doAttempt.
func TestDownload_FinalProgressTickOnSmallFile(t *testing.T) {
	payload := makePayload(64) // < 256 KB
	url, stop := freshServer(t, http.StatusOK, payload)
	defer stop()

	dest := filepath.Join(t.TempDir(), "out.bin")
	rec := &progressRecorder{}
	if err := Download(context.Background(), url, dest, "", rec.fn); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if rec.calls == 0 {
		t.Fatalf("no progress events fired")
	}
	if rec.lastF != 1.0 {
		t.Errorf("final fraction = %v, want 1.0", rec.lastF)
	}
}

// CID:download-test-015 - TestDownload_StreamedHashMatchesFileHash
// Purpose: a paranoid check that streaming the hash during io.Copy
// produces the same digest as hashing the resulting file on disk.
func TestDownload_StreamedHashMatchesFileHash(t *testing.T) {
	payload := makePayload(8192)
	url, stop := freshServer(t, http.StatusOK, payload)
	defer stop()

	dest := filepath.Join(t.TempDir(), "out.bin")
	if err := Download(context.Background(), url, dest, hashHex(payload), NopProgress); err != nil {
		t.Fatalf("Download: %v", err)
	}

	// Re-hash the file and assert the streamed digest is reproducible.
	f, err := os.Open(dest)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != hashHex(payload) {
		t.Errorf("file hash %s != payload hash %s", got, hashHex(payload))
	}
}

// CID:download-test-016 - sanity guard
// Makes the import of "fmt" survive the no-error test path so a
// future test can use it without an import churn.
var _ = fmt.Sprintf
