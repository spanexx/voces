/* Code Map: Tray Icons
 * - IconIdle: Ready state icon
 * - IconRecording: Active recording state icon
 * - IconProcessing: Transcription/TTS active state icon
 * - IconError: Error state icon
 * - IconDisabled: System disabled state icon
 *
 * CID Index:
 * CID:tray-icons-001 -> Embedded Icons
 *
 * Quick lookup: rg -n "CID:tray-icons-" internal/tray/icons.go
 */
package tray

import _ "embed"

// CID:tray-icons-001 - Embedded Icons
// Purpose: Embeds various PNG icons for application state visualization.
//
//go:embed assets/idle.png
var IconIdle []byte

//go:embed assets/record.png
var IconRecording []byte

//go:embed assets/processing.png
var IconProcessing []byte

//go:embed assets/error.png
var IconError []byte

//go:embed assets/disabled.png
var IconDisabled []byte
