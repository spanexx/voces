/* Code Map: TTS step GTK factory (rc1-hotpatch-29)
 *
 * BuildTTS materialises the TTS step's GTK widgets. Split out
 * of tts.go so the file stays under the 250-line size cap
 * enforced by scripts/check-file-size.sh.
 *
 * The step shows:
 *   - a hint label that links to the piper-voices catalogue
 *     (VOICES.md) and explains how to copy a voice link
 *   - a Voice: dropdown with the curated voices filtered by
 *     language, plus a "Custom URL..." row that reveals two
 *     text inputs
 *   - the two text inputs (hidden by default) for the .onnx
 *     and .onnx.json URLs
 *
 * Pre-select logic:
 *   - If the prior TTSVoice is a custom URL sentinel, the
 *     "Custom URL..." row is active and the inputs are
 *     pre-filled with the parsed URLs.
 *   - If the prior TTSVoice is a manifest key, that row is
 *     active. If the key is no longer in the manifest (the
 *     voice was removed in a later build), we fall back to
 *     the first curated voice for the language.
 *   - Otherwise the first curated voice is active (the
 *     default).
 *
 * CID Index:
 * CID:wizard-ttsstep-002 -> BuildTTS
 *
 * Quick lookup: rg -n "CID:wizard-ttsstep-" internal/wizard/steps/
 */
package steps

import (
	"fmt"
	"strings"

	"github.com/gotk3/gotk3/gtk"

	"voces/internal/setup"
)

// CID:wizard-ttsstep-002 - BuildTTS
// Purpose: rc1-hotpatch-29. Build the GTK step that lets the
// user pick a piper voice (curated list + custom URL escape
// hatch). The hint label explains how to copy a voice link
// from the full piper catalogue for cases the curated list
// doesn't cover. See the file header for the pre-select
// logic and the dropdown's row mapping.
func BuildTTS(win *gtk.Window, stateReader StateReader) (*Step, error) {
	manifest := setup.DefaultManifest()

	voices := filterVoicesForLanguage(manifest, stateReader.LanguageCode())
	if len(voices) == 0 {
		return nil, fmt.Errorf("tts: no voices available for language %q (manifest has no piper entries)", stateReader.LanguageCode())
	}

	// customURLMarker is the special ComboBoxText entry the
	// user picks to reveal the URL inputs. We use a sentinel
	// string ("__custom__") that's never a real voice ID.
	const customURLMarker = "__custom__"

	// Build the dropdown entries. We track (combo-index, kind, value)
	// so the Capture closure can map the active row back to
	// either a voice ID or the custom marker.
	type entry struct {
		kind  string // "voice" or "custom"
		value string // voice ID or empty
		label string // text shown in the dropdown
	}
	entries := make([]entry, 0, len(voices)+1)
	combo, err := gtk.ComboBoxTextNew()
	if err != nil {
		return nil, fmt.Errorf("tts: combobox: %w", err)
	}
	for _, v := range voices {
		combo.AppendText(v.DisplayName)
		entries = append(entries, entry{kind: "voice", value: v.ID, label: v.DisplayName})
	}
	combo.AppendText("Custom URL...")
	entries = append(entries, entry{kind: "custom", value: customURLMarker, label: "Custom URL..."})

	// onnxEntry / jsonEntry are the two URL inputs that
	// appear when "Custom URL..." is picked. The actual
	// widget assembly is extracted into
	// buildTTSCustomURLInputs to keep BuildTTS under the
	// 250-line file-size cap.
	customBox, onnxEntry, jsonEntry, err := buildTTSCustomURLInputs()
	if err != nil {
		return nil, fmt.Errorf("tts: custom url inputs: %w", err)
	}

	// Pre-select logic. The user's prior pick is either a
	// manifest voice ID, a custom URL sentinel, or empty.
	prior := ""
	if stateReader != nil {
		prior = ttsVoiceFromStateReader(stateReader)
	}
	activeIdx := 0 // default: first curated voice
	prefillOnnx := ""
	prefillJson := ""
	switch {
	case prior != "" && isCustomURLVoice(prior):
		// Prior pick was a custom URL — find the
		// "Custom URL..." row, fill the inputs.
		onnx, cfg, _ := parseCustomURLSentinel(prior)
		prefillOnnx = onnx
		prefillJson = cfg
		// The custom row is always last.
		activeIdx = len(voices)
	case prior != "":
		// Prior pick is a manifest key. Find it in
		// the dropdown, fall back to the first
		// voice if not found (e.g. user picked a
		// voice that was removed from the
		// manifest in a later build).
		activeIdx = 0
		for i, e := range entries {
			if e.kind == "voice" && e.value == prior {
				activeIdx = i
				break
			}
		}
	}
	combo.SetActive(activeIdx)
	if prefillOnnx != "" {
		onnxEntry.SetText(prefillOnnx)
	}
	if prefillJson != "" {
		jsonEntry.SetText(prefillJson)
	}
	// Initial visibility: only show the URL inputs when
	// the pre-selected row is the custom marker.
	if activeIdx == len(voices) {
		customBox.ShowAll()
	}

	// Show/hide the URL inputs whenever the dropdown
	// changes. Connect after the pre-select so the initial
	// "Hide" doesn't get clobbered.
	combo.Connect("changed", func() {
		idx := combo.GetActive()
		if idx < 0 {
			return
		}
		if entries[idx].kind == "custom" {
			customBox.ShowAll()
		} else {
			customBox.Hide()
		}
	})

	// Hint label with the VOICES.md link and how-to-copy
	// instructions. The full URL is plain text (not a
	// <a href> tag) because the user copies the URL, not
	// the link text — wrapping it in markup would make
	// the URL itself harder to read. Extracted into
	// buildTTSHintLabel so BuildTTS stays under the
	// 250-line size cap.
	hint, err := buildTTSHintLabel()
	if err != nil {
		return nil, fmt.Errorf("tts: hint label: %w", err)
	}

	// The dropdown row. Label + ComboBoxText in a single
	// horizontal box so the alignment matches the URL rows
	// below it when "Custom URL..." is selected.
	dropdownLabel, err := gtk.LabelNew("Voice:")
	if err != nil {
		return nil, fmt.Errorf("tts: dropdown label: %w", err)
	}
	dropdownLabel.SetHAlign(gtk.ALIGN_START)
	dropdownLabel.SetXAlign(0)
	dropdownRow, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	if err != nil {
		return nil, fmt.Errorf("tts: dropdown row: %w", err)
	}
	dropdownRow.PackStart(dropdownLabel, false, false, 0)
	dropdownRow.PackStart(combo, true, true, 0)

	content, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	if err != nil {
		return nil, fmt.Errorf("tts: content box: %w", err)
	}
	content.PackStart(hint, false, false, 0)
	content.PackStart(dropdownRow, false, false, 0)
	content.PackStart(customBox, false, false, 0)

	box, back, next, err := newStepContent("Pick a voice", content, "Back", "Next")
	if err != nil {
		return nil, fmt.Errorf("tts: build content: %w", err)
	}

	capture := func(setter StateSetter) error {
		idx := combo.GetActive()
		if idx < 0 || idx >= len(entries) {
			return fmt.Errorf("tts: no dropdown selection")
		}
		chosen := entries[idx]
		if chosen.kind == "custom" {
			onnx, err := onnxEntry.GetText()
			if err != nil {
				return fmt.Errorf("tts: read onnx entry: %w", err)
			}
			json, err := jsonEntry.GetText()
			if err != nil {
				return fmt.Errorf("tts: read json entry: %w", err)
			}
			onnx = strings.TrimSpace(onnx)
			json = strings.TrimSpace(json)
			if onnx == "" {
				return fmt.Errorf("tts: custom voice needs an .onnx URL")
			}
			setter.SetTTS(true)
			setter.SetTTSVoice(customURLSentinel(onnx, json))
			return nil
		}
		// Picked a curated voice — always set TTSEnabled
		// so the downloader fetches it. The wizard's
		// "skip TTS" path is the Behavior step's
		// "start on login" No + the user's ability to
		// back out before Start. If the user wants to
		// skip TTS entirely, they can re-run the
		// wizard and pick the piper-status step's
		// "Skip" option.
		setter.SetTTS(true)
		setter.SetTTSVoice(chosen.value)
		return nil
	}

	// Note: do not win.Add(box) here. The runner is the
	// single source of truth for attaching the step box
	// to the window (wizard.go showStepAt). Double-adding
	// raises: "Attempting to add a widget with type
	// GtkBox to a container of type GtkWindow, but the
	// widget is already inside a container of type
	// GtkWindow".
	return &Step{Box: box, Next: next, Back: back, Capture: capture}, nil
}
