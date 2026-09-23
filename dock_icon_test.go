package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestDockIconPadsMacOS26(t *testing.T) {
	src := solidPNG(t, 32, 32, color.NRGBA{R: 200, G: 10, B: 20, A: 255})
	padded := dockIcon(src, "27.0")
	if bytes.Equal(padded, src) {
		t.Fatal("macOS 27 icon was not padded")
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

func TestDockIconLeavesOlderMacOSUntouched(t *testing.T) {
	src := solidPNG(t, 8, 8, color.NRGBA{R: 1, A: 255})
	for _, version := range []string{"15.6", "25.0", "", "Unknown", "beta"} {
		if got := dockIcon(src, version); !bytes.Equal(got, src) {
			t.Fatalf("version %q changed the icon", version)
		}
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
