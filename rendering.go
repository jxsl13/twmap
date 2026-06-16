package twmap

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"strconv"
)

// DDNet editor checkerboard background colors.
const (
	checkerLight = 186 // RGB(186,186,186)
	checkerDark  = 153 // RGB(153,153,153)
)

// tilesetGridSize is the number of tiles per row/column in a tileset image (16×16 grid).
const tilesetGridSize = 16

// MapBounds represents the axis-aligned bounding box of non-air tiles
// in world tile coordinates.
type MapBounds struct {
	MinX, MinY, MaxX, MaxY int
}

// Width returns the width of the bounding box in tiles.
func (b MapBounds) Width() int { return b.MaxX - b.MinX }

// Height returns the height of the bounding box in tiles.
func (b MapBounds) Height() int { return b.MaxY - b.MinY }

// Bounds computes the bounding box of all non-air tiles across renderable
// layers (groups with parallax 100/100, excluding physics and detail layers).
func (m *Map) Bounds() MapBounds {
	steps := collectRenderSteps(m, nil)
	if len(steps) == 0 {
		return MapBounds{}
	}
	tileLayers := extractTileLayers(steps)
	crop := cropToNonAir(tileLayers)
	if crop.MaxX <= crop.MinX || crop.MaxY <= crop.MinY {
		return MapBounds{}
	}
	return crop
}

// RenderOption configures the rendering process.
type RenderOption func(*renderOptions)

type renderOptions struct {
	maxWidth        int        // 0 = use native resolution
	maxHeight       int        // 0 = use native resolution
	region          *MapBounds // nil = full non-air bounding box
	detail          bool       // include detail layers
	disabledBase    map[LayerKind]struct{}
	entities        bool      // render entity icons (pickups, flags, spawns)
	particles       bool      // render static particle/capability markers
	gameLayer       bool      // render game layer tiles (entities overlay)
	frontLayer      bool      // render front layer tiles (entities overlay)
	teleLayer       bool      // render tele layer tiles
	speedupLayer    bool      // render speedup layer tiles
	switchLayer     bool      // render switch layer tiles
	tuneLayer       bool      // render tune layer tiles
	invalidTiles    bool      // render problematic special-layer state similar to DDNet editor diagnostics
	overlayEntities int       // 0-100, analogous to cl_overlay_entities; controls opacity of entity overlay and design layers
	viewport        *viewport // nil = skip parallax groups
	parseOptions    []ParseOption
}

type groupClipRect struct {
	x int32
	y int32
	w int32
	h int32
}

// viewport defines the camera center for parallax rendering, in game-pixel coordinates.
type viewport struct {
	centerX, centerY float64 // tile coordinates
}

// WithMaxSize sets the maximum output image dimensions. The rendered image
// is scaled to fit within maxWidth × maxHeight while preserving aspect ratio.
// If not specified, the output uses the native tileset resolution
// (one pixel per texel as defined by the tileset images in the map).
func WithMaxSize(maxWidth, maxHeight int) RenderOption {
	return func(o *renderOptions) {
		o.maxWidth = maxWidth
		o.maxHeight = maxHeight
	}
}

// WithRegion restricts rendering to the specified tile region.
// Coordinates are in world tile units, matching the values returned by
// [Map.Bounds]. Areas outside the region are not rendered.
func WithRegion(region MapBounds) RenderOption {
	return func(o *renderOptions) {
		r := region
		o.region = &r
	}
}

// WithParseOptions passes [ParseOption] values to the map parser.
// Only used with [Render]; ignored by [RenderMap].
func WithParseOptions(opts ...ParseOption) RenderOption {
	return func(o *renderOptions) {
		o.parseOptions = append(o.parseOptions, opts...)
	}
}

// WithDetail enables rendering of detail layers, which are normally
// excluded. In the DDNet client these layers are only shown when the
// "High Detail" setting is active.
func WithDetail(detail bool) RenderOption {
	return func(o *renderOptions) {
		o.detail = detail
	}
}

// WithoutBaseLayerKinds disables layer kinds from the normal base-design pass.
// This is useful for selectively omitting quads or other regular layers while
// keeping the rest of the renderer active.
func WithoutBaseLayerKinds(kinds ...LayerKind) RenderOption {
	return func(o *renderOptions) {
		if o.disabledBase == nil {
			o.disabledBase = make(map[LayerKind]struct{}, len(kinds))
		}
		for _, kind := range kinds {
			o.disabledBase[kind] = struct{}{}
		}
	}
}

// WithEntities enables rendering of game-layer entity icons (pickups,
// flags, and DDNet weapon-removal pickups). When a game skin is registered (via
// [RegisterGameSkin] or by importing the gameskin package), the actual
// sprite images from the skin are drawn.
//
// To use a custom game skin, call [RegisterGameSkin] with your own
// 1024×512 image following the DDNet game.png layout.
//
// Default game skin:
//
//	import _ "github.com/jxsl13/twmap/external/gameskin"
func WithEntities(entities bool) RenderOption {
	return func(o *renderOptions) {
		o.entities = entities
	}
}

// WithGameLayer enables rendering of the game layer tiles only as an overlay.
// This makes invisible tiles (solid, hookable, freeze, spawns,
// checkpoints, etc.) visible.
//
// This is not the full DDNet editor entity-overlay semantics by itself,
// because the DDNet editor shows all enabled physics/entity layers together
// (game, front, tele, speedup, switch, tune). For that combined behavior,
// use [WithOverlayEntities].
//
// Requires the entities sprite sheet to be registered, which is done by
// importing the external/entities package (included in external):
//
//	import _ "github.com/jxsl13/twmap/external/entities"
func WithGameLayer(gameLayer bool) RenderOption {
	return func(o *renderOptions) {
		o.gameLayer = gameLayer
	}
}

// WithFrontLayer enables rendering of the DDNet front layer as a
// semi-transparent entities overlay.
func WithFrontLayer(frontLayer bool) RenderOption {
	return func(o *renderOptions) {
		o.frontLayer = frontLayer
	}
}

// WithTeleLayer enables rendering of the DDNet tele layer.
func WithTeleLayer(teleLayer bool) RenderOption {
	return func(o *renderOptions) {
		o.teleLayer = teleLayer
	}
}

// WithSpeedupLayer enables rendering of the DDNet speedup layer.
//
// The speedup arrow itself requires the speedup-arrow asset to be registered,
// which is done by importing:
//
//	import _ "github.com/jxsl13/twmap/external/speeduparrow"
func WithSpeedupLayer(speedupLayer bool) RenderOption {
	return func(o *renderOptions) {
		o.speedupLayer = speedupLayer
	}
}

// WithSwitchLayer enables rendering of the DDNet switch layer.
func WithSwitchLayer(switchLayer bool) RenderOption {
	return func(o *renderOptions) {
		o.switchLayer = switchLayer
	}
}

// WithTuneLayer enables rendering of the DDNet tune layer.
func WithTuneLayer(tuneLayer bool) RenderOption {
	return func(o *renderOptions) {
		o.tuneLayer = tuneLayer
	}
}

// WithOverlayEntities sets the entity overlay value (0–100), matching the
// DDNet editor's cl_overlay_entities setting.
//
// When val > 0:
//   - All DDNet physics/entity overlay layers (game, front, tele, speedup,
//     switch, tune) are enabled automatically.
//   - Both design layers and entity overlay tiles use alpha (100−val)/100,
//     exactly mirroring CRenderLayerTile::GetRenderColor in DDNet.
//     At val=50 both are at 50 % opacity; at val=100 the design layers are
//     skipped entirely and the entity overlay tiles are fully transparent
//     (checkerboard visible), matching DDNet's DoRender behavior.
//
// Requires the entities sprite sheet:
//
//	import _ "github.com/jxsl13/twmap/external/entities"
func WithOverlayEntities(val int) RenderOption {
	return func(o *renderOptions) {
		if val < 0 {
			val = 0
		}
		if val > 100 {
			val = 100
		}
		o.overlayEntities = val
		if val > 0 {
			o.gameLayer = true
			o.frontLayer = true
			o.teleLayer = true
			o.speedupLayer = true
			o.switchLayer = true
			o.tuneLayer = true
		}
	}
}

// WithParticles enables a static (non-animated) particle/capability pass.
// The sprites are sourced from the registered particles image.
func WithParticles(particles bool) RenderOption {
	return func(o *renderOptions) {
		o.particles = particles
	}
}

// WithInvalidTiles enables DDNet-editor-style diagnostics for problematic
// special-layer state where possible. This is mainly useful for broken or
// partially edited maps whose special-layer metadata should remain visible.
func WithInvalidTiles(invalid bool) RenderOption {
	return func(o *renderOptions) {
		o.invalidTiles = invalid
	}
}

// WithCameraAt sets the camera center to the middle of the given tile
// and enables rendering of groups with parallax other than 100/100.
// This is a convenience wrapper around [WithCamera] that centers on the tile.
//
// Without a camera, only groups with parallax 100/100 are rendered.
func WithCameraAt(tileX, tileY int) RenderOption {
	return WithCamera(float64(tileX)+0.5, float64(tileY)+0.5)
}

// WithCamera sets the camera center (in tile coordinates) and enables
// rendering of groups with parallax other than 100/100.  Each group's
// effective offset is computed using DDNet's world mapping projection:
//
//	effective = camera * (1 - parallax/100) - group_offset
//
// Without a camera, only groups with parallax 100/100 are rendered.
func WithCamera(x, y float64) RenderOption {
	return func(o *renderOptions) {
		o.viewport = &viewport{centerX: x, centerY: y}
	}
}

// Render parses a map from r and renders it as an image.
//
// By default the full non-air bounding box is rendered at the native
// tileset resolution. Use [WithMaxSize] to constrain the output size
// and [WithRegion] to render a sub-section of the map.
//
// The rendering approach:
//   - Tile and quad layers from groups with parallax 100/100 and offset 0/0
//   - The image is cropped to the bounding box of non-air tiles
//   - Tile flags (vflip, hflip, rotate) are handled
//   - Layer colors modulate the tileset/texture pixels
//   - Quads are rasterized with barycentric interpolation (vertex colors + textures)
//   - Physics/special layers and detail layers are excluded by default
//     (can be enabled with dedicated render options)
func Render(r io.Reader, opts ...RenderOption) (*image.NRGBA, error) {
	var ro renderOptions
	for _, fn := range opts {
		fn(&ro)
	}
	m, err := Parse(r, ro.parseOptions...)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return renderMap(m, &ro)
}

// pixelsPerTile is the number of game-pixels per tile in the DDNet coordinate system.
const pixelsPerTile = 32

// renderLayer is a collected tile layer ready for rendering.
type renderLayer struct {
	color   color.NRGBA
	imageID int // -1 = no image (use white)
	tiles   []Tile
	width   int
	height  int
	offsetX float64 // group offset in tiles (float for sub-tile precision)
	offsetY float64 // group offset in tiles (float for sub-tile precision)
	clip    *groupClipRect
}

// renderQuadLayer is a collected quad layer ready for rendering.
type renderQuadLayer struct {
	quads    []Quad
	imageID  int     // -1 = no image (vertex colors only)
	offsetX  float64 // group offset in tiles (float for sub-tile precision)
	offsetY  float64 // group offset in tiles
	alphaMul float64 // vertex-alpha multiplier from overlayEntities (0 means 1.0 = no change)
	clip     *groupClipRect
}

// renderStep represents an ordered rendering operation (either tile or quad layer).
// Layers are rendered back-to-front in the order they appear in the map.
type renderStep struct {
	isTile bool
	tile   renderLayer
	quad   renderQuadLayer
}

// overlayRenderLayer represents one DDNet physics/entity overlay layer in the
// final overlay pass. Most layers use the generic tile renderer; speedup keeps
// its source data for custom arrow rendering.
type overlayRenderLayer struct {
	kind         LayerKind
	layer        renderLayer
	teleTiles    []TeleTile
	speedupTiles []SpeedupTile
	switchTiles  []SwitchTile
	tuneTiles    []TuneTile
	showInvalid  bool
}

// RenderMap renders an already-parsed Map as an image.
//
// By default the full non-air bounding box is rendered at the native
// tileset resolution. Use [WithMaxSize] to constrain the output size
// and [WithRegion] to render a sub-section of the map.
func RenderMap(m *Map, opts ...RenderOption) (*image.NRGBA, error) {
	var ro renderOptions
	for _, fn := range opts {
		fn(&ro)
	}
	return renderMap(m, &ro)
}

// nativeTileLen returns the pixel-per-tile resolution derived from the largest
// tileset image referenced by the renderable tile layers. Falls back to 64.
func nativeTileLen(m *Map, layers []renderLayer) uint32 {
	maxSide := 0
	seen := map[int]bool{}
	for _, l := range layers {
		if l.imageID < 0 || l.imageID >= len(m.Images) || seen[l.imageID] {
			continue
		}
		seen[l.imageID] = true
		img := m.Images[l.imageID]
		side := max(img.Height, img.Width)
		if side > maxSide {
			maxSide = side
		}
	}
	if maxSide <= 0 {
		return 64
	}
	tl := max(uint32(maxSide)/tilesetGridSize, 1)
	return tl
}

// renderMap is the internal rendering pipeline shared by Render and RenderMap.
func renderMap(m *Map, ro *renderOptions) (*image.NRGBA, error) {
	// ── 1. Collect all renderable layers (tiles + quads) in order ────────
	steps := collectRenderSteps(m, ro)
	overlayLayers := collectOverlayRenderLayers(m, ro, false)
	if len(steps) == 0 && len(overlayLayers) == 0 && !ro.entities && !ro.particles {
		return image.NewNRGBA(image.Rect(0, 0, 1, 1)), nil
	}

	// ── 2. Determine crop region ─────────────────────────────────────────
	tileLayers := extractTileLayers(steps)
	boundsLayers := append([]renderLayer{}, tileLayers...)
	boundsLayers = append(boundsLayers, overlayLayersToBoundsLayers(collectOverlayRenderLayers(m, ro, true))...)
	crop := cropToNonAir(boundsLayers)
	if ro.region != nil {
		crop = *ro.region
	}
	if crop.MaxX <= crop.MinX || crop.MaxY <= crop.MinY {
		return image.NewNRGBA(image.Rect(0, 0, 1, 1)), nil
	}
	cropW := crop.MaxX - crop.MinX
	cropH := crop.MaxY - crop.MinY

	// ── 3. Determine tile resolution ─────────────────────────────────────
	var tileLen uint32
	useMaxSize := ro.maxWidth > 0 && ro.maxHeight > 0
	if useMaxSize {
		targetSize := uint32(math.Max(float64(ro.maxWidth), float64(ro.maxHeight)))
		tileLen = scaleTileLen(int32(cropW), int32(cropH), targetSize)
	} else {
		tileLen = nativeTileLen(m, tileLayers)
	}

	// ── 4. Prepare tilesets and quad images ───────────────────────────────
	tilesets := prepareTilesets(m, tileLayers, tileLen)
	quadImages := prepareQuadImages(m, steps)

	// ── 5. Render all layers onto intermediate image ─────────────────────
	imgW := cropW * int(tileLen)
	imgH := cropH * int(tileLen)
	canvas := image.NewNRGBA(image.Rect(0, 0, imgW, imgH))
	fillCheckerboard(canvas, tileLen)
	renderAllSteps(canvas, steps, tilesets, quadImages, &crop, tileLen)

	// ── 5b. Render static particle/capability markers behind overlays ────
	if ro.particles {
		renderParticles(canvas, m, ro, &crop, tileLen)
	}

	// ── 5c. Render entity icons (pickups, flags) below DDNet physics layers ──
	if ro.entities {
		renderEntities(canvas, m, ro, &crop, tileLen)
	}

	// ── 5d. Render selected DDNet physics/entity tile layers last ────────
	renderOverlayLayers(canvas, overlayLayers, &crop, tileLen)

	// ── 6. Scale to output bounding box (only when max size is set) ──────
	if useMaxSize {
		outW, outH := fitInBoundingBox(cropW, cropH, ro.maxWidth, ro.maxHeight)
		return scaleNRGBA(canvas, outW, outH), nil
	}
	return canvas, nil
}

// collectRenderSteps collects all renderable layers (tiles and quads) in
// back-to-front order.  By default only groups with parallax 100/100 are
// included and detail layers are skipped.  When a viewport is set, groups
// with other parallax values are also included and their offsets are computed
// using DDNet world mapping projection.  When detail is enabled, detail
// layers are included.
func collectRenderSteps(m *Map, ro *renderOptions) []renderStep {
	var steps []renderStep
	includeDetail := ro != nil && ro.detail

	// DDNet: design tile/quad layers use (100-EntityOverlayVal)/100 alpha.
	// At val=100 DDNet DoRender() returns false for design layers, so skip them.
	overlayVal := 0
	if ro != nil {
		overlayVal = ro.overlayEntities
	}
	if overlayVal == 100 {
		return steps // skip all design layers at full overlay
	}
	designAlphaMul := 1.0
	if overlayVal > 0 {
		designAlphaMul = float64(100-overlayVal) / 100.0
	}

	for i := range m.Groups {
		g := &m.Groups[i]
		clip := groupClipForRender(g)

		tileOffX, tileOffY, quadOffX, quadOffY, ok := computeGroupRenderOffsets(g, ro)
		if !ok {
			continue
		}

		for j := range g.Layers {
			l := &g.Layers[j]
			if l.IsPhysics() {
				continue
			}
			if baseLayerKindDisabled(ro, l.Kind) {
				continue
			}
			if l.Detail && !includeDetail {
				continue
			}

			switch l.Kind {
			case LayerKindTiles:
				if len(l.Tiles) == 0 {
					continue
				}
				layerA := l.ColorA
				if designAlphaMul != 1.0 {
					layerA = uint8(float64(l.ColorA) * designAlphaMul)
				}
				steps = append(steps, renderStep{
					isTile: true,
					tile: renderLayer{
						color: color.NRGBA{
							R: l.ColorR,
							G: l.ColorG,
							B: l.ColorB,
							A: layerA,
						},
						imageID: l.ImageID,
						tiles:   l.Tiles,
						width:   l.Width,
						height:  l.Height,
						offsetX: tileOffX,
						offsetY: tileOffY,
						clip:    clip,
					},
				})
			case LayerKindQuads:
				if len(l.Quads) == 0 {
					continue
				}
				steps = append(steps, renderStep{
					quad: renderQuadLayer{
						quads:    l.Quads,
						imageID:  l.QuadImageID,
						offsetX:  quadOffX,
						offsetY:  quadOffY,
						alphaMul: designAlphaMul,
						clip:     clip,
					},
				})
			}
		}
	}
	return steps
}

func baseLayerKindDisabled(ro *renderOptions, kind LayerKind) bool {
	if ro == nil || len(ro.disabledBase) == 0 {
		return false
	}
	_, disabled := ro.disabledBase[kind]
	return disabled
}

// computeGroupRenderOffsets converts a group's DDNet offset/parallax state
// into unified offsets used by the software renderer.
func computeGroupRenderOffsets(g *Group, ro *renderOptions) (tileOffX, tileOffY, quadOffX, quadOffY float64, ok bool) {
	hasViewport := ro != nil && ro.viewport != nil
	isParallax100 := g.ParallaxX == 100 && g.ParallaxY == 100
	if !isParallax100 && !hasViewport {
		return 0, 0, 0, 0, false
	}

	var effPixelX, effPixelY float64
	if hasViewport {
		camX := ro.viewport.centerX * float64(pixelsPerTile)
		camY := ro.viewport.centerY * float64(pixelsPerTile)
		effPixelX = camX*(1.0-float64(g.ParallaxX)/100.0) - float64(g.OffsetX)
		effPixelY = camY*(1.0-float64(g.ParallaxY)/100.0) - float64(g.OffsetY)
	} else {
		effPixelX = -float64(g.OffsetX)
		effPixelY = -float64(g.OffsetY)
	}

	tileOffX = effPixelX / float64(pixelsPerTile)
	tileOffY = effPixelY / float64(pixelsPerTile)
	quadOffX = effPixelX / float64(pixelsPerTile)
	quadOffY = effPixelY / float64(pixelsPerTile)
	return tileOffX, tileOffY, quadOffX, quadOffY, true
}

func groupClipForRender(g *Group) *groupClipRect {
	if g == nil || !g.Clipping || g.ClipW <= 0 || g.ClipH <= 0 {
		return nil
	}
	return &groupClipRect{x: g.ClipX, y: g.ClipY, w: g.ClipW, h: g.ClipH}
}

func renderClipBounds(clip *groupClipRect, crop *MapBounds, tileLen uint32, canvasBounds image.Rectangle) image.Rectangle {
	if clip == nil {
		return canvasBounds
	}
	pxPerGamePixel := float64(tileLen) / float64(pixelsPerTile)
	left := int(math.Floor((float64(clip.x) - float64(crop.MinX*pixelsPerTile)) * pxPerGamePixel))
	top := int(math.Floor((float64(clip.y) - float64(crop.MinY*pixelsPerTile)) * pxPerGamePixel))
	right := int(math.Ceil((float64(clip.x+clip.w) - float64(crop.MinX*pixelsPerTile)) * pxPerGamePixel))
	bottom := int(math.Ceil((float64(clip.y+clip.h) - float64(crop.MinY*pixelsPerTile)) * pxPerGamePixel))
	return image.Rect(left, top, right, bottom).Intersect(canvasBounds)
}

// collectOverlayRenderLayers collects DDNet physics/entity tile layers that are
// explicitly enabled via render options.
//
// When boundsOnly is true, the game layer is also included if entity or
// particle rendering is enabled, so bounds/cropping still work on maps without
// visual tile layers.
func collectOverlayRenderLayers(m *Map, ro *renderOptions, boundsOnly bool) []overlayRenderLayer {
	if m == nil || ro == nil {
		return nil
	}

	includeDetail := ro.detail
	includeGame := ro.gameLayer
	if boundsOnly && (ro.entities || ro.particles) {
		includeGame = true
	}
	includeFront := ro.frontLayer
	if boundsOnly && ro.particles {
		includeFront = true
	}
	includeTele := ro.teleLayer
	includeSpeedup := ro.speedupLayer
	includeSwitch := ro.switchLayer
	includeTune := ro.tuneLayer

	if !includeGame && !includeFront && !includeTele && !includeSpeedup && !includeSwitch && !includeTune {
		return nil
	}

	// DDNet: entity overlay tiles use the same (100-EntityOverlayVal)/100 alpha
	// as design layers (CRenderLayerEntityBase inherits CRenderLayerTile::GetRenderColor).
	overlayA := uint8(255)
	if ro.overlayEntities > 0 {
		overlayA = uint8(float64(255) * float64(100-ro.overlayEntities) / 100.0)
	}
	overlayColor := color.NRGBA{R: 255, G: 255, B: 255, A: overlayA}
	var out []overlayRenderLayer

	for i := range m.Groups {
		g := &m.Groups[i]
		clip := groupClipForRender(g)
		tileOffX, tileOffY, _, _, ok := computeGroupRenderOffsets(g, ro)
		if !ok {
			continue
		}

		for j := range g.Layers {
			l := &g.Layers[j]
			if l.Detail && !includeDetail {
				continue
			}

			switch l.Kind {
			case LayerKindGame:
				if !includeGame || len(l.Tiles) == 0 {
					continue
				}
				tiles := filterLayerTiles(l.Tiles, IsValidGameTile)
				if !hasNonAirTiles(tiles) {
					continue
				}
				out = append(out, overlayRenderLayer{kind: l.Kind, layer: renderLayer{
					color:   overlayColor,
					imageID: 0,
					tiles:   tiles,
					width:   l.Width,
					height:  l.Height,
					offsetX: tileOffX,
					offsetY: tileOffY,
					clip:    clip,
				}})
			case LayerKindFront:
				if !includeFront || len(l.Tiles) == 0 {
					continue
				}
				tiles := filterLayerTiles(l.Tiles, IsValidFrontTile)
				if !hasNonAirTiles(tiles) {
					continue
				}
				out = append(out, overlayRenderLayer{kind: l.Kind, layer: renderLayer{
					color:   overlayColor,
					imageID: 0,
					tiles:   tiles,
					width:   l.Width,
					height:  l.Height,
					offsetX: tileOffX,
					offsetY: tileOffY,
					clip:    clip,
				}})
			case LayerKindTele:
				if !includeTele || len(l.TeleTiles) == 0 {
					continue
				}
				tiles := convertTeleLayerTiles(l.TeleTiles)
				if !hasNonAirTiles(tiles) {
					continue
				}
				out = append(out, overlayRenderLayer{kind: l.Kind, layer: renderLayer{
					color:   overlayColor,
					imageID: 0,
					tiles:   tiles,
					width:   l.Width,
					height:  l.Height,
					offsetX: tileOffX,
					offsetY: tileOffY,
					clip:    clip,
				}, teleTiles: l.TeleTiles})
			case LayerKindSpeedup:
				if !includeSpeedup || len(l.SpeedupTiles) == 0 {
					continue
				}
				tiles := convertSpeedupLayerTiles(l.SpeedupTiles, ro.invalidTiles)
				if !hasNonAirTiles(tiles) {
					continue
				}
				out = append(out, overlayRenderLayer{
					kind: LayerKindSpeedup,
					layer: renderLayer{
						color:   overlayColor,
						imageID: -1,
						tiles:   tiles,
						width:   l.Width,
						height:  l.Height,
						offsetX: tileOffX,
						offsetY: tileOffY,
						clip:    clip,
					},
					speedupTiles: l.SpeedupTiles,
					showInvalid:  ro.invalidTiles,
				})
			case LayerKindSwitch:
				if !includeSwitch || len(l.SwitchTiles) == 0 {
					continue
				}
				tiles := convertSwitchLayerTiles(l.SwitchTiles)
				if !hasNonAirTiles(tiles) {
					continue
				}
				out = append(out, overlayRenderLayer{kind: l.Kind, layer: renderLayer{
					color:   overlayColor,
					imageID: 0,
					tiles:   tiles,
					width:   l.Width,
					height:  l.Height,
					offsetX: tileOffX,
					offsetY: tileOffY,
					clip:    clip,
				}, switchTiles: l.SwitchTiles})
			case LayerKindTune:
				if !includeTune || len(l.TuneTiles) == 0 {
					continue
				}
				tiles := convertTuneLayerTiles(l.TuneTiles)
				if !hasNonAirTiles(tiles) {
					continue
				}
				out = append(out, overlayRenderLayer{kind: l.Kind, layer: renderLayer{
					color:   overlayColor,
					imageID: 0,
					tiles:   tiles,
					width:   l.Width,
					height:  l.Height,
					offsetX: tileOffX,
					offsetY: tileOffY,
					clip:    clip,
				}, tuneTiles: l.TuneTiles})
			}
		}
	}

	return out
}

func overlayLayersToBoundsLayers(layers []overlayRenderLayer) []renderLayer {
	out := make([]renderLayer, 0, len(layers))
	for i := range layers {
		if hasNonAirTiles(layers[i].layer.tiles) {
			out = append(out, layers[i].layer)
		}
	}
	return out
}

func hasNonAirTiles(tiles []Tile) bool {
	for i := range tiles {
		if tiles[i].ID != TileAir {
			return true
		}
	}
	return false
}

func filterLayerTiles(src []Tile, keep func(uint8) bool) []Tile {
	out := make([]Tile, len(src))
	for i := range src {
		out[i] = src[i]
		if !keep(src[i].ID) {
			out[i] = Tile{}
		}
	}
	return out
}

func convertTeleLayerTiles(src []TeleTile) []Tile {
	out := make([]Tile, len(src))
	for i := range src {
		if src[i].ID == 0 {
			continue
		}
		out[i] = Tile{ID: src[i].ID}
	}
	return out
}

func convertTuneLayerTiles(src []TuneTile) []Tile {
	out := make([]Tile, len(src))
	for i := range src {
		if src[i].ID == 0 {
			continue
		}
		out[i] = Tile{ID: src[i].ID}
	}
	return out
}

func convertSwitchLayerTiles(src []SwitchTile) []Tile {
	out := make([]Tile, len(src))
	for i := range src {
		id := src[i].ID
		if id == 0 {
			continue
		}
		if id == TileSwitchTimedOpen {
			id = 8
		}
		flags := src[i].Flags
		if !IsSwitchTileFlagsUsed(src[i].ID) {
			flags = 0
		}
		out[i] = Tile{ID: id, Flags: flags}
	}
	return out
}

func convertSpeedupLayerTiles(src []SpeedupTile, showInvalid bool) []Tile {
	out := make([]Tile, len(src))
	for i := range src {
		id := src[i].ID
		if !shouldRenderSpeedupBase(src[i], showInvalid) {
			if !showInvalid {
				continue
			}
			if !speedupTileHasAnyData(src[i]) {
				continue
			}
			out[i] = Tile{ID: TileSpeedBoost}
			continue
		}
		out[i] = Tile{ID: id}
	}
	return out
}

func speedupTileHasAnyData(tile SpeedupTile) bool {
	return tile.ID != 0 || tile.Force != 0 || tile.MaxSpeed != 0 || tile.Angle != 0
}

func shouldRenderSpeedupBase(tile SpeedupTile, showInvalid bool) bool {
	if showInvalid {
		return IsValidSpeedupTile(tile.ID) && speedupTileHasAnyData(tile)
	}
	return (tile.Force != 0 && tile.ID == TileSpeedBoostOld) ||
		((tile.Force != 0 || tile.MaxSpeed != 0) && tile.ID == TileSpeedBoost)
}

// extractTileLayers returns only the tile renderLayers from the render steps.
func extractTileLayers(steps []renderStep) []renderLayer {
	var layers []renderLayer
	for _, s := range steps {
		if s.isTile {
			layers = append(layers, s.tile)
		}
	}
	return layers
}

// cropToNonAir computes the bounding box of all non-air tiles across all
// layers in world tile coordinates (layer position + group offset).
// Uses edge scanning: finds the first/last non-air row and column per layer
// instead of scanning every tile.
func cropToNonAir(layers []renderLayer) MapBounds {
	r := MapBounds{
		MinX: math.MaxInt,
		MinY: math.MaxInt,
		MaxX: math.MinInt,
		MaxY: math.MinInt,
	}
	for _, l := range layers {
		if len(l.tiles) == 0 {
			continue
		}
		w, h := l.width, l.height

		// Find minY: first row with a non-air tile
		var lminY, lmaxY, lminX, lmaxX int
		lminY = h // sentinel
		for y := range h {
			row := y * w
			for x := range w {
				if row+x < len(l.tiles) && l.tiles[row+x].ID != 0 {
					lminY = y
					goto foundMinY
				}
			}
		}
		continue // no non-air tiles in this layer
	foundMinY:

		// Find maxY: last row with a non-air tile
		lmaxY = lminY + 1
		for y := h - 1; y > lminY; y-- {
			row := y * w
			for x := range w {
				if row+x < len(l.tiles) && l.tiles[row+x].ID != 0 {
					lmaxY = y + 1
					goto foundMaxY
				}
			}
		}
	foundMaxY:

		// Find minX: first column with a non-air tile
		lminX = w
		for x := range w {
			for y := lminY; y < lmaxY; y++ {
				idx := y*w + x
				if idx < len(l.tiles) && l.tiles[idx].ID != 0 {
					lminX = x
					goto foundMinX
				}
			}
		}
	foundMinX:

		// Find maxX: last column with a non-air tile
		lmaxX = lminX + 1
		for x := w - 1; x > lminX; x-- {
			for y := lminY; y < lmaxY; y++ {
				idx := y*w + x
				if idx < len(l.tiles) && l.tiles[idx].ID != 0 {
					lmaxX = x + 1
					goto foundMaxX
				}
			}
		}
	foundMaxX:

		// Convert layer-local bounds to world tile coords via group offset.
		// Fractional offsets can make the layer overlap one extra tile at the
		// edges, so expand using floor/ceil like DDNet's screen mapping does.
		wMinX := int(math.Floor(float64(lminX) + l.offsetX))
		wMinY := int(math.Floor(float64(lminY) + l.offsetY))
		wMaxX := int(math.Ceil(float64(lmaxX) + l.offsetX))
		wMaxY := int(math.Ceil(float64(lmaxY) + l.offsetY))

		if l.clip != nil {
			clipMinX := int(math.Floor(float64(l.clip.x) / float64(pixelsPerTile)))
			clipMinY := int(math.Floor(float64(l.clip.y) / float64(pixelsPerTile)))
			clipMaxX := int(math.Ceil(float64(l.clip.x+l.clip.w) / float64(pixelsPerTile)))
			clipMaxY := int(math.Ceil(float64(l.clip.y+l.clip.h) / float64(pixelsPerTile)))
			if wMinX < clipMinX {
				wMinX = clipMinX
			}
			if wMinY < clipMinY {
				wMinY = clipMinY
			}
			if wMaxX > clipMaxX {
				wMaxX = clipMaxX
			}
			if wMaxY > clipMaxY {
				wMaxY = clipMaxY
			}
			if wMaxX <= wMinX || wMaxY <= wMinY {
				continue
			}
		}

		if wMinX < r.MinX {
			r.MinX = wMinX
		}
		if wMinY < r.MinY {
			r.MinY = wMinY
		}
		if wMaxX > r.MaxX {
			r.MaxX = wMaxX
		}
		if wMaxY > r.MaxY {
			r.MaxY = wMaxY
		}
	}
	return r
}

// scaleTileLen determines the pixel resolution per tile so that the intermediate
// image stays manageable. Starts at 64 and halves until the total pixel count
// fits within 2× the output area (down from 4×), trading minor quality for
// significantly less rendering and scaling work.
func scaleTileLen(cropW, cropH int32, targetSize uint32) uint32 {
	tileLen := uint32(64)
	for tileLen > 1 {
		pixels := uint64(tileLen) * uint64(tileLen) * uint64(cropW) * uint64(cropH)
		budget := uint64(2) * uint64(targetSize) * uint64(targetSize)
		if pixels <= budget {
			break
		}
		tileLen /= 2
	}
	return tileLen
}

// prepareTilesets scales each referenced tileset image to tileLen×16 square
// and clears the air tile (index 0). Returns a map from imageID → scaled NRGBA.
func prepareTilesets(m *Map, layers []renderLayer, tileLen uint32) map[int]*image.NRGBA {
	needed := map[int]bool{}
	for _, l := range layers {
		needed[l.imageID] = true
	}

	tilesets := make(map[int]*image.NRGBA, len(needed))
	resultSide := int(tileLen * tilesetGridSize)

	for imgID := range needed {
		if imgID < 0 || imgID >= len(m.Images) {
			tilesets[imgID] = newWhiteTileset(resultSide, tileLen)
			continue
		}

		src := m.Images[imgID]
		srcRGBA := src.RGBA
		if srcRGBA == nil && src.External {
			if m.Version == MapVersion07 {
				srcRGBA = resolveExternalImage07(src.Name)
			} else {
				srcRGBA = resolveExternalImage(src.Name)
			}
		}
		if srcRGBA == nil || src.Width == 0 || src.Height == 0 {
			tilesets[imgID] = newWhiteTileset(resultSide, tileLen)
			continue
		}

		// Scale tileset with the same bilinear resampler used for all other
		// scaled PNG inputs (entities, particles, final output, etc.).
		scaled := scaleImageRectNRGBA(srcRGBA, srcRGBA.Bounds(), resultSide, resultSide)

		// Clear air tile (top-left tile)
		clearAirTile(scaled, tileLen)
		tilesets[imgID] = scaled
	}

	return tilesets
}

// clearAirTile zeros the top-left tile (index 0) in a scaled tileset.
func clearAirTile(img *image.NRGBA, tileLen uint32) {
	tl := int(tileLen)
	for y := range tl {
		off := y * img.Stride
		for x := range tl {
			p := off + x*4
			img.Pix[p] = 0
			img.Pix[p+1] = 0
			img.Pix[p+2] = 0
			img.Pix[p+3] = 0
		}
	}
}

// newWhiteTileset creates a solid white tileset image of side×side pixels
// with the air tile (index 0) cleared.
func newWhiteTileset(side int, tileLen uint32) *image.NRGBA {
	white := image.NewNRGBA(image.Rect(0, 0, side, side))
	for i := 0; i < len(white.Pix); i += 4 {
		white.Pix[i] = 255
		white.Pix[i+1] = 255
		white.Pix[i+2] = 255
		white.Pix[i+3] = 255
	}
	clearAirTile(white, tileLen)
	return white
}

// prepareQuadImages returns full-resolution images for use by quad layers.
func prepareQuadImages(m *Map, steps []renderStep) map[int]*image.NRGBA {
	needed := map[int]bool{}
	for _, s := range steps {
		if !s.isTile {
			needed[s.quad.imageID] = true
		}
	}

	images := make(map[int]*image.NRGBA, len(needed))
	for imgID := range needed {
		if imgID < 0 || imgID >= len(m.Images) {
			continue
		}
		src := m.Images[imgID]
		srcRGBA := src.RGBA
		if srcRGBA == nil && src.External {
			if m.Version == MapVersion07 {
				srcRGBA = resolveExternalImage07(src.Name)
			} else {
				srcRGBA = resolveExternalImage(src.Name)
			}
		}
		if srcRGBA == nil {
			continue
		}
		images[imgID] = srcRGBA
	}
	return images
}

// renderAllSteps composites all layers onto canvas in order, handling both
// tile layers and quad layers.
func renderAllSteps(
	canvas *image.NRGBA,
	steps []renderStep,
	tilesets map[int]*image.NRGBA,
	quadImages map[int]*image.NRGBA,
	crop *MapBounds,
	tileLen uint32,
) {
	for i := range steps {
		if steps[i].isTile {
			renderSingleTileLayer(canvas, &steps[i].tile, tilesets, crop, tileLen)
		} else {
			renderSingleQuadLayer(canvas, &steps[i].quad, quadImages, crop, tileLen)
		}
	}
}

// renderSingleTileLayer composites one tile layer onto canvas.
func renderSingleTileLayer(
	canvas *image.NRGBA,
	l *renderLayer,
	tilesets map[int]*image.NRGBA,
	crop *MapBounds,
	tileLen uint32,
) {
	tl := int(tileLen)
	canvasPix := canvas.Pix
	canvasStride := canvas.Stride

	tileset := tilesets[l.imageID]
	if tileset == nil {
		return
	}
	tsPix := tileset.Pix
	tsStride := tileset.Stride
	tsPixLen := len(tsPix)

	colorIsWhite := l.color.R == 255 && l.color.G == 255 && l.color.B == 255 && l.color.A == 255
	lcR := uint32(l.color.R)
	lcG := uint32(l.color.G)
	lcB := uint32(l.color.B)
	lcA := uint32(l.color.A)

	canvasW := canvas.Bounds().Dx()
	canvasH := canvas.Bounds().Dy()
	clipBounds := renderClipBounds(l.clip, crop, tileLen, canvas.Bounds())
	if clipBounds.Empty() {
		return
	}

	// Iterate over layer tiles that overlap the crop region.
	// Layer tile (lx,ly) maps to world tile (lx+offsetX, ly+offsetY).
	// Fractional group offsets can cause partial overlap at the crop edges, so
	// use floor/ceil instead of whole-tile truncation.
	startLayerY := int(math.Floor(float64(crop.MinY) - l.offsetY))
	endLayerY := int(math.Ceil(float64(crop.MaxY) - l.offsetY))
	startLayerX := int(math.Floor(float64(crop.MinX) - l.offsetX))
	endLayerX := int(math.Ceil(float64(crop.MaxX) - l.offsetX))
	if startLayerY < 0 {
		startLayerY = 0
	}
	if endLayerY > l.height {
		endLayerY = l.height
	}
	if startLayerX < 0 {
		startLayerX = 0
	}
	if endLayerX > l.width {
		endLayerX = l.width
	}

	for layerY := startLayerY; layerY < endLayerY; layerY++ {
		for layerX := startLayerX; layerX < endLayerX; layerX++ {
			idx := layerY*l.width + layerX
			if idx >= len(l.tiles) {
				continue
			}
			tile := l.tiles[idx]
			if tile.ID == 0 {
				continue
			}

			tileX := int(tile.ID) % tilesetGridSize
			tileY := int(tile.ID) / tilesetGridSize

			// Destination on canvas = world position minus crop origin.
			// Use rounded output-pixel positions so DDNet's sub-tile group offset
			// projection is preserved instead of being truncated to full tiles.
			baseDstY := int(math.Round((float64(layerY) + l.offsetY - float64(crop.MinY)) * float64(tl)))
			baseDstX := int(math.Round((float64(layerX) + l.offsetX - float64(crop.MinX)) * float64(tl)))
			baseSrcX := tileX * tl
			baseSrcY := tileY * tl

			srcMinX, srcMinY := 0, 0
			srcMaxX, srcMaxY := tl, tl
			if baseDstX < 0 {
				srcMinX = -baseDstX
				baseDstX = 0
			}
			if baseDstY < 0 {
				srcMinY = -baseDstY
				baseDstY = 0
			}
			if baseDstX+(srcMaxX-srcMinX) > canvasW {
				srcMaxX = srcMinX + canvasW - baseDstX
			}
			if baseDstY+(srcMaxY-srcMinY) > canvasH {
				srcMaxY = srcMinY + canvasH - baseDstY
			}
			if baseDstX < clipBounds.Min.X {
				shift := clipBounds.Min.X - baseDstX
				srcMinX += shift
				baseDstX = clipBounds.Min.X
			}
			if baseDstY < clipBounds.Min.Y {
				shift := clipBounds.Min.Y - baseDstY
				srcMinY += shift
				baseDstY = clipBounds.Min.Y
			}
			if baseDstX+(srcMaxX-srcMinX) > clipBounds.Max.X {
				srcMaxX = srcMinX + clipBounds.Max.X - baseDstX
			}
			if baseDstY+(srcMaxY-srcMinY) > clipBounds.Max.Y {
				srcMaxY = srcMinY + clipBounds.Max.Y - baseDstY
			}
			if srcMinX >= srcMaxX || srcMinY >= srcMaxY {
				continue
			}
			drawW := srcMaxX - srcMinX

			// Fast path: no flags + white color → row-copy from tileset
			if tile.Flags == 0 && colorIsWhite {
				rowBytes := drawW * 4
				for iy := srcMinY; iy < srcMaxY; iy++ {
					srcRowOff := (baseSrcY+iy)*tsStride + (baseSrcX+srcMinX)*4
					if srcRowOff < 0 || srcRowOff+rowBytes > tsPixLen {
						continue
					}
					dstRowOff := (baseDstY+(iy-srcMinY))*canvasStride + baseDstX*4
					srcRow := tsPix[srcRowOff : srcRowOff+rowBytes]
					dstRow := canvasPix[dstRowOff : dstRowOff+rowBytes]
					// Check if the entire row is fully opaque (all alpha == 255)
					allOpaque := true
					for p := 3; p < rowBytes; p += 4 {
						if srcRow[p] != 255 {
							allOpaque = false
							break
						}
					}
					if allOpaque {
						// Direct row copy (RGB only, alpha stays 255)
						copy(dstRow, srcRow)
						continue
					}
					// Mixed row: pixel-by-pixel with simplified blending
					for ix := 0; ix < drawW; ix++ {
						sp := ix * 4
						pa := srcRow[sp+3]
						if pa == 0 {
							continue
						}
						dp := ix * 4
						if pa == 255 {
							dstRow[dp] = srcRow[sp]
							dstRow[dp+1] = srcRow[sp+1]
							dstRow[dp+2] = srcRow[sp+2]
							continue
						}
						sa := uint32(pa)
						inv := 255 - sa
						dstRow[dp] = uint8((uint32(srcRow[sp])*sa + uint32(dstRow[dp])*inv) / 255)
						dstRow[dp+1] = uint8((uint32(srcRow[sp+1])*sa + uint32(dstRow[dp+1])*inv) / 255)
						dstRow[dp+2] = uint8((uint32(srcRow[sp+2])*sa + uint32(dstRow[dp+2])*inv) / 255)
					}
				}
				continue
			}

			// Slow path: flags or color modulation required
			flags := tile.Flags
			// Match DDNet's texture-coordinate table layout exactly:
			// tableFlag = (X/Y flip bits) + (rotate bit shifted down)
			tableFlag := (flags & (TileFlagVFlip | TileFlagHFlip)) + ((flags & TileFlagRotate) >> 1)
			last := tileLen - 1

			for iy := srcMinY; iy < srcMaxY; iy++ {
				dstRowOff := (baseDstY + (iy - srcMinY)) * canvasStride
				for ix := srcMinX; ix < srcMaxX; ix++ {
					tx, ty := transformTileCoord(tableFlag, uint32(ix), uint32(iy), last)

					srcOff := (baseSrcY+int(ty))*tsStride + (baseSrcX+int(tx))*4
					if srcOff < 0 || srcOff+3 >= tsPixLen {
						continue
					}

					var pr, pg, pb, pa uint8
					if colorIsWhite {
						pr = tsPix[srcOff]
						pg = tsPix[srcOff+1]
						pb = tsPix[srcOff+2]
						pa = tsPix[srcOff+3]
					} else {
						pr = uint8(uint32(tsPix[srcOff]) * lcR / 255)
						pg = uint8(uint32(tsPix[srcOff+1]) * lcG / 255)
						pb = uint8(uint32(tsPix[srcOff+2]) * lcB / 255)
						pa = uint8(uint32(tsPix[srcOff+3]) * lcA / 255)
					}

					if pa == 0 {
						continue
					}

					dstOff := dstRowOff + (baseDstX+(ix-srcMinX))*4
					if pa == 255 {
						canvasPix[dstOff] = pr
						canvasPix[dstOff+1] = pg
						canvasPix[dstOff+2] = pb
						continue
					}

					sa := uint32(pa)
					inv := 255 - sa
					canvasPix[dstOff] = uint8((uint32(pr)*sa + uint32(canvasPix[dstOff])*inv) / 255)
					canvasPix[dstOff+1] = uint8((uint32(pg)*sa + uint32(canvasPix[dstOff+1])*inv) / 255)
					canvasPix[dstOff+2] = uint8((uint32(pb)*sa + uint32(canvasPix[dstOff+2])*inv) / 255)
				}
			}
		}
	}
}

// transformTileCoord maps destination tile pixel coordinates (x,y) to source
// tile pixel coordinates according to DDNet's 8-way flip/rotate table:
// tableFlag = (flags&(flip bits)) + ((flags&rotate)>>1)
func transformTileCoord(tableFlag uint8, x, y, last uint32) (tx, ty uint32) {
	switch tableFlag {
	case 0: // identity
		return x, y
	case 1: // flip X coord
		return last - x, y
	case 2: // flip Y coord
		return x, last - y
	case 3: // flip X+Y
		return last - x, last - y
	case 4: // rotate 90
		return y, last - x
	case 5: // rotate 90 + flip X
		return last - y, last - x
	case 6: // rotate 90 + flip Y
		return y, x
	case 7: // rotate 90 + flip X+Y
		return last - y, x
	default:
		return x, y
	}
}

// renderSingleQuadLayer composites one quad layer onto canvas.
func renderSingleQuadLayer(
	canvas *image.NRGBA,
	ql *renderQuadLayer,
	quadImages map[int]*image.NRGBA,
	crop *MapBounds,
	tileLen uint32,
) {
	tex := quadImages[ql.imageID] // may be nil (vertex colors only)
	mul := ql.alphaMul
	if mul == 0 {
		mul = 1.0
	}
	clipBounds := renderClipBounds(ql.clip, crop, tileLen, canvas.Bounds())
	if clipBounds.Empty() {
		return
	}
	for i := range ql.quads {
		renderQuadOnCanvas(canvas, &ql.quads[i], tex, crop, tileLen, ql.offsetX, ql.offsetY, mul, clipBounds)
	}
}

// renderQuadOnCanvas rasterizes a single quad as two triangles.
// Quad vertex layout in map data is remapped like DDNet does:
// indices 2 and 3 are swapped before rendering.
//
//	[0]=TL  [1]=TR
//	[2]=BR  [3]=BL (after remap)
//
// Triangulation: (0,1,2) and (0,2,3), matching OpenGL GL_QUADS.
func renderQuadOnCanvas(
	canvas *image.NRGBA,
	q *Quad,
	tex *image.NRGBA,
	crop *MapBounds,
	tileLen uint32,
	offsetX, offsetY float64,
	alphaMul float64,
	clipBounds image.Rectangle,
) {
	tl := float64(tileLen)
	cropMinX := float64(crop.MinX)
	cropMinY := float64(crop.MinY)
	quadIdx := [4]int{0, 1, 3, 2} // DDNet swaps 2<->3 before rendering quads

	// Convert quad corner positions from tile coords to canvas pixel coords,
	// applying the group offset.
	var px, py [4]float64
	for i := range 4 {
		idx := quadIdx[i]
		px[i] = (rawWorldCoordToTileFloat(q.Points[idx].X) + offsetX - cropMinX) * tl
		py[i] = (rawWorldCoordToTileFloat(q.Points[idx].Y) + offsetY - cropMinY) * tl
	}

	// Texture coords (normalized [0,1])
	var u, v [4]float64
	for i := range 4 {
		idx := quadIdx[i]
		u[i] = rawTexCoordToFloat(q.TexCoords[idx].X)
		v[i] = rawTexCoordToFloat(q.TexCoords[idx].Y)
	}

	var c [4]color.NRGBA
	for i := range 4 {
		c[i] = q.Colors[quadIdx[i]]
		if alphaMul != 1.0 {
			c[i].A = uint8(float64(c[i].A) * alphaMul)
		}
	}

	// Triangle 1: vertices 0, 1, 2
	rasterizeTriangle(canvas, tex,
		px[0], py[0], u[0], v[0], c[0],
		px[1], py[1], u[1], v[1], c[1],
		px[2], py[2], u[2], v[2], c[2],
		clipBounds,
	)
	// Triangle 2: vertices 0, 2, 3
	rasterizeTriangle(canvas, tex,
		px[0], py[0], u[0], v[0], c[0],
		px[2], py[2], u[2], v[2], c[2],
		px[3], py[3], u[3], v[3], c[3],
		clipBounds,
	)
}

func rawWorldCoordToTileFloat(raw int) float64 {
	return float64(raw) / 32768.0
}

func rawTexCoordToFloat(raw int) float64 {
	return float64(raw) / 1024.0
}

// rasterizeTriangle renders a textured, vertex-colored triangle onto canvas
// using scanline rasterization with barycentric interpolation.
func rasterizeTriangle(
	canvas *image.NRGBA,
	tex *image.NRGBA,
	px0, py0, u0, v0 float64, c0 color.NRGBA,
	px1, py1, u1, v1 float64, c1 color.NRGBA,
	px2, py2, u2, v2 float64, c2 color.NRGBA,
	clipBounds image.Rectangle,
) {
	// Signed area (2×)
	area := (px1-px0)*(py2-py0) - (py1-py0)*(px2-px0)
	if area == 0 {
		return
	}
	invArea := 1.0 / area

	// Bounding box clipped to canvas
	bounds := canvas.Bounds().Intersect(clipBounds)
	minPX := math.Min(px0, math.Min(px1, px2))
	maxPX := math.Max(px0, math.Max(px1, px2))
	minPY := math.Min(py0, math.Min(py1, py2))
	maxPY := math.Max(py0, math.Max(py1, py2))

	startX := int(math.Floor(minPX))
	endX := int(math.Ceil(maxPX))
	startY := int(math.Floor(minPY))
	endY := int(math.Ceil(maxPY))

	if startX < bounds.Min.X {
		startX = bounds.Min.X
	}
	if startY < bounds.Min.Y {
		startY = bounds.Min.Y
	}
	if endX > bounds.Max.X {
		endX = bounds.Max.X
	}
	if endY > bounds.Max.Y {
		endY = bounds.Max.Y
	}
	if startX >= endX || startY >= endY {
		return
	}

	canvasPix := canvas.Pix
	canvasStride := canvas.Stride

	// Pre-convert vertex colors to float64
	cr0, cg0, cb0, ca0 := float64(c0.R), float64(c0.G), float64(c0.B), float64(c0.A)
	cr1, cg1, cb1, ca1 := float64(c1.R), float64(c1.G), float64(c1.B), float64(c1.A)
	cr2, cg2, cb2, ca2 := float64(c2.R), float64(c2.G), float64(c2.B), float64(c2.A)

	for y := startY; y < endY; y++ {
		pyc := float64(y) + 0.5
		for x := startX; x < endX; x++ {
			pxc := float64(x) + 0.5

			// Barycentric coordinates via edge functions
			w0 := ((px1-pxc)*(py2-pyc) - (py1-pyc)*(px2-pxc)) * invArea
			w1 := ((px2-pxc)*(py0-pyc) - (py2-pyc)*(px0-pxc)) * invArea
			w2 := 1.0 - w0 - w1

			// Inside test (handle CW and CCW winding)
			if area > 0 {
				if w0 < 0 || w1 < 0 || w2 < 0 {
					continue
				}
			} else {
				if w0 > 0 || w1 > 0 || w2 > 0 {
					continue
				}
				w0, w1, w2 = -w0, -w1, -w2
			}

			// Interpolate vertex colors
			fR := w0*cr0 + w1*cr1 + w2*cr2
			fG := w0*cg0 + w1*cg1 + w2*cg2
			fB := w0*cb0 + w1*cb1 + w2*cb2
			fA := w0*ca0 + w1*ca1 + w2*ca2

			// Sample texture
			var tR, tG, tB, tA uint8
			if tex != nil {
				tu := w0*u0 + w1*u1 + w2*u2
				tv := w0*v0 + w1*v1 + w2*v2
				c := bilinearSampleNormalizedNRGBA(tex, tu, tv)
				tR = c.R
				tG = c.G
				tB = c.B
				tA = c.A
			} else {
				tR, tG, tB, tA = 255, 255, 255, 255
			}

			// Modulate: pixel = vertex_color × texture_color / 255
			pr := uint8(clampF64(fR * float64(tR) / 255.0))
			pg := uint8(clampF64(fG * float64(tG) / 255.0))
			pb := uint8(clampF64(fB * float64(tB) / 255.0))
			pa := uint8(clampF64(fA * float64(tA) / 255.0))

			if pa == 0 {
				continue
			}

			// Source-over compositing (opaque canvas: dst.A always 255)
			dstOff := y*canvasStride + x*4
			if pa == 255 {
				canvasPix[dstOff] = pr
				canvasPix[dstOff+1] = pg
				canvasPix[dstOff+2] = pb
				// canvasPix[dstOff+3] stays 255
				continue
			}

			sa := uint32(pa)
			inv := 255 - sa
			canvasPix[dstOff] = uint8((uint32(pr)*sa + uint32(canvasPix[dstOff])*inv) / 255)
			canvasPix[dstOff+1] = uint8((uint32(pg)*sa + uint32(canvasPix[dstOff+1])*inv) / 255)
			canvasPix[dstOff+2] = uint8((uint32(pb)*sa + uint32(canvasPix[dstOff+2])*inv) / 255)
			// canvasPix[dstOff+3] stays 255
		}
	}
}

func clampF64(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// fitInBoundingBox calculates the largest size that fits in maxW × maxH
// while preserving the aspect ratio of srcW × srcH.
func fitInBoundingBox(srcW, srcH, maxW, maxH int) (int, int) {
	if srcW <= 0 || srcH <= 0 {
		return maxW, maxH
	}
	wScale := float64(maxW) / float64(srcW)
	hScale := float64(maxH) / float64(srcH)
	scale := math.Min(wScale, hScale)
	w := int(math.Round(float64(srcW) * scale))
	h := int(math.Round(float64(srcH) * scale))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// scaleNRGBA performs bilinear interpolation to resize an NRGBA image.
// It interpolates all four channels (RGBA) in premultiplied-alpha space
// to produce smooth, correct edges on transparent images.
func scaleNRGBA(src *image.NRGBA, dstW, dstH int) *image.NRGBA {
	return scaleImageRectNRGBA(src, src.Bounds(), dstW, dstH)
}

// scaleImageRectNRGBA rescales a source rectangle from an NRGBA image using
// the shared bilinear resampler used for all PNG subimages and full-image
// resizes in the renderer.
func scaleImageRectNRGBA(src *image.NRGBA, srcRect image.Rectangle, dstW, dstH int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	if src == nil || dstW <= 0 || dstH <= 0 {
		return dst
	}
	srcRect = srcRect.Intersect(src.Bounds())
	srcW := srcRect.Dx()
	srcH := srcRect.Dy()
	if srcW == 0 || srcH == 0 {
		return dst
	}

	dstPix := dst.Pix
	dstStride := dst.Stride

	for dy := range dstH {
		sy := mappedSampleCoord(float64(dy)+0.5, 0, float64(dstH), srcRect.Min.Y, srcH)
		dstRow := dy * dstStride

		for dx := range dstW {
			dOff := dstRow + dx*4
			sx := mappedSampleCoord(float64(dx)+0.5, 0, float64(dstW), srcRect.Min.X, srcW)
			c := bilinearSampleNRGBA(src, srcRect, sx, sy)
			dstPix[dOff] = c.R
			dstPix[dOff+1] = c.G
			dstPix[dOff+2] = c.B
			dstPix[dOff+3] = c.A
		}
	}
	return dst
}

// mappedSampleCoord maps a destination pixel center to a source sample
// coordinate using the shared scaling convention for all bilinearly scaled
// PNG inputs in the renderer.
func mappedSampleCoord(dstPixelCenter, dstStart, dstSize float64, srcStart, srcSize int) float64 {
	if dstSize <= 0 || srcSize <= 0 {
		return float64(srcStart)
	}
	return (dstPixelCenter-dstStart)*(float64(srcSize)/dstSize) - 0.5 + float64(srcStart)
}

// bilinearSampleNRGBA samples src within srcRect using premultiplied-alpha
// bilinear interpolation and returns a non-premultiplied NRGBA color.
func bilinearSampleNRGBA(src *image.NRGBA, srcRect image.Rectangle, fx, fy float64) color.NRGBA {
	if src == nil || srcRect.Dx() <= 0 || srcRect.Dy() <= 0 {
		return color.NRGBA{}
	}

	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	x1 := x0 + 1
	y1 := y0 + 1
	xf := fx - float64(x0)
	yf := fy - float64(y0)
	if xf < 0 {
		xf = 0
	}
	if yf < 0 {
		yf = 0
	}

	maxSX := srcRect.Max.X - 1
	maxSY := srcRect.Max.Y - 1
	if x0 < srcRect.Min.X {
		x0 = srcRect.Min.X
	}
	if x0 > maxSX {
		x0 = maxSX
	}
	if x1 < srcRect.Min.X {
		x1 = srcRect.Min.X
	}
	if x1 > maxSX {
		x1 = maxSX
	}
	if y0 < srcRect.Min.Y {
		y0 = srcRect.Min.Y
	}
	if y0 > maxSY {
		y0 = maxSY
	}
	if y1 < srcRect.Min.Y {
		y1 = srcRect.Min.Y
	}
	if y1 > maxSY {
		y1 = maxSY
	}

	srcPix := src.Pix
	srcStride := src.Stride
	off00 := y0*srcStride + x0*4
	off10 := y0*srcStride + x1*4
	off01 := y1*srcStride + x0*4
	off11 := y1*srcStride + x1*4

	a00, a10 := float64(srcPix[off00+3]), float64(srcPix[off10+3])
	a01, a11 := float64(srcPix[off01+3]), float64(srcPix[off11+3])
	pa00, pa10 := a00/255.0, a10/255.0
	pa01, pa11 := a01/255.0, a11/255.0

	ix0 := 1 - xf
	ix1 := xf
	iy0 := 1 - yf
	iy1 := yf

	outA := a00*ix0*iy0 + a10*ix1*iy0 + a01*ix0*iy1 + a11*ix1*iy1
	if outA < 0.5 {
		return color.NRGBA{}
	}

	pr := float64(srcPix[off00+0])*pa00*ix0*iy0 + float64(srcPix[off10+0])*pa10*ix1*iy0 +
		float64(srcPix[off01+0])*pa01*ix0*iy1 + float64(srcPix[off11+0])*pa11*ix1*iy1
	pg := float64(srcPix[off00+1])*pa00*ix0*iy0 + float64(srcPix[off10+1])*pa10*ix1*iy0 +
		float64(srcPix[off01+1])*pa01*ix0*iy1 + float64(srcPix[off11+1])*pa11*ix1*iy1
	pb := float64(srcPix[off00+2])*pa00*ix0*iy0 + float64(srcPix[off10+2])*pa10*ix1*iy0 +
		float64(srcPix[off01+2])*pa01*ix0*iy1 + float64(srcPix[off11+2])*pa11*ix1*iy1

	aOut := outA / 255.0
	var r, g, b float64
	if aOut > 0 {
		r = pr / aOut
		g = pg / aOut
		b = pb / aOut
	}

	return color.NRGBA{
		R: uint8(clampF64(r)),
		G: uint8(clampF64(g)),
		B: uint8(clampF64(b)),
		A: uint8(clampF64(outA)),
	}
}

// bilinearSampleNormalizedNRGBA samples an entire NRGBA texture using
// normalized texture coordinates.
func bilinearSampleNormalizedNRGBA(src *image.NRGBA, u, v float64) color.NRGBA {
	if src == nil {
		return color.NRGBA{}
	}
	b := src.Bounds()
	fx := u*float64(b.Dx()) - 0.5 + float64(b.Min.X)
	fy := v*float64(b.Dy()) - 0.5 + float64(b.Min.Y)
	return bilinearSampleNRGBA(src, b, fx, fy)
}

// fillCheckerboard paints the DDNet editor checkerboard background onto canvas.
// Each checker square is checkerSize×checkerSize pixels (16 tiles wide).
func fillCheckerboard(canvas *image.NRGBA, tileLen uint32) {
	// DDNet uses 32px checker cells at the default view. With our tile
	// resolution that maps to tileLen pixels per tile, we use 16-tile-wide
	// squares to match the editor feel.
	checkerSize := max(int(tileLen)*16, 1)

	bounds := canvas.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()
	pix := canvas.Pix
	stride := canvas.Stride

	for y := range imgH {
		cy := (y / checkerSize) & 1
		rowOff := y * stride
		for x := range imgW {
			cx := (x / checkerSize) & 1
			off := rowOff + x*4
			if (cx ^ cy) == 0 {
				pix[off] = checkerLight
				pix[off+1] = checkerLight
				pix[off+2] = checkerLight
			} else {
				pix[off] = checkerDark
				pix[off+1] = checkerDark
				pix[off+2] = checkerDark
			}
			pix[off+3] = 255
		}
	}
}

// ── Entity rendering ─────────────────────────────────────────────────────────

// gameSkinSprite defines a sub-rectangle in the game.png sprite sheet.
// Coordinates are in grid cells (32×32 px each in a 1024×512 image).
type gameSkinSprite struct {
	x, y, w, h int
}

// bounds returns the pixel rectangle for this sprite.
func (s gameSkinSprite) bounds() image.Rectangle {
	const cell = 32
	return image.Rect(s.x*cell, s.y*cell, (s.x+s.w)*cell, (s.y+s.h)*cell)
}

// entityRenderInfo defines the sprite region and display size of an entity.
// widthTiles and heightTiles are the physical size in tile units, computed
// from the DDNet client's GetSpriteScale formula and the weapon's visual_size.
//
// DDNet formula (items.cpp + graphics_threaded.cpp):
//
//	f = sqrt(spriteW² + spriteH²)
//	scaleX, scaleY = spriteW/f, spriteH/f
//	displaySize = baseSize * scale
//	displayTiles = displaySize / 32.0  (32 game units per tile)
//
// Sources:
//   - ddnet/src/game/client/components/items.cpp (OnInit, RenderPickup, RenderFlag)
//   - ddnet/src/engine/client/graphics_threaded.cpp (GetSpriteScale)
//   - ddnet/datasrc/content.py (weapon visual_size, sprite grid coords)
type entityRenderInfo struct {
	sprite       gameSkinSprite
	widthTiles   float64 // display width in tile units
	heightTiles  float64 // display height in tile units
	offsetYTiles float64 // vertical offset (negative = up); used for flags
}

// entityInfo maps game-layer tile IDs to their sprite and DDNet display size.
// Spawns have no entity sprite in DDNet — they are only visible in the
// game layer overlay (entities.png).
var entityInfo = map[uint8]entityRenderInfo{
	// Health: sprite 2×2, base_size=64. f=√8≈2.828, scale=0.707 → 45.25 GU → 1.414 tiles
	TileHealth: {
		sprite:     gameSkinSprite{x: 10, y: 2, w: 2, h: 2},
		widthTiles: 1.414, heightTiles: 1.414,
	},
	// Armor: same as health
	TileArmor: {
		sprite:     gameSkinSprite{x: 12, y: 2, w: 2, h: 2},
		widthTiles: 1.414, heightTiles: 1.414,
	},
	// Shotgun: sprite 8×2, visual_size=96. f=√68≈8.246 → 93.1×23.3 GU → 2.91×0.73 tiles
	TileWeaponShotgun: {
		sprite:     gameSkinSprite{x: 2, y: 6, w: 8, h: 2},
		widthTiles: 2.91, heightTiles: 0.73,
	},
	// Grenade: sprite 7×2, visual_size=96. f=√53≈7.280 → 92.3×26.4 GU → 2.88×0.82 tiles
	TileWeaponGrenade: {
		sprite:     gameSkinSprite{x: 2, y: 8, w: 7, h: 2},
		widthTiles: 2.88, heightTiles: 0.82,
	},
	// Ninja: sprite 8×2, base_size=128. f=√68≈8.246 → 124.2×31.0 GU → 3.88×0.97 tiles
	TilePowerupNinja: {
		sprite:     gameSkinSprite{x: 2, y: 10, w: 8, h: 2},
		widthTiles: 3.88, heightTiles: 0.97,
	},
	// Laser: sprite 7×3, visual_size=92. f=√58≈7.616 → 84.6×36.2 GU → 2.64×1.13 tiles
	TileWeaponLaser: {
		sprite:     gameSkinSprite{x: 2, y: 12, w: 7, h: 3},
		widthTiles: 2.64, heightTiles: 1.13,
	},
	// Weapon-removal pickups (DDNet entities).
	TileEntityArmorShotgun: {
		sprite:     gameSkinSprite{x: 15, y: 2, w: 2, h: 2},
		widthTiles: 1.414, heightTiles: 1.414,
	},
	TileEntityArmorGrenade: {
		sprite:     gameSkinSprite{x: 17, y: 2, w: 2, h: 2},
		widthTiles: 1.414, heightTiles: 1.414,
	},
	TileEntityArmorNinja: {
		sprite:     gameSkinSprite{x: 10, y: 10, w: 2, h: 2},
		widthTiles: 1.414, heightTiles: 1.414,
	},
	TileEntityArmorLaser: {
		sprite:     gameSkinSprite{x: 19, y: 2, w: 2, h: 2},
		widthTiles: 1.414, heightTiles: 1.414,
	},
	// Flag blue: hardcoded 42×84 GU → 1.31×2.63 tiles, drawn at y - 42*0.75 = -0.98 tiles
	TileFlagstandBlue: {
		sprite:     gameSkinSprite{x: 12, y: 8, w: 4, h: 8},
		widthTiles: 1.31, heightTiles: 2.63,
		offsetYTiles: -0.98,
	},
	// Flag red: same dimensions
	TileFlagstandRed: {
		sprite:     gameSkinSprite{x: 16, y: 8, w: 4, h: 8},
		widthTiles: 1.31, heightTiles: 2.63,
		offsetYTiles: -0.98,
	},
}

// renderEntities draws entity icons (pickups, flags) from the game layer
// onto the canvas. When a game skin is registered (via [RegisterGameSkin]
// or importing the gameskin package), sprites from the skin are drawn at
// their DDNet client proportions.
//
// Spawns are intentionally excluded — they have no runtime sprite in DDNet
// and are only visible through the game/entity overlay
// ([WithGameLayer] or [WithOverlayEntities]).
func renderEntities(canvas *image.NRGBA, m *Map, ro *renderOptions, crop *MapBounds, tileLen uint32) {
	skin := resolveGameSkin()
	if skin == nil {
		return
	}
	tl := float64(tileLen)
	for i := range m.Groups {
		g := &m.Groups[i]
		clipBounds := renderClipBounds(groupClipForRender(g), crop, tileLen, canvas.Bounds())
		if clipBounds.Empty() {
			continue
		}
		tileOffX, tileOffY, _, _, ok := computeGroupRenderOffsets(g, ro)
		if !ok {
			continue
		}
		for j := range g.Layers {
			l := &g.Layers[j]
			if l.Kind != LayerKindGame {
				continue
			}
			for i, t := range l.Tiles {
				info, hasSprite := entityInfo[t.ID]
				if !hasSprite {
					continue
				}

				tx := float64(i%l.Width) + tileOffX
				ty := float64(i/l.Width) + tileOffY
				// Center of the entity tile in canvas pixels.
				centerX := (tx-float64(crop.MinX))*tl + tl/2.0
				centerY := (ty-float64(crop.MinY))*tl + tl/2.0

				dstW := info.widthTiles * tl
				dstH := info.heightTiles * tl
				offY := info.offsetYTiles * tl
				blitSpriteRectClipped(canvas, skin, info.sprite.bounds(),
					centerX-dstW/2, centerY-dstH/2+offY, dstW, dstH, clipBounds)
			}
		}
	}
}

// blitSpriteRect draws the srcRect region of src onto canvas at the given
// destination rectangle (dstX, dstY, dstW, dstH) in canvas pixel coordinates.
// It uses the same shared bilinear resampler as scaleNRGBA.
func blitSpriteRect(canvas *image.NRGBA, src *image.NRGBA, srcRect image.Rectangle,
	dstX, dstY, dstW, dstH float64) {
	blitSpriteRectClipped(canvas, src, srcRect, dstX, dstY, dstW, dstH, canvas.Bounds())
}

func blitSpriteRectClipped(canvas *image.NRGBA, src *image.NRGBA, srcRect image.Rectangle,
	dstX, dstY, dstW, dstH float64, clipBounds image.Rectangle) {

	sw := float64(srcRect.Dx())
	sh := float64(srcRect.Dy())
	if sw == 0 || sh == 0 || dstW <= 0 || dstH <= 0 {
		return
	}

	bounds := canvas.Bounds().Intersect(clipBounds)
	startDX := int(math.Floor(dstX))
	startDY := int(math.Floor(dstY))
	endDX := int(math.Ceil(dstX + dstW))
	endDY := int(math.Ceil(dstY + dstH))

	// Clip to canvas.
	if startDX < bounds.Min.X {
		startDX = bounds.Min.X
	}
	if startDY < bounds.Min.Y {
		startDY = bounds.Min.Y
	}
	if endDX > bounds.Max.X {
		endDX = bounds.Max.X
	}
	if endDY > bounds.Max.Y {
		endDY = bounds.Max.Y
	}

	for py := startDY; py < endDY; py++ {
		fy := mappedSampleCoord(float64(py)+0.5, dstY, dstH, srcRect.Min.Y, int(sh))
		for px := startDX; px < endDX; px++ {
			fx := mappedSampleCoord(float64(px)+0.5, dstX, dstW, srcRect.Min.X, int(sw))
			c := bilinearSampleNRGBA(src, srcRect, fx, fy)
			if c.A == 0 {
				continue
			}
			alphaBlendPixel(canvas, px, py, c)
		}
	}
}

// ── DDNet physics overlays ──────────────────────────────────────────────────

// renderOverlayLayers renders pre-collected DDNet physics/entity tile layers
// using the dedicated DDNet entity-layer sheet.
func renderOverlayLayers(canvas *image.NRGBA, layers []overlayRenderLayer, crop *MapBounds, tileLen uint32) {
	if len(layers) == 0 {
		return
	}
	entImg := resolveEntitiesImage()
	speedupArrow := resolveSpeedupArrowImage()
	speedupArrowArray := resolveSpeedupArrowArrayImage()
	var tilesets map[int]*image.NRGBA
	if entImg != nil {
		tilesets = map[int]*image.NRGBA{
			0: scaleTileset(entImg, int(tileLen)),
		}
	}
	for i := range layers {
		clipBounds := renderClipBounds(layers[i].layer.clip, crop, tileLen, canvas.Bounds())
		if clipBounds.Empty() {
			continue
		}
		switch layers[i].kind {
		case LayerKindGame:
			if len(tilesets) != 0 {
				renderGameLayerDeathBorderClipped(canvas, &layers[i].layer, crop, tileLen, tilesets[0], clipBounds)
				renderSingleTileLayer(canvas, &layers[i].layer, tilesets, crop, tileLen)
				renderOverlayLayerTextClipped(canvas, &layers[i], crop, tileLen, clipBounds)
			}
		case LayerKindSpeedup:
			renderSpeedupLayerClipped(canvas, &layers[i], crop, tileLen, clipBounds, speedupArrow, speedupArrowArray)
			if speedupArrow != nil || speedupArrowArray != nil || layers[i].showInvalid {
				renderOverlayLayerTextClipped(canvas, &layers[i], crop, tileLen, clipBounds)
			}
		default:
			if len(tilesets) != 0 {
				renderSingleTileLayer(canvas, &layers[i].layer, tilesets, crop, tileLen)
				renderOverlayLayerTextClipped(canvas, &layers[i], crop, tileLen, clipBounds)
			}
		}
	}
}

func renderGameLayerDeathBorderClipped(canvas *image.NRGBA, layer *renderLayer, crop *MapBounds, tileLen uint32, tileset *image.NRGBA, clipBounds image.Rectangle) {
	if layer == nil || tileset == nil || layer.width <= 0 || layer.height <= 0 || clipBounds.Empty() {
		return
	}

	borderColor := layer.color
	borderColor.A = uint8(math.Round(float64(borderColor.A) * 0.65))
	if borderColor.A == 0 {
		return
	}

	tilePx := int(tileLen)
	layerMinX := layer.offsetX
	layerMinY := layer.offsetY
	layerMaxX := layer.offsetX + float64(layer.width)
	layerMaxY := layer.offsetY + float64(layer.height)

	for worldY := crop.MinY; worldY < crop.MaxY; worldY++ {
		tileCenterY := float64(worldY) + 0.5
		dstY := int(math.Round(float64(worldY-crop.MinY) * float64(tilePx)))
		for worldX := crop.MinX; worldX < crop.MaxX; worldX++ {
			tileCenterX := float64(worldX) + 0.5
			if tileCenterX >= layerMinX && tileCenterX < layerMaxX && tileCenterY >= layerMinY && tileCenterY < layerMaxY {
				continue
			}
			dstX := int(math.Round(float64(worldX-crop.MinX) * float64(tilePx)))
			drawTilesetTileClipped(canvas, tileset, TileDeath, dstX, dstY, tilePx, borderColor, clipBounds)
		}
	}
}

func drawTilesetTileClipped(canvas *image.NRGBA, tileset *image.NRGBA, tileID uint8, dstX, dstY, tilePx int, mod color.NRGBA, clipBounds image.Rectangle) {
	if tileset == nil || tilePx <= 0 || mod.A == 0 {
		return
	}

	bounds := canvas.Bounds().Intersect(clipBounds)
	startX := max(dstX, bounds.Min.X)
	startY := max(dstY, bounds.Min.Y)
	endX := min(dstX+tilePx, bounds.Max.X)
	endY := min(dstY+tilePx, bounds.Max.Y)
	if startX >= endX || startY >= endY {
		return
	}

	tileX := int(tileID) % tilesetGridSize
	tileY := int(tileID) / tilesetGridSize
	baseSrcX := tileX * tilePx
	baseSrcY := tileY * tilePx

	for py := startY; py < endY; py++ {
		sy := baseSrcY + (py - dstY)
		for px := startX; px < endX; px++ {
			sx := baseSrcX + (px - dstX)
			off := tileset.PixOffset(sx, sy)
			c := color.NRGBA{
				R: uint8(uint32(tileset.Pix[off]) * uint32(mod.R) / 255),
				G: uint8(uint32(tileset.Pix[off+1]) * uint32(mod.G) / 255),
				B: uint8(uint32(tileset.Pix[off+2]) * uint32(mod.B) / 255),
				A: uint8(uint32(tileset.Pix[off+3]) * uint32(mod.A) / 255),
			}
			if c.A == 0 {
				continue
			}
			alphaBlendPixel(canvas, px, py, c)
		}
	}
}

func renderSpeedupLayer(canvas *image.NRGBA, layer *overlayRenderLayer, crop *MapBounds, tileLen uint32) {
	renderSpeedupLayerClipped(canvas, layer, crop, tileLen, canvas.Bounds(), resolveSpeedupArrowImage(), resolveSpeedupArrowArrayImage())
}

// renderSpeedupLayerClipped draws the speedup overlay arrows. When arrowArray
// (DDNet speed_arrow_array.png) is registered it is preferred for DDNet-accurate
// per-degree rotation; otherwise it falls back to the single speed_arrow.png.
func renderSpeedupLayerClipped(canvas *image.NRGBA, layer *overlayRenderLayer, crop *MapBounds, tileLen uint32, clipBounds image.Rectangle, arrow, arrowArray *image.NRGBA) {
	if layer == nil || len(layer.speedupTiles) == 0 || layer.layer.width <= 0 {
		return
	}
	if arrow == nil && arrowArray == nil {
		return
	}
	tilePx := float64(tileLen)
	for idx, speed := range layer.speedupTiles {
		if !shouldRenderSpeedupBase(speed, layer.showInvalid) {
			continue
		}
		tx := float64(idx%layer.layer.width) + layer.layer.offsetX
		ty := float64(idx/layer.layer.width) + layer.layer.offsetY
		centerX := (tx - float64(crop.MinX) + 0.5) * tilePx
		centerY := (ty - float64(crop.MinY) + 0.5) * tilePx
		if arrowArray != nil {
			renderSpeedupArrowArrayClipped(canvas, centerX, centerY, tilePx, int(speed.Angle), layer.layer.color, clipBounds, arrowArray)
			continue
		}
		renderSpeedupArrowClipped(canvas, centerX, centerY, tilePx, float64(speed.Angle), layer.layer.color, clipBounds, arrow)
	}
}

// speedupArrowArrayGrid is the sub-tile grid size of speed_arrow_array.png
// (16×16 cells of 64px in the 1024×1024 sheet).
const speedupArrowArrayGrid = 16

// speedupArrowArrayFrame maps a speedup angle (degrees) to the DDNet array
// sub-tile index (0..89: fine rotation within a quadrant) and the quadrant
// rotation count (0..3, multiples of 90°). Mirrors FillTmpTileSpeedup in DDNet
// src/game/map/render_layer.cpp.
func speedupArrowArrayFrame(angleDeg int) (subIndex, quadrant int) {
	a := angleDeg % 360
	if a < 0 {
		a += 360
	}
	return a % 90, a / 90
}

// renderSpeedupArrowArrayClipped draws one speedup arrow from the sprite array,
// selecting the per-degree sub-tile and applying the coarse quadrant rotation.
func renderSpeedupArrowArrayClipped(canvas *image.NRGBA, centerX, centerY, tilePx float64, angleDeg int, c color.NRGBA, clipBounds image.Rectangle, arrowArray *image.NRGBA) {
	if arrowArray == nil {
		return
	}
	subIndex, quadrant := speedupArrowArrayFrame(angleDeg)
	col := subIndex % speedupArrowArrayGrid
	row := subIndex / speedupArrowArrayGrid
	inv := 1.0 / float64(speedupArrowArrayGrid)
	u0 := float64(col) * inv
	u1 := u0 + inv
	v0 := float64(row) * inv
	v1 := v0 + inv

	size := tilePx * (35.0 / float64(pixelsPerTile))
	half := size / 2.0
	points := [4][2]float64{{-half, -half}, {half, -half}, {half, half}, {-half, half}}
	quadAngle := float64(quadrant) * 90.0
	for i := range points {
		points[i][0], points[i][1] = rotateAroundOrigin(points[i][0], points[i][1], quadAngle)
		points[i][0] += centerX
		points[i][1] += centerY
	}
	rasterizeTriangle(canvas, arrowArray,
		points[0][0], points[0][1], u0, v0, c,
		points[1][0], points[1][1], u1, v0, c,
		points[2][0], points[2][1], u1, v1, c,
		clipBounds,
	)
	rasterizeTriangle(canvas, arrowArray,
		points[0][0], points[0][1], u0, v0, c,
		points[2][0], points[2][1], u1, v1, c,
		points[3][0], points[3][1], u0, v1, c,
		clipBounds,
	)
}

func renderSpeedupArrow(canvas *image.NRGBA, centerX, centerY, tilePx, angleDeg float64, c color.NRGBA) {
	renderSpeedupArrowClipped(canvas, centerX, centerY, tilePx, angleDeg, c, canvas.Bounds(), resolveSpeedupArrowImage())
}

func renderSpeedupArrowClipped(canvas *image.NRGBA, centerX, centerY, tilePx, angleDeg float64, c color.NRGBA, clipBounds image.Rectangle, arrow *image.NRGBA) {
	if arrow == nil {
		return
	}
	size := tilePx * (35.0 / float64(pixelsPerTile))
	half := size / 2.0
	points := [4][2]float64{{-half, -half}, {half, -half}, {half, half}, {-half, half}}
	for i := range points {
		points[i][0], points[i][1] = rotateAroundOrigin(points[i][0], points[i][1], angleDeg)
		points[i][0] += centerX
		points[i][1] += centerY
	}
	rasterizeTriangle(canvas, arrow,
		points[0][0], points[0][1], 0, 0, c,
		points[1][0], points[1][1], 1, 0, c,
		points[2][0], points[2][1], 1, 1, c,
		clipBounds,
	)
	rasterizeTriangle(canvas, arrow,
		points[0][0], points[0][1], 0, 0, c,
		points[2][0], points[2][1], 1, 1, c,
		points[3][0], points[3][1], 0, 1, c,
		clipBounds,
	)
}

func rotateAroundOrigin(x, y, angleDeg float64) (float64, float64) {
	rad := angleDeg * math.Pi / 180.0
	sinA := math.Sin(rad)
	cosA := math.Cos(rad)
	return x*cosA - y*sinA, x*sinA + y*cosA
}

const (
	bitmapTextGlyphWidth   = 3
	bitmapTextGlyphHeight  = 5
	bitmapTextGlyphSpacing = 1
	minOverlayTextTileLen  = 12
)

var bitmapDigitFont = map[rune][bitmapTextGlyphHeight]string{
	'0': {"111", "101", "101", "101", "111"},
	'1': {"010", "110", "010", "010", "111"},
	'2': {"111", "001", "111", "100", "111"},
	'3': {"111", "001", "111", "001", "111"},
	'4': {"101", "101", "111", "001", "001"},
	'5': {"111", "100", "111", "001", "111"},
	'6': {"111", "100", "111", "101", "111"},
	'7': {"111", "001", "010", "010", "010"},
	'8': {"111", "101", "111", "101", "111"},
	'9': {"111", "101", "111", "001", "111"},
	'-': {"000", "000", "111", "000", "000"},
}

func renderOverlayLayerText(canvas *image.NRGBA, layer *overlayRenderLayer, crop *MapBounds, tileLen uint32) {
	renderOverlayLayerTextClipped(canvas, layer, crop, tileLen, canvas.Bounds())
}

func renderOverlayLayerTextClipped(canvas *image.NRGBA, layer *overlayRenderLayer, crop *MapBounds, tileLen uint32, clipBounds image.Rectangle) {
	if layer == nil || tileLen < minOverlayTextTileLen || layer.layer.width <= 0 {
		return
	}

	tilePx := float64(tileLen)
	textColor := color.NRGBA{R: 255, G: 255, B: 255, A: layer.layer.color.A}
	if textColor.A == 0 {
		textColor.A = 255
	}

	switch layer.kind {
	case LayerKindTele:
		for idx, tile := range layer.teleTiles {
			if tile.Number == 0 || !IsValidTeleTile(tile.ID) || !IsTeleTileNumberUsedAny(tile.ID) {
				continue
			}
			left, top := overlayTileCanvasOrigin(layer, idx, crop, tilePx)
			drawBitmapTextInBoxClipped(canvas, clipBounds, strconv.Itoa(int(tile.Number)), left+tilePx*0.04, top+tilePx*0.08, tilePx*0.92, tilePx*0.72, textColor)
		}
	case LayerKindSwitch:
		for idx, tile := range layer.switchTiles {
			if !IsValidSwitchTile(tile.ID) {
				continue
			}
			left, top := overlayTileCanvasOrigin(layer, idx, crop, tilePx)
			if tile.Number > 0 && IsSwitchTileNumberUsed(tile.ID) {
				drawBitmapTextInBoxClipped(canvas, clipBounds, strconv.Itoa(int(tile.Number)), left+tilePx*0.04, top+tilePx*0.04, tilePx*0.92, tilePx*0.34, textColor)
			}
			if tile.Delay > 0 && IsSwitchTileDelayUsed(tile.ID) {
				drawBitmapTextInBoxClipped(canvas, clipBounds, strconv.Itoa(int(tile.Delay)), left+tilePx*0.04, top+tilePx*0.50, tilePx*0.92, tilePx*0.34, textColor)
			}
		}
	case LayerKindTune:
		for idx, tile := range layer.tuneTiles {
			if tile.Number == 0 || !IsValidTuneTile(tile.ID) {
				continue
			}
			left, top := overlayTileCanvasOrigin(layer, idx, crop, tilePx)
			drawBitmapTextInBoxClipped(canvas, clipBounds, strconv.Itoa(int(tile.Number)), left+tilePx*0.04, top+tilePx*0.08, tilePx*0.92, tilePx*0.72, textColor)
		}
	case LayerKindSpeedup:
		for idx, tile := range layer.speedupTiles {
			if !speedupTileHasAnyData(tile) {
				continue
			}
			left, top := overlayTileCanvasOrigin(layer, idx, crop, tilePx)
			if IsValidSpeedupTile(tile.ID) {
				if !shouldRenderSpeedupBase(tile, layer.showInvalid) {
					continue
				}
				if tile.MaxSpeed > 0 {
					drawBitmapTextLineClipped(canvas, clipBounds, strconv.Itoa(int(tile.MaxSpeed)), left, top, tilePx/2.0, tilePx, textColor)
				}
				drawBitmapTextLineClipped(canvas, clipBounds, strconv.Itoa(int(tile.Force)), left, top+tilePx/2.0, tilePx/2.0, tilePx, textColor)
				continue
			}
			if !layer.showInvalid {
				continue
			}
			lineHeight := tilePx / 3.0
			drawBitmapTextLineClipped(canvas, clipBounds, strconv.Itoa(int(tile.Force)), left, top, lineHeight, tilePx, textColor)
			drawBitmapTextLineClipped(canvas, clipBounds, strconv.Itoa(int(tile.MaxSpeed)), left, top+lineHeight, lineHeight, tilePx, textColor)
			drawBitmapTextLineClipped(canvas, clipBounds, strconv.Itoa(int(tile.Angle)), left, top+lineHeight*2.0, lineHeight, tilePx, textColor)
		}
	}
}

func overlayTileCanvasOrigin(layer *overlayRenderLayer, idx int, crop *MapBounds, tilePx float64) (float64, float64) {
	tx := float64(idx%layer.layer.width) + layer.layer.offsetX
	ty := float64(idx/layer.layer.width) + layer.layer.offsetY
	left := (tx - float64(crop.MinX)) * tilePx
	top := (ty - float64(crop.MinY)) * tilePx
	return left, top
}

func drawBitmapTextInBox(canvas *image.NRGBA, text string, boxX, boxY, boxW, boxH float64, c color.NRGBA) {
	drawBitmapTextInBoxClipped(canvas, canvas.Bounds(), text, boxX, boxY, boxW, boxH, c)
}

func drawBitmapTextLineClipped(canvas *image.NRGBA, clipBounds image.Rectangle, text string, x, y, lineHeight, maxWidth float64, c color.NRGBA) {
	if text == "" || lineHeight <= 0 || maxWidth <= 0 {
		return
	}
	textW, textH := bitmapTextSize(text)
	if textW == 0 || textH == 0 {
		return
	}
	scale := lineHeight / float64(textH)
	if fitScale := maxWidth / float64(textW); fitScale < scale {
		scale = fitScale
	}
	if scale < 1.0 {
		return
	}
	baselineY := y + (lineHeight-float64(textH)*scale)/2.0
	renderBitmapTextClipped(canvas, clipBounds, text, x, baselineY, scale, c)
}

func drawBitmapTextInBoxClipped(canvas *image.NRGBA, clipBounds image.Rectangle, text string, boxX, boxY, boxW, boxH float64, c color.NRGBA) {
	if text == "" || boxW <= 0 || boxH <= 0 {
		return
	}
	textW, textH := bitmapTextSize(text)
	if textW == 0 || textH == 0 {
		return
	}
	scale := math.Min(boxW/float64(textW), boxH/float64(textH))
	if scale < 1.0 {
		return
	}
	renderW := float64(textW) * scale
	renderH := float64(textH) * scale
	x := boxX + (boxW-renderW)/2.0
	y := boxY + (boxH-renderH)/2.0
	shadowOffset := math.Max(1.0, scale*0.18)
	renderBitmapTextClipped(canvas, clipBounds, text, x+shadowOffset, y+shadowOffset, scale, color.NRGBA{A: c.A})
	renderBitmapTextClipped(canvas, clipBounds, text, x, y, scale, c)
}

func bitmapTextSize(text string) (int, int) {
	if text == "" {
		return 0, 0
	}
	width := 0
	count := 0
	for _, ch := range text {
		if _, ok := bitmapDigitFont[ch]; !ok {
			continue
		}
		if count > 0 {
			width += bitmapTextGlyphSpacing
		}
		width += bitmapTextGlyphWidth
		count++
	}
	if count == 0 {
		return 0, 0
	}
	return width, bitmapTextGlyphHeight
}

func renderBitmapText(canvas *image.NRGBA, text string, x, y, scale float64, c color.NRGBA) {
	renderBitmapTextClipped(canvas, canvas.Bounds(), text, x, y, scale, c)
}

func renderBitmapTextClipped(canvas *image.NRGBA, clipBounds image.Rectangle, text string, x, y, scale float64, c color.NRGBA) {
	if scale <= 0 || c.A == 0 {
		return
	}
	cursorX := x
	for _, ch := range text {
		glyph, ok := bitmapDigitFont[ch]
		if !ok {
			continue
		}
		for gy := 0; gy < bitmapTextGlyphHeight; gy++ {
			for gx := 0; gx < bitmapTextGlyphWidth; gx++ {
				if glyph[gy][gx] != '1' {
					continue
				}
				fillRectNRGBAClipped(canvas, clipBounds, cursorX+float64(gx)*scale, y+float64(gy)*scale, scale, scale, c)
			}
		}
		cursorX += float64(bitmapTextGlyphWidth+bitmapTextGlyphSpacing) * scale
	}
}

func fillRectNRGBA(canvas *image.NRGBA, x, y, w, h float64, c color.NRGBA) {
	fillRectNRGBAClipped(canvas, canvas.Bounds(), x, y, w, h, c)
}

func fillRectNRGBAClipped(canvas *image.NRGBA, clipBounds image.Rectangle, x, y, w, h float64, c color.NRGBA) {
	if w <= 0 || h <= 0 || c.A == 0 {
		return
	}
	bounds := canvas.Bounds().Intersect(clipBounds)
	startX := max(int(math.Floor(x)), bounds.Min.X)
	startY := max(int(math.Floor(y)), bounds.Min.Y)
	endX := min(int(math.Ceil(x+w)), bounds.Max.X)
	endY := min(int(math.Ceil(y+h)), bounds.Max.Y)
	if startX >= endX || startY >= endY {
		return
	}
	for py := startY; py < endY; py++ {
		for px := startX; px < endX; px++ {
			alphaBlendPixel(canvas, px, py, c)
		}
	}
}

// ── Particle markers ────────────────────────────────────────────────────────

type particleSprite struct {
	x, y, w, h int
}

func (s particleSprite) bounds() image.Rectangle {
	return image.Rect(s.x, s.y, s.x+s.w, s.y+s.h)
}

var particleSprites = map[string]particleSprite{
	"slice":   {x: 0, y: 0, w: 64, h: 64},
	"ball":    {x: 64, y: 0, w: 64, h: 64},
	"splat0":  {x: 128, y: 0, w: 64, h: 64},
	"splat1":  {x: 192, y: 0, w: 64, h: 64},
	"splat2":  {x: 256, y: 0, w: 64, h: 64},
	"smoke":   {x: 0, y: 64, w: 64, h: 64},
	"expl":    {x: 128, y: 64, w: 128, h: 128},
	"airjump": {x: 320, y: 64, w: 64, h: 64},
	"spiral":  {x: 0, y: 128, w: 128, h: 128},
	"spark":   {x: 128, y: 128, w: 128, h: 128},
}

func particleSpriteForTile(id uint8) (particleSprite, bool) {
	switch id {
	case TileUnlimitedJumpsOn:
		return particleSprites["airjump"], true
	case TileUnlimitedJumpsOff:
		return particleSprites["splat0"], true
	case TileJetpackOn:
		return particleSprites["smoke"], true
	case TileJetpackOff:
		return particleSprites["slice"], true
	case TileTeleGunEnable:
		return particleSprites["ball"], true
	case TileTeleGunDisable:
		return particleSprites["splat1"], true
	case TileTeleGrenadeEnable, TileTeleLaserEnable:
		return particleSprites["expl"], true
	case TileTeleGrenadeDisable, TileTeleLaserDisable:
		return particleSprites["splat2"], true
	case TileEHookEnable:
		return particleSprites["spiral"], true
	case TileEHookDisable:
		return particleSprites["slice"], true
	case TileHitEnable, TileNPHEnable, TileAllowTeleGun, TileAllowBlueTeleGun:
		return particleSprites["spark"], true
	case TileHitDisable, TileNPHDisable:
		return particleSprites["splat0"], true
	default:
		return particleSprite{}, false
	}
}

// renderParticles draws a non-animated particle marker pass on top of game/front
// tiles for selected DDNet capability/controller tiles.
func renderParticles(canvas *image.NRGBA, m *Map, ro *renderOptions, crop *MapBounds, tileLen uint32) {
	particles := resolveParticleImage()
	if particles == nil {
		return
	}

	tilePx := float64(tileLen)
	includeDetail := ro != nil && ro.detail

	for i := range m.Groups {
		g := &m.Groups[i]
		clipBounds := renderClipBounds(groupClipForRender(g), crop, tileLen, canvas.Bounds())
		if clipBounds.Empty() {
			continue
		}
		tileOffX, tileOffY, _, _, ok := computeGroupRenderOffsets(g, ro)
		if !ok {
			continue
		}

		for j := range g.Layers {
			l := &g.Layers[j]
			if l.Detail && !includeDetail {
				continue
			}
			if l.Kind != LayerKindGame && l.Kind != LayerKindFront {
				continue
			}
			for idx, t := range l.Tiles {
				spr, ok := particleSpriteForTile(t.ID)
				if !ok {
					continue
				}
				tx := float64(idx%l.Width) + tileOffX
				ty := float64(idx/l.Width) + tileOffY
				centerX := (tx - float64(crop.MinX) + 0.5) * tilePx
				centerY := (ty - float64(crop.MinY) + 0.5) * tilePx
				size := tilePx * 0.9
				blitSpriteRectClipped(canvas, particles, spr.bounds(), centerX-size/2.0, centerY-size/2.0, size, size, clipBounds)
			}
		}
	}
}

// scaleTileset scales a 16×16 tileset image so each tile is tl×tl pixels.
func scaleTileset(img *image.NRGBA, tl int) *image.NRGBA {
	targetW := tilesetGridSize * tl
	targetH := tilesetGridSize * tl
	if img.Bounds().Dx() == targetW && img.Bounds().Dy() == targetH {
		return img
	}
	return scaleNRGBA(img, targetW, targetH)
}

// alphaBlendPixel composites color c over the existing pixel at (x, y)
// using source-over alpha blending in non-premultiplied space.
func alphaBlendPixel(canvas *image.NRGBA, x, y int, c color.NRGBA) {
	off := canvas.PixOffset(x, y)
	pix := canvas.Pix
	sa := uint32(c.A)
	da := uint32(pix[off+3])
	outA := sa + da*(255-sa)/255
	if outA == 0 {
		return
	}
	pix[off+0] = uint8((uint32(c.R)*sa + uint32(pix[off+0])*da*(255-sa)/255) / outA)
	pix[off+1] = uint8((uint32(c.G)*sa + uint32(pix[off+1])*da*(255-sa)/255) / outA)
	pix[off+2] = uint8((uint32(c.B)*sa + uint32(pix[off+2])*da*(255-sa)/255) / outA)
	pix[off+3] = uint8(outA)
}
