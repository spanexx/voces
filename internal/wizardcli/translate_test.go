/* Code Map: wizardcli.StateFromWizard tests
 *
 * CID Index:
 * CID:wizardcli-translate-test-001 -> TestStateFromWizard_EnglishRoutesToSmallEn
 * CID:wizardcli-translate-test-002 -> TestStateFromWizard_NonEnglishRoutesToBase
 * CID:wizardcli-translate-test-003 -> TestStateFromWizard_TTSOnSetsPiperVoice
 * CID:wizardcli-translate-test-004 -> TestStateFromWizard_TTSOffClearsPiperVoice
 * CID:wizardcli-translate-test-005 -> TestStateFromWizard_NilUsesDefaults
 * CID:wizardcli-translate-test-006 -> TestStateFromWizard_PreservesHotkey
 */
package wizardcli

import (
	"testing"

	"whisper-voice-util/internal/setup"
	"whisper-voice-util/internal/wizard"
)

// TestStateFromWizard_EnglishRoutesToSmallEn: en → ggml-small.en.bin
// (ADR-0004: smaller, faster model for English-only).
func TestStateFromWizard_EnglishRoutesToSmallEn(t *testing.T) {
	w := &wizard.State{Language: "en"}
	got := StateFromWizard(w, "v0.1.0")
	if got.WhisperModel != "ggml-small.en.bin" {
		t.Errorf("English whisper model: got %q want %q", got.WhisperModel, "ggml-small.en.bin")
	}
}

// TestStateFromWizard_NonEnglishRoutesToBase: any non-en → ggml-base.bin
// (multilingual, larger).
func TestStateFromWizard_NonEnglishRoutesToBase(t *testing.T) {
	w := &wizard.State{Language: "de"}
	got := StateFromWizard(w, "v0.1.0")
	if got.WhisperModel != "ggml-base.bin" {
		t.Errorf("non-English whisper model: got %q want %q", got.WhisperModel, "ggml-base.bin")
	}
}

// TestStateFromWizard_TTSOnSetsPiperVoice: TTSEnabled && TTSVoice != "" →
// PiperVoice = TTSVoice.
func TestStateFromWizard_TTSOnSetsPiperVoice(t *testing.T) {
	w := &wizard.State{
		Language:   "en",
		TTSEnabled: true,
		TTSVoice:   "en_US-lessac-medium",
	}
	got := StateFromWizard(w, "v0.1.0")
	if got.PiperVoice != "en_US-lessac-medium" {
		t.Errorf("piper voice: got %q want %q", got.PiperVoice, "en_US-lessac-medium")
	}
}

// TestStateFromWizard_TTSOffClearsPiperVoice: TTSEnabled=false →
// PiperVoice = "" regardless of TTSVoice field.
func TestStateFromWizard_TTSOffClearsPiperVoice(t *testing.T) {
	w := &wizard.State{
		Language:   "en",
		TTSEnabled: false,
		TTSVoice:   "en_US-lessac-medium", // ignored when TTS off
	}
	got := StateFromWizard(w, "v0.1.0")
	if got.PiperVoice != "" {
		t.Errorf("piper voice should be empty when TTS off, got %q", got.PiperVoice)
	}
}

// TestStateFromWizard_NilUsesDefaults: nil wizard state → use
// wizard.NewState() defaults (en, ctrl-space, no TTS).
func TestStateFromWizard_NilUsesDefaults(t *testing.T) {
	got := StateFromWizard(nil, "v0.1.0")
	if got.Language != "en" {
		t.Errorf("Language from nil: got %q want %q", got.Language, "en")
	}
	if got.HotkeyPreset != setup.HotkeyPresetCtrlSpace {
		t.Errorf("HotkeyPreset from nil: got %q want %q", got.HotkeyPreset, setup.HotkeyPresetCtrlSpace)
	}
	if got.PiperVoice != "" {
		t.Errorf("PiperVoice from nil: got %q want empty", got.PiperVoice)
	}
}

// TestStateFromWizard_PreservesHotkey: custom hotkey string is
// carried over verbatim.
func TestStateFromWizard_PreservesHotkey(t *testing.T) {
	w := &wizard.State{
		Language:     "en",
		HotkeyPreset: setup.HotkeyPresetCustom,
		CustomHotkey: "ctrl+shift+r",
	}
	got := StateFromWizard(w, "v0.1.0")
	if got.HotkeyPreset != setup.HotkeyPresetCustom {
		t.Errorf("HotkeyPreset: got %q want %q", got.HotkeyPreset, setup.HotkeyPresetCustom)
	}
	if got.CustomHotkey != "ctrl+shift+r" {
		t.Errorf("CustomHotkey: got %q want %q", got.CustomHotkey, "ctrl+shift+r")
	}
}
