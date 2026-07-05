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

/* Per-step section title (rendered in newStepContent) */
.voces-step-title {
    color: #1d2940;
    font: 16px sans-serif;
    font-weight: bold;
}

/* Hint text under a section title */
.voces-step-hint {
    color: #586581;
    font: 12px sans-serif;
    margin-top: 4px;
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
