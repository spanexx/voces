/* Code Map: Event Handlers
 * - buildTrayHandlers: tray UI -> app logic
 * - buildHotkeyHandlers: global keys -> app logic
 *
 * CID:app-handlers-001 -> buildTrayHandlers
 * CID:app-handlers-002 -> buildHotkeyHandlers
 */
package app

import (
	"errors"
	"log"
	"time"

	"voces/internal/audio"
	"voces/internal/hotkey"
	"voces/internal/tray"
)

// CID:app-handlers-001 - buildTrayHandlers
// Purpose: Constructs a mapping of tray menu clicks to application actions.
func (a *Application) buildTrayHandlers() tray.ActionHandlers {
	return tray.ActionHandlers{
		OnRecordStart: func() {
			log.Println("Tray action: OnRecordStart")
			a.Tray.SetState(tray.StateRecording, "Recording (10s)...")
			if a.Overlay != nil {
				_ = a.Overlay.Start(func() {
					log.Println("Overlay: Stop clicked")
					a.Tray.SetState(tray.StateProcessing, "Stopping...")
					a.Recorder.Stop()
				})
			}

			go func() {
				// Tray manual recording defaults to 10 seconds
				audioBytes, err := a.Recorder.Record(10)
				if a.Overlay != nil {
					a.Overlay.Stop()
				}
				if err != nil {
					log.Printf("Tray recording error: %v", err)
					a.Notifier.Error("Recording Failed", err.Error())
					a.Tray.SetState(tray.StateIdle, "Idle")
				} else {
					go a.processTranscription(audioBytes)
				}
			}()
		},
		OnReadClipboard: func() {
			log.Println("Tray action: OnReadClipboard")
			a.Tray.SetState(tray.StateProcessing, "Reading clipboard...")

			go func() {
				text, err := a.Clipboard.Get()
				if err != nil {
					log.Printf("Tray clipboard error: %v", err)
					a.Tray.SetState(tray.StateIdle, "Idle")
					return
				}
				if text != "" {
					// rc27: friendly "TTS Unavailable" info when
					// piper binary/model missing (see TTS.Available).
					if a.TTS == nil || !a.TTS.Available() {
						a.Notifier.Info(
							"TTS Unavailable",
							"Install piper (system package or via the setup wizard) to enable read-clipboard speech.",
						)
					} else {
						err := a.TTS.Speak(text)
						if err != nil {
							if errors.Is(err, audio.ErrPlaybackStopped) {
								a.Tray.SetState(tray.StateIdle, "Idle")
								return
							}
							log.Printf("Tray TTS error: %v", err)
							a.Notifier.Error("TTS Failed", err.Error())
						}
					}
				}
				a.Tray.SetState(tray.StateIdle, "Idle")
			}()
		},
		OnSetTranscriptionEngine: func(engine string) {
			log.Printf("Tray action: Set Transcription Engine to %s\n", engine)
			if engine != "whisper_cpp" && engine != "openai_api" {
				log.Printf("Ignoring invalid transcription engine: %s\n", engine)
				return
			}
			a.Config.Transcription.DefaultEngine = engine
			a.Notifier.Info("Engine Changed", "Transcription engine set to "+engine)
			a.saveConfigAsync()
		},
		OnSetTTSEngine: func(engine string) {
			log.Printf("Tray action: Set TTS Engine to %s\n", engine)
			if engine != "piper" && engine != "elevenlabs" {
				log.Printf("Ignoring invalid TTS engine: %s\n", engine)
				return
			}
			a.Config.TTS.DefaultEngine = engine
			a.Notifier.Info("Engine Changed", "TTS engine set to "+engine)
			a.saveConfigAsync()
		},
		OnRunSetup: func() {
			log.Println("Tray action: OnRunSetup (re-spawning wizard)")
			go a.runSetupSubprocess()
		},
		OnCheckUpdates: func() {
			log.Println("Tray action: OnCheckUpdates")
			// Click is always user-initiated, so we always show a
			// notification (up to date / available / failed).
			a.checkForUpdates(true)
		},
		OnApplyUpdate: func() {
			log.Println("Tray action: OnApplyUpdate")
			go a.applyUpdate()
		},
		OnOpenDataDir: func() {
			log.Println("Tray action: OnOpenDataDir")
			go a.openDataDir()
		},
		OnQuit: func() {
			log.Println("Tray action: Quit requested")
			a.cancel() // Triggers the context cancellation in Run()
		},
	}
}

// CID:app-handlers-002 - buildHotkeyHandlers
// Purpose: Connects global keyboard shortcuts to record/type/read actions.
func (a *Application) buildHotkeyHandlers() hotkey.ActionHandlers {
	return hotkey.ActionHandlers{
		OnRecordStart: func() {
			if a.Recorder.IsRecording() || a.Tray.GetState() == tray.StateRecording {
				log.Println("Hotkey: Record Toggle Stop")
				a.Tray.SetState(tray.StateProcessing, "Transcribing...")
				a.Recorder.Stop()
				if a.Overlay != nil {
					a.Overlay.Stop()
				}
				return
			}
			log.Println("Hotkey: Record Start")
			a.Tray.SetState(tray.StateRecording, "Recording...")
			if a.Overlay != nil {
				_ = a.Overlay.Start(func() {
					log.Println("Overlay: Stop clicked")
					a.Tray.SetState(tray.StateProcessing, "Stopping...")
					a.Recorder.Stop()
				})
			}

			// Kick off recording asynchronously
			go func() {
				// Record up to max duration, handle error if timeout or failure
				audioBytes, err := a.Recorder.Record(300) // 5 mins max recording
				if a.Overlay != nil {
					a.Overlay.Stop()
				}
				if err != nil {
					log.Printf("Recording error: %v", err)
					a.Notifier.Error("Recording Failed", err.Error())
					a.Tray.SetState(tray.StateIdle, "Idle")
				} else {
					// Make sure we pass the payload to transcribing if it stopped successfully
					// We pass via goroutine so the hotkey release isn't blocked by transcription length
					go a.processTranscription(audioBytes)
				}
			}()
		},
		OnRecordStop: func() {
			if a.TTS != nil && a.TTS.IsPlaying() {
				log.Println("Hotkey: Stop Playback")
				a.TTS.Stop()
				a.Tray.SetState(tray.StateIdle, "Idle")
				return
			}
			if a.Recorder.IsRecording() || a.Tray.GetState() == tray.StateRecording {
				log.Println("Hotkey: Record Stop")
				a.Tray.SetState(tray.StateProcessing, "Transcribing...")
				a.Recorder.Stop() // triggers the return of the async Record() call above
				if a.Overlay != nil {
					a.Overlay.Stop()
				}
				return
			}
		},
		OnReadClipboard: func() {
			log.Println("Hotkey: Read Clipboard")
			a.Tray.SetState(tray.StateProcessing, "Reading clipboard...")

			// Kick off TTS asynchronously to not block hotkey
			go func() {
				text, err := a.Clipboard.Get()
				if err != nil {
					log.Printf("Clipboard read error: %v", err)
					a.Notifier.Error("Clipboard Failed", err.Error())
					a.Tray.SetState(tray.StateIdle, "Idle")
					return
				}

				if text == "" {
					a.Notifier.Info("TTS Skipping", "Clipboard is empty")
					a.Tray.SetState(tray.StateIdle, "Idle")
					return
				}

				// rc27: friendly "TTS Unavailable" info when
				// piper binary/model missing (see TTS.Available).
				// The release tarball doesn't bundle piper, so a
				// user who opted out of TTS in the wizard would
				// otherwise see the raw "piper binary not found"
				// error from Piper.Speak (install-deps.sh installs
				// the apt-piper fallback on the host).
				if a.TTS == nil || !a.TTS.Available() {
					a.Notifier.Info(
						"TTS Unavailable",
						"Install piper (system package or via the setup wizard) to enable read-clipboard speech.",
					)
					a.Tray.SetState(tray.StateIdle, "Idle")
					return
				}

				err = a.TTS.Speak(text)
				if err != nil {
					if errors.Is(err, audio.ErrPlaybackStopped) {
						a.Tray.SetState(tray.StateIdle, "Idle")
						return
					}
					log.Printf("TTS playback error: %v", err)
					a.Notifier.Error("TTS Failed", err.Error())
					a.Tray.SetState(tray.StateError, "TTS Error")

					time.Sleep(3 * time.Second)
				}

				a.Tray.SetState(tray.StateIdle, "Idle")
			}()
		},
		OnToggleTTS: func() {
			log.Println("Hotkey: Toggle TTS")
			// toggle logic and notify
			a.Notifier.Info("Engine Toggled", "Swapped TTS Engine")
		},
		OnToggleTranscription: func() {
			log.Println("Hotkey: Toggle Transcription")
			a.Notifier.Info("Engine Toggled", "Swapped Transcription Engine")
		},
	}
}
