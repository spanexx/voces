/* Code Map: Transcription Management
 * - Transcriber: Orchestrates multiple transcription engines
 * - New: Factory for creating the Transcriber manager
 * - Transcribe: Routes audio to the active engine
 * - SetEngine: Hot-swaps the active transcription provider
 * - ToggleEngine: Utility to cycle between engines
 *
 * CID Index:
 * CID:transcription-manager-001 -> Transcriber
 * CID:transcription-manager-002 -> New
 * CID:transcription-manager-003 -> Transcribe
 * CID:transcription-manager-004 -> SetEngine
 * CID:transcription-manager-005 -> ToggleEngine
 *
 * Quick lookup: rg -n "CID:transcription-manager-" internal/transcription/manager.go
 */
package transcription

import (
	"fmt"

	"whisper-voice-util/internal/config"
)

// CID:transcription-manager-001 - Transcriber
// Purpose: High-level manager for selecting and invoking transcription engines.
type Transcriber struct {
	cfg     *config.Config
	engine  Engine
	engines map[string]Engine
}

// CID:transcription-manager-002 - New
// Purpose: Initializes the Transcriber with all supported engines from config.
func New(cfg *config.Config) *Transcriber {
	engines := make(map[string]Engine)

	// Initialize whisper.cpp engine
	engines["whisper_cpp"] = NewWhisperCPP(cfg)

	// Initialize OpenAI API engine
	engines["openai_api"] = NewOpenAIAPI(cfg)

	// Set default engine
	defaultEngine := cfg.Transcription.DefaultEngine
	if defaultEngine == "" {
		defaultEngine = "whisper_cpp"
	}

	engine, ok := engines[defaultEngine]
	if !ok {
		engine = engines["whisper_cpp"]
	}

	return &Transcriber{
		cfg:     cfg,
		engine:  engine,
		engines: engines,
	}
}

// CID:transcription-manager-003 - Transcribe
// Purpose: Delegates audio processing to the currently selected engine.
func (t *Transcriber) Transcribe(audioPath string) (string, error) {
	return t.engine.Transcribe(audioPath)
}

// CID:transcription-manager-004 - SetEngine
// Purpose: Updates the active transcription engine and persists the choice to config.
func (t *Transcriber) SetEngine(engineName string) error {
	engine, ok := t.engines[engineName]
	if !ok {
		return fmt.Errorf("unknown engine: %s", engineName)
	}

	// Validate the new engine before switching
	if err := engine.Validate(); err != nil {
		return fmt.Errorf("engine %s validation failed: %w", engineName, err)
	}

	t.engine = engine
	t.cfg.Transcription.DefaultEngine = engineName
	return nil
}

// CID:transcription-manager-005 - ToggleEngine
// Purpose: Cycles through available engines (Local -> Cloud -> Local).
func (t *Transcriber) ToggleEngine() (string, error) {
	current := t.CurrentEngine()
	var target string

	// Toggle to the other engine
	if current == "whisper_cpp" {
		target = "openai_api"
	} else {
		target = "whisper_cpp"
	}

	if err := t.SetEngine(target); err != nil {
		return "", err
	}

	return target, nil
}

// CurrentEngine returns the name of the current engine.
func (t *Transcriber) CurrentEngine() string {
	return t.engine.Name()
}

// AvailableEngines returns a list of available engine names.
func (t *Transcriber) AvailableEngines() []string {
	names := make([]string, 0, len(t.engines))
	for name := range t.engines {
		names = append(names, name)
	}
	return names
}

// Validate checks if the current engine is properly configured.
func (t *Transcriber) Validate() error {
	return t.engine.Validate()
}
