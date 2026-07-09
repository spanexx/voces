/* Code Map: Custom URL input widget builder (rc1-hotpatch-29)
 *
 * Extracted from tts_build.go so the main BuildTTS factory
 * stays under the 250-line file-size cap enforced by
 * scripts/check-file-size.sh. The two URL inputs (and their
 * surrounding hidden box) are a self-contained piece of the
 * step UI: BuildTTS only needs to pack the box and read the
 * entries, so this helper returns both halves together.
 *
 * CID Index:
 * CID:wizard-ttsstep-011 -> buildTTSCustomURLInputs
 *
 * Quick lookup: rg -n "CID:wizard-ttsstep-" internal/wizard/steps/
 */
package steps

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

// CID:wizard-ttsstep-011 - buildTTSCustomURLInputs
// Purpose: assembles the two .onnx / .onnx.json URL inputs
// the user sees when "Custom URL..." is picked from the
// dropdown. Returns a hidden GtkBox that BuildTTS can pack
// into the content column and the two Entry widgets the
// Capture closure reads from. Extracted from BuildTTS so the
// main step factory stays under the 250-line size cap.
func buildTTSCustomURLInputs() (*gtk.Box, *gtk.Entry, *gtk.Entry, error) {
	onnxEntry, err := gtk.EntryNew()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("onnx entry: %w", err)
	}
	onnxEntry.SetPlaceholderText("https://huggingface.co/.../voice.onnx")
	onnxLabel, err := gtk.LabelNew(".onnx URL:")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("onnx label: %w", err)
	}
	onnxLabel.SetHAlign(gtk.ALIGN_START)
	onnxLabel.SetXAlign(0)

	jsonEntry, err := gtk.EntryNew()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("json entry: %w", err)
	}
	jsonEntry.SetPlaceholderText("https://huggingface.co/.../voice.onnx.json (optional)")
	jsonLabel, err := gtk.LabelNew(".onnx.json URL:")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("json label: %w", err)
	}
	jsonLabel.SetHAlign(gtk.ALIGN_START)
	jsonLabel.SetXAlign(0)

	customBox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 6)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("custom box: %w", err)
	}
	onnxRow, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("onnx row: %w", err)
	}
	onnxRow.PackStart(onnxLabel, false, false, 0)
	onnxRow.PackStart(onnxEntry, true, true, 0)
	jsonRow, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("json row: %w", err)
	}
	jsonRow.PackStart(jsonLabel, false, false, 0)
	jsonRow.PackStart(jsonEntry, true, true, 0)
	customBox.PackStart(onnxRow, false, false, 0)
	customBox.PackStart(jsonRow, false, false, 0)
	customBox.SetNoShowAll(true)
	customBox.Hide()

	return customBox, onnxEntry, jsonEntry, nil
}

// CID:wizard-ttsstep-012 - buildTTSHintLabel
// Purpose: builds the hint label that links to the piper-
// voices catalogue and explains how to copy a voice link.
// The full URL is plain text (not a <a href> tag) because
// the user copies the URL, not the link text — wrapping it
// in markup would make the URL itself harder to read.
// Extracted from BuildTTS so the main step factory stays
// under the 250-line size cap.
func buildTTSHintLabel() (*gtk.Label, error) {
	hintText := fmt.Sprintf(
		"Voces uses Piper voices hosted on HuggingFace. "+
			"Pick a voice from the dropdown, or pick "+
			"“Custom URL…” to use any voice from the full "+
			"library.\n\n"+
			"Full library: %s\n\n"+
			"To use a voice not in the list: open that "+
			"page, click a voice’s .onnx file, then copy the "+
			"URL from the address bar (right-click the “Raw” "+
			"or “download” button → “Copy link address”). "+
			"Paste the link into the “Custom URL…” field.",
		piperVoicesDocURL,
	)
	hint, err := gtk.LabelNew(hintText)
	if err != nil {
		return nil, fmt.Errorf("hint label: %w", err)
	}
	hint.SetLineWrap(true)
	hint.SetHAlign(gtk.ALIGN_START)
	hint.SetXAlign(0)
	hint.SetMarginBottom(12)
	return hint, nil
}
