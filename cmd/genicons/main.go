// Command genicons generates 22×22 menu-bar PNG icons for each Gopher mood.
// Run once: go run ./cmd/genicons
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
)

const size = 22

// Gopher palette inspired by the official Go mascot colors.
var (
	gopherBlue = color.RGBA{0x00, 0xAC, 0xD7, 0xFF}
	gopherTan  = color.RGBA{0xF4, 0xD0, 0x3B, 0xFF}
	gopherDark = color.RGBA{0x3D, 0x3D, 0x3D, 0xFF}
	white      = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	black      = color.RGBA{0x00, 0x00, 0x00, 0xFF}
	transparent = color.RGBA{0, 0, 0, 0}
)

func main() {
	outDir := filepath.Join("assets", "icons")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatal(err)
	}

	moods := []struct {
		name string
		draw func(*image.RGBA)
	}{
		{"Happy", drawHappy},
		{"Hungry", drawHungry},
		{"Tired", drawTired},
		{"Sad", drawSad},
		{"Sleeping", drawSleeping},
	}

	for _, m := range moods {
		img := image.NewRGBA(image.Rect(0, 0, size, size))
		draw.Draw(img, img.Bounds(), &image.Uniform{transparent}, image.Point{}, draw.Src)
		m.draw(img)
		path := filepath.Join(outDir, m.name+".png")
		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			log.Fatal(err)
		}
		f.Close()
		log.Printf("wrote %s", path)
	}
}

func drawGopherBase(img *image.RGBA) {
	// Ears
	fillRect(img, 4, 2, 7, 5, gopherTan)
	fillRect(img, 14, 2, 17, 5, gopherTan)
	// Head
	fillRect(img, 3, 4, 18, 13, gopherTan)
	// Body
	fillRect(img, 5, 13, 16, 20, gopherBlue)
	// Feet
	fillRect(img, 5, 20, 9, 21, gopherDark)
	fillRect(img, 12, 20, 16, 21, gopherDark)
}

func drawHappy(img *image.RGBA) {
	drawGopherBase(img)
	// Open eyes
	fillRect(img, 7, 7, 9, 9, white)
	fillRect(img, 12, 7, 14, 9, white)
	setPixel(img, 8, 8, black)
	setPixel(img, 13, 8, black)
	// Smile
	setPixel(img, 9, 11, gopherDark)
	setPixel(img, 10, 12, gopherDark)
	setPixel(img, 11, 12, gopherDark)
	setPixel(img, 12, 11, gopherDark)
}

func drawHungry(img *image.RGBA) {
	drawGopherBase(img)
	// Wide exorbitant eyes
	fillRect(img, 6, 6, 10, 10, white)
	fillRect(img, 11, 6, 15, 10, white)
	setPixel(img, 8, 8, black)
	setPixel(img, 9, 8, black)
	setPixel(img, 12, 8, black)
	setPixel(img, 13, 8, black)
	// Open mouth (stressed)
	fillRect(img, 9, 11, 12, 12, gopherDark)
}

func drawTired(img *image.RGBA) {
	drawGopherBase(img)
	// Half-closed eyes (horizontal lines)
	fillRect(img, 7, 8, 9, 8, gopherDark)
	fillRect(img, 12, 8, 14, 8, gopherDark)
	// Droopy mouth
	setPixel(img, 9, 11, gopherDark)
	setPixel(img, 10, 11, gopherDark)
	setPixel(img, 11, 11, gopherDark)
	setPixel(img, 12, 11, gopherDark)
}

func drawSad(img *image.RGBA) {
	drawGopherBase(img)
	// Downcast eyes
	fillRect(img, 7, 8, 9, 9, white)
	fillRect(img, 12, 8, 14, 9, white)
	setPixel(img, 8, 9, black)
	setPixel(img, 13, 9, black)
	// Frown
	setPixel(img, 9, 12, gopherDark)
	setPixel(img, 10, 11, gopherDark)
	setPixel(img, 11, 11, gopherDark)
	setPixel(img, 12, 12, gopherDark)
}

func drawSleeping(img *image.RGBA) {
	drawGopherBase(img)
	// Closed eyes (curved lines)
	fillRect(img, 7, 8, 9, 8, gopherDark)
	fillRect(img, 12, 8, 14, 8, gopherDark)
	// Small content smile
	setPixel(img, 10, 11, gopherDark)
	setPixel(img, 11, 11, gopherDark)
	// Zzz indicator
	setPixel(img, 16, 3, gopherDark)
	setPixel(img, 17, 2, gopherDark)
	setPixel(img, 18, 1, gopherDark)
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if x >= 0 && x < size && y >= 0 && y < size {
				img.Set(x, y, c)
			}
		}
	}
}

func setPixel(img *image.RGBA, x, y int, c color.Color) {
	if x >= 0 && x < size && y >= 0 && y < size {
		img.Set(x, y, c)
	}
}
