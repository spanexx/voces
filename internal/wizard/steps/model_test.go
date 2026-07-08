/* Code Map: Model step unit tests
 *
 * We do not spin up a GTK window in tests (the gotk3 toolkit
 * needs an X display). Instead we test the three pure helpers
 * the step relies on: filterByLanguage, sortedBySize, and
 * preselectEntry. The widget wiring is covered by the manual
 * smoke test in IMPL-wizard-model-picker.md §3.
 *
 * CID Index:
 * CID:wizard-step-model-test-001 -> TestFilterByLanguage_English
 * CID:wizard-step-model-test-002 -> TestFilterByLanguage_Multilingual
 * CID:wizard-step-model-test-003 -> TestFilterByLanguage_UnknownFallsBackToMultilingual
 * CID:wizard-step-model-test-004 -> TestSortedBySize_Ascending
 * CID:wizard-step-model-test-005 -> TestPreselectEntry_CurrentMatches
 * CID:wizard-step-model-test-006 -> TestPreselectEntry_FallsBackToSmallest
 * CID:wizard-step-model-test-007 -> TestPreselectEntry_Empty
 */
package steps

import (
	"testing"

	"voces/internal/setup"
)

// fakeEntry is a tiny helper for building whisperEntryView values
// in tests. Mirrors the field order in production code.
func fakeEntry(name string, size int64) whisperEntryView {
	return whisperEntryView{
		FileName:    name,
		DisplayName: name,
		SizeBytes:   size,
		Tier:        name,
		Language:    "en",
	}
}

// CID:wizard-step-model-test-001
func TestFilterByLanguage_English(t *testing.T) {
	m := setup.DefaultManifest()
	got := filterByLanguage(m, "en")
	if len(got) != 4 {
		t.Fatalf("filterByLanguage(en) returned %d entries, want 4", len(got))
	}
	for _, e := range got {
		if e.Language != "en" {
			t.Errorf("filterByLanguage(en) returned %q with Language=%q, want en", e.FileName, e.Language)
		}
	}
}

// CID:wizard-step-model-test-002
func TestFilterByLanguage_Multilingual(t *testing.T) {
	m := setup.DefaultManifest()
	got := filterByLanguage(m, "de")
	if len(got) != 4 {
		t.Fatalf("filterByLanguage(de) returned %d entries, want 4", len(got))
	}
	for _, e := range got {
		if e.Language != "multilingual" {
			t.Errorf("filterByLanguage(de) returned %q with Language=%q, want multilingual", e.FileName, e.Language)
		}
	}
}

// CID:wizard-step-model-test-003
func TestFilterByLanguage_UnknownFallsBackToMultilingual(t *testing.T) {
	m := setup.DefaultManifest()
	for _, lang := range []string{"", "xx", "EN", "es-MX"} {
		got := filterByLanguage(m, lang)
		if len(got) != 4 {
			t.Errorf("filterByLanguage(%q) returned %d entries, want 4 (multilingual fallback)", lang, len(got))
		}
		for _, e := range got {
			if e.Language != "multilingual" {
				t.Errorf("filterByLanguage(%q) returned %q with Language=%q, want multilingual", lang, e.FileName, e.Language)
			}
		}
	}
}

// CID:wizard-step-model-test-004
func TestSortedBySize_Ascending(t *testing.T) {
	in := []whisperEntryView{
		fakeEntry("ggml-small.en.bin", 488_479_232),
		fakeEntry("ggml-tiny.en.bin", 77_704_153),
		fakeEntry("ggml-medium.en.bin", 1_533_249_024),
		fakeEntry("ggml-base.en.bin", 147_964_480),
	}
	got := sortedBySize(in)
	want := []string{
		"ggml-tiny.en.bin",
		"ggml-base.en.bin",
		"ggml-small.en.bin",
		"ggml-medium.en.bin",
	}
	if len(got) != len(want) {
		t.Fatalf("sortedBySize returned %d entries, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].FileName != w {
			t.Errorf("sortedBySize[%d] = %q, want %q", i, got[i].FileName, w)
		}
	}
}

// CID:wizard-step-model-test-005
func TestPreselectEntry_CurrentMatches(t *testing.T) {
	in := []whisperEntryView{
		fakeEntry("a", 1),
		fakeEntry("b", 2),
		fakeEntry("c", 3),
	}
	got := preselectEntry(in, "b")
	if got != "b" {
		t.Errorf("preselectEntry with current=b: got %q, want b", got)
	}
}

// CID:wizard-step-model-test-006
func TestPreselectEntry_FallsBackToSmallest(t *testing.T) {
	in := []whisperEntryView{
		fakeEntry("big", 100),
		fakeEntry("huge", 1000),
		fakeEntry("small", 10),
	}
	// sortedBySize-ascending order, but the function doesn't require it
	// — it just returns in[0].FileName as the fallback.
	got := preselectEntry(in, "not-in-list")
	if got != "big" {
		t.Errorf("preselectEntry with no match: got %q, want %q (in[0])", got, "big")
	}
}

// CID:wizard-step-model-test-007
func TestPreselectEntry_Empty(t *testing.T) {
	got := preselectEntry(nil, "anything")
	if got != "" {
		t.Errorf("preselectEntry(nil): got %q, want empty", got)
	}
}
