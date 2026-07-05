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
 *   - Language == "en"        → PiperVoice = "en_US-lessac-medium"
 *                                (rc1-hotpatch-18: English always
 *                                installs the lessac voice so the
 *                                read-clipboard hotkey can speak
 *                                the transcript even when the user
 *                                did not opt into TTS.)
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
	// piperEnglishVoice is the default piper voice installed
	// for English installs (rc1-hotpatch-18). Used even when
	// the user does not opt into TTS so the read-clipboard
	// hotkey has a voice to speak with.
	piperEnglishVoice = "en_US-lessac-medium"
)

// CID:wizardcli-translate-001 - StateFromWizard
// Purpose: pure conversion. No disk I/O, no GTK. Tested via
// TestStateFromWizard_* in translate_test.go. The cmd layer calls
// this after wizard.RunFull returns, then calls setup.EnsureModels
// + setup.Apply.
//
// rc1-hotpatch-18: the Autostart field is gone (behavior is
// hardcoded). For English, PiperVoice is unconditionally set to
// the lessac voice so the read-clipboard hotkey works out of the
// box. The wizard's TTSEnabled choice is not persisted — PiperVoice
// being non-empty is the runtime signal that TTS is configured.
func StateFromWizard(w *wizard.State, appVersion string) *setup.State {
	if w == nil {
		w = wizard.NewState()
	}
	whisperModel := whisperMultilingualModel
	if w.Language == "en" {
		whisperModel = whisperEnglishModel
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
		StopRecordingKey:       w.StopRecordingKey,
		ReadClipboardKey:       w.ReadClipboardKey,
		ToggleTTSKey:           w.ToggleTTSKey,
		ToggleTranscriptionKey: w.ToggleTranscriptionKey,
	}
}
