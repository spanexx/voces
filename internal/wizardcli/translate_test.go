/* Code Map: wizardcli.StateFromWizard tests
 *
 * CID Index:
 * CID:wizardcli-translate-test-001 -> TestStateFromWizard_EnglishRoutesToSmallEn
 * CID:wizardcli-translate-test-002 -> TestStateFromWizard_NonEnglishRoutesToBase
 * CID:wizardcli-translate-test-003 -> TestStateFromWizard_TTSOnSetsPiperVoice
 * CID:wizardcli-translate-test-004 -> TestStateFromWizard_TTSOffClearsPiperVoice
 * CID:wizardcli-translate-test-005 -> TestStateFromWizard_NilUsesDefaults
 * CID:wizardcli-translate-test-006 -> TestStateFromWizard_PreservesHotkey
 * CID:wizardcli-translate-test-007 -> TestStateFromWizard_EnglishAutoPiper (rc1-hotpatch-19)
 * CID:wizardcli-translate-test-008 -> TestStateFromWizard_AutostartDefault (rc1-hotpatch-19)
 */
package wizardcli

import (
	"testing"

	"voces/internal/setup"
	"voces/internal/wizard"
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
// PiperVoice = TTSVoice. Uses a non-English language (rc1-hotpatch-19)
// so the English auto-piper branch does not short-circuit the case
// under test.
func TestStateFromWizard_TTSOnSetsPiperVoice(t *testing.T) {
	w := &wizard.State{
		Language:   "de",
		TTSEnabled: true,
		TTSVoice:   "de_DE-thorsten-medium",
	}
	got := StateFromWizard(w, "v0.1.0")
	if got.PiperVoice != "de_DE-thorsten-medium" {
		t.Errorf("piper voice: got %q want %q", got.PiperVoice, "de_DE-thorsten-medium")
	}
}

// TestStateFromWizard_TTSOffClearsPiperVoice: TTSEnabled=false && lang!=en
// → PiperVoice = "" regardless of TTSVoice field. Uses a non-English
// language (rc1-hotpatch-19) so the English auto-piper branch does
// not interfere.
func TestStateFromWizard_TTSOffClearsPiperVoice(t *testing.T) {
	w := &wizard.State{
		Language:   "de",
		TTSEnabled: false,
		TTSVoice:   "en_US-lessac-medium", // ignored when TTS off
	}
	got := StateFromWizard(w, "v0.1.0")
	if got.PiperVoice != "" {
		t.Errorf("piper voice should be empty when TTS off and lang!=en, got %q", got.PiperVoice)
	}
}

// TestStateFromWizard_EnglishAutoPiper (rc1-hotpatch-19): English
// with TTS off still gets the lessac piper voice installed, so the
// read-clipboard hotkey can speak the transcript. The TTS step is
// skipped in the wizard chain for English, so the user never opts
// in — but PiperVoice is set unconditionally.
func TestStateFromWizard_EnglishAutoPiper(t *testing.T) {
	w := &wizard.State{
		Language:   "en",
		TTSEnabled: false,
		TTSVoice:   "",
	}
	got := StateFromWizard(w, "v0.1.0")
	if got.PiperVoice != "en_US-lessac-medium" {
		t.Errorf("English auto-piper: got %q want %q", got.PiperVoice, "en_US-lessac-medium")
	}
}

// TestStateFromWizard_NilUsesDefaults: nil wizard state → use
// wizard.NewState() defaults (en, ctrl-space, no TTS, autostart=true).
// rc1-hotpatch-19: for English the default state now yields
// PiperVoice = "en_US-lessac-medium" (auto-piper) and Autostart = true.
func TestStateFromWizard_NilUsesDefaults(t *testing.T) {
	got := StateFromWizard(nil, "v0.1.0")
	if got.Language != "en" {
		t.Errorf("Language from nil: got %q want %q", got.Language, "en")
	}
	if got.HotkeyPreset != setup.HotkeyPresetCtrlSpace {
		t.Errorf("HotkeyPreset from nil: got %q want %q", got.HotkeyPreset, setup.HotkeyPresetCtrlSpace)
	}
	if got.PiperVoice != "en_US-lessac-medium" {
		t.Errorf("PiperVoice from nil: got %q want %q (rc1-hotpatch-19 English auto-piper)", got.PiperVoice, "en_US-lessac-medium")
	}
	if !got.Autostart {
		t.Errorf("Autostart from nil: got false want true (rc1-hotpatch-19 default)")
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

// TestStateFromWizard_AutostartDefault (rc1-hotpatch-19): wizard's
// State.Autostart flows into setup.State.Autostart verbatim. The
// default in NewState is now true (was false in rc4 + hotpatch-17),
// so this test uses the real constructor rather than a partial struct
// literal that would keep Go's zero-value false.
func TestStateFromWizard_AutostartDefault(t *testing.T) {
	w := wizard.NewState()
	got := StateFromWizard(w, "v0.1.0")
	if !got.Autostart {
		t.Errorf("Autostart from wizard.NewState(): got false want true (rc1-hotpatch-19 default)")
	}
}
