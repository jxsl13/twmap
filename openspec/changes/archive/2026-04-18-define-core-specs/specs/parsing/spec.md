## ADDED Requirements

### Requirement: Parse accepts Teeworlds datafile v3 and v4 containers
The `Parse` function SHALL accept an `io.Reader` containing a Teeworlds datafile with magic header `DATA` or big-endian `ATAD`. It SHALL support datafile version 3 (uncompressed blocks) and version 4 (zlib-compressed blocks).

#### Scenario: Valid v4 datafile
- **WHEN** `Parse` is called with a valid datafile v4 container
- **THEN** it SHALL return a fully populated `*Map` and nil error

#### Scenario: Valid v3 datafile
- **WHEN** `Parse` is called with a valid datafile v3 container (uncompressed)
- **THEN** it SHALL return a fully populated `*Map` and nil error

#### Scenario: Invalid magic bytes
- **WHEN** `Parse` is called with data whose first 4 bytes are not `DATA` or `ATAD`
- **THEN** it SHALL return a non-nil error indicating wrong magic

#### Scenario: Unsupported datafile version
- **WHEN** the datafile version is neither 3 nor 4
- **THEN** `Parse` SHALL return a non-nil error indicating unsupported version

### Requirement: Parse auto-detects map format version
The `Parse` function SHALL auto-detect Teeworlds 0.6/DDNet maps (`MapVersion06`) and Teeworlds 0.7 maps (`MapVersion07`) from the image item versions present in the datafile. `MapVersion06` corresponds to image item version 1. `MapVersion07` corresponds to image item version 2.

#### Scenario: 0.6/DDNet map
- **WHEN** the datafile contains image items with version 1
- **THEN** `Map.Version` SHALL be `MapVersion06`

#### Scenario: 0.7 map
- **WHEN** the datafile contains image items with version 2
- **THEN** `Map.Version` SHALL be `MapVersion07`

### Requirement: Parse fully decodes all map structures
`Parse` SHALL decode all items from the datafile and populate the `Map` struct. This includes: `Info` (metadata), `Images` (tileset images with embedded RGBA data for non-external images), `Envelopes` (animation envelopes with all control points), `Groups` (ordered back-to-front, each containing ordered `Layers`), and `Sounds` (DDNet sound resources).

#### Scenario: Map with all structure types
- **WHEN** a datafile contains groups, layers, images, envelopes, and sounds
- **THEN** all corresponding slices in the returned `Map` SHALL be populated with the decoded data

#### Scenario: Tile data fully decoded
- **WHEN** a tile layer is decoded
- **THEN** `Layer.Tiles` SHALL contain `Width × Height` elements, each with `ID` and `Flags`

#### Scenario: DDNet special layer tile data
- **WHEN** a DDNet special layer (tele, speedup, switch, tune) is decoded
- **THEN** the corresponding specialized tile slice (`TeleTiles`, `SpeedupTiles`, `SwitchTiles`, `TuneTiles`) SHALL be populated with `Width × Height` elements

### Requirement: Parse requires at least one game layer
`Parse` SHALL require at least one game layer after decoding. The stricter structural rules about duplicate special layers, same-group placement, and matching special-layer dimensions are guaranteed by `Validate`, not by `Parse`.

#### Scenario: No game layer
- **WHEN** the decoded map contains no game layer
- **THEN** `Parse` SHALL return `ErrNoGameLayer`

### Requirement: ParseInfo extracts metadata without full decode
`ParseInfo` SHALL parse only map metadata without decoding tile data, images, or layer contents. It SHALL return an `Info` struct containing `Author`, `Version`, `Credits`, `License`, and `Settings`.

#### Scenario: Valid map with info
- **WHEN** `ParseInfo` is called on a valid map with an info item
- **THEN** it SHALL return a populated `Info` and nil error

#### Scenario: Missing version item
- **WHEN** the datafile has no version item
- **THEN** `ParseInfo` SHALL return `ErrMissingVersion`

#### Scenario: Invalid version
- **WHEN** the version item value is not 1
- **THEN** `ParseInfo` SHALL return `ErrInvalidVersion`

### Requirement: WithRequireInfo controls info item requirement
When `WithRequireInfo(true)` is set (the default), `Parse` SHALL return `ErrMissingInfo` if the info item is absent. When `WithRequireInfo(false)` is set, `Parse` SHALL succeed without an info item, returning an empty `Info`.

#### Scenario: Missing info with require true
- **WHEN** `Parse` is called with `WithRequireInfo(true)` on a map without an info item
- **THEN** it SHALL return `ErrMissingInfo`

#### Scenario: Missing info with require false
- **WHEN** `Parse` is called with `WithRequireInfo(false)` on a map without an info item
- **THEN** it SHALL return a `*Map` with an empty `Info` and nil error

### Requirement: Public model uses Go-native types for time and enum fields
Time-like int32 fields from the datafile format SHALL be exposed as `time.Duration` in the public model. This applies to: `EnvPoint.Time`, `Layer.ColorEnvOffset`, `Quad.PosEnvOffset`, `Quad.ColorEnvOffset`, `SoundSource.Delay`, `SoundSource.PosEnvOffset`, `SoundSource.SoundEnvOffset`. Envelope channel count SHALL use the typed alias `EnvelopeChannels`. Sound shape type SHALL use the typed alias `ShapeType`.

#### Scenario: Envelope time values
- **WHEN** an envelope point has a raw millisecond value of 1500 in the datafile
- **THEN** `EnvPoint.Time` SHALL be `1500 * time.Millisecond`

#### Scenario: Sound source delay
- **WHEN** a sound source has a raw delay value of 3 (seconds) in the datafile
- **THEN** `SoundSource.Delay` SHALL be `3 * time.Second`
