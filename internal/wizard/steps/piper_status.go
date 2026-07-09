/* Code Map: Piper Status Step (rc1-hotpatch-29)
 * - BuildPiperStatus: shows the user whether the piper binary
 *   is installed, with platform-specific install instructions
 *   if it isn't. Inserted in the chain right before the
 *   TTS voice picker so the user sees the gap *before* they
 *   pick a voice to download.
 * - ShouldShow: always true. Even users who skip TTS benefit
 *   from knowing the binary is missing (the rc1-hotpatch-27
 *   friendly "TTS Unavailable" notification still fires if they
 *   hit Ctrl+U without piper).
 *
 * Visual layout:
 *
 *   ┌─────────────────────────────────────────────┐
 *   │ Piper text-to-speech engine                 │
 *   │                                             │
 *   │ Piper is a fast, local neural TTS engine.   │
 *   │ Voces uses it to read transcriptions aloud  │
 *   │ (e.g. when you press Ctrl+U to read the     │
 *   │ clipboard).                                 │
 *   │                                             │
 *   │ ✓ Piper is installed at:                    │
 *   │   /opt/voces/engines/piper                  │
 *   │                                             │
 *   │ (or, when not installed:)                   │
 *   │ ✗ Piper is not installed.                   │
 *   │                                             │
 *   │   Pick one of these options:                │
 *   │   • Debian/Ubuntu:  sudo apt-get install piper│
 *   │   • Fedora:         sudo dnf install piper   │
 *   │   • Arch:           sudo pacman -S piper     │
 *   │   • Source:         github.com/rhasspy/piper │
 *   │                                             │
 *   │   After installing, go Back and Next to     │
 *   │   re-check.                                 │
 *   │                                             │
 *   │ [ Back ]                  [ Next → ]        │
 *   └─────────────────────────────────────────────┘
 *
 * The step is rebuilt on every show, so the detection is
 * always fresh — the user just installs piper, hits Back,
 * hits Next, and the status flips.
 *
 * The Capture closure is nil — the piper status doesn't
 * change wizard state. The runner handles nil Capture.
 *
 * CID Index:
 * CID:wizard-piper-001 -> BuildPiperStatus
 * CID:wizard-piper-002 -> ShouldShow
 *
 * Quick lookup: rg -n "CID:wizard-piper-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"

	"voces/internal/setup"
)

// CID:wizard-piper-002 - ShouldShow
// Purpose: always show the piper status step. The TTS step
// is the place to skip TTS (via the "Custom URL..." option or
// the behavior step's overall flow), and surfacing the binary
// gap up-front prevents the "yes, install a voice" trap that
// the user reported on rc1-hotpatch-27.
func piperShouldShow(_ string) bool { return true }

// CID:wizard-piper-001 - BuildPiperStatus
// Purpose: detect piper and show a clean install-vs-found
// panel. The detection runs on every build so the user can
// install piper + go Back/Next to re-check.
func BuildPiperStatus(win *gtk.Window, _ StateReader) (*Step, error) {
	piperPath := setup.FindPiperBinary()

	// Two-line status: the symbol + the human text. Using
	// markup so we can colour the found / not-found labels
	// without bringing in CSS classes (the wizard's CSS is
	// tuned for the title / progress bar; a one-off colour
	// is simpler).
	var statusTitle string
	var statusBody string
	if piperPath != "" {
		statusTitle = "<span foreground=\"#2e8b3a\">✓ Piper is installed.</span>"
		statusBody = fmt.Sprintf("Found at <tt>%s</tt>. You can use any voice in the next step.", piperPath)
	} else {
		statusTitle = "<span foreground=\"#a23a3a\">✗ Piper is not installed.</span>"
		statusBody = "Voces needs piper to play back text-to-speech. Pick one of these install options for your system:"
	}

	statusTitleLabel, err := gtk.LabelNew("")
	if err != nil {
		return nil, fmt.Errorf("piper: title label: %w", err)
	}
	statusTitleLabel.SetMarkup(statusTitle)
	statusTitleLabel.SetHAlign(gtk.ALIGN_START)
	statusTitleLabel.SetXAlign(0)
	statusTitleLabel.SetMarginTop(8)
	statusTitleLabel.SetMarginBottom(4)

	statusBodyLabel, err := gtk.LabelNew(statusBody)
	if err != nil {
		return nil, fmt.Errorf("piper: body label: %w", err)
	}
	statusBodyLabel.SetHAlign(gtk.ALIGN_START)
	statusBodyLabel.SetXAlign(0)
	statusBodyLabel.SetLineWrap(true)
	statusBodyLabel.SetMarginBottom(8)

	hint, err := gtk.LabelNew(`Piper is a fast, local neural text-to-speech engine. Voces uses it to read transcriptions aloud (e.g. when you press Ctrl+U to read the clipboard).`)
	if err != nil {
		return nil, fmt.Errorf("piper: hint: %w", err)
	}
	hint.SetLineWrap(true)
	hint.SetHAlign(gtk.ALIGN_START)
	hint.SetXAlign(0)
	hint.SetMarginBottom(12)

	content, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	if err != nil {
		return nil, fmt.Errorf("piper: content: %w", err)
	}
	content.PackStart(hint, false, false, 0)
	content.PackStart(statusTitleLabel, false, false, 0)
	content.PackStart(statusBodyLabel, false, false, 0)

	// When piper is missing, append the platform-specific
	// install hint as a preformatted block. The hint text
	// contains literal newlines and indentation, so we use
	// a monospace label inside a frame.
	if piperPath == "" {
		installHint, err := gtk.LabelNew(setup.PiperInstallHintForOS())
		if err != nil {
			return nil, fmt.Errorf("piper: install hint: %w", err)
		}
		installHint.SetHAlign(gtk.ALIGN_START)
		installHint.SetXAlign(0)
		installHint.SetLineWrap(true)
		// Monospace via .voces-code class from window_css.go
		// (matches the rest of the wizard's "code-y" look).
		if hStyle, err := installHint.GetStyleContext(); err == nil {
			hStyle.AddClass("voces-code")
		}

		frame, err := gtk.FrameNew("")
		if err != nil {
			return nil, fmt.Errorf("piper: frame: %w", err)
		}
		frame.Add(installHint)
		frame.SetMarginTop(4)
		frame.SetMarginBottom(8)

		content.PackStart(frame, false, false, 0)
	}

	box, back, next, err := newStepContent("Piper text-to-speech engine", content, "Back", "Next")
	if err != nil {
		return nil, fmt.Errorf("piper: build content: %w", err)
	}

	// Optional polish: show a coloured top border on the
	// window when the build is green (piper found) so the
	// user has a clear "all good" signal. Skipped on the
	// not-installed case — the red status label is enough.
	if piperPath != "" {
		// Use a CSS class to keep the colour in window_css.go
		// (the rest of the wizard uses CSS classes for theming).
		if wStyle, err := win.GetStyleContext(); err == nil {
			wStyle.AddClass("voces-piper-ok")
		}
		_ = gdk.RGBA{} // gdk imported for the future, ignored at build
	}

	return &Step{Box: box, Next: next, Back: back, Capture: nil}, nil
}
