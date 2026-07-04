/* Code Map: Finish Step
 * - Build: assemble the finish step (summary label + Start button)
 *   and return a *Step. Capture is nil because no state is captured
 *   here — the runner commits the accumulated state on Start click.
 *
 * CID Index:
 * CID:wizard-finish-001 -> Build
 *
 * Quick lookup: rg -n "CID:wizard-finish-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

// CID:wizard-finish-001 - BuildFinish
// Purpose: build the finish step. The label summarises the user's
// choices (language, hotkey, TTS) read from the StateReader so they
// can sanity-check before clicking Start. The Start button label
// reads "Start" per the IMPL; clicking it is the signal the runner
// uses to commit state and exit the wizard.
//
// The Capture closure is nil — there is no state to write at the
// finish step; the runner already has it from prior steps.
//
// Back is set so the user can return to the previous step to
// revise. State changes propagate through the StateReader interface
// the next time the prior step's Build is called (the runner
// re-builds the prior step on Back, see wizard.go).
func BuildFinish(win *gtk.Window, stateReader StateReader) (*Step, error) {
	summaryText := "Setup complete!\n\nThe app will now download the selected model and start in the system tray.\n\n"
	if stateReader != nil {
		lang := stateReader.LanguageCode()
		if lang == "" {
			lang = "en"
		}
		preset := stateReader.Hotkey()
		if preset == "" {
			preset = "ctrl-space"
		}
		custom := stateReader.Custom()
		hotkey := preset
		if custom != "" {
			hotkey = custom
		}
		ttsLine := "TTS: bundled English voice"
		if lang != "en" {
			if stateReader.TTS() {
				ttsLine = "TTS: yes (download voice)"
			} else {
				ttsLine = "TTS: no"
			}
		}
		summaryText += fmt.Sprintf("Language: %s\nHotkey: %s\n%s", lang, hotkey, ttsLine)
	}

	label, err := gtk.LabelNew(summaryText)
	if err != nil {
		return nil, fmt.Errorf("finish: label: %w", err)
	}
	label.SetLineWrap(true)
	label.SetHAlign(gtk.ALIGN_START)

	content, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	if err != nil {
		return nil, fmt.Errorf("finish: content box: %w", err)
	}
	content.PackStart(label, true, true, 0)

	box, back, next, err := newStepContent("Setup complete", content, "Back", "Start")
	if err != nil {
		return nil, fmt.Errorf("finish: build content: %w", err)
	}

	win.Add(box)
	return &Step{Box: box, Next: next, Back: back, Capture: nil}, nil
}
