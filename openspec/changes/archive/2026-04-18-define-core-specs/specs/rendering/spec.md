## ADDED Requirements

### Requirement: Render and RenderMap produce an NRGBA image
`Render` SHALL parse the provided `io.Reader` and render the resulting map. `RenderMap` SHALL render an already-parsed `*Map`. Both SHALL return an `*image.NRGBA`. When no renderable content exists, both SHALL return a 1×1 image.

#### Scenario: Render from reader
- **WHEN** `Render` is called with a valid map reader
- **THEN** it SHALL return an `*image.NRGBA` and nil error

#### Scenario: RenderMap from parsed map
- **WHEN** `RenderMap` is called with a valid `*Map`
- **THEN** it SHALL return an `*image.NRGBA` and nil error

#### Scenario: Empty map
- **WHEN** a map has no renderable layers, no enabled overlays, no entities, and no particles
- **THEN** the result SHALL be a 1×1 `*image.NRGBA`

### Requirement: Pipeline ordering
The render pipeline SHALL execute in this order:
1. Collect design layers (tile and quad) in back-to-front order
2. Collect DDNet entity layers (game, front, tele, speedup, switch, tune) separately
3. Determine crop region from non-air tiles across both design layers and enabled DDNet entity layers
4. Determine tile resolution
5. Prepare tilesets and quad images (scaled to tile resolution)
6. Fill canvas with checkerboard background
7. Render design layers (tiles and quads in collected order)
8. Render particles (if enabled)
9. Render entity sprites (if enabled)
10. Render DDNet entity layers last
11. Scale to output size (if `WithMaxSize` specified)

#### Scenario: Overlays render above design layers
- **WHEN** a map has both design tile layers and enabled DDNet entity layers
- **THEN** the DDNet entity layers SHALL be composited on top of all design layers

#### Scenario: Entities render above particles
- **WHEN** both `WithParticles(true)` and `WithEntities(true)` are set
- **THEN** entity sprites SHALL be composited on top of particle markers

#### Scenario: Particles render above design layers
- **WHEN** `WithParticles(true)` is set
- **THEN** particle markers SHALL be composited on top of design layers but below entity sprites and overlays

### Requirement: Checkerboard background
The canvas SHALL be filled with a DDNet-editor-style checkerboard pattern before any layers are rendered. Light squares SHALL use RGB(186,186,186) and dark squares SHALL use RGB(153,153,153). Both SHALL have full opacity (A=255).

#### Scenario: Transparent area shows checkerboard
- **WHEN** no layer covers a canvas region
- **THEN** the checkerboard pattern SHALL be visible

### Requirement: Crop to non-air bounding box
By default, the output SHALL be cropped to the bounding box of all non-air tiles across design layers, enabled DDNet entity layers, and any game/front source layers needed by enabled entity-sprite or particle passes. Group clipping rectangles SHALL constrain the bounding box contribution of clipped groups. An explicit `WithRegion` SHALL override the computed bounding box.

#### Scenario: Auto crop
- **WHEN** no region is specified
- **THEN** the output SHALL span exactly the bounding box of non-air tiles

#### Scenario: Explicit region
- **WHEN** `WithRegion` is set
- **THEN** the output SHALL cover exactly the specified tile region regardless of tile content

### Requirement: WithMaxSize constrains output dimensions
When `WithMaxSize(w, h)` is set, the output SHALL be scaled to fit within `w × h` pixels while preserving aspect ratio. Without `WithMaxSize`, the output SHALL use a single global pixels-per-tile value derived from the referenced design-layer tileset images.

#### Scenario: Scaling to fit
- **WHEN** `WithMaxSize(800, 600)` is set on a map wider than 800 native pixels
- **THEN** the output width SHALL be at most 800 and height at most 600

#### Scenario: Native resolution
- **WHEN** no `WithMaxSize` is set
- **THEN** the output width and height SHALL equal the cropped tile width and height multiplied by one global pixels-per-tile value
- **THEN** that pixels-per-tile value SHALL be computed as `max(image.Width, image.Height) / 16` from the largest referenced design-layer tileset image
- **THEN** if no referenced design-layer tileset image exists, that pixels-per-tile value SHALL default to 64

### Requirement: Only parallax 100/100 groups rendered by default
Without a camera set, only groups with `ParallaxX=100` and `ParallaxY=100` SHALL be included in the design pass. Groups with other parallax values SHALL be skipped.

#### Scenario: Non-parallax group excluded
- **WHEN** no camera is set and a group has `ParallaxX=50`
- **THEN** that group's layers SHALL NOT be rendered

### Requirement: WithCamera enables parallax rendering
When `WithCamera(cx, cy)` is set, groups with any parallax value SHALL be rendered. The effective tile offset for each group SHALL be computed as `camera * (1 - parallax/100) - group_offset`, matching DDNet world-mapping projection.

#### Scenario: Parallax group included with camera
- **WHEN** `WithCamera` is set
- **THEN** a group with `ParallaxX=50` SHALL be rendered with a shifted offset

### Requirement: Group clipping
When a group has `Clipping=true` and `ClipW > 0` and `ClipH > 0`, all layers in that group SHALL be rendered only within the clip rectangle defined by `ClipX`, `ClipY`, `ClipW`, `ClipH` in game-pixel coordinates. Pixels outside the clip rectangle SHALL NOT be drawn.

#### Scenario: Clipped group
- **WHEN** a group has clipping enabled with a clip rect covering only the left half
- **THEN** layer content in the right half SHALL NOT be rendered

### Requirement: Detail layers excluded by default
Layers with `Detail=true` SHALL be excluded from rendering unless `WithDetail(true)` is set.

#### Scenario: Detail layer hidden
- **WHEN** `WithDetail` is not set
- **THEN** detail layers SHALL NOT be rendered

#### Scenario: Detail layer shown
- **WHEN** `WithDetail(true)` is set
- **THEN** detail layers SHALL be rendered

### Requirement: WithoutBaseLayerKinds excludes specific layer kinds
`WithoutBaseLayerKinds(kinds...)` SHALL prevent the specified `LayerKind` values from being included in the design layer pass.

#### Scenario: Exclude tile layers
- **WHEN** `WithoutBaseLayerKinds(LayerKindTiles)` is set
- **THEN** regular tile layers SHALL NOT be rendered in the design pass

### Requirement: Tile flag transforms
Tile rendering SHALL respect the `Flags` field on each tile. `VFlip` (1) SHALL flip horizontally, `HFlip` (2) SHALL flip vertically, `Rotate` (8) SHALL rotate 90° clockwise. Combinations SHALL be applied in the DDNet-compatible transform order.

#### Scenario: Rotated tile
- **WHEN** a tile has `Flags = Rotate`
- **THEN** the tile SHALL be drawn rotated 90° clockwise

#### Scenario: Combined flags
- **WHEN** a tile has `Flags = VFlip | HFlip`
- **THEN** the tile SHALL be drawn flipped both horizontally and vertically

### Requirement: Quad rasterization with barycentric interpolation
Quad layers SHALL be rasterized by splitting each quad into two triangles and using barycentric interpolation for vertex colors and texture coordinates. When a quad has an associated image, texture sampling SHALL be performed with bilinear filtering. When no image is set, the quad SHALL be rendered using vertex colors only.

#### Scenario: Textured quad
- **WHEN** a quad has an associated image
- **THEN** texture pixels SHALL be sampled with bilinear interpolation modulated by vertex colors

#### Scenario: Vertex-color-only quad
- **WHEN** a quad has `QuadImageID = -1`
- **THEN** the quad SHALL be rendered with interpolated vertex colors only

### Requirement: Layer color modulation
Tile layers SHALL multiply each pixel by the layer's RGBA color (`ColorR`, `ColorG`, `ColorB`, `ColorA`). Alpha composition SHALL use standard source-over blending.

#### Scenario: Semi-transparent layer
- **WHEN** a tile layer has `ColorA = 128`
- **THEN** all tile pixels in that layer SHALL be rendered at half opacity

### Requirement: Missing tileset fallback
When a tile layer references an image that is not available (neither embedded nor registered via `RegisterExternalImage`), the renderer SHALL use a solid white fallback tileset.

#### Scenario: Unregistered external image
- **WHEN** a tile layer references an external image not registered
- **THEN** the layer SHALL render using white tiles

### Requirement: WithEntities renders game-skin entity sprites
When `WithEntities(true)` is set and a game skin is registered, entity sprites (pickups, flags, DDNet weapon-removal pickups) SHALL be drawn from the game skin at DDNet client proportions. Spawns SHALL NOT be rendered as entity sprites. Without a registered game skin, no entity sprites SHALL be rendered.

#### Scenario: Entities with game skin
- **WHEN** `WithEntities(true)` is set and a game skin is registered
- **THEN** pickup and flag tiles in the game layer SHALL be drawn as entity sprites from the skin

#### Scenario: Entities without game skin
- **WHEN** `WithEntities(true)` is set but no game skin is registered
- **THEN** no entity sprites SHALL be drawn

#### Scenario: Spawns excluded
- **WHEN** a game layer tile is a spawn tile
- **THEN** it SHALL NOT be rendered by the entity pass regardless of game skin registration

### Requirement: WithParticles renders static particle markers
When `WithParticles(true)` is set and a particle image is registered, static particle/capability markers SHALL be rendered. Without a registered particle image, the particle pass SHALL be skipped.

#### Scenario: Particles with registered image
- **WHEN** `WithParticles(true)` is set and a particle image is registered
- **THEN** particle markers SHALL be rendered

#### Scenario: Particles without registered image
- **WHEN** no particle image is registered
- **THEN** the particle pass SHALL be skipped silently

### Requirement: WithGameLayer renders game layer overlay
When `WithGameLayer(true)` is set, the game layer, a DDNet entity layer, SHALL be rendered as an overlay using the entities sprite sheet. This requires a registered entities image. The game layer overlay SHALL also render a death-border effect for tiles outside the game layer bounds within the crop region.

#### Scenario: Game layer with entities sheet
- **WHEN** `WithGameLayer(true)` is set and an entities image is registered
- **THEN** game layer tiles SHALL be rendered as an overlay

#### Scenario: Death border
- **WHEN** `WithGameLayer(true)` is set and the crop region extends beyond the game layer bounds
- **THEN** tiles outside the game layer SHALL be drawn using the death tile from the entities sheet

### Requirement: WithFrontLayer renders front layer overlay
When `WithFrontLayer(true)` is set, the front layer, a DDNet entity layer, SHALL be rendered as an overlay using the entities sprite sheet.

#### Scenario: Front layer rendered
- **WHEN** `WithFrontLayer(true)` is set and an entities image is registered
- **THEN** front layer tiles SHALL be visible

### Requirement: WithTeleLayer renders tele layer overlay
When `WithTeleLayer(true)` is set, the tele layer, a DDNet entity layer, SHALL be rendered as an overlay using the entities sprite sheet. Numeric labels (tele number) SHALL be rendered when tile size permits.

#### Scenario: Tele layer with numbers
- **WHEN** `WithTeleLayer(true)` is set and tile size is sufficient
- **THEN** tele tiles SHALL show their number as a text label

### Requirement: WithSpeedupLayer renders speedup layer overlay
When `WithSpeedupLayer(true)` is set, the speedup layer, a DDNet entity layer, SHALL be rendered. The speedup arrow requires a registered speedup-arrow asset. Without the asset, valid speedup tiles SHALL NOT be rendered. Numeric labels (force, max speed) for valid speedup tiles SHALL be rendered when the arrow asset is registered and tile size permits. When `WithInvalidTiles(true)` is also set, invalid speedup tiles MAY still render diagnostic text without the arrow asset.

#### Scenario: Speedup with arrow asset
- **WHEN** `WithSpeedupLayer(true)` is set and a speedup-arrow image is registered
- **THEN** speedup tiles SHALL show a rotated arrow sprite and numeric labels

#### Scenario: Speedup without arrow asset
- **WHEN** no speedup-arrow image is registered
- **THEN** valid speedup tiles SHALL NOT be rendered

### Requirement: WithSwitchLayer renders switch layer overlay
When `WithSwitchLayer(true)` is set, the switch layer, a DDNet entity layer, SHALL be rendered using the entities sprite sheet. Timed-open switch tiles SHALL be remapped to the open visual as DDNet does. Numeric labels (number, delay) SHALL be rendered when tile size permits and the tile type uses those fields.

#### Scenario: Switch layer with labels
- **WHEN** `WithSwitchLayer(true)` is set and tile size is sufficient
- **THEN** switch tiles SHALL show number and delay labels where applicable

### Requirement: WithTuneLayer renders tune layer overlay
When `WithTuneLayer(true)` is set, the tune layer, a DDNet entity layer, SHALL be rendered using the entities sprite sheet. Numeric labels (tune number) SHALL be rendered when tile size permits.

#### Scenario: Tune layer with number
- **WHEN** `WithTuneLayer(true)` is set and tile size is sufficient
- **THEN** tune tiles SHALL show their tune number

### Requirement: WithOverlayEntities sets combined entity overlay mode
When `WithOverlayEntities(val)` is set with `val > 0`:
- All DDNet entity layers (game, front, tele, speedup, switch, tune) SHALL be enabled automatically.
- Design layer alpha SHALL be multiplied by `(100 - val) / 100`.
- Overlay tile alpha SHALL also be multiplied by `(100 - val) / 100`.
- At `val = 100`, design layers SHALL be skipped entirely.
- `val` SHALL be clamped to the range 0–100.

#### Scenario: Overlay entities at 50
- **WHEN** `WithOverlayEntities(50)` is set
- **THEN** design layers and overlay tiles SHALL both render at 50% opacity

#### Scenario: Overlay entities at 100
- **WHEN** `WithOverlayEntities(100)` is set
- **THEN** design layers SHALL NOT be rendered

#### Scenario: Overlay enables all DDNet entity layers
- **WHEN** `WithOverlayEntities(50)` is set
- **THEN** game, front, tele, speedup, switch, and tune DDNet entity layers SHALL all be enabled

### Requirement: WithInvalidTiles enables diagnostic rendering
When `WithInvalidTiles(true)` is set, speedup tiles with invalid tile IDs but non-zero data (force, max speed, angle) SHALL be rendered as diagnostic markers. Without this option, such tiles SHALL be hidden.

#### Scenario: Invalid speedup visible with diagnostics
- **WHEN** a speedup tile has an unrecognized ID but non-zero force/angle, and `WithInvalidTiles(true)` is set
- **THEN** the tile SHALL be rendered with diagnostic text (force, max speed, angle)

#### Scenario: Invalid speedup hidden without diagnostics
- **WHEN** a speedup tile has an unrecognized ID and `WithInvalidTiles` is not set
- **THEN** the tile SHALL NOT be rendered

### Requirement: Overlay text labels use internal bitmap font
Tele, switch, tune, and speedup overlays SHALL render numeric text labels using an internal bitmap font when tile size is at least the minimum threshold. The text pass SHALL not depend on external font assets.

#### Scenario: Sufficient tile size
- **WHEN** overlay layers are enabled and tile size meets the minimum threshold
- **THEN** numeric labels SHALL be rendered

#### Scenario: Insufficient tile size
- **WHEN** tile size is below the minimum threshold
- **THEN** numeric labels SHALL NOT be rendered

### Requirement: Accepted limitation — text rendering approximation
Overlay text labels use an internal bitmap font, not DDNet's FreeType-based text renderer. Visual parity with the DDNet editor is approximate, not pixel-identical. This is a known and accepted limitation.

#### Scenario: Text is readable but not pixel-identical
- **WHEN** overlay text labels are rendered
- **THEN** they SHALL be legible at sufficient tile sizes but are NOT required to match DDNet editor output pixel-for-pixel
