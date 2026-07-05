/* Code Map: wizard "Start" → commit progress step
 * - startCommit: swap to a Downloading view, run commit in a goroutine
 * - buildDownloadingView: label + progress bar
 * - formatProgress: human-readable bytes/percent string
 *
 * CID Index:
 * CID:wizard-commit-001 -> startCommit
 * CID:wizard-commit-002 -> buildDownloadingView
 * CID:wizard-commit-003 -> formatProgress
 *
 * Quick lookup: rg -n "CID:wizard-commit-" internal/wizard/commit.go
 */

package wizard

import (
	"context"
	"fmt"
	"log"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"voces/internal/wizard/steps"
)

// CID:wizard-commit-001 - startCommit
// Purpose: build a "Downloading..." view, run commit from a
// goroutine, and finish the wizard when commit returns. The
// progress bar updates from any goroutine via glib.IdleAdd.
//
// Called from the finish step's "Start" click handler. This is
// the rc1-hotpatch-13 fix for the "Voces is not responding"
// overlay: before this, EnsureModels ran on the main thread
// after the wizard returned, blocking GTK and leaving the wizard
// window visible-but-frozen.
//
// On commit error, the user is sent back to the finish step
// (rebuilt fresh) so they can click Start again after fixing
// the problem (e.g. checking the network).
func startCommit(
	win *gtk.Window,
	contentBox *gtk.Box,
	currentBoxRef **gtk.Box,
	state *State,
	commit CommitFunc,
	finish func(v *State),
) {
	progressBox, progressBar, progressLabel, err := buildDownloadingView()
	if err != nil {
		log.Printf("wizard: build download view: %v", err)
		showError(win, err)
		return
	}
	if *currentBoxRef != nil {
		contentBox.Remove(*currentBoxRef)
	}
	contentBox.Add(progressBox)
	*currentBoxRef = progressBox
	progressBox.ShowAll()
	log.Printf("wizard: starting commit, progress view shown")

	// progress: called by commit on whatever cadence it likes;
	// we hop to the GTK main thread via glib.IdleAdd so the
	// bar updates are safe.
	progress := func(fraction float64, bytesDone, total int64) {
		glib.IdleAdd(func() {
			if fraction >= 0 {
				progressBar.SetFraction(fraction)
			}
			progressLabel.SetText(formatProgress(fraction, bytesDone, total))
		})
	}
	done := make(chan error, 1)
	go func() {
		done <- commit(context.Background(), state, progress)
	}()
	go func() {
		err := <-done
		glib.IdleAdd(func() {
			if err != nil {
				log.Printf("wizard: commit error: %v", err)
				showError(win, err)
				// Send the user back to the finish step
				// so they can click Start again after
				// fixing the problem (e.g. checking the
				// network).
				if *currentBoxRef != nil {
					contentBox.Remove(*currentBoxRef)
				}
				finishStep, stepErr := steps.BuildFinish(win, state)
				if stepErr != nil {
					log.Printf("wizard: re-build finish: %v", stepErr)
					return
				}
				contentBox.Add(finishStep.Box)
				*currentBoxRef = finishStep.Box
				finishStep.Box.ShowAll()
				return
			}
			log.Printf("wizard: commit OK, finishing")
			finish(state)
		})
	}()
}

// CID:wizard-commit-002 - buildDownloadingView
// Purpose: assemble the "Downloading..." view: a label + a
// determinate-progress bar. Kept separate so the progress-step
// state machine (startCommit) is readable.
func buildDownloadingView() (*gtk.Box, *gtk.ProgressBar, *gtk.Label, error) {
	label, err := gtk.LabelNew("Downloading model...")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("wizard: download label: %w", err)
	}
	label.SetHAlign(gtk.ALIGN_START)
	bar, err := gtk.ProgressBarNew()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("wizard: progress bar: %w", err)
	}
	bar.SetFraction(0)
	bar.SetShowText(false)
	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("wizard: progress box: %w", err)
	}
	box.SetMarginTop(16)
	box.SetMarginBottom(16)
	box.SetMarginStart(16)
	box.SetMarginEnd(16)
	box.PackStart(label, false, false, 0)
	box.PackStart(bar, false, false, 0)
	return box, bar, label, nil
}

// CID:wizard-commit-003 - formatProgress
// Purpose: render "Downloading model... 12.3 MB / 488 MB (50%)"
// or "Downloading model... 12.3 MB" when the server did not
// return Content-Length (total <= 0).
func formatProgress(fraction float64, done, total int64) string {
	const (
		KiB = 1024
		MiB = 1024 * 1024
	)
	fmtBytes := func(b int64) string {
		switch {
		case b < KiB:
			return fmt.Sprintf("%d B", b)
		case b < MiB:
			return fmt.Sprintf("%.1f KB", float64(b)/float64(KiB))
		default:
			return fmt.Sprintf("%.1f MB", float64(b)/float64(MiB))
		}
	}
	if total > 0 {
		return fmt.Sprintf("Downloading model... %s / %s (%.0f%%)",
			fmtBytes(done), fmtBytes(total), fraction*100)
	}
	return fmt.Sprintf("Downloading model... %s", fmtBytes(done))
}
