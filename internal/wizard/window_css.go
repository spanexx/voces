/* Code Map: Wizard Window CSS
 * - vocesCSS: the global stylesheet for every step in the wizard
 * - installCSS: attach the provider to the default screen
 *
 * Split out from window.go so the file stays under the 250-line
 * cap enforced by scripts/check-file-size.sh.
 *
 * rc1-hotpatch-14: small palette (Voces blue, off-white
 * header, subtle gradient) that matches the tray app icon so
 * the user gets a coherent first impression.
 *
 * CID Index:
 * CID:wizard-win-css-001 -> vocesCSS
 * CID:wizard-win-css-002 -> installCSS
 *
 * Quick lookup: rg -n "CID:wizard-win-css-" internal/wizard/
 */
package wizard

import (
	"fmt"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

// CID:wizard-win-css-001 - vocesCSS
// Purpose: the global stylesheet for every step in the wizard.
// The rules do the work:
//   - .voces-header            — gradient + soft bottom border
//   - .voces-title             — large bold display text
//   - .voces-subtitle          — small muted line under the title
//   - .voces-next-btn          — primary-action emphasis
//   - .voces-step-title        — 16px bold section heading
//   - .voces-step-hint         — 12px muted hint text
//   - .voces-accent            — the thin colored strip below the
//                                header
//   - .voces-download-label    — the "Downloading model..." text
//                                above the progress bar
//   - .voces-progress          — the determinate progress bar
//                                (rc1-hotpatch-26)
//
// rc1-hotpatch-26: dropped the hardcoded `color: #1d2940` on
// .voces-step-title and `color: #586581` on .voces-step-hint.
// The fixed dark blue did not respect the system theme; on a
// dark GTK theme the body background is dark and the dark-blue
// title was unreadable. Without the explicit color, GTK falls
// back to the theme's default text color, which is dark on
// light themes and light on dark themes. The header gradient
// stays dark — it is a brand element, and a dark bar is
// legible against both light and dark window backgrounds.
//
// Also added the .voces-download-label and .voces-progress
// classes so the commit progress view stays readable on dark
// themes: a default GtkLabel/GtkProgressBar can render with
// theme_text_color = theme_bg_color (invisible) on misconfigured
// themes, and rc1-hotpatch-26's "white window with a black box"
// was the default GtkProgressBar at ~0% on a white background.
//
// Kept as a single CSS blob (rather than split across files)
// because the wizard window is one screen; splitting would
// require multiple gtk.CssProviderNew + getStyleContext calls
// and add nothing for the user.
const vocesCSS = `
/* Header bar — gradient + 1px bottom border */
.voces-header {
    background-image: linear-gradient(to bottom, #2a3a52, #1d2940);
    border-bottom: 1px solid #0d1525;
    padding: 16px 20px;
}

/* Large app title */
.voces-title {
    color: #ffffff;
    font: 18px sans-serif;
    font-weight: bold;
}

/* Subtitle / version line */
.voces-subtitle {
    color: #a8b8d0;
    font: 11px sans-serif;
    margin-top: 2px;
}

/* Accent strip below the header (Voces blue) */
.voces-accent {
    background-color: #3b82f6;
    min-height: 3px;
}

/* Primary action button (Next / Start) */
.voces-next-btn {
    background-image: linear-gradient(to bottom, #3b82f6, #2563eb);
    color: #ffffff;
    border: 1px solid #1d4ed8;
    border-radius: 4px;
    padding: 6px 18px;
    font: 13px sans-serif;
    font-weight: bold;
}
.voces-next-btn:hover {
    background-image: linear-gradient(to bottom, #4b8ff6, #316bef);
}

/* Per-step section title (rendered in newStepContent).
 * No explicit color — uses the theme's default text color so the
 * title is readable on both light and dark system themes. The
 * pre-rc1-hotpatch-26 hardcoded #1d2940 made the title invisible
 * on dark themes (and very low contrast on some light themes).
 */
.voces-step-title {
    font: 16px sans-serif;
    font-weight: bold;
}

/* Hint text under a section title.
 * No explicit color — same theme-aware rationale as .voces-step-title.
 */
.voces-step-hint {
    font: 12px sans-serif;
    margin-top: 4px;
    opacity: 0.75;
}

/* Downloading progress view (rc1-hotpatch-26).
 * - label uses theme default text color (no override) so it
 *   is visible on both light and dark themes.
 * - progress bar is styled explicitly so the trough is
 *   visible on any background: light grey trough + Voces
 *   blue fill. Without this, a default GtkProgressBar on a
 *   light theme renders the trough as white-on-white, which
 *   is the "white window with a black box" the user saw
 *   in the rc1-hotpatch-26 testing.
 */
.voces-download-label {
    font: 12px sans-serif;
    margin-bottom: 6px;
}
.voces-progress {
    /* Trough = light grey on both light + dark themes. The
     * value (filled portion) is the Voces blue so the user
     * can read the progress at a glance.
     */
    color: #3b82f6;
    background-color: #dde3ec;
    border-radius: 3px;
    min-height: 14px;
}
.voces-progress progress {
    background-color: #3b82f6;
    border-radius: 3px;
}
.voces-progress trough {
    background-color: #dde3ec;
    border-radius: 3px;
}
`

// CID:wizard-win-css-002 - installCSS
// Purpose: install the wizard's stylesheet into GTK's default
// screen. Called once per NewWindow. The provider is attached at
// the application-priority level so step widgets automatically
// inherit the .voces-* class rules when they add the matching
// style class (no per-widget loadCSS calls needed).
func installCSS(screen *gdk.Screen) error {
	provider, err := gtk.CssProviderNew()
	if err != nil {
		return fmt.Errorf("wizard: css provider: %w", err)
	}
	if err := provider.LoadFromData(vocesCSS); err != nil {
		return fmt.Errorf("wizard: css load: %w", err)
	}
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	return nil
}
