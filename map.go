package twmap

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

// ── map item type IDs (from datafile spec) ───────────────────────────────────

const (
	mapItemTypeVersion   uint16 = 0
	mapItemTypeInfo      uint16 = 1
	mapItemTypeImage     uint16 = 2
	mapItemTypeEnvelope  uint16 = 3
	mapItemTypeGroup     uint16 = 4
	mapItemTypeLayer     uint16 = 5
	mapItemTypeEnvPoints uint16 = 6
	mapItemTypeSound     uint16 = 7
)

// ── layer type constants ─────────────────────────────────────────────────────

const (
	layerTypeTilemap            int32 = 2
	layerTypeQuads              int32 = 3
	layerTypeDdraceSoundsLegacy int32 = 9
	layerTypeDdraceSounds       int32 = 10
)

// ── tile layer flag constants ────────────────────────────────────────────────

const (
	tileLayerFlagGame    uint32 = 1
	tileLayerFlagTele    uint32 = 2
	tileLayerFlagSpeedup uint32 = 4
	tileLayerFlagFront   uint32 = 8
	tileLayerFlagSwitch  uint32 = 16
	tileLayerFlagTune    uint32 = 32
)

// ── Tile is 4 bytes: id, flags, skip, unused ─────────────────────────────────

// Tile represents a single tile in a tilemap layer.
type Tile struct {
	ID    uint8
	Flags uint8
}

// TileFlags
const (
	TileFlagVFlip  uint8 = 1
	TileFlagHFlip  uint8 = 2
	TileFlagOpaque uint8 = 4
	TileFlagRotate uint8 = 8
)

// ── DDNet special tile types ─────────────────────────────────────────────────

// TeleTile is a teleporter tile (DDNet).
type TeleTile struct {
	Number uint8
	ID     uint8
}

// SpeedupTile is a speed-boost tile (DDNet).
type SpeedupTile struct {
	Force    uint8
	MaxSpeed uint8
	ID       uint8
	Angle    int16
}

// SwitchTile is a switch tile (DDNet).
type SwitchTile struct {
	Number uint8
	ID     uint8
	Flags  uint8
	Delay  uint8
}

// TuneTile is a tune-zone tile (DDNet).
type TuneTile struct {
	Number uint8
	ID     uint8
}

// ── Envelope types ───────────────────────────────────────────────────────────

// CurveType identifies the interpolation between envelope points.
type CurveType int32

const (
	CurveStep   CurveType = 0
	CurveLinear CurveType = 1
	CurveSlow   CurveType = 2
	CurveFast   CurveType = 3
	CurveSmooth CurveType = 4
	CurveBezier CurveType = 5
)

// EnvPoint represents a single control point in an envelope.
type EnvPoint struct {
	Time      int32 // milliseconds
	CurveType CurveType
	Values    [4]int32 // 22.10 fixed-point; meaning depends on envelope type
}

// Envelope represents an animation envelope (position, color, or sound).
type Envelope struct {
	Name         string
	Channels     int32 // 1=sound, 3=position, 4=color
	Synchronized bool
	Points       []EnvPoint
}

// ── Sound types ──────────────────────────────────────────────────────────────

// Sound is a sound resource stored in the map (DDNet only).
type Sound struct {
	Name string
	Data []byte // opus encoded; nil for external sounds
}

// SoundSource represents a sound emitter in a sounds layer.
type SoundSource struct {
	Position       Point
	Loop           bool
	Panning        bool
	Delay          int32 // seconds
	Falloff        uint8
	PosEnv         int32
	PosEnvOffset   int32
	SoundEnv       int32
	SoundEnvOffset int32
	ShapeType      int32 // 0=rectangle, 1=circle
	ShapeWidth     int32 // 22.10 fxp (rect width or circle radius)
	ShapeHeight    int32 // 22.10 fxp (rect height; unused for circle)
}

// ── Quad ─────────────────────────────────────────────────────────────────────

// Quad represents a single quad in a quads layer.
// Positions use 17.15 fixed-point.
type Quad struct {
	Points         [5]Point // corners[4] + position
	Colors         [4]color.NRGBA
	TexCoords      [4]Point
	PosEnv         int32
	PosEnvOffset   int32
	ColorEnv       int32
	ColorEnvOffset int32
}

// Point is a 2D coordinate (fixed-point 17.15 in the datafile).
type Point struct {
	X, Y float64
}

// ── Map version ──────────────────────────────────────────────────────────────

// MapVersion identifies the Teeworlds map format variant.
// The value corresponds to the CMapItemImage version found in the map:
// version 1 images are used in TW 0.6/DDNet, version 2 in TW 0.7.
type MapVersion int

const (
	MapVersion06 MapVersion = 1 // CMapItemImage v1 (Teeworlds 0.6 / DDNet)
	MapVersion07 MapVersion = 2 // CMapItemImage v2 (Teeworlds 0.7)
)

// ── Map model (mirrors ddnet-rs/twmap TwMap) ─────────────────────────────────

// Map holds all parsed information from a Teeworlds/DDNet map file.
type Map struct {
	Version   MapVersion // MapVersion06 or MapVersion07
	Info      Info
	Images    []Image
	Envelopes []Envelope
	Groups    []Group
	Sounds    []Sound // DDNet only
}

// Info holds map metadata.
type Info struct {
	Author   string
	Version  string
	Credits  string
	License  string
	Settings []string // DDNet map commands/settings
}

// Image is either embedded (with RGBA pixel data) or external (name only).
type Image struct {
	Name     string
	Width    int
	Height   int
	External bool
	RGBA     *image.NRGBA // nil for external images
}

// Group is a collection of layers with a shared offset/parallax.
type Group struct {
	Name      string
	OffsetX   int32
	OffsetY   int32
	ParallaxX int32
	ParallaxY int32
	Clipping  bool
	ClipX     int32
	ClipY     int32
	ClipW     int32
	ClipH     int32
	Layers    []Layer
}

// IsPhysicsGroup returns true if this group contains any physics layers.
func (g *Group) IsPhysicsGroup() bool {
	for _, l := range g.Layers {
		if l.IsPhysics() {
			return true
		}
	}
	return false
}

// LayerKind identifies the type of layer.
type LayerKind int

const (
	LayerKindTiles LayerKind = iota
	LayerKindGame
	LayerKindFront
	LayerKindTele
	LayerKindSpeedup
	LayerKindSwitch
	LayerKindTune
	LayerKindQuads
	LayerKindSounds
	LayerKindInvalid
)

// Layer represents a single layer in a group.
type Layer struct {
	Kind   LayerKind
	Detail bool

	// Tilemap fields (used when Kind is tiles/game/front/tele/speedup/switch/tune)
	Width          int
	Height         int
	ColorR         uint8
	ColorG         uint8
	ColorB         uint8
	ColorA         uint8
	ImageID        int   // -1 = no image
	ColorEnv       int32 // envelope index, -1 = none
	ColorEnvOffset int32
	Tiles          []Tile

	// DDNet special tile data (only set for the corresponding layer kind)
	TeleTiles    []TeleTile
	SpeedupTiles []SpeedupTile
	SwitchTiles  []SwitchTile
	TuneTiles    []TuneTile

	// Quads fields (used when Kind is LayerKindQuads)
	Quads       []Quad
	QuadImageID int // -1 = no image

	// Sound layer fields (used when Kind is LayerKindSounds)
	SoundSources []SoundSource
	SoundID      int // index into Map.Sounds, -1 = no sound

	// Name
	Name string
}

// IsPhysics returns true for game/front/tele/speedup/switch/tune layers.
func (l *Layer) IsPhysics() bool {
	switch l.Kind {
	case LayerKindGame, LayerKindFront, LayerKindTele, LayerKindSpeedup, LayerKindSwitch, LayerKindTune:
		return true
	}
	return false
}

// SpawnType classifies a spawn point.
type SpawnType uint8

const (
	SpawnGeneric SpawnType = iota // DM / generic spawn (tile 192)
	SpawnRed                      // red team spawn (tile 193)
	SpawnBlue                     // blue team spawn (tile 194)
)

// SpawnPoint is a spawn location in the game layer.
type SpawnPoint struct {
	X, Y int // tile coordinates (column, row)
	Type SpawnType
}

// SpawnPoints returns all spawn points found in the game layer, ordered
// from top-left to bottom-right (first by Y ascending, then by X ascending).
func (m *Map) SpawnPoints() []SpawnPoint {
	var spawns []SpawnPoint
	for _, g := range m.Groups {
		for _, l := range g.Layers {
			if l.Kind != LayerKindGame {
				continue
			}
			for i, t := range l.Tiles {
				if !IsSpawn(t.ID) {
					continue
				}
				var st SpawnType
				switch t.ID {
				case TileSpawnRed:
					st = SpawnRed
				case TileSpawnBlue:
					st = SpawnBlue
				default:
					st = SpawnGeneric
				}
				spawns = append(spawns, SpawnPoint{
					X:    i % l.Width,
					Y:    i / l.Width,
					Type: st,
				})
			}
		}
	}
	// Tiles are iterated row-by-row, so the result is already
	// sorted top-left to bottom-right (Y asc, then X asc).
	return spawns
}

// GameLayers returns all game layers (LayerKindGame) found in the map.
func (m *Map) GameLayers() []Layer {
	var layers []Layer
	for _, g := range m.Groups {
		for _, l := range g.Layers {
			if l.Kind == LayerKindGame {
				layers = append(layers, l)
			}
		}
	}
	return layers
}

// IsTilemap returns true for any tilemap-based layer.
func (l *Layer) IsTilemap() bool {
	switch l.Kind {
	case LayerKindTiles, LayerKindGame, LayerKindFront, LayerKindTele, LayerKindSpeedup, LayerKindSwitch, LayerKindTune:
		return true
	}
	return false
}

// ── Parse: construct Map from an io.Reader ───────────────────────────────────

// ParseOption configures the map parser.
type ParseOption func(*parseOptions)

type parseOptions struct {
	requireInfo bool
}

func defaultParseOptions() parseOptions {
	return parseOptions{requireInfo: true}
}

// WithRequireInfo controls whether the parser returns an error when the
// map info item is missing. Default is true.
func WithRequireInfo(require bool) ParseOption {
	return func(o *parseOptions) { o.requireInfo = require }
}

// Parse reads and parses a Teeworlds/DDNet map from r.
// It fully decodes all tile data and embedded images so the returned Map
// can be used for thumbnail generation.
func Parse(r io.Reader, opts ...ParseOption) (*Map, error) {
	df, err := parseDatafile(r)
	if err != nil {
		return nil, fmt.Errorf("datafile: %w", err)
	}
	o := defaultParseOptions()
	for _, fn := range opts {
		fn(&o)
	}
	return parseMap(df, &o)
}

// ParseInfo reads only the metadata (author, version, credits, license)
// from a Teeworlds/DDNet map without decoding images or layers.
func ParseInfo(r io.Reader) (Info, error) {
	df, err := parseDatafile(r)
	if err != nil {
		return Info{}, fmt.Errorf("datafile: %w", err)
	}
	if err := checkMapVersion(df); err != nil {
		return Info{}, err
	}
	return parseInfo(df, true)
}

func parseMap(df *datafile, opts *parseOptions) (*Map, error) {
	m := &Map{}

	// ── version ──────────────────────────────────────────────────────────
	if err := checkMapVersion(df); err != nil {
		return nil, err
	}

	// ── info ─────────────────────────────────────────────────────────────
	info, err := parseInfo(df, opts.requireInfo)
	if err != nil {
		return nil, err
	}
	m.Info = info

	// ── images ───────────────────────────────────────────────────────────
	images, ver, err := parseImages(df)
	if err != nil {
		return nil, err
	}
	m.Images = images
	m.Version = ver

	// ── envelopes ────────────────────────────────────────────────────────
	envelopes, err := parseEnvelopes(df)
	if err != nil {
		return nil, err
	}
	m.Envelopes = envelopes

	// ── sounds (DDNet only) ──────────────────────────────────────────────
	sounds, err := parseSounds(df)
	if err != nil {
		return nil, err
	}
	m.Sounds = sounds

	// ── groups & layers ──────────────────────────────────────────────────
	groups, err := parseGroups(df)
	if err != nil {
		return nil, err
	}
	m.Groups = groups

	return m, nil
}

// ── version check ────────────────────────────────────────────────────────────

func checkMapVersion(df *datafile) error {
	item := df.findItem(mapItemTypeVersion, 0)
	if item == nil {
		return ErrMissingVersion
	}
	if len(item.Data) < 1 {
		return fmt.Errorf("%w: empty version item", ErrInvalidVersion)
	}
	version := item.Data[0]
	if version != 1 {
		return fmt.Errorf("%w: got %d, want 1", ErrInvalidVersion, version)
	}
	return nil
}

// ── info parsing ─────────────────────────────────────────────────────────────

func parseInfo(df *datafile, requireInfo bool) (Info, error) {
	item := df.findItem(mapItemTypeInfo, 0)
	if item == nil {
		if requireInfo {
			return Info{}, ErrMissingInfo
		}
		return Info{}, nil
	}
	if len(item.Data) < 5 {
		return Info{}, fmt.Errorf("info item too short: %d int32s, want >= 5", len(item.Data))
	}

	readStr := func(dataIdx int32) string {
		if dataIdx < 0 || int(dataIdx) >= df.numData {
			return ""
		}
		data, err := df.readData(int(dataIdx))
		if err != nil || len(data) == 0 {
			return ""
		}
		// c-string: strip trailing NUL
		if data[len(data)-1] == 0 {
			data = data[:len(data)-1]
		}
		return string(data)
	}

	info := Info{
		Author:  readStr(item.Data[1]),
		Version: readStr(item.Data[2]),
		Credits: readStr(item.Data[3]),
		License: readStr(item.Data[4]),
	}

	// DDNet settings (item version 1, 6th field)
	if len(item.Data) >= 6 {
		settingsIdx := item.Data[5]
		if settingsIdx >= 0 && int(settingsIdx) < df.numData {
			rawSettings, err := df.readData(int(settingsIdx))
			if err == nil && len(rawSettings) > 0 {
				info.Settings = parseCStringArray(rawSettings)
			}
		}
	}

	return info, nil
}

// parseCStringArray splits a NUL-delimited byte sequence into strings.
func parseCStringArray(data []byte) []string {
	var result []string
	var cur []byte
	for _, b := range data {
		if b == 0 {
			if len(cur) > 0 {
				result = append(result, string(cur))
				cur = cur[:0]
			}
		} else {
			cur = append(cur, b)
		}
	}
	if len(cur) > 0 {
		result = append(result, string(cur))
	}
	return result
}

// ── image parsing ────────────────────────────────────────────────────────────

func parseImages(df *datafile) ([]Image, MapVersion, error) {
	items := df.itemsOfType(mapItemTypeImage)
	images := make([]Image, 0, len(items))
	mapVer := MapVersion06

	for i, item := range items {
		if len(item.Data) < 6 {
			return nil, 0, fmt.Errorf("image %d: too short (%d int32s)", i, len(item.Data))
		}
		ver := MapVersion(item.Data[0])
		if ver < MapVersion06 {
			return nil, 0, fmt.Errorf("image %d: invalid version %d", i, ver)
		}
		if ver >= MapVersion07 {
			mapVer = MapVersion07
		}

		width := int(item.Data[1])
		height := int(item.Data[2])
		external := item.Data[3] != 0
		nameIdx := item.Data[4]
		dataIdx := item.Data[5]

		// TW 0.7 image variant: 0=RGB, 1=RGBA (default for v1)
		bytesPerPixel := 4
		if ver >= MapVersion07 && len(item.Data) >= 7 {
			if item.Data[6] == 0 {
				bytesPerPixel = 3 // RGB
			}
		}

		// Validate embedded image dimensions
		if !external {
			if width <= 0 {
				return nil, 0, fmt.Errorf("image %d: invalid width %d", i, width)
			}
			if height <= 0 {
				return nil, 0, fmt.Errorf("image %d: invalid height %d", i, height)
			}
		}

		// Read name
		name := ""
		if nameIdx >= 0 && int(nameIdx) < df.numData {
			nameData, err := df.readData(int(nameIdx))
			if err == nil && len(nameData) > 0 {
				if nameData[len(nameData)-1] == 0 {
					nameData = nameData[:len(nameData)-1]
				}
				name = string(nameData)
			}
		}

		img := Image{
			Name:     name,
			Width:    width,
			Height:   height,
			External: external,
		}

		// Decode embedded image data
		if !external && dataIdx >= 0 && int(dataIdx) < df.numData {
			pixelData, err := df.readData(int(dataIdx))
			if err == nil && len(pixelData) == width*height*bytesPerPixel {
				nrgba := image.NewNRGBA(image.Rect(0, 0, width, height))
				if bytesPerPixel == 4 {
					copy(nrgba.Pix, pixelData)
				} else {
					// Convert RGB to NRGBA
					for j := 0; j < width*height; j++ {
						src := j * 3
						dst := j * 4
						nrgba.Pix[dst] = pixelData[src]
						nrgba.Pix[dst+1] = pixelData[src+1]
						nrgba.Pix[dst+2] = pixelData[src+2]
						nrgba.Pix[dst+3] = 255
					}
				}
				img.RGBA = nrgba
			}
		}

		images = append(images, img)
	}
	return images, mapVer, nil
}

// ── group & layer parsing ────────────────────────────────────────────────────

func parseGroups(df *datafile) ([]Group, error) {
	groupItems := df.itemsOfType(mapItemTypeGroup)
	layerItems := df.itemsOfType(mapItemTypeLayer)

	var hasGame bool
	groups := make([]Group, 0, len(groupItems))

	for gi, gItem := range groupItems {
		if len(gItem.Data) < 7 {
			return nil, fmt.Errorf("group %d: too short (%d int32s)", gi, len(gItem.Data))
		}
		ver := gItem.Data[0]
		if ver < 1 {
			return nil, fmt.Errorf("group %d: invalid version %d", gi, ver)
		}

		g := Group{
			OffsetX:   gItem.Data[1],
			OffsetY:   gItem.Data[2],
			ParallaxX: gItem.Data[3],
			ParallaxY: gItem.Data[4],
		}

		startLayer := int(gItem.Data[5])
		numLayers := int(gItem.Data[6])

		if startLayer < 0 || numLayers < 0 {
			return nil, fmt.Errorf("group %d: invalid layer range", gi)
		}

		// v2: clipping
		if ver >= 2 && len(gItem.Data) >= 12 {
			g.Clipping = gItem.Data[7] != 0
			g.ClipX = gItem.Data[8]
			g.ClipY = gItem.Data[9]
			g.ClipW = gItem.Data[10]
			g.ClipH = gItem.Data[11]
		}

		// v3: name
		if ver >= 3 && len(gItem.Data) >= 15 {
			g.Name = parseI32String(gItem.Data[12:15])
		}

		// Parse layers belonging to this group
		absEnd := startLayer + numLayers
		if absEnd > len(layerItems) {
			return nil, fmt.Errorf("group %d: layer range [%d, %d) exceeds layer count %d", gi, startLayer, absEnd, len(layerItems))
		}

		for li := startLayer; li < absEnd; li++ {
			layer, err := parseLayer(df, &layerItems[li])
			if err != nil {
				return nil, fmt.Errorf("group %d, layer %d: %w", gi, li, err)
			}
			if layer.Kind == LayerKindGame {
				hasGame = true
			}
			g.Layers = append(g.Layers, layer)
		}

		groups = append(groups, g)
	}

	if !hasGame {
		return nil, ErrNoGameLayer
	}
	return groups, nil
}

func parseLayer(df *datafile, item *itemView) (Layer, error) {
	if len(item.Data) < 3 {
		return Layer{}, fmt.Errorf("layer too short (%d int32s)", len(item.Data))
	}

	layerType := item.Data[1]
	flags := uint32(item.Data[2])

	switch layerType {
	case layerTypeTilemap:
		return parseTilemapLayer(df, item.Data, flags)
	case layerTypeQuads:
		return parseQuadsLayer(df, item.Data)
	case layerTypeDdraceSounds:
		return parseSoundsLayer(df, item.Data, false)
	case layerTypeDdraceSoundsLegacy:
		return parseSoundsLayer(df, item.Data, true)
	default:
		return Layer{Kind: LayerKindInvalid}, nil
	}
}

func parseTilemapLayer(df *datafile, data []int32, layerFlags uint32) (Layer, error) {
	// data[0]=layerVersion, [1]=type, [2]=flags
	// data[3]=tilemapVersion, [4]=width, [5]=height, [6]=tileFlags
	// data[7..10]=color, [11]=colorEnv, [12]=colorEnvOffset, [13]=image, [14]=dataIdx
	if len(data) < 15 {
		return Layer{}, fmt.Errorf("tilemap layer too short (%d int32s)", len(data))
	}

	ver := data[3]
	if ver < 2 {
		return Layer{}, fmt.Errorf("unsupported tilemap version %d", ver)
	}

	width := int(data[4])
	height := int(data[5])
	tileFlags := uint32(data[6])

	if width <= 0 || height <= 0 {
		return Layer{}, fmt.Errorf("invalid dimensions %dx%d", width, height)
	}

	detail := layerFlags&1 != 0

	l := Layer{
		Width:          width,
		Height:         height,
		Detail:         detail,
		ColorR:         uint8(data[7]),
		ColorG:         uint8(data[8]),
		ColorB:         uint8(data[9]),
		ColorA:         uint8(data[10]),
		ColorEnv:       data[11],
		ColorEnvOffset: data[12],
		ImageID:        int(data[13]),
	}

	// Determine kind from tile flags
	switch {
	case tileFlags&tileLayerFlagGame != 0:
		l.Kind = LayerKindGame
	case tileFlags&tileLayerFlagFront != 0:
		l.Kind = LayerKindFront
	case tileFlags&tileLayerFlagTele != 0:
		l.Kind = LayerKindTele
	case tileFlags&tileLayerFlagSpeedup != 0:
		l.Kind = LayerKindSpeedup
	case tileFlags&tileLayerFlagSwitch != 0:
		l.Kind = LayerKindSwitch
	case tileFlags&tileLayerFlagTune != 0:
		l.Kind = LayerKindTune
	default:
		l.Kind = LayerKindTiles
	}

	// v3: name
	if ver >= 3 && len(data) >= 18 {
		l.Name = parseI32String(data[15:18])
	}

	totalTiles := width * height

	// Read base tile data (data[14]) for game/tiles/front layers and as
	// vanilla-compat dummy for DDNet physics layers.
	readBaseTiles := func() {
		rawIdx := data[14]
		if rawIdx < 0 || int(rawIdx) >= df.numData {
			return
		}
		tileData, err := df.readData(int(rawIdx))
		if err != nil {
			return
		}
		l.Tiles = decodeTiles(tileData, totalTiles, l.Kind, ver)
	}

	switch l.Kind {
	case LayerKindTiles, LayerKindGame:
		readBaseTiles()
	case LayerKindFront:
		// Front layer: actual tile data is in the DDNet front data index
		frontIdx := specialDataOffset(ver, tileFlags)
		if frontIdx >= 0 && frontIdx < len(data) {
			rawIdx := data[frontIdx]
			if rawIdx >= 0 && int(rawIdx) < df.numData {
				tileData, err := df.readData(int(rawIdx))
				if err == nil {
					l.Tiles = decodeTiles(tileData, totalTiles, l.Kind, ver)
				}
			}
		}
	case LayerKindTele:
		teleIdx := specialDataOffset(ver, tileFlags)
		if teleIdx >= 0 && teleIdx < len(data) {
			rawIdx := data[teleIdx]
			if rawIdx >= 0 && int(rawIdx) < df.numData {
				tileData, err := df.readData(int(rawIdx))
				if err == nil {
					l.TeleTiles = decodeTeleTiles(tileData, totalTiles)
				}
			}
		}
	case LayerKindSpeedup:
		speedupIdx := specialDataOffset(ver, tileFlags)
		if speedupIdx >= 0 && speedupIdx < len(data) {
			rawIdx := data[speedupIdx]
			if rawIdx >= 0 && int(rawIdx) < df.numData {
				tileData, err := df.readData(int(rawIdx))
				if err == nil {
					l.SpeedupTiles = decodeSpeedupTiles(tileData, totalTiles)
				}
			}
		}
	case LayerKindSwitch:
		switchIdx := specialDataOffset(ver, tileFlags)
		if switchIdx >= 0 && switchIdx < len(data) {
			rawIdx := data[switchIdx]
			if rawIdx >= 0 && int(rawIdx) < df.numData {
				tileData, err := df.readData(int(rawIdx))
				if err == nil {
					l.SwitchTiles = decodeSwitchTiles(tileData, totalTiles)
				}
			}
		}
	case LayerKindTune:
		tuneIdx := specialDataOffset(ver, tileFlags)
		if tuneIdx >= 0 && tuneIdx < len(data) {
			rawIdx := data[tuneIdx]
			if rawIdx >= 0 && int(rawIdx) < df.numData {
				tileData, err := df.readData(int(rawIdx))
				if err == nil {
					l.TuneTiles = decodeTuneTiles(tileData, totalTiles)
				}
			}
		}
	}

	return l, nil
}

// specialDataOffset returns the item data offset for a special layer's data index.
func specialDataOffset(version int32, tileFlags uint32) int {
	var base int
	switch {
	case version >= 3:
		base = 18 // after name[3] at [15,16,17]
	case version >= 2:
		base = 15 // no name
	default:
		return -1
	}
	switch {
	case tileFlags&tileLayerFlagTele != 0:
		return base + 0
	case tileFlags&tileLayerFlagSpeedup != 0:
		return base + 1
	case tileFlags&tileLayerFlagFront != 0:
		return base + 2
	case tileFlags&tileLayerFlagSwitch != 0:
		return base + 3
	case tileFlags&tileLayerFlagTune != 0:
		return base + 4
	default:
		return 14
	}
}

// decodeTiles converts raw decompressed tile bytes into Tile structs.
// For game/tiles layers with version >= 4, data may be TW-compressed.
func decodeTiles(data []byte, expectedCount int, kind LayerKind, version int32) []Tile {
	// TW 0.7 compression: version >= 4, game/tiles layers only
	if version >= 4 && (kind == LayerKindGame || kind == LayerKindTiles) {
		data = decompressTWTiles(data)
	}

	// Standard tiles: 4 bytes per tile (id, flags, skip, unused)
	if len(data) < expectedCount*4 {
		return nil
	}
	tiles := make([]Tile, expectedCount)
	for i := range expectedCount {
		off := i * 4
		tiles[i] = Tile{
			ID:    data[off],
			Flags: data[off+1],
		}
	}
	return tiles
}

// decompressTWTiles implements Teeworlds' custom tile compression.
// Format: each 4-byte entry is [id, flags, skip_count, unused].
// "skip_count" means repeat the tile that many additional times.
func decompressTWTiles(data []byte) []byte {
	if len(data)%4 != 0 {
		return data
	}
	var out []byte
	for i := 0; i+3 < len(data); i += 4 {
		id := data[i]
		flags := data[i+1]
		skip := int(data[i+2])
		// write the tile once
		out = append(out, id, flags, 0, data[i+3])
		// then repeat it "skip" more times
		for range skip {
			out = append(out, id, flags, 0, data[i+3])
		}
	}
	return out
}

// decodeTeleTiles decodes 2-byte TeleTile entries: [number, id].
func decodeTeleTiles(data []byte, expectedCount int) []TeleTile {
	const size = 2
	if len(data) < expectedCount*size {
		return nil
	}
	tiles := make([]TeleTile, expectedCount)
	for i := range expectedCount {
		off := i * size
		tiles[i] = TeleTile{
			Number: data[off],
			ID:     data[off+1],
		}
	}
	return tiles
}

// decodeSpeedupTiles decodes 6-byte SpeedupTile entries:
// [force, max_speed, id, padding, angle_lo, angle_hi].
func decodeSpeedupTiles(data []byte, expectedCount int) []SpeedupTile {
	const size = 6
	if len(data) < expectedCount*size {
		return nil
	}
	tiles := make([]SpeedupTile, expectedCount)
	for i := range expectedCount {
		off := i * size
		tiles[i] = SpeedupTile{
			Force:    data[off],
			MaxSpeed: data[off+1],
			ID:       data[off+2],
			Angle:    int16(binary.LittleEndian.Uint16(data[off+4 : off+6])),
		}
	}
	return tiles
}

// decodeSwitchTiles decodes 4-byte SwitchTile entries: [number, id, flags, delay].
func decodeSwitchTiles(data []byte, expectedCount int) []SwitchTile {
	const size = 4
	if len(data) < expectedCount*size {
		return nil
	}
	tiles := make([]SwitchTile, expectedCount)
	for i := range expectedCount {
		off := i * size
		tiles[i] = SwitchTile{
			Number: data[off],
			ID:     data[off+1],
			Flags:  data[off+2],
			Delay:  data[off+3],
		}
	}
	return tiles
}

// decodeTuneTiles decodes 2-byte TuneTile entries: [number, id].
func decodeTuneTiles(data []byte, expectedCount int) []TuneTile {
	const size = 2
	if len(data) < expectedCount*size {
		return nil
	}
	tiles := make([]TuneTile, expectedCount)
	for i := range expectedCount {
		off := i * size
		tiles[i] = TuneTile{
			Number: data[off],
			ID:     data[off+1],
		}
	}
	return tiles
}

// parseSoundsLayer parses a sounds layer (DDNet only).
// legacy=true for the deprecated sound source format (type 9).
func parseSoundsLayer(df *datafile, data []int32, legacy bool) (Layer, error) {
	// data[0]=layerVersion, [1]=type, [2]=flags
	// data[3]=soundsVersion, [4]=numSources, [5]=dataIdx, [6]=soundIdx, [7..9]=name
	if len(data) < 7 {
		return Layer{}, fmt.Errorf("sounds layer too short (%d int32s)", len(data))
	}

	numSources := int(data[4])
	if numSources < 0 {
		return Layer{}, fmt.Errorf("invalid num_sources %d", numSources)
	}

	dataIdx := data[5]
	soundIdx := int(data[6])
	detail := uint32(data[2])&1 != 0

	l := Layer{
		Kind:    LayerKindSounds,
		Detail:  detail,
		SoundID: soundIdx,
	}

	// name
	if len(data) >= 10 {
		l.Name = parseI32String(data[7:10])
	}

	if dataIdx >= 0 && int(dataIdx) < df.numData && numSources > 0 {
		srcData, err := df.readData(int(dataIdx))
		if err == nil {
			if legacy {
				l.SoundSources = decodeSoundSourcesLegacy(srcData, numSources)
			} else {
				l.SoundSources = decodeSoundSources(srcData, numSources)
			}
		}
	}

	return l, nil
}

const soundSourceSize = 52       // 13 int32s
const soundSourceLegacySize = 36 // 9 int32s

func decodeSoundSources(data []byte, numSources int) []SoundSource {
	if len(data) < numSources*soundSourceSize {
		return nil
	}
	sources := make([]SoundSource, numSources)
	for i := range numSources {
		off := i * soundSourceSize
		sources[i] = SoundSource{
			Position:       readPoint(data[off:]),
			Loop:           readI32(data[off+8:]) != 0,
			Panning:        readI32(data[off+12:]) != 0,
			Delay:          readI32(data[off+16:]),
			Falloff:        uint8(readI32(data[off+20:])),
			PosEnv:         readI32(data[off+24:]),
			PosEnvOffset:   readI32(data[off+28:]),
			SoundEnv:       readI32(data[off+32:]),
			SoundEnvOffset: readI32(data[off+36:]),
			ShapeType:      readI32(data[off+40:]),
			ShapeWidth:     readI32(data[off+44:]),
			ShapeHeight:    readI32(data[off+48:]),
		}
	}
	return sources
}

func decodeSoundSourcesLegacy(data []byte, numSources int) []SoundSource {
	if len(data) < numSources*soundSourceLegacySize {
		return nil
	}
	sources := make([]SoundSource, numSources)
	for i := range numSources {
		off := i * soundSourceLegacySize
		radius := readI32(data[off+16:])
		sources[i] = SoundSource{
			Position:       readPoint(data[off:]),
			Loop:           readI32(data[off+8:]) != 0,
			Panning:        true,
			Delay:          readI32(data[off+12:]),
			Falloff:        0,
			PosEnv:         readI32(data[off+20:]),
			PosEnvOffset:   readI32(data[off+24:]),
			SoundEnv:       readI32(data[off+28:]),
			SoundEnvOffset: readI32(data[off+32:]),
			ShapeType:      1, // circle
			ShapeWidth:     radius,
			ShapeHeight:    0,
		}
	}
	return sources
}

func readI32(data []byte) int32 {
	return int32(binary.LittleEndian.Uint32(data[0:4]))
}

// ── envelope parsing ─────────────────────────────────────────────────────────

func parseEnvelopes(df *datafile) ([]Envelope, error) {
	envItems := df.itemsOfType(mapItemTypeEnvelope)
	if len(envItems) == 0 {
		return nil, nil
	}

	// Check if any envelope has version >= 3 (TW 0.7 bezier).
	// This changes the size of envelope points from 6 to 22 int32s.
	hasBezier := false
	for _, item := range envItems {
		if len(item.Data) >= 1 && item.Data[0] >= 3 {
			hasBezier = true
			break
		}
	}

	// Parse all envelope points
	allPoints, err := parseEnvPoints(df, hasBezier)
	if err != nil {
		return nil, err
	}

	envelopes := make([]Envelope, 0, len(envItems))
	for i, item := range envItems {
		if len(item.Data) < 5 {
			return nil, fmt.Errorf("envelope %d: too short (%d int32s)", i, len(item.Data))
		}
		ver := item.Data[0]
		channels := item.Data[1]
		startPoint := int(item.Data[2])
		numPoints := int(item.Data[3])

		env := Envelope{
			Channels: channels,
		}

		// name (8 int32s, starting at index 4)
		if len(item.Data) >= 12 {
			env.Name = parseI32String(item.Data[4:12])
		}

		// v2: synchronized flag
		if ver >= 2 && len(item.Data) >= 13 {
			env.Synchronized = item.Data[12] != 0
		}

		// Extract this envelope's points from the global array
		if startPoint >= 0 && numPoints > 0 && startPoint+numPoints <= len(allPoints) {
			env.Points = allPoints[startPoint : startPoint+numPoints]
		}

		envelopes = append(envelopes, env)
	}
	return envelopes, nil
}

func parseEnvPoints(df *datafile, hasBezier bool) ([]EnvPoint, error) {
	pointItem := df.findItem(mapItemTypeEnvPoints, 0)
	if pointItem == nil {
		return nil, nil
	}

	pointSize := 6 // standard: time, curvetype, values[4]
	if hasBezier {
		pointSize = 22 // 6 + 16 bezier tangent values
	}

	numPoints := len(pointItem.Data) / pointSize
	if numPoints == 0 {
		return nil, nil
	}

	points := make([]EnvPoint, numPoints)
	for i := range numPoints {
		base := i * pointSize
		if base+6 > len(pointItem.Data) {
			break
		}
		points[i] = EnvPoint{
			Time:      pointItem.Data[base],
			CurveType: CurveType(pointItem.Data[base+1]),
			Values: [4]int32{
				pointItem.Data[base+2],
				pointItem.Data[base+3],
				pointItem.Data[base+4],
				pointItem.Data[base+5],
			},
		}
	}
	return points, nil
}

// ── sound item parsing (DDNet only) ──────────────────────────────────────────

func parseSounds(df *datafile) ([]Sound, error) {
	items := df.itemsOfType(mapItemTypeSound)
	if len(items) == 0 {
		return nil, nil
	}

	sounds := make([]Sound, 0, len(items))
	for i, item := range items {
		if len(item.Data) < 5 {
			return nil, fmt.Errorf("sound %d: too short (%d int32s)", i, len(item.Data))
		}
		// item.Data[0]=version, [1]=external, [2]=nameIdx, [3]=dataIdx, [4]=dataSize
		nameIdx := item.Data[2]
		dataIdx := item.Data[3]

		name := ""
		if nameIdx >= 0 && int(nameIdx) < df.numData {
			nameData, err := df.readData(int(nameIdx))
			if err == nil && len(nameData) > 0 {
				if nameData[len(nameData)-1] == 0 {
					nameData = nameData[:len(nameData)-1]
				}
				name = string(nameData)
			}
		}

		s := Sound{Name: name}
		if dataIdx >= 0 && int(dataIdx) < df.numData {
			sndData, err := df.readData(int(dataIdx))
			if err == nil {
				s.Data = sndData
			}
		}
		sounds = append(sounds, s)
	}
	return sounds, nil
}

// parseI32String decodes a Teeworlds "3 x int32" encoded string (big-endian, XOR 128).
func parseI32String(nums []int32) string {
	buf := make([]byte, 0, len(nums)*4)
	for _, n := range nums {
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(n))
		buf = append(buf, b[:]...)
	}
	// All zeros = empty
	allZero := true
	for _, b := range buf {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return ""
	}
	// Strip guaranteed trailing NUL
	if len(buf) > 0 && buf[len(buf)-1] == 0 {
		buf = buf[:len(buf)-1]
	}
	// XOR decode
	for i := range buf {
		buf[i] = byte(int(buf[i]) + 128)
	}
	// Strip trailing NULs
	for len(buf) > 0 && buf[len(buf)-1] == 0 {
		buf = buf[:len(buf)-1]
	}
	return string(buf)
}

func parseQuadsLayer(df *datafile, data []int32) (Layer, error) {
	// data[0]=layerVersion, [1]=type, [2]=flags
	// data[3]=quadsVersion, [4]=numQuads, [5]=dataIdx, [6]=imageIdx
	if len(data) < 7 {
		return Layer{}, fmt.Errorf("quads layer too short (%d int32s)", len(data))
	}

	ver := data[3]
	if ver < 1 {
		return Layer{}, fmt.Errorf("invalid quads version %d", ver)
	}

	numQuads := int(data[4])
	if numQuads < 0 {
		return Layer{}, fmt.Errorf("invalid num_quads %d", numQuads)
	}

	dataIdx := data[5]
	imageIdx := int(data[6])
	detail := uint32(data[2])&1 != 0

	l := Layer{
		Kind:        LayerKindQuads,
		Detail:      detail,
		QuadImageID: imageIdx,
	}

	// v2: name
	if ver >= 2 && len(data) >= 10 {
		l.Name = parseI32String(data[7:10])
	}

	// Decode quads
	if dataIdx >= 0 && int(dataIdx) < df.numData && numQuads > 0 {
		quadData, err := df.readData(int(dataIdx))
		if err == nil {
			l.Quads = decodeQuads(quadData, numQuads)
		}
	}

	return l, nil
}

const quadBinarySize = 152 // 5 positions * 8 + 4 colors * 4 + 4 texcoords * 8 + 4 env fields * 4

func decodeQuads(data []byte, numQuads int) []Quad {
	// Binary quad: 5 points (2*i32 each) = 40 bytes
	//   + 4 colors (4*i32 each) = 64 bytes
	//   + 4 tex coords (2*i32 each) = 32 bytes
	//   + position_env + position_env_offset + color_env + color_env_offset = 16 bytes
	//   = 152 bytes total
	const (
		pointSize = 8  // 2 * int32
		colorSize = 16 // 4 * int32
	)

	if len(data) < numQuads*quadBinarySize {
		return nil
	}

	quads := make([]Quad, numQuads)
	for qi := range numQuads {
		base := qi * quadBinarySize
		off := base

		var q Quad

		// 5 corners (4 corners + 1 position)
		for i := range 5 {
			q.Points[i] = readPoint(data[off:])
			off += pointSize
		}

		// 4 colors
		for i := range 4 {
			q.Colors[i] = readColor(data[off:])
			off += colorSize
		}

		// 4 texture coordinates
		for i := range 4 {
			q.TexCoords[i] = readTexCoord(data[off:])
			off += pointSize
		}

		// envelope references (4 int32s)
		q.PosEnv = readI32(data[off:])
		q.PosEnvOffset = readI32(data[off+4:])
		q.ColorEnv = readI32(data[off+8:])
		q.ColorEnvOffset = readI32(data[off+12:])

		quads[qi] = q
	}
	return quads
}

// readPoint reads a 22.10 fixed-point 2D point, returning tile coordinates
// (world coords / 32). This is equivalent to dividing the raw int32 by 32768.
func readPoint(data []byte) Point {
	x := int32(binary.LittleEndian.Uint32(data[0:4]))
	y := int32(binary.LittleEndian.Uint32(data[4:8]))
	return Point{
		X: float64(x) / 32768.0,
		Y: float64(y) / 32768.0,
	}
}

// readTexCoord reads a 22.10 fixed-point texture coordinate, returning
// normalized [0,1] values (standard fx2f: raw / 1024).
func readTexCoord(data []byte) Point {
	x := int32(binary.LittleEndian.Uint32(data[0:4]))
	y := int32(binary.LittleEndian.Uint32(data[4:8]))
	return Point{
		X: float64(x) / 1024.0,
		Y: float64(y) / 1024.0,
	}
}

// readColor reads an RGBA color from 4 consecutive int32s.
func readColor(data []byte) color.NRGBA {
	r := int32(binary.LittleEndian.Uint32(data[0:4]))
	g := int32(binary.LittleEndian.Uint32(data[4:8]))
	b := int32(binary.LittleEndian.Uint32(data[8:12]))
	a := int32(binary.LittleEndian.Uint32(data[12:16]))
	return color.NRGBA{
		R: clampByte(r),
		G: clampByte(g),
		B: clampByte(b),
		A: clampByte(a),
	}
}

func clampByte(v int32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
