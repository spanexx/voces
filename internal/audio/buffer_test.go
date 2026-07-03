package audio

import (
	"testing"
)

func TestRingBuffer_Operations(t *testing.T) {
	b := NewRingBuffer(5)
	if b.Cap() != 5 {
		t.Errorf("Expected capacity 5, got %d", b.Cap())
	}

	// 1. Initial read should fail
	p := make([]byte, 2)
	_, err := b.Read(p)
	if err == nil {
		t.Error("Read on empty buffer should fail")
	}

	// 2. Write and Read
	b.Write([]byte{1, 2, 3})
	if b.Len() != 3 {
		t.Errorf("Expected length 3, got %d", b.Len())
	}

	n, _ := b.Read(p)
	if n != 2 || p[0] != 1 || p[1] != 2 {
		t.Errorf("Incorrect read result: %v", p)
	}

	// 3. Overwrite
	b.Write([]byte{4, 5, 6})
	// Original: [1, 2, 3] -> adding [4, 5, 6] -> [6, 2, 3, 4, 5] (circular)
	// Start was at 0. After reading 0,1, start is at 0? No, Read doesn't increment start!
	// Looking at Read code: it uses (b.start+i)%b.capacity. It doesn't modify start.
	// So Read is more like Peek.

	all := b.ReadAll()
	// Count is 5 (maxed out). start should have moved.
	// Write 1, 2, 3 -> count 3, start 0, end 3
	// Write 4, 5 -> count 5, start 0, end 0
	// Write 6 -> count 5, start 1, end 1
	if b.Len() != 5 {
		t.Errorf("Expected maxed length 5, got %d", b.Len())
	}
	if len(all) != 5 || all[0] != 2 {
		t.Errorf("Incorrect ReadAll: %v", all)
	}

	// 4. Clear
	b.Clear()
	if b.Len() != 0 {
		t.Error("Clear failed")
	}
}

func TestBufferManager_Lifecycle(t *testing.T) {
	m := NewBufferManager(2, 10)

	if m.GetCurrent() != nil {
		t.Error("Initial current buffer should be nil")
	}

	b1 := m.Allocate()
	if b1 == nil || b1.Cap() != 10 {
		t.Error("Allocation failed")
	}

	if m.GetCurrent() != b1 {
		t.Error("GetCurrent did not return allocated buffer")
	}

	b2 := m.Allocate()
	if b2 == nil || len(m.buffers) != 2 {
		t.Error("Allocation or pooling failed")
	}

	b3 := m.Allocate()
	if len(m.buffers) != 2 {
		t.Error("Should not exceed maxBuffers")
	}
	if m.GetCurrent() != b3 {
		t.Error("GetCurrent should return the latest allocated buffer even if not pooled")
	}

	m.ReleaseAll()
	if m.GetCurrent() != nil || len(m.buffers) != 0 {
		t.Error("ReleaseAll failed")
	}
}
