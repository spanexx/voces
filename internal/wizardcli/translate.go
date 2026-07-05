/* Code Map: wizard → setup state translation
 * - StateFromWizard: pure function that converts a wizard.State
 *   (user choices collected in the GTK wizard) into a setup.State
 *   (the on-disk record). Lives in wizardcli so we don't create
 *   a setup ↔ wizard import cycle (wizard already imports setup
 *   for the HotkeyPreset* constants).
 *
 * Routing rule (ADR-0004):
 *   - Language == "en"        → whisper model ggml-small.en.bin
 *   - Language != "en"        → whisper model ggml-base.bin
 *   - TTSEnabled && TTSVoice  → PiperVoice = TTSVoice
 *   - !TTSEnabled             → PiperVoice = ""
 *
 * CID Index:
 * CID:wizardcli-translate-001 -> StateFromWizard
 *
 * Quick lookup: rg -n "CID:wizardcli-translate-" internal/wizardcli/
 */
package wizardcli

import (
	"voces/internal/setup"
	"voces/internal/wizard"
)

// whisperEnglishModel and whisperMultilingualModel are the file names
// from the DefaultManifest. Kept as named constants (not magic strings)
// so a future change in either is a one-line edit.
const (
	whisperEnglishModel      = "ggml-small.en.bin"
	whisperMultilingualModel = "ggml-base.bin"
)

// CID:wizardcli-translate-001 - StateFromWizard
// Purpose: pure conversion. No disk I/O, no GTK. Tested via
// TestStateFromWizard_* in translate_test.go. The cmd layer calls
// this after wizard.RunFull returns, then calls setup.EnsureModels
// + setup.Apply.
func StateFromWizard(w *wizard.State, appVersion string) *setup.State {
	if w == nil {
		w = wizard.NewState()
	}
	whisperModel := whisperMultilingualModel
	if w.Language == "en" {
		whisperModel = whisperEnglishModel
	}
	piperVoice := ""
	if w.TTSEnabled && w.TTSVoice != "" {
		piperVoice = w.TTSVoice
	}
	return &setup.State{
		SchemaVersion:          "1",
		AppVersion:             appVersion,
		Language:               w.Language,
		WhisperModel:           whisperModel,
		PiperVoice:             piperVoice,
		HotkeyPreset:           w.HotkeyPreset,
		CustomHotkey:           w.CustomHotkey,
		Autostart:              w.Autostart,
		StopRecordingKey:       w.StopRecordingKey,
		ReadClipboardKey:       w.ReadClipboardKey,
		ToggleTTSKey:           w.ToggleTTSKey,
		ToggleTranscriptionKey: w.ToggleTranscriptionKey,
	}
}
