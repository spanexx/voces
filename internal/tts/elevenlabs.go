/* Code Map: ElevenLabs TTS Engine
 * - ElevenLabs: Cloud-based TTS engine via API
 * - NewElevenLabs: Factory for creating an ElevenLabs engine
 * - Speak: Converts text to MP3 via ElevenLabs API
 *
 * CID Index:
 * CID:tts-elevenlabs-001 -> ElevenLabs
 * CID:tts-elevenlabs-002 -> NewElevenLabs
 * CID:tts-elevenlabs-003 -> Speak
 *
 * Quick lookup: rg -n "CID:tts-elevenlabs-" internal/tts/elevenlabs.go
 */
package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"whisper-voice-util/internal/audio"
	"whisper-voice-util/internal/config"
)

// CID:tts-elevenlabs-001 - ElevenLabs
// Purpose: Implements the Engine interface for the ElevenLabs cloud API.
type ElevenLabs struct {
	apiKey          string
	voiceID         string
	model           string
	stability       float64
	similarityBoost float64
	player          *audio.Player
}

// CID:tts-elevenlabs-002 - NewElevenLabs
// Purpose: Initializes an ElevenLabs engine with credentials from config.
func NewElevenLabs(cfg *config.Config) *ElevenLabs {
	return &ElevenLabs{
		apiKey:          cfg.TTS.ElevenLabs.APIKey,
		voiceID:         cfg.TTS.ElevenLabs.VoiceID,
		model:           cfg.TTS.ElevenLabs.Model,
		stability:       cfg.TTS.ElevenLabs.Stability,
		similarityBoost: cfg.TTS.ElevenLabs.SimilarityBoost,
		player:          audio.NewPlayer(),
	}
}

// Name returns the engine name.
func (e *ElevenLabs) Name() string {
	return "elevenlabs"
}

func (e *ElevenLabs) IsPlaying() bool {
	return e.player.IsPlaying()
}

func (e *ElevenLabs) Stop() {
	e.player.Stop()
}

// CID:tts-elevenlabs-003 - Speak
// Purpose: Sends a text-to-speech request to ElevenLabs and plays the returned MP3.
func (e *ElevenLabs) Speak(text string) error {
	// Validate API key
	if e.apiKey == "" {
		return fmt.Errorf("ElevenLabs API key not configured")
	}

	// Build API endpoint
	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", e.voiceID)

	// Create request body
	requestBody := map[string]interface{}{
		"text":     text,
		"model_id": e.model,
		"voice_settings": map[string]float64{
			"stability":        e.stability,
			"similarity_boost": e.similarityBoost,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("xi-api-key", e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ElevenLabs API error: %s (%s)", resp.Status, string(respBody))
	}

	// Read audio data
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read audio data: %w", err)
	}

	// Play the audio using centralized player
	return e.player.PlayMP3(audioData)
}

// Validate checks if the ElevenLabs API is properly configured.
func (e *ElevenLabs) Validate() error {
	if e.apiKey == "" {
		return fmt.Errorf("ElevenLabs API key not configured")
	}
	if e.voiceID == "" {
		return fmt.Errorf("ElevenLabs voice ID not configured")
	}
	return nil
}
