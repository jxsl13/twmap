# twmap

[![Go Reference](https://pkg.go.dev/badge/github.com/jxsl13/twmap.svg)](https://pkg.go.dev/github.com/jxsl13/twmap)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Package `twmap` implements parsing, writing, validation, and thumbnail
generation for **Teeworlds 0.6**, **Teeworlds 0.7**, and **DDNet** map files.

The Teeworlds map format is a "datafile" container holding typed items
(metadata) and zlib-compressed data blocks (tile data, image data, etc.).
This package fully decodes all tile data, embedded images, quad layers,
envelopes, sounds, and map metadata, producing an in-memory `Map` struct
suitable for inspection, modification, writing, validation, or rendering.

## Features

- **Parse** — fully decode a `.map` file into groups, layers, tiles, quads,
  images, envelopes, sounds, and metadata. Supports both 0.6/DDNet and 0.7
  map formats (auto-detected from image item versions).
- **ParseInfo** — extract only map metadata (author, version, credits,
  license, DDNet settings) without decoding layers or images.
- **Write** — serialise a `Map` back into the Teeworlds datafile (v4)
  format, producing output loadable by TW/DDNet clients.
- **Validate** — verify structural integrity: checks the datafile container,
  map version, game layer presence, and DDNet special-layer consistency.
- **Render / RenderMap** — generate an `image.NRGBA` thumbnail with
  configurable bounding box, including tile flags (flip, rotate), layer
  colors, checkerboard background, and barycentric quad rasterization.
- **External tilesets** — optional `external/mapres` sub-package ships
  embedded PNGs for common DDNet/Teeworlds tilesets, registered
  automatically via blank import (like `image/png` and `image/jpeg`).
- **Entity-layer sprites** — optional `external/entities` sub-package embeds
  DDNet's `entities.png` overlay sheet for rendering game/front/tele/
  speedup/switch/tune layers.
- **Speedup arrow sprite** — optional `external/speeduparrow` sub-package
  embeds DDNet's `speed_arrow.png` for the speedup-layer arrow rendering path.
- **Game skin sprites** — optional `external/gameskin` sub-package embeds
  the DDNet game.png sprite sheet for rendering pickups, flags, and spawns
  with actual game sprites via `WithEntities(true)`. Override with your own
  skin via `RegisterGameSkin`.
- **Particle sprites** — optional `external/particles` sub-package embeds
  the DDNet particles.png sprite sheet.
- **RegisterExternalImage / RegisterEntitiesImage / RegisterSpeedupArrowImage / RegisterGameSkin** — public
  APIs for registering custom tilesets, entity overlay sheets, speedup-arrow assets,
  and game skins from your own packages.
- **Game-layer tile IDs** — exported constants for all DDNet game-layer
  tile types (`TileAir`, `TileSolid`, `TileFreeze`, …) and helper
  functions (`IsSolid`, `IsPassable`).

## Installation

```sh
go get github.com/jxsl13/twmap@latest
```

## Quick start

```go
package main

import (
    "fmt"
    "image/png"
    "log"
    "os"

    "github.com/jxsl13/twmap"
    _ "github.com/jxsl13/twmap/external" // register default tilesets
)

func main() {
    f, err := os.Open("mymap.map")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    // Parse the full map.
    m, err := twmap.Parse(f)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Author:", m.Info.Author)
    fmt.Println("Version:", m.Version) // MapVersion06 or MapVersion07
    fmt.Println("Groups:", len(m.Groups))
    fmt.Println("Images:", len(m.Images))

    // Generate a 800×600 thumbnail.
    thumb, err := twmap.RenderMap(m, twmap.WithMaxSize(800, 600))
    if err != nil {
        log.Fatal(err)
    }

    out, _ := os.Create("thumbnail.png")
    defer out.Close()
    png.Encode(out, thumb)
}
```

## API overview

### Parsing

| Function                                                 | Description                                                                |
| -------------------------------------------------------- | -------------------------------------------------------------------------- |
| `Parse(r io.Reader, opts ...ParseOption) (*Map, error)`  | Full parse — decodes all items, layers, tiles, quads, and embedded images. |
| `ParseInfo(r io.Reader) (Info, error)`                   | Lightweight parse — extracts only map metadata.                            |
| `WithRequireInfo(require bool) ParseOption`              | Controls whether a missing info item is an error (default: `true`).        |

### Writing

- `(*Map).Write(w io.Writer) error` — Serialise the map into the Teeworlds datafile (v4) format, written to `w`.

### Building maps from scratch

Thin constructors and fluent helpers assemble a `Map` with correct defaults
(reference ids set to the `-1` "none" sentinel, opaque-white tile color,
parallax 100/100, pre-sized air-filled tile slices). There is no parallel
builder type — the constructors return plain `Map`/`Group`/`Layer` values.

| Function                                          | Description                                                       |
| ------------------------------------------------- | ----------------------------------------------------------------- |
| `NewMap(v MapVersion) *Map`                       | Empty map; zero-value version defaults to `MapVersion06`.         |
| `NewGroup(name string) Group`                     | Group with default parallax 100/100 (normal scroll).             |
| `(*Map).AddGroup(g Group) *Group`                 | Append a group; returns a pointer for chaining.                   |
| `(*Group).AddLayer(l Layer) *Layer`               | Append a layer; returns a pointer for in-place tile edits.        |
| `NewTileLayer(name string, w, h int) Layer`       | Regular `w×h` visual tile layer, air-filled.                      |
| `NewGameLayer(w, h int) Layer`                    | `w×h` game (physics) layer, air-filled.                           |
| `NewFrontLayer(w, h int) Layer`                   | DDNet front layer (front tiles live in `Tiles`).                  |
| `NewTeleLayer / NewSpeedupLayer / NewSwitchLayer / NewTuneLayer(w, h int) Layer` | DDNet special layers with their matching special-tile grid. |
| `NewQuadsLayer(name string) Layer`                | Empty quad layer (append to `Quads`).                             |
| `(*Layer).SetTile(x, y int, t Tile)`              | Set a tile; panics if out of bounds.                             |
| `(*Layer).TileAt(x, y int) Tile`                  | Read a tile; panics if out of bounds.                            |
| `(*Layer).Fill(t Tile)`                           | Set every tile in the `Tiles` grid to `t`.                       |

```go
m := twmap.NewMap(twmap.MapVersion06)
g := m.AddGroup(twmap.NewGroup("Game"))

bg := g.AddLayer(twmap.NewTileLayer("bg", 50, 30))
bg.Fill(twmap.Tile{ID: twmap.TileSolid})

game := g.AddLayer(twmap.NewGameLayer(50, 30))
game.SetTile(0, 29, twmap.Tile{ID: twmap.TileSolid})

_ = m.Write(out) // serialise to a Teeworlds .map datafile
```

### Validation

- `Validate(r io.Reader, opts ...ParseOption) error` — Parses and validates the structural integrity of a map file.

**Validation checks:**

- Datafile container integrity (magic bytes, header, compressed data)
- Map version is 1
- Info item present with required fields (configurable via `WithRequireInfo`)
- All groups and layers parse successfully
- All images parse successfully
- Exactly one game layer exists
- DDNet special layers (teleport, speedup, front, switch, tune) share
  the game layer dimensions

### Rendering

- `Render(r io.Reader, opts ...RenderOption) (*image.NRGBA, error)` — Parse + render in one step.
- `RenderMap(m *Map, opts ...RenderOption) (*image.NRGBA, error)` — Render from an already-parsed `Map`.
- `(*Map).Bounds() MapBounds` — Bounding box (in tile coords) of all non-air tiles across renderable layers.
- `MapBounds{MinX, MinY, MaxX, MaxY int}` — Axis-aligned bounding box with `Width()` and `Height()` helpers.
- `WithMaxSize(maxW, maxH int) RenderOption` — Constrain output to maxW×maxH (default: native tileset resolution).
- `WithRegion(region MapBounds) RenderOption` — Render only a sub-section of the map.
- `WithParseOptions(opts ...ParseOption) RenderOption` — Pass parse options to `Render` (ignored by `RenderMap`).
- `RegisterExternalImage(name string, img *image.NRGBA)` — Register a tileset for use during rendering.
- `RegisterEntitiesImage(img *image.NRGBA)` — Register a DDNet entity-layer sprite sheet (`entities.png`).
- `RegisterSpeedupArrowImage(img *image.NRGBA)` — Register the DDNet speedup arrow image (`speed_arrow.png`).
- `WithEntities(entities bool) RenderOption` — Render game-layer entity sprites (pickups/flags and DDNet weapon-removal pickups) at DDNet proportions.
- `WithGameLayer(gameLayer bool) RenderOption` — Render the game layer only as an overlay.
- `WithFrontLayer(frontLayer bool) RenderOption` — Render the DDNet front layer as a semi-transparent entities overlay.
- `WithTeleLayer(teleLayer bool) RenderOption` — Render the DDNet tele layer.
- `WithSpeedupLayer(speedupLayer bool) RenderOption` — Render the DDNet speedup layer (requires a registered speedup-arrow asset for the arrow sprite).
- `WithSwitchLayer(switchLayer bool) RenderOption` — Render the DDNet switch layer.
- `WithTuneLayer(tuneLayer bool) RenderOption` — Render the DDNet tune layer.
- `WithOverlayEntities(val int) RenderOption` — Render the combined DDNet editor-style entity overlay (`cl_overlay_entities`) across game/front/tele/speedup/switch/tune.
- `WithParticles(particles bool) RenderOption` — Render a static (non-animated) particle/capability marker pass from particles.png.
- `WithInvalidTiles(invalid bool) RenderOption` — Render DDNet-editor-style diagnostics for problematic special-layer state where supported.
- `RegisterGameSkin(img *image.NRGBA)` — Register a custom game skin image (1024×512, 32×16 grid) for entity rendering.
- `RegisterParticleImage(img *image.NRGBA)` — Register a particle sprite sheet.

To make the default DDNet/Teeworlds assets available, add a blank import:

```go
import _ "github.com/jxsl13/twmap/external" // registers mapres + entities + speeduparrow + gameskin + particles
```

Or import only what you need:

```go
import _ "github.com/jxsl13/twmap/external/mapres"     // tileset images
import _ "github.com/jxsl13/twmap/external/entities"   // DDNet entity-layer overlay sheet
import _ "github.com/jxsl13/twmap/external/speeduparrow" // DDNet speedup arrow
import _ "github.com/jxsl13/twmap/external/gameskin"   // game skin (pickups, flags, spawns)
import _ "github.com/jxsl13/twmap/external/particles"  // particle sprites
```

This follows the same pattern as `image/png` and `image/jpeg`: each
sub-package's `init()` function registers its assets with the
corresponding `twmap.Register*` function. You can create your own
asset packages the same way, or call `RegisterGameSkin` directly to
override the default game skin with a custom one.

DDNet itself treats these asset families separately:

- `mapres` contains visual map tilesets used by regular tile layers.
- `entities.png` is a dedicated entity-layer overlay sheet used for game/front/tele/speedup/switch/tune visualization.
- `speed_arrow.png` is the DDNet speedup-arrow sprite used by speedup overlays.
- `game.png` is the runtime game skin used for pickups and flags.
- `particles.png` is the particle/effects sheet.

**Rendering details:**

- Without `WithCamera(...)`, only groups with parallax 100/100 are rendered.
  Group clipping is applied when present in the map data.
- Physics layers (game/front/tele/speedup/switch/tune) are excluded by
  default and can be enabled individually with dedicated options.
  Detail layers are also excluded by default (enable via `WithDetail(true)`).
- When `WithEntities(true)` is set and a game skin is registered, entity
  sprites (hearts, shields, weapons, flags) are drawn from the game skin
  at their DDNet client proportions (spanning multiple tiles). Without a
  game skin, entity sprites are not rendered. Spawns are not rendered
  as entity sprites — use `WithGameLayer(true)` to make them visible.
- When `WithGameLayer(true)` is set, only the game layer tiles (solid,
  hookable, freeze, spawns, checkpoints, etc.) are rendered as an overlay
  using the dedicated entity-layer sheet from `external/entities`.
- When `WithOverlayEntities(val)` is set, the combined DDNet editor-style
  entity overlay is rendered across game/front/tele/speedup/switch/tune,
  using DDNet-style overlay alpha semantics.
- `WithFrontLayer`, `WithTeleLayer`, `WithSpeedupLayer`, `WithSwitchLayer`,
  and `WithTuneLayer` enable rendering of the corresponding DDNet physics
  layers individually. Tele, switch, tune, and speedup overlays also render
  DDNet-style numeric labels when there is enough tile space available.
- `WithInvalidTiles(true)` keeps problematic speedup-layer state renderable as
  diagnostics even when the entry would normally disappear from the standard
  overlay path.
- When `WithParticles(true)` is set and a particle sheet is registered,
  static (non-animated) particle/capability markers are rendered.
- The output is cropped to the bounding box of non-air tiles (or the region
  specified via `WithRegion`) and, when `WithMaxSize` is used, scaled to fit
  within the requested dimensions while preserving aspect ratio.
  Without `WithMaxSize`, the output uses the native tileset resolution.
- A checkerboard background is drawn behind all layers.
- Tile flags (`VFlip`, `HFlip`, `Rotate`) are applied per-tile.
- Quad layers are rasterized with barycentric vertex-color and texture
  interpolation.

### Types

```text
Map
├── Version        — MapVersion06 (0.6/DDNet) or MapVersion07 (0.7)
├── Info           — Author, Version, Credits, License, Settings
├── Images[]       — Name, Width, Height, External, RGBA
├── Envelopes[]    — Name, Channels, Synchronized, Points[]
│   └── EnvPoint   — Time, CurveType, Values[4]
├── Sounds[]       — Name, Data (DDNet only)
└── Groups[]       — Name, Offset, Parallax, Clipping, ClipRect
    └── Layers[]
        ├── Tile layers   — Name, Width, Height, Color, ImageID, ColorEnv, Detail, Tiles[]
        │   └── Tile      — ID, Flags
        ├── DDNet layers  — TeleTiles[], SpeedupTiles[], SwitchTiles[], TuneTiles[]
        ├── Quad layers   — Name, Quads[], QuadImageID
        │   └── Quad      — Points[5], Colors[4], TexCoords[4], PosEnv, ColorEnv
        └── Sound layers  — Name, SoundSources[], SoundID
            └── SoundSource — Position, Loop, Panning, Delay, Falloff, Shape, Envelopes
```

#### Map version

- `MapVersion06` = `1` — Teeworlds 0.6 / DDNet
- `MapVersion07` = `2` — Teeworlds 0.7

#### Layer kinds

| Kind               | Description                  |
| ------------------ | ---------------------------- |
| `LayerKindTiles`   | Regular visual tilemap layer |
| `LayerKindGame`    | Game layer (physics)         |
| `LayerKindFront`   | DDNet front layer            |
| `LayerKindTele`    | DDNet teleport layer         |
| `LayerKindSpeedup` | DDNet speedup layer          |
| `LayerKindSwitch`  | DDNet switch layer           |
| `LayerKindTune`    | DDNet tune layer             |
| `LayerKindQuads`   | Quad layer                   |
| `LayerKindSounds`  | Sound layer                  |
| `LayerKindInvalid` | Unrecognised layer type      |

#### Helper methods

- `(*Layer).IsPhysics() bool` — True for game/front/tele/speedup/switch/tune layers.
- `(*Layer).IsTilemap() bool` — True for any tilemap-based layer (physics or regular).
- `(*Map).GameLayers() []Layer` — Returns all game layers found in the map.
- `(*Group).IsPhysicsGroup() bool` — True if the group contains any physics layers.

#### Tile flags

| Flag             | Value | Description     |
| ---------------- | ----- | --------------- |
| `TileFlagVFlip`  | 1     | Vertical flip   |
| `TileFlagHFlip`  | 2     | Horizontal flip |
| `TileFlagOpaque` | 4     | Opaque tile     |
| `TileFlagRotate` | 8     | 90° rotation    |

#### Game-layer tile IDs

The package exports constants for all DDNet game-layer tile types
(e.g. `TileAir`, `TileSolid`, `TileDeath`, `TileUnhookable`, `TileFreeze`,
`TileStart`, `TileFinish`, …) and two helper functions:

- `IsSolid(id uint8) bool` — True if the tile blocks player movement (solid or unhookable).
- `IsPassable(id uint8) bool` — True if a player can move through the tile (not solid/death/freeze).

#### Envelope curve types

| Constant      | Value | Description           |
| ------------- | ----- | --------------------- |
| `CurveStep`   | 0     | Step interpolation    |
| `CurveLinear` | 1     | Linear interpolation  |
| `CurveSlow`   | 2     | Slow-in               |
| `CurveFast`   | 3     | Fast-in               |
| `CurveSmooth` | 4     | Smooth interpolation  |
| `CurveBezier` | 5     | Bézier interpolation  |

### Sentinel errors

| Error                                | Description                                       |
| ------------------------------------ | ------------------------------------------------- |
| `ErrMissingVersion`                  | Map version item not found                        |
| `ErrInvalidVersion`                  | Map version is not 1                              |
| `ErrMissingInfo`                     | Map info item not found                           |
| `ErrNoGameLayer`                     | No game layer in the map                          |
| `ErrTooManyGameGroups`               | Game layers span multiple groups                  |
| `ErrTooManyGameLayers`               | Duplicate game or special layer                   |
| `ErrInconsistentGameLayerDimensions` | Special layers differ in size from the game layer |

## Testable examples

See [example_test.go](example_test.go) for runnable `Example` functions
recognised by `go test` and rendered on [pkg.go.dev](https://pkg.go.dev/github.com/jxsl13/twmap).

```sh
go test -v -run ^Example
```

## Building & checking

```sh
make        # runs go build ./... && go vet ./...
make build  # go build ./...
make vet    # go vet ./...
```

## License

[MIT](LICENSE) — Copyright (c) 2026 John Behm

### Embedded assets from DDNet

The image assets bundled in the `external/` sub-packages originate from the
[DDNet project](https://github.com/ddnet/ddnet) and are released under
**CC-BY-SA 3.0** ([creativecommons.org/licenses/by-sa/3.0/](https://creativecommons.org/licenses/by-sa/3.0/)) as stated
in DDNet's [license.txt](https://github.com/ddnet/ddnet/blob/master/license.txt).

| Package | Image(s) | Source path in DDNet | License file |
| ------- | -------- | -------------------- | ------------ |
| `external/gameskin` | `game.png` — game skin sprite sheet (pickups, flags, spawns) | [`data/game.png`](https://github.com/ddnet/ddnet/blob/master/data/game.png) | [external/gameskin/LICENSE](external/gameskin/LICENSE) |
| `external/mapres` | `*.png` — tileset images (grass, desert, jungle, winter, …) | [`data/mapres/`](https://github.com/ddnet/ddnet/tree/master/data/mapres) | [external/mapres/LICENSE](external/mapres/LICENSE) |
| `external/particles` | `particles.png` — particle sprite sheet | [`data/particles.png`](https://github.com/ddnet/ddnet/blob/master/data/particles.png) | [external/particles/LICENSE](external/particles/LICENSE) |
| `external/speeduparrow` | `speed_arrow.png` — DDNet speedup arrow sprite | [`data/editor/speed_arrow.png`](https://github.com/ddnet/ddnet/blob/master/data/editor/speed_arrow.png) | [external/speeduparrow/LICENSE](external/speeduparrow/LICENSE) |

## References

- [DDNet map format (ddnet-rs/twmap)](https://gitlab.com/ddnet-rs/twmap)
- [Teeworlds datafile spec](https://teeworlds.com/)
