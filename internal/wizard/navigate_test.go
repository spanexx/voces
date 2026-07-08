/* Code Map: Wizard navigation tests
 *
 * These tests pin the order of buildStepChain. The chain shape
 * is a user-visible contract (changing it is a UX change), so
 * any reorder that drops a step, inserts a duplicate, or omits
 * the Model step should fail CI rather than be discovered by
 * the next manual smoke test.
 *
 * rc1-hotpatch-24: the chain gains a Model step after
 * stepLanguage and before stepHotkey/TTS. Tests assert:
 *   - The new step is present in the right position for both
 *     English and non-English chains.
 *   - "Back" from the next step lands on the new step.
 *   - The English chain still omits TTS (rc1-hotpatch-14).
 *   - The non-English chain still includes TTS at its position.
 *
 * CID Index:
 * CID:wizard-nav-test-001 -> TestBuildStepChain_English
 * CID:wizard-nav-test-002 -> TestBuildStepChain_NonEnglish
 * CID:wizard-nav-test-003 -> TestBuildStepChain_ModelPosition
 */
package wizard

import "testing"

// CID:wizard-nav-test-001
func TestBuildStepChain_English(t *testing.T) {
	s := NewState()
	s.Language = "en"
	got := buildStepChain(s)
	want := []stepKey{
		stepWelcome, stepLanguage, stepModel, stepHotkey, stepBehavior, stepSecondaryHotkeys, stepFinish,
	}
	if len(got) != len(want) {
		t.Fatalf("English chain length = %d, want %d\n  got:  %v\n  want: %v", len(got), len(want), got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("English chain[%d] = %v, want %v\n  got:  %v\n  want: %v", i, got[i], w, got, want)
		}
	}
	for _, k := range got {
		if k == stepTTS {
			t.Errorf("English chain unexpectedly includes TTS step: %v", got)
		}
	}
}

// CID:wizard-nav-test-002
func TestBuildStepChain_NonEnglish(t *testing.T) {
	s := NewState()
	s.Language = "de"
	got := buildStepChain(s)
	want := []stepKey{
		stepWelcome, stepLanguage, stepModel, stepHotkey, stepTTS, stepBehavior, stepSecondaryHotkeys, stepFinish,
	}
	if len(got) != len(want) {
		t.Fatalf("non-English chain length = %d, want %d\n  got:  %v\n  want: %v", len(got), len(want), got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("non-English chain[%d] = %v, want %v\n  got:  %v\n  want: %v", i, got[i], w, got, want)
		}
	}
}

// CID:wizard-nav-test-003
func TestBuildStepChain_ModelPosition(t *testing.T) {
	// The Model step is between Language and Hotkey in both chains.
	// This test guards the contract that "back from Hotkey lands on
	// Model, not on Language" — which is what the user experiences
	// when they change their mind about the model.
	for _, lang := range []string{"en", "de", "es"} {
		s := NewState()
		s.Language = lang
		got := buildStepChain(s)
		var langIdx, modelIdx, hotkeyIdx int = -1, -1, -1
		for i, k := range got {
			switch k {
			case stepLanguage:
				langIdx = i
			case stepModel:
				modelIdx = i
			case stepHotkey:
				hotkeyIdx = i
			}
		}
		if langIdx < 0 || modelIdx < 0 || hotkeyIdx < 0 {
			t.Errorf("language=%q chain missing one of Language/Model/Hotkey: %v", lang, got)
			continue
		}
		if !(langIdx < modelIdx && modelIdx < hotkeyIdx) {
			t.Errorf("language=%q chain order wrong: Language@%d Model@%d Hotkey@%d; want Language < Model < Hotkey\n  chain: %v",
				lang, langIdx, modelIdx, hotkeyIdx, got)
		}
	}
}
