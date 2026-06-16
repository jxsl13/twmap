package twmap

import (
	"bytes"
	"image"
	"testing"
)

// V3, V6 — NewMap/NewGroup defaults.
func TestNewMapAndGroupDefaults(t *testing.T) {
	if m := NewMap(0); m.Version != MapVersion06 {
		t.Errorf("NewMap(0) version: got %d, want %d (MapVersion06)", m.Version, MapVersion06)
	}
	if m := NewMap(MapVersion07); m.Version != MapVersion07 {
		t.Errorf("NewMap(MapVersion07) version: got %d", m.Version)
	}
	g := NewGroup("Game")
	if g.ParallaxX != 100 || g.ParallaxY != 100 {
		t.Errorf("NewGroup parallax: got (%d,%d), want (100,100)", g.ParallaxX, g.ParallaxY)
	}
}

// V1, V2, V4, V8 — tile-layer ctor defaults.
func TestNewTileLayerDefaults(t *testing.T) {
	l := NewTileLayer("bg", 7, 5)
	if got := len(l.Tiles); got != 7*5 {
		t.Errorf("Tiles len: got %d, want %d", got, 7*5) // V1
	}
	if l.ColorR != 255 || l.ColorG != 255 || l.ColorB != 255 || l.ColorA != 255 {
		t.Errorf("color: got (%d,%d,%d,%d), want opaque white", l.ColorR, l.ColorG, l.ColorB, l.ColorA) // V4
	}
	if l.ImageID != -1 || l.ColorEnv != -1 || l.QuadImageID != -1 || l.SoundID != -1 {
		t.Errorf("ref ids not all -1: img=%d env=%d quad=%d snd=%d", l.ImageID, l.ColorEnv, l.QuadImageID, l.SoundID) // V2
	}
}

// V8 — physics ctors set kind + matching special-tile slice length.
func TestPhysicsLayerCtors(t *testing.T) {
	const w, h = 4, 3
	cases := []struct {
		name string
		l    Layer
		kind LayerKind
		n    int
	}{
		{"game", NewGameLayer(w, h), LayerKindGame, len(NewGameLayer(w, h).Tiles)},
		{"front", NewFrontLayer(w, h), LayerKindFront, len(NewFrontLayer(w, h).Tiles)},
		{"tele", NewTeleLayer(w, h), LayerKindTele, len(NewTeleLayer(w, h).TeleTiles)},
		{"speedup", NewSpeedupLayer(w, h), LayerKindSpeedup, len(NewSpeedupLayer(w, h).SpeedupTiles)},
		{"switch", NewSwitchLayer(w, h), LayerKindSwitch, len(NewSwitchLayer(w, h).SwitchTiles)},
		{"tune", NewTuneLayer(w, h), LayerKindTune, len(NewTuneLayer(w, h).TuneTiles)},
	}
	for _, c := range cases {
		if c.l.Kind != c.kind {
			t.Errorf("%s: kind got %d, want %d", c.name, c.l.Kind, c.kind)
		}
		if c.n != w*h {
			t.Errorf("%s: special slice len got %d, want %d", c.name, c.n, w*h)
		}
	}
	if l := NewQuadsLayer("q"); l.Kind != LayerKindQuads || l.QuadImageID != -1 {
		t.Errorf("NewQuadsLayer: kind=%d quadImageID=%d", l.Kind, l.QuadImageID)
	}
}

// V1, V7 — accessors index correctly and bounds-panic.
func TestTileAccessors(t *testing.T) {
	l := NewTileLayer("l", 3, 2)
	l.SetTile(2, 1, Tile{ID: TileSolid})
	if got := l.TileAt(2, 1); got.ID != TileSolid {
		t.Errorf("TileAt(2,1): got id %d, want %d", got.ID, TileSolid)
	}
	if l.Tiles[1*3+2].ID != TileSolid { // V1 row-major index
		t.Error("SetTile wrote wrong index")
	}
	l.Fill(Tile{ID: TileFreeze})
	for i, tile := range l.Tiles {
		if tile.ID != TileFreeze {
			t.Fatalf("Fill missed index %d", i)
		}
	}
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {3, 0}, {0, 2}} { // V7
		func(x, y int) {
			defer func() {
				if recover() == nil {
					t.Errorf("SetTile(%d,%d): expected out-of-bounds panic", x, y)
				}
			}()
			l.SetTile(x, y, Tile{})
		}(p[0], p[1])
	}
}

// V12 — NewSoundLayer defaults.
func TestNewSoundLayer(t *testing.T) {
	l := NewSoundLayer("amb")
	if l.Kind != LayerKindSounds {
		t.Errorf("kind: got %d, want %d", l.Kind, LayerKindSounds)
	}
	if l.SoundID != -1 || l.ImageID != -1 || l.ColorEnv != -1 || l.QuadImageID != -1 {
		t.Errorf("ref ids not -1: snd=%d img=%d env=%d quad=%d", l.SoundID, l.ImageID, l.ColorEnv, l.QuadImageID)
	}
	if len(l.SoundSources) != 0 {
		t.Errorf("SoundSources: got %d, want 0", len(l.SoundSources))
	}
}

// V9 — special-tile accessors write the matching grid and bounds-panic.
func TestSpecialTileAccessors(t *testing.T) {
	tele := NewTeleLayer(3, 2)
	tele.SetTeleTile(2, 1, TeleTile{Number: 7, ID: TileTeleInEvil})
	if got := tele.TeleTileAt(2, 1); got.Number != 7 || got.ID != TileTeleInEvil {
		t.Errorf("tele: got %+v", got)
	}
	if tele.TeleTiles[1*3+2].Number != 7 { // matching grid, row-major
		t.Error("SetTeleTile wrote wrong grid/index")
	}

	su := NewSpeedupLayer(2, 2)
	su.SetSpeedupTile(1, 1, SpeedupTile{Force: 50, Angle: 90})
	if su.SpeedupTileAt(1, 1).Force != 50 {
		t.Error("speedup accessor mismatch")
	}
	sw := NewSwitchLayer(2, 2)
	sw.SetSwitchTile(0, 1, SwitchTile{Number: 3})
	if sw.SwitchTileAt(0, 1).Number != 3 {
		t.Error("switch accessor mismatch")
	}
	tn := NewTuneLayer(2, 2)
	tn.SetTuneTile(1, 0, TuneTile{Number: 4})
	if tn.TuneTileAt(1, 0).Number != 4 {
		t.Error("tune accessor mismatch")
	}

	func() { // V9 bounds-panic
		defer func() {
			if recover() == nil {
				t.Error("SetTeleTile out of bounds: expected panic")
			}
		}()
		tele.SetTeleTile(3, 0, TeleTile{})
	}()
}

// V10 — NewQuad geometry, colors, texcoords, envelopes.
func TestNewQuad(t *testing.T) {
	q := NewQuad(4, 2, 2, 2) // center (4,2), 2x2 tiles
	const u = 1 << 15
	wantPts := [5]Point{
		{X: 3 * u, Y: 1 * u}, // TL
		{X: 5 * u, Y: 1 * u}, // TR
		{X: 3 * u, Y: 3 * u}, // BL
		{X: 5 * u, Y: 3 * u}, // BR
		{X: 4 * u, Y: 2 * u}, // center
	}
	if q.Points != wantPts {
		t.Errorf("points: got %+v, want %+v", q.Points, wantPts)
	}
	for i, c := range q.Colors {
		if c.R != 255 || c.G != 255 || c.B != 255 || c.A != 255 {
			t.Errorf("color %d not opaque white: %+v", i, c)
		}
	}
	const tu = 1 << 10
	wantTex := [4]Point{{0, 0}, {tu, 0}, {0, tu}, {tu, tu}}
	if q.TexCoords != wantTex {
		t.Errorf("texcoords: got %+v, want %+v", q.TexCoords, wantTex)
	}
	if q.PosEnv != -1 || q.ColorEnv != -1 {
		t.Errorf("envs: posEnv=%d colorEnv=%d, want -1", q.PosEnv, q.ColorEnv)
	}
}

// V11 — AddImage/AddExternalImage index wiring and fields.
func TestAddImages(t *testing.T) {
	m := NewMap(MapVersion06)
	rgba := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	i0 := m.AddImage("grass", rgba)
	i1 := m.AddExternalImage("desert", 128, 96)
	if i0 != 0 || i1 != 1 {
		t.Fatalf("indices: got %d,%d want 0,1", i0, i1)
	}
	if len(m.Images) != 2 {
		t.Fatalf("images len: %d", len(m.Images))
	}
	emb := m.Images[i0]
	if emb.External || emb.RGBA == nil || emb.Width != 64 || emb.Height != 32 {
		t.Errorf("embedded image wrong: %+v", emb)
	}
	ext := m.Images[i1]
	if !ext.External || ext.RGBA != nil || ext.Width != 128 || ext.Height != 96 {
		t.Errorf("external image wrong: %+v", ext)
	}
}

// V5, C3 — a map built from scratch survives Write -> Parse and validates.
func TestBuildRoundTrip(t *testing.T) {
	const w, h = 12, 8
	m := NewMap(MapVersion06)
	m.Info.Author = "builder-test"

	g := m.AddGroup(NewGroup("Game"))
	bg := g.AddLayer(NewTileLayer("bg", w, h))
	bg.Fill(Tile{ID: TileSolid})
	game := g.AddLayer(NewGameLayer(w, h))
	game.SetTile(0, 0, Tile{ID: TileSolid})
	game.SetTile(w-1, h-1, Tile{ID: TileSolid})
	g.AddLayer(NewQuadsLayer("decor"))

	var buf bytes.Buffer
	if err := m.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := Validate(bytes.NewReader(buf.Bytes()), WithRequireInfo(false)); err != nil {
		t.Fatalf("Validate built map: %v", err)
	}

	m2, err := Parse(bytes.NewReader(buf.Bytes()), WithRequireInfo(false))
	if err != nil {
		t.Fatalf("re-Parse: %v", err)
	}
	if len(m2.Groups) != 1 || len(m2.Groups[0].Layers) != 3 {
		t.Fatalf("structure: groups=%d layers=%d", len(m2.Groups), len(m2.Groups[0].Layers))
	}
	g2 := m2.Groups[0]
	if g2.ParallaxX != 100 || g2.ParallaxY != 100 {
		t.Errorf("parallax round-trip: got (%d,%d)", g2.ParallaxX, g2.ParallaxY)
	}
	var gameOut *Layer
	for i := range g2.Layers {
		if g2.Layers[i].Kind == LayerKindGame {
			gameOut = &g2.Layers[i]
		}
	}
	if gameOut == nil {
		t.Fatal("game layer missing after round-trip")
	}
	if gameOut.Width != w || gameOut.Height != h {
		t.Errorf("game dims: got %dx%d, want %dx%d", gameOut.Width, gameOut.Height, w, h)
	}
	if gameOut.TileAt(0, 0).ID != TileSolid || gameOut.TileAt(w-1, h-1).ID != TileSolid {
		t.Error("game tiles not preserved across round-trip")
	}
}
