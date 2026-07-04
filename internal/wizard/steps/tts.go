/* Code Map: TTS Step
 * - ShouldShow: the TTS step is skipped for English per the IMPL
 *   ("Only consulted when Language != en").
 * - Build: assemble the TTS yes/no radios and return a *Step with
 *   a Capture that commits the boolean to the wizard State.
 *
 * CID Index:
 * CID:wizard-ttsstep-001 -> ShouldShow
 * CID:wizard-ttsstep-002 -> Build
 *
 * Quick lookup: rg -n "CID:wizard-ttsstep-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

// CID:wizard-ttsstep-001 - ShouldShow
// Purpose: returns true when the TTS step should be inserted into
// the chain. The IMPL-public-setup §3 says the TTS step is only
// consulted for non-English languages; English users get the
// bundled "lessac" piper voice automatically. An empty language
// code is treated as "not yet set" and the prompt is shown — the
// language step is the only place a code is written, so empty in
// practice means the language step has not been visited.
func ShouldShow(languageCode string) bool {
	return languageCode != "en"
}

// CID:wizard-ttsstep-002 - BuildTTS
// Purpose: build the TTS step. Two radio buttons: "Yes, install a
// voice" and "No, skip TTS". Default is "No" so users who do not
// need TTS never see a download. The "Yes" radio marks
// TTSEnabled=true on the State; the Phase 5 download step reads
// that flag to decide whether to fetch a piper voice.
//
// The step is hidden when stateReader.LanguageCode() == "en"; the
// wizard runner is responsible for skipping the call to BuildTTS
// in that case (see ShouldShow).
//
// Back is set so the user can return to the hotkey step.
func BuildTTS(win *gtk.Window, stateReader StateReader) (*Step, error) {
	yesBtn, err := gtk.RadioButtonNewWithLabel(nil, "Yes, install a voice")
	if err != nil {
		return nil, fmt.Errorf("tts: yes radio: %w", err)
	}
	noBtn, err := gtk.RadioButtonNewWithLabelFromWidget(yesBtn, "No, skip TTS")
	if err != nil {
		return nil, fmt.Errorf("tts: no radio: %w", err)
	}
	// Default: No.
	noBtn.SetActive(true)
	if stateReader != nil && stateReader.TTS() {
		yesBtn.SetActive(true)
	}

	hint, err := gtk.LabelNew(
		"Text-to-speech reads the transcribed text back to you. " +
			"Only non-English languages prompt here; English uses the bundled voice. " +
			"You can change this later by re-running the setup.",
	)
	if err != nil {
		return nil, fmt.Errorf("tts: hint label: %w", err)
	}
	hint.SetLineWrap(true)
	hint.SetHAlign(gtk.ALIGN_START)

	radioBox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	if err != nil {
		return nil, fmt.Errorf("tts: radio box: %w", err)
	}
	radioBox.PackStart(yesBtn, false, false, 0)
	radioBox.PackStart(noBtn, false, false, 0)

	content, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	if err != nil {
		return nil, fmt.Errorf("tts: content box: %w", err)
	}
	content.PackStart(hint, false, false, 0)
	content.PackStart(radioBox, false, false, 0)

	box, back, next, err := newStepContent("Enable text-to-speech?", content, "Back", "Next")
	if err != nil {
		return nil, fmt.Errorf("tts: build content: %w", err)
	}

	capture := func(setter StateSetter) error {
		setter.SetTTS(yesBtn.GetActive())
		return nil
	}

	win.Add(box)
	return &Step{Box: box, Next: next, Back: back, Capture: capture}, nil
}
