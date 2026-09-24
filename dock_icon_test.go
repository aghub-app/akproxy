package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestDockIconPadsCurrentMacOS(t *testing.T) {
	src := solidPNG(t, 32, 32, color.NRGBA{R: 200, G: 10, B: 20, A: 255})
	padded := dockIcon(src)
	if bytes.Equal(padded, src) {
		t.Fatal("macOS icon was not padded")
	}
	img := decodePNG(t, padded)
	if got := img.Bounds().Size(); got != image.Pt(dockIconCanvas, dockIconCanvas) {
		t.Fatalf("size = %v", got)
	}
	if img.NRGBAAt(0, 0).A != 0 || img.NRGBAAt(dockIconCanvas-1, dockIconMargin-1).A != 0 {
		t.Fatal("margin is not transparent")
	}
	if img.NRGBAAt(dockIconMargin, dockIconMargin).A == 0 {
		t.Fatal("artwork does not start at the 100px inset")
	}
}

func TestDockIconLeavesInvalidPNGUntouched(t *testing.T) {
	src := []byte("not a PNG")
	if got := dockIcon(src); !bytes.Equal(got, src) {
		t.Fatal("invalid PNG changed")
	}
}

func solidPNG(t *testing.T, w, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func decodePNG(t *testing.T, raw []byte) *image.NRGBA {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	nrgba, ok := img.(*image.NRGBA)
	if !ok {
		t.Fatalf("png type %T", img)
	}
	return nrgba
}
