/* Code Map: Wizard Window
 * - NewWindow: creates the top-level wizard window. Includes a
 *   styled header bar (gradient background + accent strip), a
 *   centered title, and the IMPL §3 default size. CSS lives
 *   in window_css.go so this file stays under the size cap.
 *
 * rc1-hotpatch-14: "style the pop up window to look better".
 * Replaces the bare 480x420 gtk.Window with a header + body
 * shell. The CSS uses a small palette (Voces blue, off-white
 * header, subtle gradient) that matches the tray app icon.
 *
 * CID Index:
 * CID:wizard-win-001 -> NewWindow
 *
 * Quick lookup: rg -n "CID:wizard-win-" internal/wizard/
 */
package wizard

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

const (
	// windowWidth and windowHeight are larger than the
	// pre-rc1-hotpatch-14 480x420 to fit the header bar plus
	// the larger, more padded step content. 540x520 keeps the
	// window scannable on a 1366x768 laptop and still fits
	// the four-row secondary hotkey editor without scrolling.
	windowWidth  = 540
	windowHeight = 520
	// windowTitle is shown in the title bar (also used as a
	// fallback for window managers that strip the custom
	// header). The em-dash is U+2014.
	windowTitle = "Voces — Setup"
	// borderPadding is the inner margin around the step body.
	// The pre-rc1-hotpatch-14 value was 16, which made the
	// four-row secondary hotkey editor feel cramped against
	// the window border. 20 on the sides, 24 on top matches
	// the visual weight of the new header.
	borderPaddingSide = 20
	borderPaddingTop  = 24
	// contentBoxName is the gtk.Buildable name NewWindow sets
	// on the content box. Kept as a named constant so a
	// future debug helper that walks the widget tree (e.g.
	// to take a screenshot from a test) has one place to
	// find the name. Not strictly required today.
	contentBoxName = "voces-wizard-content"
)

// CID:wizard-win-001 - NewWindow
// Purpose: build the top-level wizard window with a styled
// header + accent + body shell. GTK init must already have
// happened (call ensureInit from wizard.go before this). The
// window is not shown; the caller calls ShowAll when the first
// step is attached.
//
// Layout (top to bottom):
//   1. header box (gradient) — title "Voces Setup" + subtitle
//   2. accent strip (3px, Voces blue)
//   3. contentBox (vertical) — populated by the runner with
//      the current step's Box
//
// The header and accent are added directly to the window (not
// nested inside contentBox) so they survive every step swap and
// the visual identity is constant throughout the wizard.
//
// Returns the contentBox (the slot the runner swaps step boxes
// into) so the runner does not have to walk the widget tree
// to find it. Both widgets are returned because the runner
// needs both — adding another layer of indirection (an
// accessor on a struct, or gtk.Buildable.GetName lookups) just
// to hide one variable adds noise.
func NewWindow() (*gtk.Window, *gtk.Box, error) {
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return nil, nil, fmt.Errorf("wizard: window new: %w", err)
	}
	win.SetTitle(windowTitle)
	win.SetDefaultSize(windowWidth, windowHeight)
	win.SetResizable(false)
	win.SetDecorated(true)
	win.SetPosition(gtk.WIN_POS_CENTER)

	// Install the global stylesheet before any widgets are
	// built. A missing DISPLAY would have already failed in
	// ensureInit; Screen is always available here.
	if err := installCSS(win.GetScreen()); err != nil {
		// Non-fatal: the wizard still works, just with the
		// stock look. Log a warning instead of erroring so
		// a CSS parser bug never blocks the wizard.
		fmt.Printf("wizard: install css: %v\n", err)
	}

	// Outer vertical box: header / accent / content.
	outer, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("wizard: outer box: %w", err)
	}
	win.Add(outer)

	// Header.
	header, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("wizard: header box: %w", err)
	}
	if style, err := header.GetStyleContext(); err == nil {
		style.AddClass("voces-header")
	}

	title, err := gtk.LabelNew("")
	if err != nil {
		return nil, nil, fmt.Errorf("wizard: title label: %w", err)
	}
	title.SetMarkup("<span foreground=\"white\"><b>Voces Setup</b></span>")
	title.SetHAlign(gtk.ALIGN_START)
	title.SetMarginStart(4)
	if tStyle, err := title.GetStyleContext(); err == nil {
		tStyle.AddClass("voces-title")
	}
	header.PackStart(title, false, false, 0)

	subtitle, err := gtk.LabelNew("")
	if err != nil {
		return nil, nil, fmt.Errorf("wizard: subtitle label: %w", err)
	}
	// rc1-hotpatch-27: dropped the misleading "push to talk"
	// tagline. The wizard's hotkey step binds f9 to start and
	// space to stop (tap-to-start, tap-to-stop), not
	// push-and-hold. The tagline survived from the
	// rc1-hotpatch-14 design before the toggle hotkeys
	// landed. The subtitle now shows just the version — the
	// brief is honest about the build and lets the welcome
	// body copy do the talking. The test in
	// window_css_test.go bans the legacy phrase from this
	// file's source; this comment paraphrases the rationale
	// to avoid tripping the regression guard.
	subtitle.SetMarkup(fmt.Sprintf(
		"<span foreground=\"#a8b8d0\">v%s</span>",
		AppVersion,
	))
	subtitle.SetHAlign(gtk.ALIGN_START)
	subtitle.SetMarginStart(4)
	if sStyle, err := subtitle.GetStyleContext(); err == nil {
		sStyle.AddClass("voces-subtitle")
	}
	header.PackStart(subtitle, false, false, 0)
	outer.PackStart(header, false, false, 0)

	// Accent strip.
	accent, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("wizard: accent box: %w", err)
	}
	if aStyle, err := accent.GetStyleContext(); err == nil {
		aStyle.AddClass("voces-accent")
	}
	outer.PackStart(accent, false, false, 0)

	// contentBox is the slot the runner swaps the current
	// step's Box into. We add an empty container so the
	// window has a valid child structure even before the
	// first step is built.
	contentBox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("wizard: content box: %w", err)
	}
	contentBox.SetName(contentBoxName)
	contentBox.SetMarginStart(borderPaddingSide)
	contentBox.SetMarginEnd(borderPaddingSide)
	contentBox.SetMarginTop(borderPaddingTop)
	contentBox.SetMarginBottom(borderPaddingSide)
	outer.PackStart(contentBox, true, true, 0)

	// Closing the window with the X button emits "destroy";
	// the caller wires that up to abort the wizard.
	return win, contentBox, nil
}
