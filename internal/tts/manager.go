/* Code Map: TTS Management
 * - TTS: Orchestrates multiple TTS engines
 * - New: Factory for creating the TTS manager
 * - Speak: Routes text to the active engine
 * - SetEngine: Hot-swaps the active TTS provider
 *
 * CID Index:
 * CID:tts-manager-001 -> TTS
 * CID:tts-manager-002 -> New
 * CID:tts-manager-003 -> Speak
 * CID:tts-manager-004 -> SetEngine
 *
 * Quick lookup: rg -n "CID:tts-manager-" internal/tts/manager.go
 */
package tts

import (
	"fmt"

	"whisper-voice-util/internal/config"
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
