/* Code Map: Recording Overlay Binary
 * - main: GTK init, builds the overlay window, runs the animation loop
 * - positionBottomCenter: places the window at the bottom-center of the screen
 * - roundedRect: cairo helper for the rounded bar background and bars
 *
 * The overlay is a tiny standalone GTK process spawned by
 * internal/overlay.Manager when a recording starts. It animates a
 * green "bar" pulse to give the user visual feedback. Clicking the
 * window dials a unix socket back to the manager, which stops the
 * recording. No go modules here — only gotk3 + cgo.
 *
 * CID Index:
 * CID:overlay-bin-001 -> main
 * CID:overlay-bin-002 -> positionBottomCenter
 * CID:overlay-bin-003 -> roundedRect
 *
 * Quick lookup: rg -n "CID:overlay-bin-" cmd/whisper-voice-overlay/main.go
 */
package main

// #cgo pkg-config: gdk-3.0
// #cgo CFLAGS: -Wno-deprecated-declarations
// #include <gdk/gdk.h>
import "C"

import (
	"bufio"
	"flag"
	"math"
	"net"
	"os"
	"unsafe"

	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

// CID:overlay-bin-001 - main
// Purpose: GTK init + window construction + animation loop. Blocks
// in gtk.Main() until the user clicks the overlay (sends STOP over
// the unix socket) or the window is destroyed.
func main() {
	socketPath := flag.String("socket", "", "Unix socket to signal STOP on click")
	flag.Parse()
	if *socketPath == "" {
		os.Exit(2)
	}

	gtk.Init(nil)

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetDecorated(false)
	win.SetResizable(false)
	win.SetSkipTaskbarHint(true)
	win.SetSkipPagerHint(true)
	win.SetKeepAbove(true)
	win.SetAcceptFocus(false)
	win.SetTypeHint(gdk.WINDOW_TYPE_HINT_NOTIFICATION)

	// Best-effort transparency
	if screen, _ := gdk.ScreenGetDefault(); screen != nil {
		if visual, _ := screen.GetRGBAVisual(); visual != nil {
			win.SetVisual(visual)
		}
	}
	win.SetAppPaintable(true)

	width := 180
	height := 42
	win.SetDefaultSize(width, height)

	da, _ := gtk.DrawingAreaNew()
	win.Add(da)

	phase := 0.0
	da.Connect("draw", func(_ *gtk.DrawingArea, cr *cairo.Context) {
		// background
		cr.SetSourceRGBA(0, 0, 0, 0.55)
		radius := 14.0
		roundedRect(cr, 0, 0, float64(width), float64(height), radius)
		cr.Fill()

		// bars
		barCount := 9
		gap := 5.0
		barW := 7.0
		totalW := float64(barCount)*barW + float64(barCount-1)*gap
		startX := (float64(width) - totalW) / 2
		midY := float64(height) / 2
		maxH := float64(height) * 0.50

		cr.SetSourceRGBA(0.2, 0.9, 0.4, 0.95)
		for i := 0; i < barCount; i++ {
			x := startX + float64(i)*(barW+gap)
			// simple wave
			v := 0.5 + 0.5*math.Sin(phase+float64(i)*0.7)
			h := 6 + v*maxH
			y := midY - h/2
			roundedRect(cr, x, y, barW, h, 3)
			cr.Fill()
		}
	})

	// click-to-stop
	win.Connect("button-press-event", func(_ *gtk.Window, _ *gdk.Event) {
		go func() {
			if c, err := net.Dial("unix", *socketPath); err == nil {
				w := bufio.NewWriter(c)
				_, _ = w.WriteString("STOP\n")
				_ = w.Flush()
				_ = c.Close()
			}
		}()
		win.Hide()
		gtk.MainQuit()
	})
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	positionBottomCenter(win, width, height)
	win.ShowAll()

	// animate
	glib.TimeoutAdd(33, func() bool {
		phase += 0.18
		da.QueueDraw()
		return true
	})

	gtk.Main()
}

func positionBottomCenter(win *gtk.Window, w, h int) {
	screen, _ := gdk.ScreenGetDefault()
	if screen == nil {
		win.Move(0, 0)
		return
	}
	// Use screen dimensions; these C APIs are widely available across GTK3 versions.
	sx := int(C.gdk_screen_get_width((*C.GdkScreen)(unsafe.Pointer(screen.Native()))))
	sy := int(C.gdk_screen_get_height((*C.GdkScreen)(unsafe.Pointer(screen.Native()))))

	x := (sx - w) / 2
	y := sy - h - 24
	win.Move(x, y)
}

func roundedRect(cr *cairo.Context, x, y, w, h, r float64) {
	cr.NewPath()
	cr.Arc(x+w-r, y+r, r, -1.5708, 0)
	cr.Arc(x+w-r, y+h-r, r, 0, 1.5708)
	cr.Arc(x+r, y+h-r, r, 1.5708, 3.1416)
	cr.Arc(x+r, y+r, r, 3.1416, 4.7124)
	cr.ClosePath()
}
