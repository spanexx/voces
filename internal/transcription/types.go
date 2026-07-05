/* Code Map: Transcription Types
 * - Engine: Interface for all audio-to-text providers
 * - ErrNoSpeechDetected: sentinel for the "user said nothing
 *   during the recording" case (rc1-hotpatch-14 R3)
 *
 * CID Index:
 * CID:transcription-types-001 -> Engine
 * CID:transcription-types-002 -> ErrNoSpeechDetected
 *
 * Quick lookup: rg -n "CID:transcription-types-" internal/transcription/types.go
 */
package transcription

import "errors"

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

// CID:transcription-types-002 - ErrNoSpeechDetected
// Purpose: sentinel returned by Engine.Transcribe when the
// binary exited successfully but produced no text and no
// stderr. The tray layer branches on errors.Is(err,
// transcription.ErrNoSpeechDetected) to show "I didn't catch
// that — try again" instead of "transcription failed". All
// other empty-output cases (binary failed, model load error,
// etc.) still return a wrapped error with the underlying
// stderr text so the user has something to debug.
//
// rc1-hotpatch-14 R3: replaces the previous opaque
// "whisper.cpp produced no transcription output (binary=...)"
// string.
var ErrNoSpeechDetected = errors.New("no speech detected")
