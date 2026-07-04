/* Code Map: Download Readers
 * - countingReader: wraps an io.Reader, tracks total bytes, fires progress
 * - hashingReader: wraps an io.Reader, updates a running hash on every Read
 *
 * CID Index:
 * CID:download-006 -> countingReader
 * CID:download-007 -> hashingReader
 *
 * Quick lookup: rg -n "CID:download-" internal/download/readers.go
 */
package download

import (
	"hash"
	"io"
)

// CID:download-006 - countingReader
// Purpose: wraps an io.Reader, tracks total bytes, and fires the
// progress callback each time we cross an interval boundary.
type countingReader struct {
	r        io.Reader
	total    int64
	bytes    int64
	next     int64
	fn       ProgressFunc
	interval int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.bytes += int64(n)
	for c.fn != nil && c.interval > 0 && c.bytes >= c.next {
		c.report()
		c.next += c.interval
	}
	return n, err
}

func (c *countingReader) report() {
	if c.total > 0 {
		c.fn(float64(c.bytes)/float64(c.total), c.bytes, c.total)
	} else {
		c.fn(-1, c.bytes, -1)
	}
}

// CID:download-007 - hashingReader
// Purpose: streams bytes through a hash.Hash so we verify SHA-256
// without a second pass over the file.
type hashingReader struct {
	r io.Reader
	h hash.Hash
}

func (h *hashingReader) Read(p []byte) (int, error) {
	n, err := h.r.Read(p)
	if n > 0 {
		_, _ = h.h.Write(p[:n])
	}
	return n, err
}
