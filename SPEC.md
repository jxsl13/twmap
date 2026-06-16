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

## §V Invariants

- V1 — tile-layer `len(Tiles) == Width*Height` after every ctor + accessor. Never desync.
- V2 — ctor sets ref ids `ImageID/ColorEnv/QuadImageID/SoundID = -1` (no accidental index 0).
- V3 — `NewGroup` defaults `ParallaxX=ParallaxY=100` (normal scroll).
- V4 — `NewTileLayer` defaults color opaque white `255,255,255,255`.
- V5 — built map round-trips: `Write` then `Parse` yields struct-equal map (valid datafile v4).
- V6 — `NewMap` defaults `Version=MapVersion06` when arg zero-value; Info empty but writable.
- V7 — `SetTile`/`TileAt` out of `[0,Width)×[0,Height)` panic slice-style (bounds-checked, no silent clobber).
- V8 — physics ctors set `Kind` correct + alloc matching special-tile slice (`TeleTiles` len W*H for tele, etc.), tile-layer ctors alloc `Tiles` len W*H air.

## §T Tasks

id|status|task|cites
T1|x|add builder.go skeleton + NewMap/NewGroup/AddGroup|I.builder,I.api.NewMap,V3,V6
T2|x|NewTileLayer + NewGameLayer air-filled sized slice|I.api.NewTileLayer,V1,V2,V4,V8
T3|x|NewQuadsLayer + physics-layer ctors (front/tele/speedup/switch/tune)|I.api.NewQuadsLayer,I.api.NewPhysLayer,V2,V8
T4|x|AddLayer + tile accessors SetTile/TileAt/Fill/tileIndex|I.api.AddLayer,I.api.SetTile,V1,V7
T5|x|round-trip test: build map scratch → Write → Parse → Validate equal|V5,C3
T6|x|example_test ExampleNewMap + README API-overview section|I.builder,C2
T7|x|table tests for defaults/bounds-panic invariants|V1,V2,V3,V4,V7

## §B Bugs

id|date|cause|fix
