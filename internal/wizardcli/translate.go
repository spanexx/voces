/* Code Map: wizard → setup state translation
 * - StateFromWizard: pure function that converts a wizard.State
 *   (user choices collected in the GTK wizard) into a setup.State
 *   (the on-disk record). Lives in wizardcli so we don't create
 *   a setup ↔ wizard import cycle (wizard already imports setup
 *   for the HotkeyPreset* constants).
 *
 * Routing rule (rc1-hotpatch-24):
 *   - Model comes from w.Model verbatim. The Language step seeds
 *     w.Model with wizard.DefaultModelForLanguage(w.Language) so
 *     a user who never reaches the Model step still gets the
 *     ADR-0004 routing (en → ggml-small.en.bin, anything else
 *     → ggml-base.bin). The Model step lets the user override.
 *   - TTSEnabled && TTSVoice  → PiperVoice = TTSVoice
 *   - !TTSEnabled             → PiperVoice = ""
 *   - Language == "en"        → PiperVoice = "en_US-lessac-medium"
 *                                (rc1-hotpatch-19: English always
 *                                installs the lessac voice so the
 *                                read-clipboard hotkey can speak
 *                                the transcript even when the user
 *                                did not opt into TTS. The TTS step
 *                                is still skipped for English in
 *                                steps/languages.go; the user is
 *                                not asked to choose a voice.)
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

// piperEnglishVoice is the default piper voice installed for
// English installs (rc1-hotpatch-19). Used even when the user
// does not opt into TTS so the read-clipboard hotkey has a
// voice to speak with.
const piperEnglishVoice = "en_US-lessac-medium"

// CID:wizardcli-translate-001 - StateFromWizard
// Purpose: pure conversion. No disk I/O, no GTK. Tested via
// TestStateFromWizard_* in translate_test.go. The cmd layer calls
// this after wizard.RunFull returns, then calls setup.EnsureModels
// + setup.Apply.
//
// rc1-hotpatch-24: model comes from w.Model (the wizard's Model
// step writes there). When w.Model is empty (e.g. a hand-rolled
// State that pre-dates the picker), falls back to
// wizard.DefaultModelForLanguage(w.Language) so the ADR-0004
// routing still applies.
func StateFromWizard(w *wizard.State, appVersion string) *setup.State {
	if w == nil {
		w = wizard.NewState()
	}
	whisperModel := w.Model
	if whisperModel == "" {
		whisperModel = wizard.DefaultModelForLanguage(w.Language)
	}
	piperVoice := ""
	switch {
	case w.TTSEnabled && w.TTSVoice != "":
		piperVoice = w.TTSVoice
	case w.Language == "en":
		// English: TTS step is skipped, but install the
		// lessac voice so read-clipboard can speak.
		piperVoice = piperEnglishVoice
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
