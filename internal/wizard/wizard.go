/* Code Map: Wizard Driver
 * - AppVersion: version string rendered in the welcome footer
 * - ensureInit: idempotent GTK init
 * - RunWelcome: welcome step only (returns true on Next, false on close)
 * - RunFull: 4-5 step chain (welcome → language → hotkey → tts? → finish)
 *
 * CID Index:
 * CID:wizard-001 -> AppVersion
 * CID:wizard-002 -> ensureInit
 * CID:wizard-003 -> RunWelcome
 * CID:wizard-004 -> RunFull
 *
 * Quick lookup: rg -n "CID:wizard-" internal/wizard/
 */
package wizard

import (
	"fmt"
	"sync"

	"github.com/gotk3/gotk3/gtk"

	"voces/internal/wizard/steps"
)

// CID:wizard-001 - AppVersion
// Purpose: rendered in the welcome footer. A future phase will wire
// this to a ldflags -X override at build time.
const AppVersion = "0.1.0"

// gtkInitOnce guards gtk.Init so we never call it twice (gotk3
// aborts on the second call).
var gtkInitOnce sync.Once
var gtkInitErr error

// CID:wizard-002 - ensureInit
// Purpose: idempotently initialize GTK. Returns the first init
// error (typically a missing DISPLAY).
func ensureInit() error {
	gtkInitOnce.Do(func() {
		gtkInitErr = gtk.InitCheck(nil)
	})
	return gtkInitErr
}

// CID:wizard-003 - RunWelcome
// Purpose: present the welcome step only, block the calling goroutine
// on the GTK main loop, and return when the user clicks "Get started"
// (completed=true) or closes the window (completed=false). Provided
// for backwards compatibility; the full multi-step entry point is
// RunFull.
func RunWelcome() (bool, error) {
	if err := ensureInit(); err != nil {
		return false, fmt.Errorf("wizard: gtk init: %w", err)
	}

	win, err := NewWindow()
	if err != nil {
		return false, err
	}

	step, err := steps.BuildWelcome(win, AppVersion)
	if err != nil {
		win.Destroy()
		return false, fmt.Errorf("wizard: build welcome step: %w", err)
	}

	// result is buffered so the close handler can never block waiting
	// for a reader that is not yet on the main loop.
	result := make(chan bool, 1)
	// quitOnce prevents a double gtk.MainQuit (see RunFull for the
	// full explanation). The destroy event fires twice — once when
	// the user closes the window, once when win.Destroy() below
	// runs after gtk.Main() returns.
	var quitOnce sync.Once
	finish := func(v bool) {
		quitOnce.Do(func() {
			select {
			case result <- v:
			default:
			}
			gtk.MainQuit()
		})
	}

	step.Next.Connect("clicked", func() { finish(true) })
	win.Connect("destroy", func() { finish(false) })

	win.ShowAll()
	gtk.Main()
	win.Destroy()

	return <-result, nil
}

// CID:wizard-004 - RunFull
// Purpose: present the 4-5 step wizard (welcome → language → hotkey →
// tts? → finish), block on the GTK main loop, return the accumulated
// State when the user clicks "Start" (or nil on window close).
//
// The chain is rebuilt on every transition so a Back into language +
// change of language inserts/removes the TTS step on the way forward.
// On Next click: step.Capture(state) commits, then showStepAt swaps
// in the next step's box inside the single contentBox wrapper.
// On Back click: showStepAt rebuilds and re-shows the prior step.
func RunFull() (*State, error) {
	if err := ensureInit(); err != nil {
		return nil, fmt.Errorf("wizard: gtk init: %w", err)
	}

	win, err := NewWindow()
	if err != nil {
		return nil, err
	}

	// GtkWindow is a GtkBin (one direct child). Every step's box
	// lives inside this single wrapper; showStepAt swaps them in
	// and out, so the window itself only ever has one child and
	// step transitions don't trigger the Gtk-WARNING.
	contentBox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		win.Destroy()
		return nil, fmt.Errorf("wizard: build content box: %w", err)
	}
	win.Add(contentBox)

	state := NewState()
	result := make(chan *State, 1)

	registry := map[stepKey]stepRenderer{
		stepWelcome: func(win *gtk.Window, _ *State) (*steps.Step, error) {
			return steps.BuildWelcome(win, AppVersion)
		},
		stepLanguage: func(win *gtk.Window, s *State) (*steps.Step, error) {
			return steps.BuildLanguage(win, s)
		},
		stepHotkey: func(win *gtk.Window, s *State) (*steps.Step, error) {
			return steps.BuildHotkey(win, s)
		},
		stepTTS: func(win *gtk.Window, s *State) (*steps.Step, error) {
			return steps.BuildTTS(win, s)
		},
		stepFinish: func(win *gtk.Window, s *State) (*steps.Step, error) {
			return steps.BuildFinish(win, s)
		},
	}

	// chain returns the ordered stepKeys for the current state.
	chain := func() []stepKey {
		keys := []stepKey{stepWelcome, stepLanguage, stepHotkey}
		if steps.ShouldShow(state.LanguageCode()) {
			keys = append(keys, stepTTS)
		}
		keys = append(keys, stepFinish)
		return keys
	}

	idx := 0
	keys := chain()

	// currentBox tracks the step's box currently parented under
	// contentBox. showStepAt removes it before adding the new one,
	// so the wrapper (and the window) always has exactly one child.
	var currentBox *gtk.Box

	// finish is the single exit point. sync.Once prevents a double
	// gtk.MainQuit (which would trigger GTK's
	// "main_loops != NULL" assertion and kill the process with
	// SIGKILL): the explicit win.Destroy() after gtk.Main() returns
	// re-fires the destroy event — finish() is then a no-op.
	var quitOnce sync.Once
	finish := func(v *State) {
		quitOnce.Do(func() {
			select {
			case result <- v:
			default:
			}
			gtk.MainQuit()
		})
	}

	// showStepAt: declared as var + assignment (not := with the
	// function literal) so the closure can recursively call itself.
	var showStepAt func() error
	showStepAt = func() error {
		if idx < 0 || idx >= len(keys) {
			return nil
		}
		k := keys[idx]
		step, err := registry[k](win, state)
		if err != nil {
			return err
		}
		// Swap the step's box into the wrapper. Removing the
		// previous box (if any) is what was missing — Hide() alone
		// left it parented and triggered the GtkWindow warning.
		if currentBox != nil {
			contentBox.Remove(currentBox)
		}
		contentBox.Add(step.Box)
		currentBox = step.Box
		step.Box.ShowAll()

		step.Next.Connect("clicked", func() {
			if step.Capture != nil {
				if err := step.Capture(state); err != nil {
					showError(win, err)
					return
				}
			}
			keys = chain()
			if idx+1 >= len(keys) {
				finish(state)
				return
			}
			idx++
			if err := showStepAt(); err != nil {
				showError(win, err)
			}
		})
		if step.Back != nil {
			step.Back.Connect("clicked", func() {
				keys = chain()
				if idx > 0 {
					idx--
				}
				if err := showStepAt(); err != nil {
					showError(win, err)
				}
			})
		}
		return nil
	}

	if err := showStepAt(); err != nil {
		win.Destroy()
		return nil, fmt.Errorf("wizard: build step %d: %w", idx, err)
	}

	// Show the window + the first step. ShowAll on step.Box alone
	// doesn't render (the GtkWindow is still hidden). Mirrors
	// what RunWelcome does at the end of its setup.
	win.ShowAll()
	win.Connect("destroy", func() { finish(nil) })

	gtk.Main()
	win.Destroy()
	return <-result, nil
}
