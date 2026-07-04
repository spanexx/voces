/* Code Map: Piper TTS Engine
 * - Piper: Subprocess-based local TTS engine
 * - NewPiper: Factory for creating a Piper engine
 * - Speak: Converts text to raw PCM via Piper
 *
 * CID Index:
 * CID:tts-piper-001 -> Piper
 * CID:tts-piper-002 -> NewPiper
 * CID:tts-piper-003 -> Speak
 *
 * Quick lookup: rg -n "CID:tts-piper-" internal/tts/piper.go
 */
// Package tts provides the Piper text-to-speech engine implementation.
package tts

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"voces/internal/audio"
	"voces/internal/config"
)

// CID:tts-piper-001 - Piper
// Purpose: Implements the Engine interface for the local Piper TTS utility.
type Piper struct {
	binaryPath   string
	modelPath    string
	voiceConfig  string
	outputDevice string
	player       *audio.Player
}

// CID:tts-piper-002 - NewPiper
// Purpose: Initializes a Piper engine with paths from application config.
func NewPiper(cfg *config.Config) *Piper {
	return &Piper{
		binaryPath:   cfg.TTS.Piper.BinaryPath,
		modelPath:    cfg.TTS.Piper.Model,
		voiceConfig:  cfg.TTS.Piper.VoiceConfig,
		outputDevice: cfg.TTS.Piper.OutputDevice,
		player:       audio.NewPlayer(),
	}
}

// Name returns the engine name.
func (p *Piper) Name() string {
	return "piper"
}

func (p *Piper) IsPlaying() bool {
	return p.player.IsPlaying()
}

func (p *Piper) Stop() {
	p.player.Stop()
}

// CID:tts-piper-003 - Speak
// Purpose: Executes the Piper binary and pipes result to the audio player.
func (p *Piper) Speak(text string) error {
	// Check if binary exists
	if _, err := os.Stat(p.binaryPath); err != nil {
		return fmt.Errorf("piper binary not found: %s", p.binaryPath)
	}

	// Check if model exists
	if _, err := os.Stat(p.modelPath); err != nil {
		return fmt.Errorf("piper model not found: %s", p.modelPath)
	}

	// Check if voice config exists
	if _, err := os.Stat(p.voiceConfig); err != nil {
		return fmt.Errorf("piper voice config not found: %s", p.voiceConfig)
	}

	// Build command
	args := []string{
		"-m", p.modelPath,
		"-c", p.voiceConfig,
		"--output-raw",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.binaryPath, args...)
	cmd.Dir = filepath.Dir(p.binaryPath)

	// Send text to stdin
	cmd.Stdin = strings.NewReader(text)

	// Capture raw audio output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("piper failed: %w, stderr: %s", err, stderr.String())
	}

	// Play the audio output using centralized player
	return p.player.PlayRaw(stdout.Bytes(), 22050)
}

// Validate checks if the Piper binary and model are accessible.
func (p *Piper) Validate() error {
	if _, err := os.Stat(p.binaryPath); err != nil {
		return fmt.Errorf("piper binary not found: %s", p.binaryPath)
	}
	if _, err := os.Stat(p.modelPath); err != nil {
		return fmt.Errorf("piper model not found: %s", p.modelPath)
	}
	if _, err := os.Stat(p.voiceConfig); err != nil {
		return fmt.Errorf("piper voice config not found: %s", p.voiceConfig)
	}
	return nil
}
