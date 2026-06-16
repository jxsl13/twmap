## ADDED Requirements

### Requirement: Write produces a datafile matching the map version
`(*Map).Write` SHALL serialize the map into a Teeworlds datafile version 4 container with zlib-compressed data blocks, written to the provided `io.Writer`. The image item version written for each image SHALL correspond to `Map.Version`: version 1 for `MapVersion06`, version 2 for `MapVersion07`. TW 0.7 images SHALL additionally include the pixel-format field. The output SHALL be loadable by the TW/DDNet client that matches the map version.

#### Scenario: Write a MapVersion06 map
- **WHEN** `Write` is called on a `Map` with `Version = MapVersion06`
- **THEN** all image items SHALL use image-item version 1 and the output SHALL be loadable by TW 0.6/DDNet clients

#### Scenario: Write a MapVersion07 map
- **WHEN** `Write` is called on a `Map` with `Version = MapVersion07`
- **THEN** all image items SHALL use image-item version 2 with a pixel-format field and the output SHALL be loadable by TW 0.7 clients

### Requirement: Write serializes all map structures
`Write` SHALL serialize the version item (always 1), info item with all string fields, image items with embedded pixel data, envelope items with all control points, sound items, group items, and layer items. In the final datafile, items SHALL be grouped by item type in the container, while preserving the map's original order within each type family. The image item layout SHALL match the map version (6-field items for 0.6, 7-field items for 0.7).

#### Scenario: Complete structure serialization
- **WHEN** a `Map` with info, images, envelopes, groups, layers, and sounds is written
- **THEN** the output SHALL contain items for every structure, with original order preserved within each item type family

#### Scenario: DDNet special layer data preserved
- **WHEN** a map with tele, speedup, switch, and tune layers is written
- **THEN** each special layer's tile data SHALL be serialized using the DDNet-specific encoding (TeleTile: 2 bytes, SpeedupTile: 6 bytes, SwitchTile: 4 bytes, TuneTile: 2 bytes)

### Requirement: Semantic roundtrip fidelity
For any valid map input `B`, `Parse(Write(Parse(B)))` SHALL produce a `Map` semantically equal to `Parse(B)`. Byte-level identity of the written file is explicitly NOT required because zlib compression output is implementation-dependent. Semantic equality means every public `Map` field compares equal after a write/re-parse cycle.

#### Scenario: Info roundtrip
- **WHEN** a `Map` with Author, Version, Credits, License, and Settings is written and re-parsed
- **THEN** all `Info` fields SHALL be identical

#### Scenario: Tile data roundtrip
- **WHEN** a tile layer with specific ID and Flags values is written and re-parsed
- **THEN** every `Tile` SHALL have identical ID and Flags

#### Scenario: DDNet tile data roundtrip
- **WHEN** a map with TeleTiles, SpeedupTiles, SwitchTiles, and TuneTiles is written and re-parsed
- **THEN** every special tile SHALL have identical field values

#### Scenario: Quad data roundtrip
- **WHEN** quads with specific points, colors, and tex coords are written and re-parsed
- **THEN** points, colors, tex coords, and envelope references SHALL be preserved

#### Scenario: Image pixel data roundtrip
- **WHEN** an embedded image with specific RGBA data is written and re-parsed
- **THEN** the pixel data SHALL be byte-identical

#### Scenario: Map version roundtrip
- **WHEN** a `MapVersion07` map is written and re-parsed
- **THEN** the re-parsed `Map.Version` SHALL be `MapVersion07`

#### Scenario: Group and layer structure roundtrip
- **WHEN** a map with multiple groups, parallax, clipping, and DDNet special layers is written and re-parsed
- **THEN** group count, layer count, layer kinds, dimensions, and all group fields SHALL be identical

#### Scenario: Envelope roundtrip
- **WHEN** a map with envelopes is written and re-parsed
- **THEN** envelope count, channels, point count, point times, curve types, and values SHALL be identical

### Requirement: Fixed-point values use plain int raw representation
Point coordinates and texture coordinates SHALL use plain `int` fields as source-of-truth, carrying raw datafile fixed-point values (`int32` domain). Position values SHALL represent 17.15 fixed-point raw integers and texture coordinates SHALL represent 22.10 fixed-point raw integers. `Write` SHALL serialize these raw values directly via range-safe cast to `int32`.

#### Scenario: Position raw value preserved
- **WHEN** a quad point raw 17.15 value is parsed and written again without modification
- **THEN** the same raw `int32` value SHALL be serialized

#### Scenario: Texture raw value preserved
- **WHEN** a texture coordinate raw 22.10 value is parsed and written again without modification
- **THEN** the same raw `int32` value SHALL be serialized

#### Scenario: Range-safe write cast
- **WHEN** a fixed-point raw `int` value is outside the `int32` range at write time
- **THEN** `Write` SHALL return a non-nil error instead of silently truncating

#### Scenario: Float conversion is derived behavior only
- **WHEN** callers derive float-space values from raw fixed-point integers (for example `float64(raw)/32768.0`)
- **THEN** this conversion SHALL NOT change the stored raw value unless the caller explicitly updates that raw field

### Requirement: String encoding uses NUL-terminated data blocks
All string fields SHALL be written as NUL-terminated byte sequences in individual data blocks. String references in items SHALL use the DDNet int32 encoding convention.

#### Scenario: Empty string
- **WHEN** an Info field is an empty string
- **THEN** it SHALL be written as a single NUL byte in its data block

#### Scenario: Multi-byte string
- **WHEN** an Info field contains UTF-8 text
- **THEN** the data block SHALL contain the UTF-8 bytes followed by a NUL terminator
