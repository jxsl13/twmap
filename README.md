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
- **External tilesets** — optional `external` sub-package ships embedded
  PNGs for common DDNet/Teeworlds tilesets, registered automatically via
  blank import (like `image/png` and `image/jpeg`).
- **RegisterExternalImage** — public API for registering custom tilesets
  from your own packages.
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

| Function                         | Description                                                                  |
| -------------------------------- | ---------------------------------------------------------------------------- |
| `(*Map).Write(w io.Writer) error` | Serialise the map into the Teeworlds datafile (v4) format, written to `w`. |

### Validation

| Function                                          | Description                                                  |
| ------------------------------------------------- | ------------------------------------------------------------ |
| `Validate(r io.Reader, opts ...ParseOption) error` | Parses and validates the structural integrity of a map file. |

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

| Function / Type                                                            | Description                                                                                           |
| -------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `Render(r io.Reader, opts ...RenderOption) (*image.NRGBA, error)`          | Parse + render in one step.                                                                           |
| `RenderMap(m *Map, opts ...RenderOption) (*image.NRGBA, error)`            | Render from an already-parsed `Map`.                                                                  |
| `(*Map).Bounds() MapBounds`                                                | Bounding box (in tile coords) of all non-air tiles across renderable layers.                          |
| `MapBounds{MinX, MinY, MaxX, MaxY int}`                                    | Axis-aligned bounding box with `Width()` and `Height()` helpers.                                      |
| `WithMaxSize(maxW, maxH int) RenderOption`                                 | Constrain output to maxW×maxH (default: native tileset resolution).                                   |
| `WithRegion(region MapBounds) RenderOption`                                | Render only a sub-section of the map.                                                                 |
| `WithParseOptions(opts ...ParseOption) RenderOption`                       | Pass parse options to `Render` (ignored by `RenderMap`).                                              |
| `RegisterExternalImage(name string, img *image.NRGBA)`                     | Register a tileset for use during rendering.                                                          |

To make the default DDNet/Teeworlds tilesets available, add a blank import:

```go
import _ "github.com/jxsl13/twmap/external"
```

This follows the same pattern as `image/png` and `image/jpeg`: the
sub-package's `init()` function registers all its tilesets with
`twmap.RegisterExternalImage`. You can create your own tileset packages
the same way.

**Rendering details:**

- Only groups with parallax 100/100 and no clipping are rendered.
- Physics layers (game, tele, speedup, front, switch, tune) and detail
  layers are excluded.
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

| Constant       | Value | Description                     |
| -------------- | ----- | ------------------------------- |
| `MapVersion06` | 1     | Teeworlds 0.6 / DDNet          |
| `MapVersion07` | 2     | Teeworlds 0.7                  |

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

| Method / Function              | Description                                               |
| ------------------------------ | --------------------------------------------------------- |
| `(*Layer).IsPhysics() bool`    | True for game/front/tele/speedup/switch/tune layers.      |
| `(*Layer).IsTilemap() bool`    | True for any tilemap-based layer (physics or regular).    |
| `(*Map).GameLayers() []Layer`  | Returns all game layers found in the map.                 |
| `(*Group).IsPhysicsGroup() bool` | True if the group contains any physics layers.          |

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

| Function                    | Description                                                      |
| --------------------------- | ---------------------------------------------------------------- |
| `IsSolid(id uint8) bool`   | True if the tile blocks player movement (solid or unhookable).   |
| `IsPassable(id uint8) bool` | True if a player can move through the tile (not solid/death/freeze). |

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

Embedded tilesets in `external/mapres/` are subject to their own [license](external/mapres/LICENSE).

## References

- [DDNet map format (ddnet-rs/twmap)](https://gitlab.com/ddnet-rs/twmap)
- [Teeworlds datafile spec](https://teeworlds.com/)
