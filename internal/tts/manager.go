/* Code Map: TTS Management
 * - TTS: Orchestrates multiple TTS engines
 * - New: Factory for creating the TTS manager
 * - Speak: Routes text to the active engine
 * - SetEngine: Hot-swaps the active TTS provider
 * - Available: rc1-hotpatch-27 — true when the active engine's
 *              binary/model/voice-config all resolve. Used by
 *              the read_clipboard handlers so the user gets a
 *              friendly "TTS not configured" notification when
 *              piper isn't installed (the release tarball
 *              doesn't bundle piper — see Makefile piper-build)
 *              instead of the raw "piper binary not found:
 *              /opt/voces/engines/piper" error from Piper.Speak.
 *
 * CID Index:
 * CID:tts-manager-001 -> TTS
 * CID:tts-manager-002 -> New
 * CID:tts-manager-003 -> Speak
 * CID:tts-manager-004 -> SetEngine
 * CID:tts-manager-005 -> Available (rc1-hotpatch-27)
 *
 * Quick lookup: rg -n "CID:tts-manager-" internal/tts/manager.go
 */
package tts

import (
	"fmt"

	"voces/internal/config"
)

// CID:tts-manager-001 - TTS
// Purpose: High-level manager for selecting and invoking TTS engines.
type TTS struct {
	cfg     *config.Config
	engine  Engine
	engines map[string]Engine
}

// CID:tts-manager-002 - New
// Purpose: Initializes the TTS manager with all supported engines from config.
func New(cfg *config.Config) *TTS {
	engines := make(map[string]Engine)

	// Initialize Piper engine
	engines["piper"] = NewPiper(cfg)

	// Initialize ElevenLabs engine
	engines["elevenlabs"] = NewElevenLabs(cfg)

	// Set default engine
	defaultEngine := cfg.TTS.DefaultEngine
	if defaultEngine == "" {
		defaultEngine = "piper"
	}

	engine, ok := engines[defaultEngine]
	if !ok {
		engine = engines["piper"]
	}

	return &TTS{
		cfg:     cfg,
		engine:  engine,
		engines: engines,
	}
}

// CID:tts-manager-003 - Speak
// Purpose: Delegates speech synthesis to the currently selected engine.
func (t *TTS) Speak(text string) error {
	return t.engine.Speak(text)
}

func (t *TTS) IsPlaying() bool {
	ps, ok := t.engine.(PlayingStatus)
	if !ok {
		return false
	}
	return ps.IsPlaying()
}

func (t *TTS) Stop() {
	s, ok := t.engine.(Stoppable)
	if !ok {
		return
	}
	s.Stop()
}

// CID:tts-manager-004 - SetEngine
// Purpose: Updates the active TTS engine and persists the choice to config.
func (t *TTS) SetEngine(engineName string) error {
	engine, ok := t.engines[engineName]
	if !ok {
		return fmt.Errorf("unknown engine: %s", engineName)
	}

	// Validate the new engine before switching
	if err := engine.Validate(); err != nil {
		return fmt.Errorf("engine %s validation failed: %w", engineName, err)
	}

	t.engine = engine
	t.cfg.TTS.DefaultEngine = engineName
	return nil
}

// CurrentEngine returns the name of the current engine.
// CurrentEngine returns the name of the current engine.
func (t *TTS) CurrentEngine() string {
	return t.engine.Name()
}

// AvailableEngines returns a list of available engine names.
// AvailableEngines returns a list of available engine names.
func (t *TTS) AvailableEngines() []string {
	names := make([]string, 0, len(t.engines))
	for name := range t.engines {
		names = append(names, name)
	}
	return names
}

// Validate checks if the current engine is properly configured.
// Validate checks if the current engine is properly configured.
func (t *TTS) Validate() error {
	return t.engine.Validate()
}

// CID:tts-manager-005 - Available
// Purpose: rc1-hotpatch-27. Cheap "is the active TTS engine
// ready to speak?" check for the read_clipboard hotkey path.
// Piper's Validate() does three os.Stat calls (binary, model,
// voice config). For the hotkey use-case we only care whether
// the engine can be invoked, so we wrap Validate and translate
// any non-nil error to false. The full error message is
// preserved on the engine itself for the user-facing
// notification ("Piper binary not found: /opt/voces/..."), but
// the boolean is what the hotkey handler dispatches on.
//
// Returns true if the active engine is configured and false
// otherwise. Re-evaluated on every call (no caching) — the
// user might install piper mid-session and the next hotkey
// press should pick that up. The three os.Stat calls are
// microseconds; caching is not worth the staleness.
func (t *TTS) Available() bool {
	return t.engine.Validate() == nil
}
