package twmap

import (
	"image"
	"image/color"
	"reflect"
	"testing"
)

func makeEntitiesTileset(tileColors map[uint8]color.NRGBA) *image.NRGBA {
	const side = 256
	const grid = 16
	tile := side / grid
	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	for id, c := range tileColors {
		tx := int(id) % grid
		ty := int(id) / grid
		baseX := tx * tile
		baseY := ty * tile
		for y := 0; y < tile; y++ {
			for x := 0; x < tile; x++ {
				off := img.PixOffset(baseX+x, baseY+y)
				img.Pix[off+0] = c.R
				img.Pix[off+1] = c.G
				img.Pix[off+2] = c.B
				img.Pix[off+3] = c.A
			}
		}
	}
	return img
}

func makeParticlesSheetAirJump(c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 512, 512))
	for y := 64; y < 128; y++ {
		for x := 320; x < 384; x++ {
			off := img.PixOffset(x, y)
			img.Pix[off+0] = c.R
			img.Pix[off+1] = c.G
			img.Pix[off+2] = c.B
			img.Pix[off+3] = c.A
		}
	}
	return img
}

func centerPixel(img *image.NRGBA) color.NRGBA {
	cx := img.Bounds().Dx() / 2
	cy := img.Bounds().Dy() / 2
	off := img.PixOffset(cx, cy)
	return color.NRGBA{
		R: img.Pix[off+0],
		G: img.Pix[off+1],
		B: img.Pix[off+2],
		A: img.Pix[off+3],
	}
}

func hasPixelInRect(img *image.NRGBA, rect image.Rectangle, match func(color.NRGBA) bool) bool {
	rect = rect.Intersect(img.Bounds())
	if rect.Empty() {
		return false
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			off := img.PixOffset(x, y)
			px := color.NRGBA{
				R: img.Pix[off+0],
				G: img.Pix[off+1],
				B: img.Pix[off+2],
				A: img.Pix[off+3],
			}
			if match(px) {
				return true
			}
		}
	}
	return false
}

func TestRenderMapTeleLayerOnly(t *testing.T) {
	RegisterEntitiesImage(makeEntitiesTileset(map[uint8]color.NRGBA{
		TileTeleIn: {R: 255, A: 255},
	}))

	m := &Map{
		Groups: []Group{{
			ParallaxX: 100,
			ParallaxY: 100,
			Layers: []Layer{{
				Kind:   LayerKindTele,
				Width:  1,
				Height: 1,
				TeleTiles: []TeleTile{{
					Number: 1,
					ID:     TileTeleIn,
				}},
			}},
		}},
	}

	img, err := RenderMap(m, WithTeleLayer(true))
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if img.Bounds().Dx() <= 1 || img.Bounds().Dy() <= 1 {
		t.Fatalf("unexpected tiny output: %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}

	if !hasPixelInRect(img, img.Bounds(), func(px color.NRGBA) bool {
		return px.R > 220 && px.G < 80 && px.B < 80
	}) {
		t.Fatalf("expected tele tile sprite pixels to be present")
	}
	if !hasPixelInRect(img, image.Rect(img.Bounds().Dx()/4, img.Bounds().Dy()/4, img.Bounds().Dx()*3/4, img.Bounds().Dy()*3/4), func(px color.NRGBA) bool {
		return px.R > 220 && px.G > 220 && px.B > 220
	}) {
		t.Fatalf("expected tele number text pixels to be present")
	}
}

func TestRenderMapParticlesLayerOnly(t *testing.T) {
	RegisterParticleImage(makeParticlesSheetAirJump(color.NRGBA{G: 255, A: 255}))

	m := &Map{
		Groups: []Group{{
			ParallaxX: 100,
			ParallaxY: 100,
			Layers: []Layer{{
				Kind:   LayerKindGame,
				Width:  1,
				Height: 1,
				Tiles: []Tile{{
					ID: TileUnlimitedJumpsOn,
				}},
			}},
		}},
	}

	img, err := RenderMap(m, WithParticles(true))
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if img.Bounds().Dx() <= 1 || img.Bounds().Dy() <= 1 {
		t.Fatalf("unexpected tiny output: %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}

	px := centerPixel(img)
	if px.G <= 180 || px.R >= 120 || px.B >= 120 {
		t.Fatalf("unexpected center pixel for particle marker: %#v", px)
	}
}

func TestRenderMapParticlesBehindGameLayer(t *testing.T) {
	RegisterEntitiesImage(makeEntitiesTileset(map[uint8]color.NRGBA{
		TileUnlimitedJumpsOn: {R: 255, A: 255},
	}))
	RegisterParticleImage(makeParticlesSheetAirJump(color.NRGBA{G: 255, A: 255}))

	m := &Map{
		Groups: []Group{{
			ParallaxX: 100,
			ParallaxY: 100,
			Layers: []Layer{{
				Kind:   LayerKindGame,
				Width:  1,
				Height: 1,
				Tiles: []Tile{{
					ID: TileUnlimitedJumpsOn,
				}},
			}},
		}},
	}

	img, err := RenderMap(m, WithParticles(true), WithGameLayer(true))
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	px := centerPixel(img)
	if px.R <= px.G {
		t.Fatalf("expected game-layer overlay to render above particle marker, got %#v", px)
	}
}

func TestRenderMapGameLayerOverEntities(t *testing.T) {
	RegisterEntitiesImage(makeEntitiesTileset(map[uint8]color.NRGBA{
		TileHealth: {B: 255, A: 255},
	}))

	m := &Map{
		Groups: []Group{{
			ParallaxX: 100,
			ParallaxY: 100,
			Layers: []Layer{{
				Kind:   LayerKindGame,
				Width:  1,
				Height: 1,
				Tiles: []Tile{{
					ID: TileHealth,
				}},
			}},
		}},
	}

	img, err := RenderMap(m, WithEntities(true), WithGameLayer(true))
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	px := centerPixel(img)
	if px.B <= px.R {
		t.Fatalf("expected game-layer overlay to render above entity sprite, got %#v", px)
	}
}

func TestRenderMapWithoutBaseLayerKindsSkipsTileLayers(t *testing.T) {
	m := &Map{
		Images: []Image{{
			Name:   "test",
			Width:  256,
			Height: 256,
			RGBA: makeEntitiesTileset(map[uint8]color.NRGBA{
				1: {R: 255, A: 255},
			}),
		}},
		Groups: []Group{{
			ParallaxX: 100,
			ParallaxY: 100,
			Layers: []Layer{{
				Kind:    LayerKindTiles,
				ImageID: 0,
				Width:   1,
				Height:  1,
				ColorR:  255,
				ColorG:  255,
				ColorB:  255,
				ColorA:  255,
				Tiles: []Tile{{
					ID: 1,
				}},
			}},
		}},
	}

	img, err := RenderMap(m)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if px := centerPixel(img); px.R <= 180 {
		t.Fatalf("expected base tile layer to render, got %#v", px)
	}

	img, err = RenderMap(m, WithoutBaseLayerKinds(LayerKindTiles))
	if err != nil {
		t.Fatalf("filtered render failed: %v", err)
	}
	if img.Bounds().Dx() != 1 || img.Bounds().Dy() != 1 {
		t.Fatalf("expected filtered render to collapse to 1x1 output, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestBlitSpriteRectMatchesSharedScaler(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			off := src.PixOffset(x, y)
			src.Pix[off+0] = uint8(20 + x*40)
			src.Pix[off+1] = uint8(30 + y*50)
			src.Pix[off+2] = uint8(40 + (x+y)*20)
			src.Pix[off+3] = uint8(90 + x*25 + y*20)
		}
	}

	srcRect := image.Rect(1, 1, 4, 4)
	w, h := 7, 5
	expectedScaled := scaleImageRectNRGBA(src, srcRect, w, h)
	expected := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := expectedScaled.PixOffset(x, y)
			alphaBlendPixel(expected, x, y, color.NRGBA{
				R: expectedScaled.Pix[off+0],
				G: expectedScaled.Pix[off+1],
				B: expectedScaled.Pix[off+2],
				A: expectedScaled.Pix[off+3],
			})
		}
	}
	canvas := image.NewNRGBA(image.Rect(0, 0, w, h))
	blitSpriteRect(canvas, src, srcRect, 0, 0, float64(w), float64(h))

	if !reflect.DeepEqual(canvas.Pix, expected.Pix) {
		for i := 0; i < len(canvas.Pix); i += 4 {
			if canvas.Pix[i] == expected.Pix[i] &&
				canvas.Pix[i+1] == expected.Pix[i+1] &&
				canvas.Pix[i+2] == expected.Pix[i+2] &&
				canvas.Pix[i+3] == expected.Pix[i+3] {
				continue
			}
			px := (i / 4) % w
			py := (i / 4) / w
			t.Fatalf("blitSpriteRect mismatch at (%d,%d): got=%v want=%v", px, py,
				canvas.Pix[i:i+4], expected.Pix[i:i+4])
		}
		t.Fatalf("blitSpriteRect did not match shared bilinear scaler")
	}
}

func TestBilinearSampleNormalizedNRGBA(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255})
	src.SetNRGBA(0, 1, color.NRGBA{B: 255, A: 255})
	src.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 255, A: 255})

	c := bilinearSampleNormalizedNRGBA(src, 0.5, 0.5)
	if c.R < 120 || c.G < 120 || c.B < 60 || c.A != 255 {
		t.Fatalf("unexpected bilinear normalized sample: %#v", c)
	}
}

func TestRenderMapPreservesFractionalParallaxTileOffset(t *testing.T) {
	m := &Map{
		Images: []Image{{
			Name:   "test",
			Width:  256,
			Height: 256,
			RGBA: makeEntitiesTileset(map[uint8]color.NRGBA{
				1: {R: 255, A: 255},
			}),
		}},
		Groups: []Group{{
			ParallaxX: 50,
			ParallaxY: 100,
			Layers: []Layer{{
				Kind:    LayerKindTiles,
				ImageID: 0,
				Width:   1,
				Height:  1,
				ColorR:  255,
				ColorG:  255,
				ColorB:  255,
				ColorA:  255,
				Tiles: []Tile{{
					ID: 1,
				}},
			}},
		}},
	}

	img, err := RenderMap(m,
		WithRegion(MapBounds{MinX: 0, MinY: 0, MaxX: 2, MaxY: 1}),
		WithCamera(1, 0.5),
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	firstRed := -1
	for x := 0; x < img.Bounds().Dx(); x++ {
		off := img.PixOffset(x, 0)
		if img.Pix[off] > 200 && img.Pix[off+1] < 50 && img.Pix[off+2] < 50 {
			firstRed = x
			break
		}
	}
	if firstRed != 8 {
		t.Fatalf("expected first tile pixel at x=8 for half-tile parallax shift, got %d", firstRed)
	}
}

func TestRenderOverlayLayerTextTele(t *testing.T) {
	canvas := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	layer := overlayRenderLayer{
		kind: LayerKindTele,
		layer: renderLayer{
			width:  1,
			height: 1,
			color:  color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		},
		teleTiles: []TeleTile{{Number: 8, ID: TileTeleIn}},
	}

	renderOverlayLayerText(canvas, &layer, &MapBounds{MinX: 0, MinY: 0, MaxX: 1, MaxY: 1}, 64)

	if !hasPixelInRect(canvas, image.Rect(16, 10, 48, 50), func(px color.NRGBA) bool {
		return px.R > 220 && px.G > 220 && px.B > 220
	}) {
		t.Fatalf("expected tele overlay text in the center of the tile")
	}
}

func TestRenderOverlayLayerTextSwitch(t *testing.T) {
	canvas := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	layer := overlayRenderLayer{
		kind: LayerKindSwitch,
		layer: renderLayer{
			width:  1,
			height: 1,
			color:  color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		},
		switchTiles: []SwitchTile{{Number: 8, ID: TileSwitchOpen, Delay: 6}},
	}

	renderOverlayLayerText(canvas, &layer, &MapBounds{MinX: 0, MinY: 0, MaxX: 1, MaxY: 1}, 64)

	if !hasPixelInRect(canvas, image.Rect(12, 4, 52, 26), func(px color.NRGBA) bool {
		return px.R > 220 && px.G > 220 && px.B > 220
	}) {
		t.Fatalf("expected switch number text in the top half of the tile")
	}
	if !hasPixelInRect(canvas, image.Rect(12, 34, 52, 58), func(px color.NRGBA) bool {
		return px.R > 220 && px.G > 220 && px.B > 220
	}) {
		t.Fatalf("expected switch delay text in the bottom half of the tile")
	}
}

func TestRenderOverlayLayerTextTune(t *testing.T) {
	canvas := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	layer := overlayRenderLayer{
		kind: LayerKindTune,
		layer: renderLayer{
			width:  1,
			height: 1,
			color:  color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		},
		tuneTiles: []TuneTile{{Number: 7, ID: TileTune}},
	}

	renderOverlayLayerText(canvas, &layer, &MapBounds{MinX: 0, MinY: 0, MaxX: 1, MaxY: 1}, 64)

	if !hasPixelInRect(canvas, image.Rect(16, 10, 48, 50), func(px color.NRGBA) bool {
		return px.R > 220 && px.G > 220 && px.B > 220
	}) {
		t.Fatalf("expected tune overlay text in the center of the tile")
	}
}

func TestRenderOverlayLayerTextSpeedup(t *testing.T) {
	canvas := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	layer := overlayRenderLayer{
		kind: LayerKindSpeedup,
		layer: renderLayer{
			width:  1,
			height: 1,
			color:  color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		},
		speedupTiles: []SpeedupTile{{Force: 8, MaxSpeed: 6, ID: TileSpeedBoost, Angle: 0}},
	}

	renderOverlayLayerText(canvas, &layer, &MapBounds{MinX: 0, MinY: 0, MaxX: 1, MaxY: 1}, 64)

	if !hasPixelInRect(canvas, image.Rect(12, 2, 52, 22), func(px color.NRGBA) bool {
		return px.R > 220 && px.G > 220 && px.B > 220
	}) {
		t.Fatalf("expected speedup max-speed text in the top band of the tile")
	}
	if !hasPixelInRect(canvas, image.Rect(12, 38, 52, 60), func(px color.NRGBA) bool {
		return px.R > 220 && px.G > 220 && px.B > 220
	}) {
		t.Fatalf("expected speedup force text in the bottom band of the tile")
	}
}
