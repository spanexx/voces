/* Code Map: Shared Step Helpers
 * - Step: the value every step Build() returns. It carries the
 *   container the wizard swaps in/out plus the Next/Back buttons
 *   and a Capture function for committing the user's selection to
 *   the wizard State.
 * - newStepContent: builds the standard vertical-box layout
 *   (title + content + Back/Next row) that every step uses.
 * - StateReader: read-only view of wizard.State the steps use to
 *   initialize widgets (e.g. preselect the current language).
 * - StateSetter: write view of wizard.State the steps use in their
 *   Capture closure to commit the user's choice.
 *
 * The interfaces live here (not in the wizard package) to break the
 * import cycle: wizard imports steps, so steps cannot import wizard.
 * Go's structural typing makes *wizard.State satisfy both interfaces
 * automatically once the methods are defined.
 *
 * CID Index:
 * CID:wizard-step-001 -> Step
 * CID:wizard-step-002 -> newStepContent
 * CID:wizard-step-003 -> StateReader
 * CID:wizard-step-004 -> StateSetter
 *
 * Quick lookup: rg -n "CID:wizard-step-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

// CID:wizard-step-003 - StateReader
// Purpose: read-only view of the wizard State a step's Build takes
// so the step can initialize its widgets from the current values
// (e.g. preselect the user's previously picked language). The wizard
// runner passes *wizard.State which satisfies this interface.
type StateReader interface {
	LanguageCode() string
	Hotkey() string
	Custom() string
	TTS() bool
	AutostartDesired() bool
	StopRecordingKeyCode() string
	ReadClipboardKeyCode() string
	ToggleTTSKeyCode() string
	ToggleTranscriptionKeyCode() string
	// ModelFile returns the whisper model file the user has
	// chosen (or the language-implied default).
	// rc1-hotpatch-24 — the model step preselects the radio
	// whose file name equals this value.
	ModelFile() string
	// TTSVoiceID returns the piper voice the user has chosen.
	// Empty when the user has not reached the TTS step yet, or
	// when they picked the "no TTS" path. May be a manifest
	// key (e.g. "en_US-lessac-medium") or a custom-URL
	// sentinel (see steps.customURLSentinel).
	// rc1-hotpatch-29 — the TTS step preselects the dropdown
	// row whose ID matches this value.
	TTSVoiceID() string
}

// CID:wizard-step-004 - StateSetter
// Purpose: write view of the wizard State a step's Capture closure
// takes to commit the user's choice. SetLanguageCode/SetHotkey are
// no-ops on empty input so a step that the user skipped does not
// erase the State. SetTTS / SetAutostart are always applied because
// their zero value (false) is meaningful.
type StateSetter interface {
	SetLanguageCode(code string)
	SetHotkey(preset, custom string)
	SetTTS(enabled bool)
	SetAutostart(desired bool)
	SetSecondaryHotkeys(stop, read, toggleTTS, toggleTranscription string)
	// SetModel commits the chosen whisper model file. Empty
	// input is a no-op so a skipped model step does not erase
	// the State. rc1-hotpatch-24.
	SetModel(filename string)
	// SetTTSVoice commits the chosen piper voice ID. Empty
	// input is a no-op so a skipped TTS step does not erase
	// the State. rc1-hotpatch-29.
	SetTTSVoice(id string)
}

// CID:wizard-step-001 - Step
// Purpose: every step Build() returns one of these. The wizard.go
// runner attaches Box to the window on show, calls Capture on Next
// (which may return an error to abort the advance), and reads the
// button labels to know when to move forward or back.
// Back is nil for the first step (welcome).
// Capture is nil when the step has no state to commit (e.g. welcome,
// finish).
type Step struct {
	Box     *gtk.Box
	Next    *gtk.Button
	Back    *gtk.Button
	Capture func(setter StateSetter) error
}

// CID:wizard-step-002 - newStepContent
// Purpose: builds the common vertical-box skeleton every step uses:
// a title label, a content slot filled by the caller, and a footer
// row with the Back and Next buttons (Back omitted when nil).
// The returned *gtk.Box is the Step.Box the wizard attaches to the
// window. The returned buttons are wired by the runner, not here.
//
// rc1-hotpatch-14: title label uses the .voces-step-title CSS
// class (16px bold) instead of an inline <b> markup; Next uses
// the .voces-next-btn class for the primary-action emphasis.
// The BorderWidth bump (16 -> 20) gives the four-row secondary
// hotkey editor enough room to breathe.
func newStepContent(title string, content gtk.IWidget, backLabel, nextLabel string) (*gtk.Box, *gtk.Button, *gtk.Button, error) {
	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 14)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("steps: box: %w", err)
	}
	box.SetBorderWidth(0)

	// The title uses the .voces-step-title CSS class (16px
	// bold, near-black) so it reads as a section header.
	titleLabel, err := gtk.LabelNew("")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("steps: title label: %w", err)
	}
	titleLabel.SetMarkup(fmt.Sprintf("<b>%s</b>", title))
	titleLabel.SetHAlign(gtk.ALIGN_START)
	titleLabel.SetMarginBottom(4)
	if tStyle, err := titleLabel.GetStyleContext(); err == nil {
		tStyle.AddClass("voces-step-title")
	}
	box.PackStart(titleLabel, false, false, 0)

	// The caller-supplied content is the only widget that expands.
	box.PackStart(content, true, true, 0)

	// Footer row: Back on the left, Next on the right. Next
	// gets the .voces-next-btn class so it picks up the
	// primary-action gradient.
	footer, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("steps: footer box: %w", err)
	}
	footer.SetMarginTop(8)
	var back *gtk.Button
	if backLabel != "" {
		back, err = gtk.ButtonNewWithLabel(backLabel)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("steps: back button: %w", err)
		}
	}
	next, err := gtk.ButtonNewWithLabel(nextLabel)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("steps: next button: %w", err)
	}
	if nStyle, err := next.GetStyleContext(); err == nil {
		nStyle.AddClass("voces-next-btn")
	}
	spacer, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("steps: spacer: %w", err)
	}
	if back != nil {
		footer.PackStart(back, false, false, 0)
	}
	footer.PackStart(spacer, true, true, 0)
	footer.PackStart(next, false, false, 0)
	box.PackStart(footer, false, false, 0)

	return box, back, next, nil
}
