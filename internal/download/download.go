/* Code Map: HTTP Downloader with Progress
 * Files in this package:
 *   download.go  - public API, constants, retry loop
 *   attempt.go   - one HTTP GET attempt
 *   readers.go   - counting + hashing io.Reader wrappers
 *   errors.go    - typed errors, retry classification, digest compare
 *
 * CID Index:
 * CID:download-001 -> ProgressFunc
 * CID:download-002 -> NopProgress
 * CID:download-003 -> Download
 *
 * Quick lookup: rg -n "CID:download-" internal/download/
 */
package download

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	// progressIntervalBytes is the granularity at which ProgressFunc fires.
	progressIntervalBytes int64 = 256 * 1024
	// maxAttempts caps the number of HTTP attempts per Download() call.
	maxAttempts = 3
	// initialBackoff is the wait before the second attempt. Doubles each time.
	initialBackoff = 1 * time.Second
	// partialSuffix is appended to destPath for the in-flight file.
	partialSuffix = ".partial"
)

// CID:download-001 - ProgressFunc
// Purpose: callback invoked during the download.
// fraction is 0.0-1.0 when total is known, -1 otherwise.
// bytesDone and total are -1 when the server did not provide a
// Content-Length header.
type ProgressFunc func(fraction float64, bytesDone, total int64)

// CID:download-002 - NopProgress
// Purpose: a ProgressFunc that discards every event. Useful as the
// default value when the caller does not care about progress.
var NopProgress ProgressFunc = func(float64, int64, int64) {}

// CID:download-003 - Download
// Purpose: fetch url to destPath with retry, progress, and optional
// SHA-256 verification. Writes to destPath+".partial" while in flight
// and renames to destPath on success. On permanent failure, the
// .partial file is left in place for manual recovery (resume is
// deferred per IMPL-public-setup §6).
// Retries up to 3 attempts with 1s/2s/4s backoff on transient errors
// (5xx, network). Does not retry on 4xx.
// If sha256Hex is non-empty, the downloaded bytes are verified against
// it; a mismatch is a permanent error and the .partial is removed.
// ctx is honored for both backoff sleeps and the in-flight request.
func Download(ctx context.Context, url, destPath, sha256Hex string, progress ProgressFunc) error {
	log.Printf("download: starting url=%s dest=%s", url, destPath)
	if url == "" {
		return errors.New("download: empty url")
	}
	if destPath == "" {
		return errors.New("download: empty destPath")
	}
	if progress == nil {
		progress = NopProgress
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("download: mkdir %s: %w", filepath.Dir(destPath), err)
	}

	partialPath := destPath + partialSuffix
	backoff := initialBackoff
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			log.Printf("download: retry %d/%d after backoff", attempt, maxAttempts)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			// Resume is deferred; start each retry from a clean file.
			_ = os.Remove(partialPath)
		}

		err := doAttempt(ctx, url, partialPath, sha256Hex, progress)
		if err == nil {
			log.Printf("download: success, renaming to %s", destPath)
			return os.Rename(partialPath, destPath)
		}
		if !isRetryable(err) {
			log.Printf("download: permanent error: %v", err)
			return err
		}
		log.Printf("download: retryable error: %v", err)
		lastErr = err
	}
	return fmt.Errorf("download: gave up after %d attempts: %w", maxAttempts, lastErr)
}
