/* Code Map: Wizard State tests
 * - TestDefaultModelForLanguage: confirms the ADR-0004 routing
 *   (en → small.en, anything else → base) is the single source of
 *   truth.
 * - TestNewState_ModelDefault: NewState pre-fills Model with the
 *   English default so the model step's preselect works on a
 *   fresh run.
 * - TestSetModel_PreservesOnEmpty: SetModel is a no-op on empty
 *   input (skipped step does not erase State).
 * - TestSetModel_OverwritesOnValue: SetModel writes the new value.
 *
 * CID Index:
 * CID:wizard-state-test-001 -> TestDefaultModelForLanguage
 * CID:wizard-state-test-002 -> TestNewState_ModelDefault
 * CID:wizard-state-test-003 -> TestSetModel_PreservesOnEmpty
 * CID:wizard-state-test-004 -> TestSetModel_OverwritesOnValue
 */
package wizard

import "testing"

// CID:wizard-state-test-001
func TestDefaultModelForLanguage(t *testing.T) {
	cases := []struct {
		lang string
		want string
	}{
		{"en", "ggml-small.en.bin"},
		{"de", "ggml-base.bin"},
		{"es", "ggml-base.bin"},
		{"", "ggml-base.bin"},
		{"EN", "ggml-base.bin"}, // case-sensitive; only exact "en" is English
	}
	for _, c := range cases {
		got := DefaultModelForLanguage(c.lang)
		if got != c.want {
			t.Errorf("DefaultModelForLanguage(%q) = %q, want %q", c.lang, got, c.want)
		}
	}
}

// CID:wizard-state-test-002
func TestNewState_ModelDefault(t *testing.T) {
	s := NewState()
	if s.Model != "ggml-small.en.bin" {
		t.Errorf("NewState().Model = %q, want %q", s.Model, "ggml-small.en.bin")
	}
}

// CID:wizard-state-test-003
func TestSetModel_PreservesOnEmpty(t *testing.T) {
	s := NewState()
	prior := s.Model
	s.SetModel("")
	if s.Model != prior {
		t.Errorf("SetModel(\"\") overwrote Model: was %q, now %q", prior, s.Model)
	}
}

// CID:wizard-state-test-004
func TestSetModel_OverwritesOnValue(t *testing.T) {
	s := NewState()
	s.SetModel("ggml-base.en.bin")
	if s.Model != "ggml-base.en.bin" {
		t.Errorf("SetModel(ggml-base.en.bin): Model = %q, want ggml-base.en.bin", s.Model)
	}
	s.SetModel("ggml-tiny.bin")
	if s.Model != "ggml-tiny.bin" {
		t.Errorf("SetModel(ggml-tiny.bin): Model = %q, want ggml-tiny.bin", s.Model)
	}
}
