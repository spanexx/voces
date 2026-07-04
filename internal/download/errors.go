/* Code Map: Download Errors
 * - httpStatusError: typed error carrying the HTTP status code
 * - isRetryable: classifies an error as retryable (5xx, network) or not (4xx)
 * - compareDigest: verifies a downloaded SHA-256 against a known-good hex digest
 *
 * CID Index:
 * CID:download-008 -> httpStatusError
 * CID:download-009 -> isRetryable
 * CID:download-010 -> compareDigest
 *
 * Quick lookup: rg -n "CID:download-" internal/download/errors.go
 */
package download

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"strings"
)

// CID:download-008 - httpStatusError
// Purpose: typed error carrying the HTTP status. Lets isRetryable
// distinguish 4xx (no retry) from 5xx (retry).
type httpStatusError struct {
	Status int
	Msg    string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("download: http %d %s", e.Status, e.Msg)
}

// CID:download-009 - isRetryable
// Purpose: classify an error as transient. 5xx and common network
// failures are retried. 4xx is permanent.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	var he *httpStatusError
	if errors.As(err, &he) {
		return he.Status >= 500
	}
	msg := err.Error()
	for _, needle := range []string{
		"connection refused",
		"no such host",
		"timeout",
		"EOF",
		"reset by peer",
		"broken pipe",
		"connection reset",
		"i/o timeout",
		"network is unreachable",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// CID:download-010 - compareDigest
// Purpose: compare an in-flight hash to an expected hex string.
// Case-insensitive so manifests can mix cases.
func compareDigest(h hash.Hash, expectedHex string) error {
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expectedHex) {
		return fmt.Errorf("download: sha256 mismatch: got %s, want %s", got, expectedHex)
	}
	return nil
}
