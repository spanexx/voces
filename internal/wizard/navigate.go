/* Code Map: Wizard Step Navigation
 * - buildStepRegistry: assembles the stepKey -> stepRenderer map.
 *   All BuildX functions share the same signature; centralising
 *   the map here keeps the runner ignorant of step types.
 * - buildStepChain: returns the ordered stepKeys for a State.
 *   The TTS step is omitted when the language is English (TTS
 *   is only useful for non-English transcription output).
 * - showStepAt: renders the step at idx, swaps its box into
 *   contentBox, and wires Next/Back click handlers. Next calls
 *   Capture, rebuilds the chain (a language change may add or
 *   remove the TTS step), and recurses to the next step. On the
 *   finish step with a non-nil commit, swaps in a Downloading
 *   view and hands off to startCommit. Back rebuilds the chain
 *   and recurses to the prior step.
 *
 * Split out from wizard.go so the main file stays under the
 * 250-line size cap enforced by scripts/check-file-size.sh.
 *
 * CID Index:
 * CID:wizard-nav-001 -> buildStepRegistry
 * CID:wizard-nav-002 -> buildStepChain
 * CID:wizard-nav-003 -> showStepAt
 *
 * Quick lookup: rg -n "CID:wizard-nav-" internal/wizard/
 */
package wizard

import (
	"log"

	"github.com/gotk3/gotk3/gtk"

	"voces/internal/wizard/steps"
)

// CID:wizard-nav-001 - buildStepRegistry
// Purpose: returns the stepKey -> stepRenderer map. Each renderer
// calls the matching steps.BuildX(win, state) factory. Centralised
// here so the runner iterates a map and the chain builder only
// needs the key list.
func buildStepRegistry() map[stepKey]stepRenderer {
	return map[stepKey]stepRenderer{
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
		stepBehavior: func(win *gtk.Window, s *State) (*steps.Step, error) {
			return steps.BuildBehavior(win, s)
		},
		stepSecondaryHotkeys: func(win *gtk.Window, s *State) (*steps.Step, error) {
			return steps.BuildSecondaryHotkeys(win, s)
		},
		stepFinish: func(win *gtk.Window, s *State) (*steps.Step, error) {
			return steps.BuildFinish(win, s)
		},
	}
}

// CID:wizard-nav-002 - buildStepChain
// Purpose: returns the ordered stepKeys for the user's current
// state. The TTS step is omitted when the language is English.
// The chain is rebuilt on every transition so a Back into
// language + change of language inserts/removes the TTS step
// on the way forward.
//
// rc1-hotpatch-14: the Behavior and SecondaryHotkeys steps are
// always shown (they are short and have sensible defaults for
// users who want to skip the customization).
func buildStepChain(state *State) []stepKey {
	keys := []stepKey{stepWelcome, stepLanguage, stepHotkey}
	if steps.ShouldShow(state.LanguageCode()) {
		keys = append(keys, stepTTS)
	}
	keys = append(keys, stepBehavior, stepSecondaryHotkeys, stepFinish)
	return keys
}

// CID:wizard-nav-003 - showStepAt
// Purpose: render the step at the (mutated) idx, swap its box
// into contentBox, and wire the Next/Back click handlers.
//
// Next calls step.Capture(state) when defined, rebuilds the
// chain (a language change may add or remove the TTS step),
// then recurses to the next step. On the finish step with a
// non-nil commit, swaps in a Downloading view and hands off
// to startCommit. Back rebuilds the chain and recurses to
// the prior step.
//
// The pointers to currentBox, keys, and idx are mutated here
// so the caller (RunFull) sees the same state on the next
// show. Passing them as pointers keeps the runner's state
// visible to the closure without a wrapper struct.
func showStepAt(
	win *gtk.Window,
	contentBox *gtk.Box,
	currentBoxRef **gtk.Box,
	state *State,
	registry map[stepKey]stepRenderer,
	keys *[]stepKey,
	idx *int,
	commit CommitFunc,
	finish func(*State),
) error {
	if *idx < 0 || *idx >= len(*keys) {
		return nil
	}
	k := (*keys)[*idx]
	step, err := registry[k](win, state)
	if err != nil {
		return err
	}
	// Swap the step's box into the wrapper. Removing the
	// previous box (if any) is what was missing - Hide() alone
	// left it parented and triggered the GtkWindow warning.
	if *currentBoxRef != nil {
		contentBox.Remove(*currentBoxRef)
	}
	contentBox.Add(step.Box)
	*currentBoxRef = step.Box
	step.Box.ShowAll()

	step.Next.Connect("clicked", func() {
		log.Printf("wizard: Next clicked on step idx=%d k=%v len(keys)=%d", *idx, k, len(*keys))
		if step.Capture != nil {
			if err := step.Capture(state); err != nil {
				log.Printf("wizard: Capture error: %v", err)
				showError(win, err)
				return
			}
		}
		*keys = buildStepChain(state)
		log.Printf("wizard: rebuilt chain len=%d", len(*keys))
		if *idx+1 >= len(*keys) {
			// Last step: the user clicked "Start" on
			// the finish step. If commit is wired up,
			// run it in-place with a progress view.
			// Otherwise just close the wizard.
			log.Printf("wizard: finish step, commit=%v", commit != nil)
			if commit != nil {
				startCommit(win, contentBox, currentBoxRef, state, commit, finish)
				return
			}
			log.Printf("wizard: finish() called from click handler (no commit)")
			finish(state)
			log.Printf("wizard: finish() returned, waiting for gtk.Main() to exit")
			return
		}
		*idx++
		if err := showStepAt(win, contentBox, currentBoxRef, state, registry, keys, idx, commit, finish); err != nil {
			showError(win, err)
		}
	})
	if step.Back != nil {
		step.Back.Connect("clicked", func() {
			*keys = buildStepChain(state)
			if *idx > 0 {
				*idx--
			}
			if err := showStepAt(win, contentBox, currentBoxRef, state, registry, keys, idx, commit, finish); err != nil {
				showError(win, err)
			}
		})
	}
	return nil
}
