// Package wizard drives the GTK setup wizard.
// RunWelcome = single step (back-compat). RunFull = 4-5 step chain.
package wizard

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/gotk3/gotk3/gtk"

	"voces/internal/download"
	"voces/internal/wizard/steps"
)

// CommitFunc runs after the user clicks "Start" on the finish step.
// It runs in a background goroutine. The progress callback may be
// invoked from any goroutine; the wizard marshals UI updates onto
// the GTK main thread via glib.IdleAdd. A non-nil error is shown
// in a GTK error dialog and the user is sent back to the finish
// step to retry.
//
// Pass nil from the wizard-only entry point (the tray's "Run setup
// again..." menu), which spawns a subprocess and does the download
// in the parent after the wizard returns.
type CommitFunc func(ctx context.Context, state *State, progress download.ProgressFunc) error

// CID:wizard-001 - AppVersion
// Purpose: rendered in the welcome footer. A future phase will wire
// this to a ldflags -X override at build time.
//
// rc1-hotpatch-26: changed from `const` to `var` and seeded with
// "dev" so the cmd entrypoint can inject the build's Version at
// process start (cmd/voces/main.go calls
// `wizard.AppVersion = stripV(Version)` before RunFull). The
// "0.1.0" hardcode was the rc1-hotpatch-14 initial value; with
// the 6-step wizard + model picker (rc24) the hardcoded value
// was visibly stale, so callers that want the build's real
// version (e.g. v0.2.0-rc11) set it explicitly. The default
// "dev" matches the ldflags default in main.go and keeps tests
// independent of any build flag.
var AppVersion = "dev"

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

	win, _, err := NewWindow()
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
	// full explanation). The destroy event fires twice - once when
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
// Purpose: present the 4-5 step wizard (welcome -> language -> hotkey
// -> tts? -> finish), block on the GTK main loop, return the
// accumulated State when the user clicks "Start" (or nil on window
// close).
//
// The chain is rebuilt on every transition so a Back into language +
// change of language inserts/removes the TTS step on the way forward.
// The step-rendering, chain-building, and click-wiring logic lives
// in navigate.go; this function owns the GTK window, the wrapper
// contentBox, the result channel, the finish closure, and the
// gtk.Main loop.
//
// When the user clicks "Start" on the finish step, the wizard swaps
// in a "Downloading..." view and calls commit (when non-nil) from a
// goroutine. commit's progress callback updates a progress bar from
// the GTK main thread via glib.IdleAdd. When commit returns, the
// wizard finishes. This avoids the "Voces is not responding" overlay
// that happened when EnsureModels ran on the main thread after the
// wizard returned (rc1-hotpatch-13).
//
// commit may be nil - for example, the wizard-only entry point from
// the tray menu spawns a subprocess and does the download in the
// parent. When nil, "Start" closes the wizard without a download.
func RunFull(commit CommitFunc) (*State, error) {
	if err := ensureInit(); err != nil {
		return nil, fmt.Errorf("wizard: gtk init: %w", err)
	}

	// NewWindow returns the contentBox (the step-swap slot)
	// alongside the window. The window now ships with a
	// styled header + accent strip baked in, so the runner
	// does not own the outer container anymore.
	win, contentBox, err := NewWindow()
	if err != nil {
		return nil, err
	}

	state := NewState()
	result := make(chan *State, 1)

	// currentBox tracks the step's box currently parented under
	// contentBox. showStepAt removes it before adding the new one,
	// so the wrapper (and the window) always has exactly one child.
	var currentBox *gtk.Box

	// finish is the single exit point. sync.Once prevents a double
	// gtk.MainQuit (which would trigger GTK's
	// "main_loops != NULL" assertion and kill the process with
	// SIGKILL): the explicit win.Destroy() after gtk.Main() returns
	// re-fires the destroy event - finish() is then a no-op.
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

	registry := buildStepRegistry()
	keys := buildStepChain(state)
	idx := 0

	if err := showStepAt(win, contentBox, &currentBox, state, registry, &keys, &idx, commit, finish); err != nil {
		win.Destroy()
		return nil, fmt.Errorf("wizard: build step %d: %w", idx, err)
	}

	// Show the window + the first step. ShowAll on step.Box alone
	// doesn't render (the GtkWindow is still hidden). Mirrors
	// what RunWelcome does at the end of its setup.
	log.Printf("wizard: about to show window and enter gtk.Main()")
	win.ShowAll()
	win.Connect("destroy", func() {
		log.Printf("wizard: destroy event fired")
		finish(nil)
	})

	gtk.Main()
	log.Printf("wizard: gtk.Main() returned, calling win.Destroy()")
	win.Destroy()
	log.Printf("wizard: win.Destroy() returned, reading result channel")
	return <-result, nil
}
