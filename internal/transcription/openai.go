/* Code Map: OpenAI Transcription Engine
 * - OpenAIAPI: Cloud-based transcription engine via API
 * - NewOpenAIAPI: Factory for creating an OpenAIAPI engine
 * - Transcribe: Converts audio to text via OpenAI Whisper API
 *
 * CID Index:
 * CID:transcription-openai-001 -> OpenAIAPI
 * CID:transcription-openai-002 -> NewOpenAIAPI
 * CID:transcription-openai-003 -> Transcribe
 *
 * Quick lookup: rg -n "CID:transcription-openai-" internal/transcription/openai.go
 */
package transcription

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"whisper-voice-util/internal/config"
)

// CID:transcription-openai-001 - OpenAIAPI
// Purpose: Implements the Engine interface for the OpenAI Whisper cloud API.
type OpenAIAPI struct {
	apiKey string
	model  string
	prompt string
}

// CID:transcription-openai-002 - NewOpenAIAPI
// Purpose: Initializes an OpenAI transcription engine with credentials from config.
func NewOpenAIAPI(cfg *config.Config) *OpenAIAPI {
	return &OpenAIAPI{
		apiKey: cfg.Transcription.OpenAIAPI.APIKey,
		model:  cfg.Transcription.OpenAIAPI.Model,
		prompt: cfg.Transcription.OpenAIAPI.Prompt,
	}
}

// Name returns the engine name.
func (o *OpenAIAPI) Name() string {
	return "openai_api"
}

// CID:transcription-openai-003 - Transcribe
// Purpose: Sends an audio file to OpenAI and returns the transcribed text.
func (o *OpenAIAPI) Transcribe(audioPath string) (string, error) {
	// Validate API key
	if o.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	// Open audio file
	audioFile, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer audioFile.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, audioFile); err != nil {
		return "", fmt.Errorf("failed to copy file data: %w", err)
	}

	// Add model
	if err := writer.WriteField("model", o.model); err != nil {
		return "", fmt.Errorf("failed to write model field: %w", err)
	}

	// Add prompt if specified
	if o.prompt != "" {
		if err := writer.WriteField("prompt", o.prompt); err != nil {
			return "", fmt.Errorf("failed to write prompt field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: %s (%s)", resp.Status, string(respBody))
	}

	// Parse response
	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Text, nil
}

// Validate checks if the OpenAI API is properly configured.
func (o *OpenAIAPI) Validate() error {
	if o.apiKey == "" {
		return fmt.Errorf("OpenAI API key not configured")
	}
	return nil
}
