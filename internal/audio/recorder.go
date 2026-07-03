/* Code Map: Audio Recorder
 * - Recorder: Captures audio from system microphone
 * - NewRecorder: Factory for creating a Recorder
 * - Record: Orchestrates the recording process
 *
 * CID Index:
 * CID:audio-recorder-001 -> Recorder
 * CID:audio-recorder-002 -> NewRecorder
 * CID:audio-recorder-003 -> Record
 *
 * Quick lookup: rg -n "CID:audio-recorder-" internal/audio/recorder.go
 */
package audio

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// CID:audio-recorder-001 - Recorder
// Purpose: Manages microphone access and recording state using arecord.
type Recorder struct {
	device      string
	sampleRate  int
	channels    int
	maxDuration time.Duration
	isRecording bool

	mu            sync.Mutex
	cmd           *exec.Cmd
	cancel        context.CancelFunc
	stopRequested bool
}

// CID:audio-recorder-002 - NewRecorder
// Purpose: Initializes a recorder with default audio parameters.
func NewRecorder() *Recorder {
	return &Recorder{
		sampleRate:  16000,
		channels:    1,
		maxDuration: 5 * time.Minute,
	}
}

// CID:audio-recorder-003 - Record
// Purpose: Captures a fixed duration of audio and returns it as WAV data.
func (r *Recorder) Record(durationSeconds int) ([]byte, error) {
	if durationSeconds <= 0 {
		return nil, fmt.Errorf("duration must be positive")
	}

	// Limit duration to maxDuration
	duration := time.Duration(durationSeconds) * time.Second
	if duration > r.maxDuration {
		duration = r.maxDuration
	}

	// Create temp file for recording
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("recording-%d.wav", time.Now().UnixNano()))

	// Build arecord command
	// -f cd: 16-bit, little-endian, mono
	// -r 16000: 16kHz sample rate
	// -c 1: mono channel
	// -d: duration in seconds
	args := []string{
		"-f", "cd",
		"-r", fmt.Sprintf("%d", r.sampleRate),
		"-c", fmt.Sprintf("%d", r.channels),
		"-d", fmt.Sprintf("%d", durationSeconds),
		"-t", "wav",
		tmpFile,
	}

	if r.device != "" {
		args = append([]string{"-D", r.device}, args...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration+10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "arecord", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	r.mu.Lock()
	r.isRecording = true
	r.cmd = cmd
	r.cancel = cancel
	r.stopRequested = false
	r.mu.Unlock()

	startErr := cmd.Start()
	if startErr != nil {
		r.mu.Lock()
		r.isRecording = false
		r.cmd = nil
		r.cancel = nil
		r.mu.Unlock()
		os.Remove(tmpFile)
		return nil, fmt.Errorf("arecord failed to start: %w, stderr: %s", startErr, stderr.String())
	}

	err := cmd.Wait()

	r.mu.Lock()
	stopped := r.stopRequested
	r.isRecording = false
	r.cmd = nil
	r.cancel = nil
	r.stopRequested = false
	r.mu.Unlock()

	if err != nil {
		// If we intentionally stopped the recording, arecord typically exits non-zero due to signal.
		// Treat this as success if a file was produced.
		if !(stopped && fileExistsNonEmpty(tmpFile)) {
			// Clean up temp file on error
			os.Remove(tmpFile)
			if ctx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("recording timed out")
			}
			return nil, fmt.Errorf("arecord failed: %w, stderr: %s", err, stderr.String())
		}
	}

	// Read the WAV file
	audioData, err := os.ReadFile(tmpFile)
	if err != nil {
		os.Remove(tmpFile)
		return nil, fmt.Errorf("failed to read recorded audio: %w", err)
	}

	// Clean up temp file
	os.Remove(tmpFile)

	return audioData, nil
}

// Stop stops the current recording.
func (r *Recorder) Stop() {
	r.mu.Lock()
	r.stopRequested = true
	cmd := r.cmd
	cancel := r.cancel
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGINT)
		// If it doesn't stop quickly, force kill.
		go func(p *os.Process) {
			t := time.NewTimer(800 * time.Millisecond)
			defer t.Stop()
			<-t.C
			_ = p.Kill()
		}(cmd.Process)
	}
}

// IsRecording returns true if currently recording.
func (r *Recorder) IsRecording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isRecording
}

func fileExistsNonEmpty(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	if st.IsDir() {
		return false
	}
	return st.Size() > 0
}
