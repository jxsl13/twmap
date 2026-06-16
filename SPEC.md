# SPEC — twmap map-builder API

## §G Goal

Add public helper API to build multi-layer TW/DDNet maps from scratch — correct defaults, sized tile slices, fluent layer/group assembly — output loadable by clients.

## §C Constraints

- C1 — Go stdlib only. No new deps (match existing module).
- C2 — Additive. No break existing exported types/funcs. Plain-struct model (`Map`/`Group`/`Layer`) stays; builders = thin ctors, no parallel type system.
- C3 — Built map must pass `Validate` + survive `Write`→`Parse` round-trip.
- C4 — Idiomatic Go: ctors return values, append-helpers return `*pointer` for chaining, tile accessors slice-like.
- C5 — Mirror reference design (rust twmap `new(shape)` + setters; python mapgen group→layer→tilearray).

## §I Surfaces

- I.builder — new file `builder.go`, package `twmap` (public API below).
- I.map — extends existing types in `map.go` (methods only, no field change).
- I.write — `(*Map).Write` already serializes; builder output feeds it unchanged.
- I.api.NewMap        — `NewMap(v MapVersion) *Map`
- I.api.NewGroup      — `NewGroup(name string) Group` (parallax 100/100)
- I.api.AddGroup      — `(*Map).AddGroup(g Group) *Group`
- I.api.NewTileLayer  — `NewTileLayer(name string, w, h int) Layer`
- I.api.NewGameLayer  — `NewGameLayer(w, h int) Layer`
- I.api.NewPhysLayer  — `NewFrontLayer/NewTeleLayer/NewSpeedupLayer/NewSwitchLayer/NewTuneLayer(w, h int) Layer`
- I.api.NewQuadsLayer — `NewQuadsLayer(name string) Layer`
- I.api.AddLayer      — `(*Group).AddLayer(l Layer) *Layer`
- I.api.SetTile       — `(*Layer).SetTile(x, y int, t Tile)`
- I.api.TileAt        — `(*Layer).TileAt(x, y int) Tile`
- I.api.Fill          — `(*Layer).Fill(t Tile)`
- I.api.tileIndex     — `(*Layer).tileIndex(x, y int) int` (unexported, y*W+x)
- I.api.NewSoundLayer — `NewSoundLayer(name string) Layer` (Kind=Sounds, SoundID=-1)
- I.api.SetTeleTile   — `(*Layer).SetTeleTile/TeleTileAt(x,y int[, t TeleTile])`
- I.api.SetSpeedup    — `(*Layer).SetSpeedupTile/SpeedupTileAt`
- I.api.SetSwitchTile — `(*Layer).SetSwitchTile/SwitchTileAt`
- I.api.SetTuneTile   — `(*Layer).SetTuneTile/TuneTileAt`
- I.api.NewQuad       — `NewQuad(cx, cy, w, h int) Quad` (tile units; white corners, unit texcoords)
- I.api.AddImage      — `(*Map).AddImage(name string, rgba *image.NRGBA) int` (embedded, returns index)
- I.api.AddExtImage   — `(*Map).AddExternalImage(name string, w, h int) int` (external ref, returns index)

## §V Invariants

- V1 — tile-layer `len(Tiles) == Width*Height` after every ctor + accessor. Never desync.
- V2 — ctor sets ref ids `ImageID/ColorEnv/QuadImageID/SoundID = -1` (no accidental index 0).
- V3 — `NewGroup` defaults `ParallaxX=ParallaxY=100` (normal scroll).
- V4 — `NewTileLayer` defaults color opaque white `255,255,255,255`.
- V5 — built map round-trips: `Write` then `Parse` yields struct-equal map (valid datafile v4).
- V6 — `NewMap` defaults `Version=MapVersion06` when arg zero-value; Info empty but writable.
- V7 — `SetTile`/`TileAt` out of `[0,Width)×[0,Height)` panic slice-style (bounds-checked, no silent clobber).
- V8 — physics ctors set `Kind` correct + alloc matching special-tile slice (`TeleTiles` len W*H for tele, etc.), tile-layer ctors alloc `Tiles` len W*H air.
- V9 — special-tile accessors (`SetTeleTile`/`SetSpeedupTile`/`SetSwitchTile`/`SetTuneTile` + `*At`) write/read matching special slice (not base `Tiles`), bounds-panic like V7.
- V10 — `NewQuad` default: 4 corners axis-aligned rect around center, `Points[4]`=center, all `Colors` opaque white, `TexCoords` unit square, `PosEnv=ColorEnv=-1`. Tile→raw factor confirmed: world 17.15 → 1 tile = 1<<15 raw (rendering `rawWorldCoordToTileFloat=raw/32768`); tex 22.10 → 1.0 = 1<<10 raw.
- V11 — `AddImage`/`AddExternalImage` append to `Map.Images`, return index == slice position. Embedded: `External=false`, `RGBA` set, W/H from image bounds. External: `External=true`, `RGBA=nil`.
- V12 — `NewSoundLayer` sets `Kind=Sounds`, `SoundID=-1`, empty `SoundSources`, ref ids -1.

## §T Tasks

id|status|task|cites
T1|x|add builder.go skeleton + NewMap/NewGroup/AddGroup|I.builder,I.api.NewMap,V3,V6
T2|x|NewTileLayer + NewGameLayer air-filled sized slice|I.api.NewTileLayer,V1,V2,V4,V8
T3|x|NewQuadsLayer + physics-layer ctors (front/tele/speedup/switch/tune)|I.api.NewQuadsLayer,I.api.NewPhysLayer,V2,V8
T4|x|AddLayer + tile accessors SetTile/TileAt/Fill/tileIndex|I.api.AddLayer,I.api.SetTile,V1,V7
T5|x|round-trip test: build map scratch → Write → Parse → Validate equal|V5,C3
T6|x|example_test ExampleNewMap + README API-overview section|I.builder,C2
T7|x|table tests for defaults/bounds-panic invariants|V1,V2,V3,V4,V7
T8|x|NewSoundLayer ctor|I.api.NewSoundLayer,V12
T9|x|special-tile accessors Set/At tele/speedup/switch/tune|I.api.SetTeleTile,I.api.SetSpeedup,I.api.SetSwitchTile,I.api.SetTuneTile,V9
T10|x|NewQuad helper (confirm tile→raw fxp factor vs rendering)|I.api.NewQuad,V10
T11|x|AddImage + AddExternalImage index wiring|I.api.AddImage,I.api.AddExtImage,V11
T12|x|tests + README/example for T8-T11|V9,V10,V11,V12,C2,C3

## §B Bugs

id|date|cause|fix
