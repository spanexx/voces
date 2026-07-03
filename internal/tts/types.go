/* Code Map: TTS Types
 * - Engine: Interface for all text-to-speech providers
 *
 * CID Index:
 * CID:tts-types-001 -> Engine
 *
 * Quick lookup: rg -n "CID:tts-types-" internal/tts/types.go
 */
package tts

// CID:tts-types-001 - Engine
// Purpose: Common interface for swapping TTS implementations (Piper, ElevenLabs).
type Engine interface {
	// Speak converts text to speech and plays the audio
	Speak(text string) error
	// Validate checks if the engine is properly configured
	Validate() error
	// Name returns the engine name
	Name() string
}

type Stoppable interface {
	Stop()
}

type PlayingStatus interface {
	IsPlaying() bool
}
