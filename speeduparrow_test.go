package twmap

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// V20 — angle → (sub-tile index, quadrant) mapping mirrors DDNet
// FillTmpTileSpeedup (render_layer.cpp): subIndex = angle%90, quadrant = angle/90.
func TestSpeedupArrowArrayFrame(t *testing.T) {
	cases := []struct{ angle, sub, quad int }{
		{0, 0, 0}, {45, 45, 0}, {89, 89, 0},
		{90, 0, 1}, {135, 45, 1},
		{180, 0, 2}, {270, 0, 3}, {359, 89, 3},
		{360, 0, 0}, {-1, 89, 3}, {-90, 0, 3},
	}
	for _, c := range cases {
		sub, quad := speedupArrowArrayFrame(c.angle)
		if sub != c.sub || quad != c.quad {
			t.Errorf("angle %d: got (sub=%d,quad=%d), want (%d,%d)", c.angle, sub, quad, c.sub, c.quad)
		}
		if sub < 0 || sub >= 90 || quad < 0 || quad >= 4 {
			t.Errorf("angle %d: out-of-range frame (sub=%d,quad=%d)", c.angle, sub, quad)
		}
	}
}

// V20, I.speeduparrowarray — register/resolve round-trip.
func TestRegisterSpeedupArrowArrayImage(t *testing.T) {
	prev := resolveSpeedupArrowArrayImage()
	t.Cleanup(func() { RegisterSpeedupArrowArrayImage(prev) })

	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	RegisterSpeedupArrowArrayImage(img)
	if got := resolveSpeedupArrowArrayImage(); got != img {
		t.Fatalf("resolve != registered image")
	}
}

// V20 — the array render path actually draws an arrow from the sprite sheet.
func TestRenderSpeedupArrowArrayDraws(t *testing.T) {
	f, err := os.Open("external/speeduparrow/speed_arrow_array.png")
	if err != nil {
		t.Skipf("asset missing: %v", err)
	}
	defer f.Close()
	dec, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	arr := ToNRGBA(dec)

	// rasterizeTriangle writes RGB only and assumes an opaque canvas, so start
	// from opaque black and look for any RGB change where the arrow drew.
	canvas := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for i := 3; i < len(canvas.Pix); i += 4 {
		canvas.Pix[i] = 255 // opaque, RGB stays 0
	}
	renderSpeedupArrowArrayClipped(canvas, 32, 32, 32, 45, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, canvas.Bounds(), arr)

	drew := false
	for i := 0; i < len(canvas.Pix); i += 4 {
		if canvas.Pix[i] != 0 || canvas.Pix[i+1] != 0 || canvas.Pix[i+2] != 0 {
			drew = true
			break
		}
	}
	if !drew {
		t.Error("array arrow path drew nothing onto the canvas")
	}
}
