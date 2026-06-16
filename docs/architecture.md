# Architecture

## Scope

twmap has four main responsibilities:

1. Read the Teeworlds/DDNet datafile container.
2. Decode that container into a Go-native public model.
3. Validate and write the public model back to the map format.
4. Render the visual map design plus optional DDNet physics overlays.

The code is intentionally split so container handling, public model handling,
and rendering logic stay separable.

## Repository Layout

- datafile.go
  Low-level datafile container parsing and decompression.
  This code knows about headers, item tables, offsets, compressed payloads, and
  lazy data access.
- map.go
  Public model definitions plus container-item decoding into Map, Group, Layer,
  Quad, Envelope, SoundSource, and related types.
  This is the main translation boundary from map-format integers to Go-native
  types.
- write.go
  Reverse translation from the public model back into map items and data blocks.
  This file is the write-side counterpart to map.go.
- validate.go
  Structural validation on top of Parse.
  Validation is intentionally model-based, not byte-offset based.
- rendering.go
  Software renderer.
  This file contains the render pipeline, quad rasterization, tile compositing,
  DDNet overlay rendering, particle markers, and entity sprite rendering.
- tile_id.go
  Exported DDNet tile IDs and layer-specific validity helpers such as
  IsValidGameTile, IsValidTeleTile, and IsValidSwitchTile.
  The renderer uses these helpers to avoid interpreting the same raw numeric ID
  identically across unrelated layer types.
- external.go
  Runtime registries for external visual assets.
- external/
  Embedded default assets:
  - mapres: regular tilesets
  - entities: DDNet entity-layer sheet
  - gameskin: runtime entity sprites for pickups/flags
  - particles: particle/effect sprites
- cmd/render_tutorial/main.go
  Utility that renders the Tutorial map into a set of inspection variants.
  It now only emits variants for categories that actually change the output on
  the given map.

## Data Model Boundary

The public model prefers Go-native semantics over raw map-format integers where
that improves correctness or API clarity.

- Envelope.Channels uses the typed uint32 alias EnvelopeChannels.
- SoundSource.ShapeType uses the typed uint32 alias ShapeType.
- Time-like int32 fields are exposed as time.Duration:
  - EnvPoint.Time
  - Layer.ColorEnvOffset
  - Quad.PosEnvOffset
  - Quad.ColorEnvOffset
  - SoundSource.Delay
  - SoundSource.PosEnvOffset
  - SoundSource.SoundEnvOffset

The parser still reads the original int32 values from the map format first and
then converts them into the public model.
The writer performs the inverse conversion.

This keeps the file format compatibility layer localized while the public API
stays idiomatic for Go callers.

## Render Pipeline

RenderMap follows this high-level order:

1. collectRenderSteps
   Collect regular tile and quad layers for the base design pass.
   Physics layers are excluded here.
2. collectOverlayRenderLayers
   Collect optional DDNet physics overlays separately.
   This pass is layer-type aware.
3. cropToNonAir
   Compute the world-tile crop bounds from both base layers and enabled overlay
   layers.
4. prepareTilesets / prepareQuadImages
   Scale source assets to the current tile size.
5. renderAllSteps
   Render normal tile layers and quads.
6. renderParticles
   Render static particle/capability markers.
7. renderEntities
   Render game-skin based pickups/flags from the game layer.
8. renderOverlayLayers
   Render DDNet physics overlays last, matching the editor expectation that
   game/front/tele/switch/tune/speedup stay above the rest.

## Where Specific Things Render

- Regular tile layers:
  collectRenderSteps -> renderAllSteps -> renderSingleTileLayer
- Quads:
  collectRenderSteps -> renderAllSteps -> renderSingleQuadLayer ->
  renderQuadOnCanvas -> rasterizeTriangle
- Particle markers:
  renderParticles
- Game-skin pickups and flags:
  renderEntities
- DDNet physics overlays:
  renderOverlayLayers
  - game/front/tele/switch/tune use the entities sheet
  - speedup uses a dedicated procedural arrow path
  - tele/switch/tune/speedup text labels are drawn by an internal bitmap-font
    overlay pass when tile size permits

This answers the practical debugging questions "where are quads rendered?" and
"where are particles rendered?" directly:

- Quads are rendered in the quad branch of renderAllSteps.
- Particles are rendered in renderParticles before entities and before the
  physics overlay pass.

## Overlay Architecture

The renderer does not treat all physics layers as generic tile layers.
Instead it keeps them in a dedicated overlay collection so it can apply
DDNet-specific rules:

- Game layer tiles are filtered with IsValidGameTile.
- Front layer tiles are filtered with IsValidFrontTile.
- Tele layer tiles are filtered with IsValidTeleTile.
- Switch layer tiles are filtered with IsValidSwitchTile and timed-open is
  remapped like DDNet does.
- Tune layer tiles are filtered with IsValidTuneTile.
- Speedup tiles are only considered active if the layer data actually carries a
  usable force/max-speed payload.
- Tele, switch, tune, and speedup text labels are rendered in a separate text
  sub-pass so they stay available even if the entities texture is not
  registered.

This separation exists because several raw numeric tile IDs overlap between
different DDNet layer types.
Without the layer-specific validity helpers the renderer can easily show the
wrong semantic symbol for the same byte value.

## Parallax And Offsets

computeGroupRenderOffsets is the single place where group offset/parallax state
is translated into render-space offsets.

- Base tile layers and quads share the same group-offset calculation.
- Tile rendering keeps fractional offsets so sub-tile parallax shifts do not
  get truncated away.
- Entity and particle rendering now also uses the same group offset path, so
  they stay aligned with the rest of the group.

## Writer Symmetry

Write symmetry is a deliberate design constraint:

- map.go converts file-format primitives into the public model.
- write.go converts the same fields back into file-format primitives.
- map_test.go exercises write/parse round-trips.

When changing the public model, the parse and write side should be updated in
the same change unless the field is explicitly read-only.

## Decision Log

Future architectural decisions must be appended here with an ISO 8601
timestamp and a short rationale.

- 2026-04-12T00:00:00Z
  Public time-like int32 fields were moved to time.Duration in the public model.
  Rationale: callers should not have to remember which values are milliseconds
  or seconds when Go already has a native duration type.
- 2026-04-12T00:00:00Z
  Envelope channel count and sound shape type were changed to typed uint32
  aliases (EnvelopeChannels and ShapeType).
  Rationale: these are small closed enums and should not travel through the API
  as anonymous integers.
- 2026-04-12T00:00:00Z
  DDNet physics layers were split from the normal base pass into a dedicated
  overlay pass that renders last.
  Rationale: this matches the editor layering model more closely and avoids the
  old problem where entity sprites could visually cover the game layer.
- 2026-04-12T00:00:00Z
  Layer-type-specific validity helpers were introduced and used by the renderer.
  Rationale: raw tile IDs are not globally unique in meaning across game/front/
  tele/switch/tune/speedup layers.
- 2026-04-12T00:00:00Z
  Tele, switch, tune, and speedup overlays gained a dedicated numeric text
  sub-pass implemented with an internal bitmap font.
  Rationale: the renderer needed DDNet-style layer numbers without introducing a
  full text-renderer dependency into the software rendering pipeline.

## External References Used For The Current Design

The current DDNet-aligned renderer decisions were derived from these DDNet
source areas:

- src/game/mapitems.cpp
- src/game/map/render_layer.cpp
- src/game/map/render_map.cpp
- src/game/editor/map_view.cpp

Those references drove the layer-validity rules, overlay ordering, timed-switch
remapping, and the distinction between generic entity-sheet overlays and the
special speedup arrow path.
