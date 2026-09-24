package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

const (
	dockIconCanvas = 1024
	dockIconMargin = 100
)

// SetIcon draws this bitmap directly in the Dock, so leave space around the artwork.
func dockIcon(pngBytes []byte) []byte {
	inset, err := insetIcon(pngBytes, dockIconCanvas, dockIconMargin)
	if err != nil {
		return pngBytes
	}
	return inset
}

func insetIcon(pngBytes []byte, canvas, margin int) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, err
	}
	body := canvas - margin*2
	if body <= 0 {
		return nil, image.ErrFormat
	}
	dst := image.NewNRGBA(image.Rect(0, 0, canvas, canvas))
	scaled := scaleNRGBA(toNRGBA(src), body, body)
	for y := 0; y < body; y++ {
		copy(dst.Pix[(y+margin)*dst.Stride+margin*4:], scaled.Pix[y*scaled.Stride:y*scaled.Stride+body*4])
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func toNRGBA(src image.Image) *image.NRGBA {
	if img, ok := src.(*image.NRGBA); ok {
		return img
	}
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(x, y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func scaleNRGBA(src *image.NRGBA, width, height int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw == 0 || sh == 0 || width == 0 || height == 0 {
		return dst
	}
	for y := 0; y < height; y++ {
		fy := (float64(y)+0.5)*float64(sh)/float64(height) - 0.5
		y0 := int(math.Floor(fy))
		y1 := y0 + 1
		ty := fy - float64(y0)
		y0 = clamp(y0, 0, sh-1)
		y1 = clamp(y1, 0, sh-1)
		for x := 0; x < width; x++ {
			fx := (float64(x)+0.5)*float64(sw)/float64(width) - 0.5
			x0 := int(math.Floor(fx))
			x1 := x0 + 1
			tx := fx - float64(x0)
			x0 = clamp(x0, 0, sw-1)
			x1 = clamp(x1, 0, sw-1)
			c00 := src.NRGBAAt(src.Bounds().Min.X+x0, src.Bounds().Min.Y+y0)
			c10 := src.NRGBAAt(src.Bounds().Min.X+x1, src.Bounds().Min.Y+y0)
			c01 := src.NRGBAAt(src.Bounds().Min.X+x0, src.Bounds().Min.Y+y1)
			c11 := src.NRGBAAt(src.Bounds().Min.X+x1, src.Bounds().Min.Y+y1)
			dst.SetNRGBA(x, y, blend(blend(c00, c10, tx), blend(c01, c11, tx), ty))
		}
	}
	return dst
}

func blend(a, b color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{
		R: lerp(a.R, b.R, t),
		G: lerp(a.G, b.G, t),
		B: lerp(a.B, b.B, t),
		A: lerp(a.A, b.A, t),
	}
}

func lerp(a, b uint8, t float64) uint8 {
	return uint8(math.Round(float64(a) + (float64(b)-float64(a))*t))
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
