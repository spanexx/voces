/* Code Map: Audio Player
 * - Player: Handles audio playback via system commands
 * - NewPlayer: Factory for creating a Player
 * - PlayRaw: Plays uncompressed PCM data
 * - PlayMP3: Plays compressed MP3 data
 *
 * CID Index:
 * CID:audio-player-001 -> Player
 * CID:audio-player-002 -> NewPlayer
 * CID:audio-player-003 -> PlayRaw
 * CID:audio-player-004 -> PlayMP3
 *
 * Quick lookup: rg -n "CID:audio-player-" internal/audio/player.go
 */
package audio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

var ErrPlaybackStopped = errors.New("playback stopped")

// CID:audio-player-001 - Player
// Purpose: Orchestrates audio playback using system utilities like aplay or paplay.
type Player struct {
	outputDevice string

	mu            sync.Mutex
	cmd           *exec.Cmd
	cancel        context.CancelFunc
	stopRequested bool
}

// CID:audio-player-002 - NewPlayer
// Purpose: Initializes a new audio player.
func NewPlayer() *Player {
	return &Player{}
}

func (p *Player) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cmd != nil
}

func (p *Player) Stop() {
	p.mu.Lock()
	p.stopRequested = true
	cmd := p.cmd
	cancel := p.cancel
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func (p *Player) runCmd(cmd *exec.Cmd, cancel context.CancelFunc) error {
	p.mu.Lock()
	p.cmd = cmd
	p.cancel = cancel
	p.stopRequested = false
	p.mu.Unlock()

	err := cmd.Run()

	p.mu.Lock()
	stopped := p.stopRequested
	p.cmd = nil
	p.cancel = nil
	p.stopRequested = false
	p.mu.Unlock()

	if stopped {
		return ErrPlaybackStopped
	}
	return err
}

// CID:audio-player-003 - PlayRaw
// Purpose: Plays raw PCM audio data by piping it to aplay/paplay.
func (p *Player) PlayRaw(audioData []byte, sampleRate int) error {
	// Derive a timeout from the audio duration to avoid cutting off long TTS output.
	// PCM is s16le mono, so bytes per sample = 2.
	// Add a small safety margin for process startup/scheduling.
	bytesPerSample := 2
	if sampleRate <= 0 {
		sampleRate = 22050
	}
	durationSeconds := float64(len(audioData)) / float64(bytesPerSample*sampleRate)
	if durationSeconds < 0 {
		durationSeconds = 0
	}

	timeout := time.Duration(durationSeconds*float64(time.Second)) + 5*time.Second
	if timeout < 30*time.Second {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Prefer paplay when available (PulseAudio/PipeWire setups are common on desktop Linux).
	if _, lookErr := exec.LookPath("paplay"); lookErr == nil {
		// paplay can play raw PCM if we provide explicit parameters.
		paplayArgs := []string{
			"--raw",
			"--format=s16le",
			"--rate", fmt.Sprintf("%d", sampleRate),
			"--channels", "1",
			"-",
		}
		paplayCmd := exec.CommandContext(ctx, "paplay", paplayArgs...)
		paplayCmd.Stdin = bytes.NewReader(audioData)
		var paplayStderr bytes.Buffer
		paplayCmd.Stderr = &paplayStderr
		if err := p.runCmd(paplayCmd, cancel); err == nil {
			return nil
		} else {
			// Fall back to ALSA aplay if paplay fails (e.g., no server).
			// Continue below.
		}
	}

	// ALSA fallback: aplay
	aplayArgs := []string{
		"-r", fmt.Sprintf("%d", sampleRate),
		"-f", "S16_LE",
		"-t", "raw",
		"-c", "1",
	}
	if p.outputDevice != "" {
		aplayArgs = append(aplayArgs, "-D", p.outputDevice)
	}
	aplayCmd := exec.CommandContext(ctx, "aplay", aplayArgs...)
	aplayCmd.Stdin = bytes.NewReader(audioData)
	var aplayStderr bytes.Buffer
	aplayCmd.Stderr = &aplayStderr
	if err := p.runCmd(aplayCmd, cancel); err != nil {
		if errors.Is(err, ErrPlaybackStopped) {
			return err
		}
		return fmt.Errorf("audio playback failed (aplay: %v, aplay_stderr: %s)", err, aplayStderr.String())
	}

	return nil
}

// CID:audio-player-004 - PlayMP3
// Purpose: Plays MP3 data by writing to a disk file and calling a system player.
func (p *Player) PlayMP3(audioData []byte) error {
	// Write to temp file
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("audio-%d.mp3", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, audioData, 0o644); err != nil {
		return fmt.Errorf("failed to write temp audio file: %w", err)
	}
	defer os.Remove(tmpFile)

	// Try to play using common Linux audio players
	players := []struct {
		cmd  string
		args []string
	}{
		{"paplay", []string{tmpFile}},
		{"ffplay", []string{"-nodisp", "-autoexit", "-loglevel", "quiet", tmpFile}},
		{"mpv", []string{"--no-video", tmpFile}},
	}

	for _, player := range players {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, player.cmd, player.args...)
		if err := p.runCmd(cmd, cancel); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no MP3 player available (tried: paplay, ffplay, mpv)")
}
