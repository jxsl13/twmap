package twmap

import "fmt"

// This file provides the public map-builder API: thin constructors and
// fluent assembly helpers for creating Teeworlds/DDNet maps from scratch.
//
// The model mirrors the parsed representation exactly — there is no parallel
// builder type. Constructors return plain Map/Group/Layer values with correct
// defaults (sentinel reference ids of -1, opaque-white tile color, parallax
// 100/100, pre-sized tile slices); the Add* helpers append and return a
// pointer into the owning slice so calls can be chained:
//
//	m := twmap.NewMap(twmap.MapVersion06)
//	g := m.AddGroup(twmap.NewGroup("Game"))
//	game := g.AddLayer(twmap.NewGameLayer(50, 30))
//	game.SetTile(5, 5, twmap.Tile{ID: twmap.TileSolid})
//	_ = m.Write(w)

// NewMap returns an empty map ready for assembly. A zero-value version
// defaults to MapVersion06 (Teeworlds 0.6 / DDNet).
func NewMap(v MapVersion) *Map {
	if v == 0 {
		v = MapVersion06
	}
	return &Map{Version: v}
}

// NewGroup returns a group with default parallax 100/100 (normal scroll) and
// no clipping. Layers are added with (*Group).AddLayer.
func NewGroup(name string) Group {
	return Group{Name: name, ParallaxX: 100, ParallaxY: 100}
}

// AddGroup appends g to the map and returns a pointer to the stored group so
// further layers can be added to it.
func (m *Map) AddGroup(g Group) *Group {
	m.Groups = append(m.Groups, g)
	return &m.Groups[len(m.Groups)-1]
}

// AddLayer appends l to the group and returns a pointer to the stored layer so
// its tiles can be edited in place.
func (g *Group) AddLayer(l Layer) *Layer {
	g.Layers = append(g.Layers, l)
	return &g.Layers[len(g.Layers)-1]
}

// newLayerBase returns a layer with the shared default fields: opaque-white
// color and all reference ids set to the -1 "none" sentinel. It allocates no
// tile data — callers fill the appropriate slice.
func newLayerBase(kind LayerKind, name string, w, h int) Layer {
	return Layer{
		Kind:        kind,
		Name:        name,
		Width:       w,
		Height:      h,
		ColorR:      255,
		ColorG:      255,
		ColorB:      255,
		ColorA:      255,
		ImageID:     -1,
		ColorEnv:    -1,
		QuadImageID: -1,
		SoundID:     -1,
	}
}

// NewTileLayer returns a regular w×h visual tile layer filled with air tiles.
func NewTileLayer(name string, w, h int) Layer {
	l := newLayerBase(LayerKindTiles, name, w, h)
	l.Tiles = make([]Tile, w*h)
	return l
}

// NewGameLayer returns a w×h game (physics) layer filled with air tiles.
func NewGameLayer(w, h int) Layer {
	l := newLayerBase(LayerKindGame, "Game", w, h)
	l.Tiles = make([]Tile, w*h)
	return l
}

// NewFrontLayer returns a w×h DDNet front layer filled with air tiles. Front
// tiles share the regular Tile type and live in (*Layer).Tiles.
func NewFrontLayer(w, h int) Layer {
	l := newLayerBase(LayerKindFront, "Front", w, h)
	l.Tiles = make([]Tile, w*h)
	return l
}

// NewTeleLayer returns a w×h DDNet teleport layer with an empty TeleTiles grid.
func NewTeleLayer(w, h int) Layer {
	l := newLayerBase(LayerKindTele, "Tele", w, h)
	l.TeleTiles = make([]TeleTile, w*h)
	return l
}

// NewSpeedupLayer returns a w×h DDNet speedup layer with an empty SpeedupTiles grid.
func NewSpeedupLayer(w, h int) Layer {
	l := newLayerBase(LayerKindSpeedup, "Speedup", w, h)
	l.SpeedupTiles = make([]SpeedupTile, w*h)
	return l
}

// NewSwitchLayer returns a w×h DDNet switch layer with an empty SwitchTiles grid.
func NewSwitchLayer(w, h int) Layer {
	l := newLayerBase(LayerKindSwitch, "Switch", w, h)
	l.SwitchTiles = make([]SwitchTile, w*h)
	return l
}

// NewTuneLayer returns a w×h DDNet tune layer with an empty TuneTiles grid.
func NewTuneLayer(w, h int) Layer {
	l := newLayerBase(LayerKindTune, "Tune", w, h)
	l.TuneTiles = make([]TuneTile, w*h)
	return l
}

// NewQuadsLayer returns an empty quad layer. Append quads to (*Layer).Quads
// and set QuadImageID to reference a tileset image.
func NewQuadsLayer(name string) Layer {
	return newLayerBase(LayerKindQuads, name, 0, 0)
}

// tileIndex maps (x,y) to an index into the row-major Tiles slice, panicking
// slice-style if the coordinate is outside [0,Width)×[0,Height).
func (l *Layer) tileIndex(x, y int) int {
	if x < 0 || y < 0 || x >= l.Width || y >= l.Height {
		panic(fmt.Sprintf("twmap: tile (%d,%d) out of bounds for %dx%d layer", x, y, l.Width, l.Height))
	}
	return y*l.Width + x
}

// SetTile sets the tile at column x, row y. It panics if the coordinate is out
// of bounds. Operates on the regular Tiles grid (tiles/game/front layers).
func (l *Layer) SetTile(x, y int, t Tile) {
	l.Tiles[l.tileIndex(x, y)] = t
}

// TileAt returns the tile at column x, row y. It panics if the coordinate is
// out of bounds.
func (l *Layer) TileAt(x, y int) Tile {
	return l.Tiles[l.tileIndex(x, y)]
}

// Fill sets every tile in the regular Tiles grid to t.
func (l *Layer) Fill(t Tile) {
	for i := range l.Tiles {
		l.Tiles[i] = t
	}
}
