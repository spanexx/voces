/* Code Map: Wizard Driver
 * - AppVersion: the version string rendered in the welcome footer
 * - ensureInit: idempotent GTK initialization
 * - RunWelcome: blocks on the main loop until the user clicks
 *   "Get started" or closes the window. Backwards-compatible
 *   wrapper for the first step only.
 * - RunFull: blocks on the main loop until the user clicks "Start"
 *   on the finish step (returns the accumulated State) or closes
 *   the window (returns nil). Back/Next chain the 4-5 steps
 *   (welcome → language → hotkey → tts? → finish).
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

// gtkInitOnce guards gtk.Init so we never call it twice in the same
// process. Gotk3's Init aborts on the second call.
var gtkInitOnce sync.Once
var gtkInitErr error

// CID:wizard-002 - ensureInit
// Purpose: idempotently initialize GTK. Returns the first init error
// (typically a missing DISPLAY). All entry points that touch GTK
// call this before constructing widgets.
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

	step.Next.Connect("clicked", func() {
		select {
		case result <- true:
		default:
		}
		gtk.MainQuit()
	})

	win.Connect("destroy", func() {
		select {
		case result <- false:
		default:
		}
		gtk.MainQuit()
	})

	win.ShowAll()
	gtk.Main()
	win.Destroy()

	return <-result, nil
}

// CID:wizard-004 - RunFull
// Purpose: present the 4-5 step wizard (welcome → language → hotkey
// → tts? → finish), block the calling goroutine on the GTK main loop,
// and return the accumulated State when the user clicks "Start".
//
// The runner walks a registry of step renderers. The chain is
// rebuilt on every transition so a Back into language followed by
// a switch from "en" to "de" inserts the TTS step on the way
// forward (and vice versa).
//
// On Next click the runner:
//   1. Calls step.Capture(state) to commit the user's choice.
//   2. If Capture returns an error, the advance is aborted and the
//      error is shown via a small dialog so the user can fix it.
//   3. Otherwise the next step's box is added and shown.
//
// On Back click the previous step's box is re-shown (the prior step
// is re-built so its widgets reflect the current State).
//
// On window close (X button) the runner returns nil and the state is
// not committed.
func RunFull() (*State, error) {
	if err := ensureInit(); err != nil {
		return nil, fmt.Errorf("wizard: gtk init: %w", err)
	}

	win, err := NewWindow()
	if err != nil {
		return nil, err
	}

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
		win.Add(step.Box)
		step.Box.ShowAll()

		step.Next.Connect("clicked", func() {
			if step.Capture != nil {
				if err := step.Capture(state); err != nil {
					showError(win, err)
					return
				}
			}
			step.Box.Hide()
			keys = chain()
			if idx+1 >= len(keys) {
				select {
				case result <- state:
				default:
				}
				gtk.MainQuit()
				return
			}
			idx++
			if err := showStepAt(); err != nil {
				showError(win, err)
			}
		})
		if step.Back != nil {
			step.Back.Connect("clicked", func() {
				step.Box.Hide()
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

	// Show the window + the first step. Without this call, gtk.Main()
	// blocks indefinitely on a hidden window (the step box's ShowAll
	// only marks the box and its children visible — the top-level
	// GtkWindow stays hidden and nothing is rendered to the screen).
	// Mirrors what RunWelcome does at the end of its setup.
	win.ShowAll()

	win.Connect("destroy", func() {
		select {
		case result <- nil:
		default:
		}
		gtk.MainQuit()
	})

	gtk.Main()
	win.Destroy()
	return <-result, nil
}
