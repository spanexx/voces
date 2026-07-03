/* Code Map: Audio Buffers
 * - RingBuffer: Circular byte buffer for streaming audio
 * - NewRingBuffer: Factory for creating a RingBuffer
 * - BufferManager: Lifecycle control for multiple buffers
 * - NewBufferManager: Factory for creating a BufferManager
 *
 * CID Index:
 * CID:audio-buffer-001 -> RingBuffer
 * CID:audio-buffer-002 -> NewRingBuffer
 * CID:audio-buffer-003 -> BufferManager
 * CID:audio-buffer-004 -> NewBufferManager
 *
 * Quick lookup: rg -n "CID:audio-buffer-" internal/audio/buffer.go
 */
package audio

import (
	"fmt"
	"sync"
)

// CID:audio-buffer-001 - RingBuffer
// Purpose: Thread-safe circular buffer for storing transient audio data.
type RingBuffer struct {
	data     []byte
	size     int
	start    int
	end      int
	count    int
	capacity int
	mu       sync.RWMutex
}

// CID:audio-buffer-002 - NewRingBuffer
// Purpose: Initializes a ring buffer with a fixed capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		data:     make([]byte, capacity),
		capacity: capacity,
	}
}

// Write appends data to the buffer. If the buffer is full, oldest data is overwritten.
func (b *RingBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	written := 0
	for _, byte := range data {
		b.data[b.end] = byte
		b.end = (b.end + 1) % b.capacity
		if b.count < b.capacity {
			b.count++
		} else {
			b.start = (b.start + 1) % b.capacity
		}
		written++
	}
	return written, nil
}

// Read reads data from the buffer.
func (b *RingBuffer) Read(p []byte) (int, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.count == 0 {
		return 0, fmt.Errorf("buffer empty")
	}

	n := 0
	for i := 0; i < len(p) && i < b.count; i++ {
		p[i] = b.data[(b.start+i)%b.capacity]
		n++
	}
	return n, nil
}

// ReadAll returns all data currently in the buffer.
func (b *RingBuffer) ReadAll() []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]byte, b.count)
	for i := 0; i < b.count; i++ {
		result[i] = b.data[(b.start+i)%b.capacity]
	}
	return result
}

// Clear resets the buffer.
func (b *RingBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.start = 0
	b.end = 0
	b.count = 0
}

// Len returns the number of bytes currently in the buffer.
func (b *RingBuffer) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.count
}

// Cap returns the total capacity of the buffer.
func (b *RingBuffer) Cap() int {
	return b.capacity
}

// CID:audio-buffer-003 - BufferManager
// Purpose: Maintains a pool of ring buffers for efficient memory reuse.
type BufferManager struct {
	buffers    []*RingBuffer
	currentBuf *RingBuffer
	maxBuffers int
	bufSize    int
	mu         sync.Mutex
}

// CID:audio-buffer-004 - NewBufferManager
// Purpose: Initializes a buffer manager with pool size constraints.
func NewBufferManager(maxBuffers, bufSize int) *BufferManager {
	return &BufferManager{
		buffers:    make([]*RingBuffer, 0, maxBuffers),
		maxBuffers: maxBuffers,
		bufSize:    bufSize,
	}
}

// Allocate allocates a new buffer.
func (m *BufferManager) Allocate() *RingBuffer {
	m.mu.Lock()
	defer m.mu.Unlock()

	buf := NewRingBuffer(m.bufSize)
	if len(m.buffers) < m.maxBuffers {
		m.buffers = append(m.buffers, buf)
	}
	m.currentBuf = buf
	return buf
}

// GetCurrent returns the current buffer.
func (m *BufferManager) GetCurrent() *RingBuffer {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentBuf
}

// ReleaseAll releases all buffers.
func (m *BufferManager) ReleaseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buffers = make([]*RingBuffer, 0, m.maxBuffers)
	m.currentBuf = nil
}
