/* Code Map: Icon Generator
 * - main: Entry point for generating application icons
 */
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

func generateSolidColorIcon(path string, c color.Color) error {
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, c)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

func main() {
	outDir := "."
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	icons := map[string]color.Color{
		"idle.png":       color.RGBA{0, 255, 0, 255},     // Green
		"record.png":     color.RGBA{255, 0, 0, 255},     // Red
		"processing.png": color.RGBA{255, 255, 0, 255},   // Yellow
		"error.png":      color.RGBA{128, 128, 128, 255}, // Gray
		"disabled.png":   color.RGBA{200, 200, 200, 255}, // Light Gray
	}

	os.MkdirAll(outDir, 0755)

	for name, c := range icons {
		err := generateSolidColorIcon(filepath.Join(outDir, name), c)
		if err != nil {
			panic(err)
		}
	}
}
