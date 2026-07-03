/* Code Map: Transcription Types
 * - Engine: Interface for all audio-to-text providers
 *
 * CID Index:
 * CID:transcription-types-001 -> Engine
 *
 * Quick lookup: rg -n "CID:transcription-types-" internal/transcription/types.go
 */
package transcription

// CID:transcription-types-001 - Engine
// Purpose: Common interface for swapping transcription implementations (WhisperCPP, OpenAI).
type Engine interface {
	// Transcribe transcribes audio file at audioPath to text
	Transcribe(audioPath string) (string, error)
	// Validate checks if the engine is properly configured
	Validate() error
	// Name returns the engine name
	Name() string
}
