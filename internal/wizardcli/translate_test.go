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
 * CID:wizardcli-translate-test-009 -> TestStateFromWizard_HonorsExplicitModel (rc1-hotpatch-24)
 * CID:wizardcli-translate-test-010 -> TestStateFromWizard_EmptyModelFallsBack (rc1-hotpatch-24)
 * CID:wizardcli-translate-test-011 -> TestStateFromWizard_NonEnglishExplicitMultilingual (rc1-hotpatch-24)
 */
package wizardcli

import (
	"testing"

	"voces/internal/setup"
	"voces/internal/wizard"
)

// TestStateFromWizard_EnglishRoutesToSmallEn: en → ggml-small.en.bin.
// rc1-hotpatch-24: now tests the empty-Model fallback path — the
// wizard State has no explicit pick, so translate.go falls back to
// wizard.DefaultModelForLanguage("en") = "ggml-small.en.bin". The
// visible result is unchanged.
func TestStateFromWizard_EnglishRoutesToSmallEn(t *testing.T) {
	w := &wizard.State{Language: "en"}
	got := StateFromWizard(w, "v0.1.0")
	if got.WhisperModel != "ggml-small.en.bin" {
		t.Errorf("English whisper model: got %q want %q", got.WhisperModel, "ggml-small.en.bin")
	}
}

// TestStateFromWizard_NonEnglishRoutesToBase: any non-en → ggml-base.bin
// (multilingual, larger). rc1-hotpatch-24: now tests the empty-Model
// fallback path for non-English.
func TestStateFromWizard_NonEnglishRoutesToBase(t *testing.T) {
	w := &wizard.State{Language: "de"}
	got := StateFromWizard(w, "v0.1.0")
	if got.WhisperModel != "ggml-base.bin" {
		t.Errorf("non-English whisper model: got %q want %q", got.WhisperModel, "ggml-base.bin")
	}
}

// TestStateFromWizard_HonorsExplicitModel (rc1-hotpatch-24): the
// model picker step writes the user's pick into State.Model. That
// pick wins over the language-implied default — that's the whole
// point of adding the picker. This test pins the contract.
func TestStateFromWizard_HonorsExplicitModel(t *testing.T) {
	cases := []struct {
		name string
		lang string
		pick string
	}{
		{"english-picks-base.en", "en", "ggml-base.en.bin"},
		{"english-picks-tiny.en", "en", "ggml-tiny.en.bin"},
		{"english-picks-medium.en", "en", "ggml-medium.en.bin"},
		{"multilingual-picks-small", "de", "ggml-small.bin"},
		{"multilingual-picks-medium", "fr", "ggml-medium.bin"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &wizard.State{Language: tc.lang, Model: tc.pick}
			got := StateFromWizard(w, "v0.1.0")
			if got.WhisperModel != tc.pick {
				t.Errorf("explicit model: got %q want %q", got.WhisperModel, tc.pick)
			}
		})
	}
}

// TestStateFromWizard_EmptyModelFallsBack (rc1-hotpatch-24):
// back-compat path. A wizard State with no Model field (e.g. a
// hand-rolled struct that pre-dates the picker) still resolves
// to a sensible default via DefaultModelForLanguage.
func TestStateFromWizard_EmptyModelFallsBack(t *testing.T) {
	cases := []struct {
		lang string
		want string
	}{
		{"en", "ggml-small.en.bin"},
		{"de", "ggml-base.bin"},
		{"fr", "ggml-base.bin"},
		{"", "ggml-base.bin"}, // unknown lang falls through to multilingual default
	}
	for _, tc := range cases {
		t.Run(tc.lang, func(t *testing.T) {
			w := &wizard.State{Language: tc.lang, Model: ""}
			got := StateFromWizard(w, "v0.1.0")
			if got.WhisperModel != tc.want {
				t.Errorf("empty-model fallback for %q: got %q want %q", tc.lang, got.WhisperModel, tc.want)
			}
		})
	}
}

// TestStateFromWizard_NonEnglishExplicitMultilingual (rc1-hotpatch-24):
// a non-English user can still pick an English-only model if they
// want (e.g. the user knows their audio is English). The picker
// doesn't enforce the filter on commit — that happens at the
// wizard-UI layer. translate.go is pure plumbing: whatever the
// State says, it carries through.
func TestStateFromWizard_NonEnglishExplicitMultilingual(t *testing.T) {
	w := &wizard.State{Language: "de", Model: "ggml-tiny.en.bin"}
	got := StateFromWizard(w, "v0.1.0")
	if got.WhisperModel != "ggml-tiny.en.bin" {
		t.Errorf("explicit en model under de lang: got %q want %q", got.WhisperModel, "ggml-tiny.en.bin")
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
