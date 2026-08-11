package iconutil

import (
	"bytes"
	"image"
	"image/png"
)

// menuBarPx is the @2x pixel size for a 22pt menu bar icon (Retina).
const menuBarPx = 44

// PrepareForMenuBar trims transparent padding, scales the artwork to fill
// the menu bar slot as much as possible, and returns PNG bytes ready for systray.
// Source files in assets/ are never modified.
func PrepareForMenuBar(pngData []byte) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, err
	}

	trimmed := cropTransparent(src)
	if trimmed.Bounds().Dx() == 0 || trimmed.Bounds().Dy() == 0 {
		return pngData, nil
	}

	scaled := scaleToFit(trimmed, menuBarPx, menuBarPx)
	canvas := image.NewRGBA(image.Rect(0, 0, menuBarPx, menuBarPx))
	pasteCentered(canvas, scaled)

	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func cropTransparent(src image.Image) image.Image {
	b := src.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X, b.Min.Y

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if a > 0x1000 {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	if minX > maxX || minY > maxY {
		return src
	}

	rect := image.Rect(minX, minY, maxX+1, maxY+1)
	dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			dst.Set(x-rect.Min.X, y-rect.Min.Y, src.At(x, y))
		}
	}
	return dst
}

func scaleToFit(src image.Image, maxW, maxH int) image.Image {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw == 0 || sh == 0 {
		return src
	}

	scale := float64(maxW) / float64(sw)
	if hScale := float64(maxH) / float64(sh); hScale < scale {
		scale = hScale
	}

	dw := max(1, int(float64(sw)*scale+0.5))
	dh := max(1, int(float64(sh)*scale+0.5))

	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			sx := sb.Min.X + x*sw/dw
			sy := sb.Min.Y + y*sh/dh
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func pasteCentered(canvas *image.RGBA, src image.Image) {
	sb := src.Bounds()
	cb := canvas.Bounds()
	offsetX := cb.Min.X + (cb.Dx()-sb.Dx())/2
	offsetY := cb.Min.Y + (cb.Dy()-sb.Dy())/2
	for y := 0; y < sb.Dy(); y++ {
		for x := 0; x < sb.Dx(); x++ {
			canvas.Set(offsetX+x, offsetY+y, src.At(sb.Min.X+x, sb.Min.Y+y))
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
