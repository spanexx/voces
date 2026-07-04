/* Code Map: TTS Types
 * - Engine: Interface for all text-to-speech providers
 * - Stoppable: Optional interface for engines that can be interrupted mid-speak
 * - PlayingStatus: Optional interface for engines that can report playback state
 *
 * CID Index:
 * CID:tts-types-001 -> Engine
 * CID:tts-types-002 -> Stoppable
 * CID:tts-types-003 -> PlayingStatus
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

// CID:tts-types-002 - Stoppable
// Purpose: Optional capability for engines that support interrupting
// playback mid-utterance (e.g. on a "stop TTS" hotkey).
type Stoppable interface {
	Stop()
}

// CID:tts-types-003 - PlayingStatus
// Purpose: Optional capability for engines that can report whether
// they are currently playing audio (used by the tray icon and
// the "toggle TTS" hotkey).
type PlayingStatus interface {
	IsPlaying() bool
}
