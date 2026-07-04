/* Code Map: Download Attempts
 * - doAttempt: one HTTP GET to the partial file path
 * - contentLength: HEAD probe for the server's Content-Length
 *
 * CID Index:
 * CID:download-004 -> doAttempt
 * CID:download-005 -> contentLength
 *
 * Quick lookup: rg -n "CID:download-" internal/download/attempt.go
 */
package download

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
)

// CID:download-004 - doAttempt
// Purpose: one HTTP GET to partialPath. Streams bytes through a hash
// (if sha256Hex is non-empty) and a counter, calling progress every
// progressIntervalBytes. Returns nil on success, a typed error
// otherwise. The caller decides whether the error is retryable.
func doAttempt(ctx context.Context, url, partialPath, sha256Hex string, progress ProgressFunc) error {
	var total int64 = -1
	if cl, err := contentLength(ctx, url); err == nil && cl > 0 {
		total = cl
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("download: build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &httpStatusError{Status: resp.StatusCode, Msg: resp.Status}
	}
	if cl := resp.ContentLength; cl > 0 {
		total = cl
	}

	f, err := os.Create(partialPath)
	if err != nil {
		return fmt.Errorf("download: create %s: %w", partialPath, err)
	}

	hasher := sha256.New()
	// Wrap body so we hash as we copy.
	src := io.Reader(resp.Body)
	if sha256Hex != "" {
		src = &hashingReader{r: resp.Body, h: hasher}
	}
	cr := &countingReader{r: src, total: total, fn: progress, interval: progressIntervalBytes}

	if _, err := io.Copy(f, cr); err != nil {
		_ = f.Close()
		return fmt.Errorf("download: copy body: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("download: close %s: %w", partialPath, err)
	}

	// Final progress tick so the UI hits 1.0 even on small files that
	// never cross a full interval.
	if progress != nil {
		if total > 0 {
			progress(1.0, total, total)
		} else {
			progress(-1, cr.bytes, -1)
		}
	}

	if sha256Hex != "" {
		if err := compareDigest(hasher, sha256Hex); err != nil {
			// Mismatch: drop the bad partial so resume does not reuse it.
			_ = os.Remove(partialPath)
			return err
		}
	}
	return nil
}

// CID:download-005 - contentLength
// Purpose: HEAD request to discover total size. Returns -1 on any
// non-success; the caller treats unknown size as "no progress bar".
func contentLength(ctx context.Context, url string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return -1, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return -1, &httpStatusError{Status: resp.StatusCode, Msg: resp.Status}
	}
	return resp.ContentLength, nil
}
