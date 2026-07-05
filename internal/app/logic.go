/* Code Map: Application Logic
 * - processTranscription: Core pipeline (File -> Transcribe -> Type)
 *
 * CID Index:
 * CID:app-logic-001 -> processTranscription
 *
 * Quick lookup: rg -n "CID:app-logic-" internal/app/logic.go
 */
package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"voces/internal/transcription"
	"voces/internal/tray"
)

// CID:app-logic-001 - processTranscription
// Purpose: Manages the end-to-end transcription flow: saves audio to disk, runs engine, and triggers auto-typing.
func (a *Application) processTranscription(audioData []byte) {
	if len(audioData) == 0 {
		a.Tray.SetState(tray.StateIdle, "Idle")
		return
	}

	// The transcription engines require a physical file path
	tmpDir := os.TempDir()
	audioPath := filepath.Join(tmpDir, fmt.Sprintf("whisper_record_%d.wav", time.Now().UnixNano()))
	if err := os.WriteFile(audioPath, audioData, 0644); err != nil {
		log.Printf("Failed to write temp audio for transcription: %v", err)
		a.Tray.SetState(tray.StateError, "File Error")
		return
	}
	defer os.Remove(audioPath) // Cleanup the wav file after transcribe attempt

	text, err := a.Transcriber.Transcribe(audioPath)
	if err != nil {
		// rc1-hotpatch-14 R3: branch on the no-speech sentinel
		// so the tray + notification copy is friendly ("I
		// didn't catch that — try again") instead of the
		// generic "Transcription Failed: no speech
		// detected" the previous error string produced. All
		// other errors (binary failed, model load, OOM,
		// etc.) still surface as before with the underlying
		// error text so the user has something to debug.
		if errors.Is(err, transcription.ErrNoSpeechDetected) {
			log.Printf("Transcription: %v", err)
			a.Notifier.Info("Voces", "I didn't catch that — try again")
			a.Tray.SetState(tray.StateIdle, "Idle")
			return
		}
		log.Printf("Transcription failed: %v", err)
		a.Notifier.Error("Transcription Failed", err.Error())
		a.Tray.SetState(tray.StateError, "Transcription Error")

		// Revert to idle after a few seconds
		go func() {
			time.Sleep(3 * time.Second)
			a.Tray.SetState(tray.StateIdle, "Idle")
		}()
		return
	}

	if text != "" {
		log.Printf("Transcription successful: %q", text)
		a.Notifier.SuccessTranscriptionComplete(text)

		// Auto-type output
		if a.Config.Behavior.AutoType {
			a.AutoTyper.Type(text)
		}
	}

	a.Tray.SetState(tray.StateIdle, "Idle")
}
