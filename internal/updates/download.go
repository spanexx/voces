/* Code Map: Update Notifier — Asset Download
 * Companion to updates.go. PickAsset selects the right tarball for
 * the current host; Download fetches it to a staged path next to the
 * running binary.
 *
 * CID Index:
 * CID:updates-006 -> Download
 * CID:updates-006b -> PickAsset
 * CID:updates-006c -> NewAssetPath
 *
 * Quick lookup: rg -n "CID:updates-006" internal/updates/download.go
 */
package updates

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"whisper-voice-util/internal/download"
)

// updateFileSuffix is appended to the destination path for the
// in-flight download. Matches the .partial pattern in
// internal/download but renamed to .new so the user can see the
// staged file in the install dir.
const updateFileSuffix = ".new"

// downloadTimeout caps the asset GET so a stalled connection cannot
// hold the tray click handler open forever. 5 min matches the
// worst-case download time on a slow connection for a ~50 MB binary.
const downloadTimeout = 5 * time.Minute

// CID:updates-006b - PickAsset
// Purpose: Return the most appropriate Asset for the current host.
// For v1 we ship a single linux-amd64 tarball per release and look
// it up by suffix. If no asset matches, returns nil — Download will
// then report "no compatible asset in release vX.Y.Z".
//
// Naming convention documented in Phase 8 (build pipeline):
//
//	whisper-voice-util-vX.Y.Z-linux-amd64.tar.gz
func (r *Release) PickAsset(goos, goarch string) *Asset {
	if r == nil {
		return nil
	}
	want := fmt.Sprintf("-%s-%s.tar.gz", goos, goarch)
	for i := range r.Assets {
		a := &r.Assets[i]
		if !strings.HasSuffix(a.Name, want) {
			continue
		}
		return a
	}
	return nil
}

// CID:updates-006 - Download
// Purpose: Download the release's primary asset to `destPath + ".new"`,
// verify against the asset's Content-Length (download.Download does
// this), and return the path to the staged file. The caller (the
// restart flow) is responsible for renaming it into place and exec'ing
// the new binary.
//
// We do NOT rename in place here because the running binary is the
// old version — overwriting the running binary would be a Windows-only
// error, but on Linux the rename would still succeed and the old
// process would be reading from the now-overwritten inode. Safer to
// keep the staged file as `.new` and have Restart do the swap.
func (r *Release) Download(ctx context.Context, destPath string) (string, error) {
	if r == nil {
		return "", errors.New("updates: nil release")
	}
	if destPath == "" {
		return "", errors.New("updates: empty destPath")
	}
	asset := r.PickAsset("linux", "amd64")
	if asset == nil {
		return "", fmt.Errorf("updates: no linux/amd64 tarball in release %s", r.TagName)
	}
	staged := destPath + updateFileSuffix
	// download.Download writes to destPath+".partial" and renames on
	// success. We point it at staged so the .partial lives next to the
	// .new file.
	if err := download.Download(ctx, asset.BrowserDownloadURL, staged, "", nil); err != nil {
		return "", fmt.Errorf("updates: download %s: %w", asset.Name, err)
	}
	// Best-effort size sanity check vs the asset metadata. download.Download
	// already enforces Content-Length during the GET; this is just a
	// paranoia check for the rare case where the server lied.
	if asset.Size > 0 {
		fi, err := os.Stat(staged)
		if err != nil {
			return "", fmt.Errorf("updates: stat staged: %w", err)
		}
		if fi.Size() != asset.Size {
			_ = os.Remove(staged)
			return "", fmt.Errorf("updates: staged size %d != asset size %d", fi.Size(), asset.Size)
		}
	}
	return staged, nil
}

// CID:updates-006c - NewAssetPath
// Purpose: Build the canonical staged path without re-deriving the
// suffix. Used by tests and the restart flow.
func NewAssetPath(destPath string) string {
	return filepath.Join(filepath.Dir(destPath), filepath.Base(destPath)+updateFileSuffix)
}
