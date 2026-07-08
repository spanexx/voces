/* Code Map: Model picker wizard step
 * - BuildModel: builds the "Speech recognition model" step. Shows
 *   one radio per matching entry in the manifest (filtered by the
 *   language scope chosen on the prior step). Each radio shows the
 *   entry's DisplayName which already includes a size hint
 *   (e.g. "Base (English, ~141 MB)"). The default selection is
 *   the entry whose file name equals the state's existing pick
 *   (or the smallest entry if the pick is empty/unknown).
 *   Capture writes the chosen file name back into state via
 *   SetModel, which is a no-op on empty input.
 *
 * - sortedBySize: returns a copy of the manifest's whisper entries
 *   sorted ascending by SizeBytes. Stable enough for the picker
 *   to show "smallest first" and pre-select the smallest when
 *   nothing else is set.
 *
 * - preselectEntry: returns the file name that should be
 *   pre-selected. If the user's current pick matches an entry,
 *   use it (so re-runs preserve the prior choice). Otherwise
 *   use the smallest entry. Returns empty string if the list is
 *   empty.
 *
 * - filterByLanguage: returns the entries whose Language matches
 *   the chosen scope (en for English, multilingual otherwise).
 *   Unknown languages fall through to multilingual, matching
 *   the fallback in DefaultModelForLanguage.
 *
 * rc1-hotpatch-24 — the wizard gains a model picker so the user
 * can override the language-implied default. See
 * docs/wizard-model-picker/PRD-wizard-model-picker.md and
 * docs/wizard-model-picker/IMPL-wizard-model-picker.md.
 *
 * CID Index:
 * CID:wizard-step-model-001 -> BuildModel
 * CID:wizard-step-model-002 -> sortedBySize
 * CID:wizard-step-model-003 -> preselectEntry
 * CID:wizard-step-model-004 -> filterByLanguage
 *
 * Quick lookup: rg -n "CID:wizard-step-model-" internal/wizard/steps/
 */
package steps

import (
	"sort"

	"github.com/gotk3/gotk3/gtk"

	"voces/internal/setup"
)

// CID:wizard-step-model-001 - BuildModel
// Purpose: builds the "Speech recognition model" wizard step. The
// manifest is the canonical source of available models (8 entries
// after rc1-hotpatch-24). The picker filters by language scope:
// "en" picks the 4 .en variants, any other language picks the 4
// multilingual variants. Radios are sorted ascending by size so
// the smallest model is at the top.
//
// Preselect logic: if the state's existing pick matches one of the
// filtered entries, pre-select it (preserves the user's previous
// choice on a wizard re-run). Otherwise pre-select the smallest
// entry. This means a user with no prior pick sees a sensible
// starting point (ADR-0004 routing has already been applied by
// NewState / language commit).
func BuildModel(_ *gtk.Window, s StateReader) (*Step, error) {
	manifest := setup.DefaultManifest()
	filtered := filterByLanguage(manifest, s.LanguageCode())
	sorted := sortedBySize(filtered)
	defaultPick := preselectEntry(sorted, s.ModelFile())

	// Track each radio button with the file name it represents.
	// Capture walks the list to find the active one and commits
	// the matching file name. Keeping the lookup in the step
	// (instead of widget user-data) avoids the gotk3 SetData
	// finalizer dance and keeps the Step a pure-Go value.
	type radioBinding struct {
		btn      *gtk.RadioButton
		fileName string
	}
	bindings := make([]radioBinding, 0, len(sorted))

	content, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	if err != nil {
		return nil, err
	}
	hint, err := gtk.LabelNew(
		"Pick a model. Larger models give better accuracy but take longer to download and run. The default works for most users.",
	)
	if err != nil {
		return nil, err
	}
	hint.SetLineWrap(true)
	hint.SetHAlign(gtk.ALIGN_START)
	hint.SetMarginBottom(12)
	content.PackStart(hint, false, false, 0)

	var firstBtn *gtk.RadioButton
	for _, entry := range sorted {
		btn, err := gtk.RadioButtonNewWithLabelFromWidget(firstBtn, entry.DisplayName)
		if err != nil {
			return nil, err
		}
		if firstBtn == nil {
			firstBtn = btn
		}
		if entry.FileName == defaultPick {
			btn.SetActive(true)
		}
		content.PackStart(btn, false, false, 0)
		bindings = append(bindings, radioBinding{btn: btn, fileName: entry.FileName})
	}

	box, back, next, err := newStepContent("Speech recognition model", content, "Back", "Next")
	if err != nil {
		return nil, err
	}

	return &Step{
		Box:  box,
		Next: next,
		Back: back,
		Capture: func(setter StateSetter) error {
			for _, b := range bindings {
				if b.btn.GetActive() {
					setter.SetModel(b.fileName)
					return nil
				}
			}
			// No radio active (shouldn't happen — firstBtn is
			// always preselected). Leave the state untouched so
			// the runtime falls back to DefaultModelForLanguage.
			return nil
		},
	}, nil
}

// CID:wizard-step-model-002 - sortedBySize
// Purpose: returns a copy of the entries sorted ascending by
// SizeBytes. Pure helper so the test in model_test.go can pin
// the order without spinning up a wizard.
func sortedBySize(in []whisperEntryView) []whisperEntryView {
	out := make([]whisperEntryView, len(in))
	copy(out, in)
	sort.Slice(out, func(i, j int) bool { return out[i].SizeBytes < out[j].SizeBytes })
	return out
}

// CID:wizard-step-model-003 - preselectEntry
// Purpose: returns the file name that should be pre-selected.
// If the user's current pick matches an entry in `in`, use it
// (so re-runs preserve the prior choice). Otherwise use the
// smallest entry (sortedBySize guarantees sorted ascending).
// Returns empty string if `in` is empty.
func preselectEntry(in []whisperEntryView, current string) string {
	if len(in) == 0 {
		return ""
	}
	if current != "" {
		for _, e := range in {
			if e.FileName == current {
				return current
			}
		}
	}
	return in[0].FileName
}

// CID:wizard-step-model-004 - filterByLanguage
// Purpose: returns the whisper entries that match the chosen
// language. "en" → 4 .en variants. Anything else → 4
// multilingual variants. Unknown languages (empty, future
// locales) fall through to multilingual, matching the
// fallback in DefaultModelForLanguage.
func filterByLanguage(m *setup.Manifest, lang string) []whisperEntryView {
	scope := "multilingual"
	if lang == "en" {
		scope = "en"
	}
	var out []whisperEntryView
	for name, e := range m.Whisper {
		if e.Language != scope {
			continue
		}
		out = append(out, whisperEntryView{
			FileName:    name,
			DisplayName: e.DisplayName,
			SizeBytes:   e.SizeBytes,
			Tier:        e.Tier,
			Language:    e.Language,
		})
	}
	return out
}

// whisperEntryView is a flattened view of a single whisper
// manifest entry for the model step. The picker only needs a
// handful of fields; passing the whole setup.WhisperModelMeta
// would couple the step to the manifest type and pull the
// Piper map in transitively.
type whisperEntryView struct {
	FileName    string
	DisplayName string
	SizeBytes   int64
	Tier        string
	Language    string
}
