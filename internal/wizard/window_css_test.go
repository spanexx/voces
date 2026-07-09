/* Code Map: vocesCSS regression tests
 * - TestVocessCSS_NoHardcodedDarkOnTheme: the rc1-hotpatch-26 fix
 *   dropped the explicit dark colors from .voces-step-title and
 *   .voces-step-hint so the wizard respects the user's system
 *   theme. This test pins that behaviour at the source: any
 *   future PR that re-adds an explicit `color:` to those
 *   selectors fails the build, forcing the author to think
 *   about whether a theme override is really intended.
 * - TestVocessCSS_HeaderStillDark: the brand identity lives in
 *   the dark header gradient; pin that it stays.
 * - TestVocessCSS_ProgressBarStyled: the rc1-hotpatch-26 fix
 *   also added explicit styles for the progress bar so the
 *   "white window with a black box" bug does not regress.
 *
 * Pure regex test — no GTK, no DISPLAY, runs in <1ms. Lives
 * here (next to window_css.go) so a regression lands in the
 * same diff as the fix.
 *
 * CID Index:
 * CID:wizard-win-css-test-001 -> TestVocessCSS_NoHardcodedDarkOnTheme
 * CID:wizard-win-css-test-002 -> TestVocessCSS_HeaderStillDark
 * CID:wizard-win-css-test-003 -> TestVocessCSS_ProgressBarStyled
 */
package wizard

import (
	"regexp"
	"testing"
)

// CID:wizard-win-css-test-001 - TestVocessCSS_NoHardcodedDarkOnTheme
// Purpose: the wizard's per-step body text used to be hardcoded
// dark blue (#1d2940) and dark grey (#586581). On a dark GTK
// theme the body background is also dark, which made the title
// and hint unreadable. The rc1-hotpatch-26 fix dropped the
// explicit `color:` lines from .voces-step-title and
// .voces-step-hint; this test pins that behaviour at the CSS
// source so a future drive-by edit cannot regress it.
//
// Strategy: find the body of each selector and assert it does
// not contain a `color:` declaration. The header gradient's
// `background-image:` line and the accent strip's
// `background-color:` are NOT covered by this rule — those
// are brand elements that should stay dark.
func TestVocessCSS_NoHardcodedDarkOnTheme(t *testing.T) {
	selectors := []string{".voces-step-title", ".voces-step-hint"}
	for _, sel := range selectors {
		rule := extractCSSRule(vocesCSS, sel)
		if rule == "" {
			t.Errorf("CSS rule for %s not found in vocesCSS; was the selector removed?", sel)
			continue
		}
		if matched, _ := regexp.MatchString(`(?m)^\s*color\s*:`, rule); matched {
			t.Errorf("%s still has an explicit `color:` declaration:\n%s\n\n"+
				"rc1-hotpatch-26: drop the color so the wizard uses the system "+
				"theme's text color. The pre-rc26 hardcoded #1d2940 was unreadable "+
				"on dark themes.", sel, rule)
		}
	}
}

// CID:wizard-win-css-test-002 - TestVocessCSS_HeaderStillDark
// Purpose: pin the brand identity. The wizard header is a dark
// blue gradient that matches the tray app icon. A future
// "let's use theme colors everywhere" refactor must NOT touch
// .voces-header or .voces-title, or the wizard loses its visual
// coherence with the rest of the app.
func TestVocessCSS_HeaderStillDark(t *testing.T) {
	for _, sel := range []string{".voces-header", ".voces-title"} {
		rule := extractCSSRule(vocesCSS, sel)
		if rule == "" {
			t.Errorf("CSS rule for %s not found; the header is the brand element and must stay", sel)
			continue
		}
	}
	// Header must use the dark gradient (any of #1d2940, #2a3a52,
	// or #0d1525 — the three tokens the rc14 palette introduced).
	header := extractCSSRule(vocesCSS, ".voces-header")
	hasDarkGradient := regexp.MustCompile(`(?i)linear-gradient.*#1d2940|#2a3a52|#0d1525`).MatchString(header)
	if !hasDarkGradient {
		t.Errorf(".voces-header has lost its dark gradient:\n%s\n\n"+
			"the header is the brand element; restore the dark gradient or "+
			"update the tray icon to match the new palette.", header)
	}
	// .voces-title must keep its white text on the dark header.
	title := extractCSSRule(vocesCSS, ".voces-title")
	if matched, _ := regexp.MatchString(`(?m)color\s*:\s*#fff(?:fff)?\b`, title); !matched {
		t.Errorf(".voces-title no longer has white text on the dark header:\n%s", title)
	}
}

// CID:wizard-win-css-test-003 - TestVocessCSS_ProgressBarStyled
// Purpose: the rc1-hotpatch-26 fix added explicit styles for
// the GtkProgressBar in the downloading view. Before the fix
// the default trough was theme_bg_color (often white) with a
// theme_text_color (often also white) progress fill, producing
// the "white window with a black box" the user reported.
// Pin the presence of:
//   - .voces-progress rule with a light-grey trough (#dde3ec)
//   - .voces-progress progress rule with the Voces blue fill
//     (#3b82f6)
//   - .voces-progress trough rule (defence in depth: some
//     themes only honour the trough sub-element, not the
//     top-level background-color).
func TestVocessCSS_ProgressBarStyled(t *testing.T) {
	prog := extractCSSRule(vocesCSS, ".voces-progress")
	if prog == "" {
		t.Fatal(".voces-progress rule not found; the progress bar in buildDownloadingView " +
			"would render as a default GtkProgressBar (invisible trough on light themes)")
	}
	// Trough must be light grey #dde3ec. GTK honours this as
	// the "background" of the .voces-progress node, which
	// appears behind the trough sub-element.
	if !regexp.MustCompile(`(?m)background-color\s*:\s*#dde3ec`).MatchString(prog) {
		t.Errorf(".voces-progress is missing the rc1-hotpatch-26 light-grey trough "+
			"background-color: #dde3ec. Rule:\n%s", prog)
	}
	// The fill (the "progress" sub-element) must be the
	// Voces blue. Look for it in the nested .voces-progress
	// progress rule, not the top-level .voces-progress rule
	// — the top-level has the *trough* color, the nested
	// progress rule has the *fill* color.
	nested := extractCSSRule(vocesCSS, ".voces-progress progress")
	if nested == "" {
		t.Fatal(".voces-progress progress rule not found; the Voces blue fill is " +
			"not defined anywhere in vocesCSS")
	}
	if !regexp.MustCompile(`(?m)background-color\s*:\s*#3b82f6`).MatchString(nested) {
		t.Errorf(".voces-progress progress is missing the rc1-hotpatch-26 Voces blue fill "+
			"background-color: #3b82f6. Rule:\n%s", nested)
	}
	// Trough sub-element rule (defence in depth).
	trough := extractCSSRule(vocesCSS, ".voces-progress trough")
	if trough == "" {
		t.Fatal(".voces-progress trough rule not found; the light-grey trough is " +
			"not reinforced on the trough sub-element")
	}
}

// extractCSSRule returns the body of the CSS rule for the given
// selector, from the opening `{` to the matching `}`. The
// matching is brace-balanced so nested `:` (e.g. in
// `linear-gradient(to bottom, #a, #b)`) are not treated as
// block delimiters. If the selector is not present, returns "".
func extractCSSRule(css, selector string) string {
	// Find the selector. CSS is whitespace-tolerant at the
	// selector level but the body always starts with `{`.
	startIdx := -1
	for i := 0; i+len(selector) <= len(css); i++ {
		if css[i:i+len(selector)] != selector {
			continue
		}
		// Word boundary — selector must not be a prefix of a
		// longer class name.
		prevOK := i == 0 || isCSSWhitespace(css[i-1]) || css[i-1] == ',' || css[i-1] == '}'
		nextOK := i+len(selector) == len(css) || isCSSWhitespace(css[i+len(selector)]) || css[i+len(selector)] == '{' || css[i+len(selector)] == ','
		if !prevOK || !nextOK {
			continue
		}
		// Find the opening brace after the selector.
		j := i + len(selector)
		for j < len(css) && isCSSWhitespace(css[j]) {
			j++
		}
		if j < len(css) && css[j] == '{' {
			startIdx = j
			break
		}
	}
	if startIdx < 0 {
		return ""
	}
	// Brace-balanced scan to find the matching `}`.
	depth := 0
	for k := startIdx; k < len(css); k++ {
		switch css[k] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return css[startIdx+1 : k]
			}
		}
	}
	return ""
}

func isCSSWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
