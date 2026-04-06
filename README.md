# twmap

[![Go Reference](https://pkg.go.dev/badge/github.com/jxsl13/twmap.svg)](https://pkg.go.dev/github.com/jxsl13/twmap)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Package `twmap` implements parsing, validation, and thumbnail generation for
**Teeworlds 0.6.x** and **DDNet** map files.

The Teeworlds map format is a "datafile" container holding typed items
(metadata) and zlib-compressed data blocks (tile data, image data, etc.).
This package fully decodes all tile data, embedded images, quad layers, and
map metadata, producing an in-memory `Map` struct suitable for inspection,
validation, or rendering.

## Features

- **Parse** — fully decode a `.map` file into groups, layers, tiles, quads,
  images, and metadata.
- **ParseInfo** — extract only map metadata (author, version, credits,
  license) without decoding layers or images.
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
    fmt.Println("Groups:", len(m.Groups))
    fmt.Println("Images:", len(m.Images))

    // Generate a 800×600 thumbnail.
    thumb, err := twmap.RenderMap(m, 800, 600)
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

| Function                               | Description                                                                |
| -------------------------------------- | -------------------------------------------------------------------------- |
| `Parse(r io.Reader) (*Map, error)`     | Full parse — decodes all items, layers, tiles, quads, and embedded images. |
| `ParseInfo(r io.Reader) (Info, error)` | Lightweight parse — extracts only map metadata.                            |

### Validation

| Function                      | Description                                                  |
| ----------------------------- | ------------------------------------------------------------ |
| `Validate(r io.Reader) error` | Parses and validates the structural integrity of a map file. |

**Validation checks:**

- Datafile container integrity (magic bytes, header, compressed data)
- Map version is 1
- Info item present with required fields
- All groups and layers parse successfully
- All images parse successfully
- Exactly one game layer exists
- DDNet special layers (teleport, speedup, front, switch, tune) share
  the game layer dimensions

### Rendering

| Function                                                    | Description                                  |
| ----------------------------------------------------------- | -------------------------------------------- |
| `Render(r io.Reader, maxW, maxH int) (*image.NRGBA, error)` | Parse + render in one step.                  |
| `RenderMap(m *Map, maxW, maxH int) (*image.NRGBA, error)`   | Render from an already-parsed `Map`.         |
| `RegisterExternalImage(name string, img *image.NRGBA)`      | Register a tileset for use during rendering. |

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
- The output is cropped to the bounding box of non-air tiles and scaled
  to fit within the requested dimensions while preserving aspect ratio.
- A checkerboard background is drawn behind all layers.
- Tile flags (`VFlip`, `HFlip`, `Rotate`) are applied per-tile.
- Quad layers are rasterized with barycentric vertex-color and texture
  interpolation.

### Types

```text
Map
├── Info        — Author, Version, Credits, License
├── Images[]    — Name, Width, Height, External, RGBA
└── Groups[]    — Name, Offset, Parallax, Clipping, ClipRect
    └── Layers[]
        ├── Tile layers  — Width, Height, Color, ImageID, Tiles[]
        │   └── Tile     — ID, Flags
        └── Quad layers  — Quads[], QuadImageID
            └── Quad     — Points[5], Colors[4], TexCoords[4]
```

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

#### Tile flags

| Flag             | Value | Description     |
| ---------------- | ----- | --------------- |
| `TileFlagVFlip`  | 1     | Vertical flip   |
| `TileFlagHFlip`  | 2     | Horizontal flip |
| `TileFlagOpaque` | 4     | Opaque tile     |
| `TileFlagRotate` | 8     | 90° rotation    |

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
