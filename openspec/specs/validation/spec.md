## ADDED Requirements

### Requirement: Validate checks datafile container integrity
`Validate` SHALL parse the datafile container and verify its structural integrity. This includes magic bytes, datafile version, header field bounds, item block alignment, and data block decompression.

#### Scenario: Valid map passes
- **WHEN** `Validate` is called on a structurally valid map file
- **THEN** it SHALL return nil

#### Scenario: Corrupt container fails
- **WHEN** the datafile has invalid magic bytes or corrupted headers
- **THEN** `Validate` SHALL return a non-nil error

### Requirement: Validate requires exactly one game layer
The map SHALL contain exactly one layer with `LayerKindGame`. Zero or more than one game layer is invalid.

#### Scenario: No game layer
- **WHEN** the map contains no game layer
- **THEN** `Validate` SHALL return `ErrNoGameLayer`

#### Scenario: Duplicate game layers
- **WHEN** the map contains two or more game layers
- **THEN** `Validate` SHALL return `ErrTooManyGameLayers`

#### Scenario: Exactly one game layer
- **WHEN** the map contains exactly one game layer
- **THEN** `Validate` SHALL NOT return a game-layer-related error

### Requirement: Validate requires all DDNet entity layers in the same group
All DDNet entity layers (game, front, tele, speedup, switch, tune) SHALL be located in the same group. An entity layer in a different group than the game layer is invalid.

#### Scenario: DDNet entity layers in game group
- **WHEN** game, front, tele, speedup, switch, and tune layers are all in the same group
- **THEN** `Validate` SHALL NOT return a group-related error

#### Scenario: DDNet entity layer in separate group
- **WHEN** a tele layer is in a different group than the game layer
- **THEN** `Validate` SHALL return `ErrTooManyGameGroups`

### Requirement: Validate requires special layer dimensions to match game layer
Every DDNet special layer (front, tele, speedup, switch, tune) SHALL have the same `Width` and `Height` as the game layer. A dimension mismatch is invalid.

#### Scenario: Matching dimensions
- **WHEN** the game layer is 50×50 and the tele layer is 50×50
- **THEN** `Validate` SHALL NOT return a dimension error

#### Scenario: Width mismatch
- **WHEN** the game layer is 50×50 and the speedup layer is 100×50
- **THEN** `Validate` SHALL return `ErrInconsistentGameLayerDimensions`

### Requirement: Validate respects WithRequireInfo
`Validate` SHALL accept `ParseOption` values. When `WithRequireInfo(true)` is active (the default), a missing info item SHALL cause `Validate` to return `ErrMissingInfo`. When `WithRequireInfo(false)` is active, a missing info item SHALL NOT cause an error.

#### Scenario: Missing info with default options
- **WHEN** `Validate` is called on a map without an info item, using default options
- **THEN** it SHALL return `ErrMissingInfo`

#### Scenario: Missing info with require disabled
- **WHEN** `Validate` is called with `WithRequireInfo(false)` on a map without an info item
- **THEN** it SHALL return nil (assuming the map is otherwise valid)

### Requirement: Validate checks map version
The map version item SHALL be present and its value SHALL be 1. A missing version item or a version other than 1 is invalid.

#### Scenario: Missing version
- **WHEN** the datafile has no version item
- **THEN** `Validate` SHALL return `ErrMissingVersion`

#### Scenario: Invalid version value
- **WHEN** the version item value is 2
- **THEN** `Validate` SHALL return `ErrInvalidVersion`

#### Scenario: Valid version
- **WHEN** the version item value is 1
- **THEN** `Validate` SHALL NOT return a version-related error

### Requirement: Validate checks all groups and layers parse successfully
All group items and layer items SHALL decode without error. Invalid item sizes, unsupported layer versions, and invalid dimension values SHALL cause validation to fail.

#### Scenario: Layer with invalid dimensions
- **WHEN** a tile layer has width 0 or height 0
- **THEN** `Validate` SHALL return a non-nil error

#### Scenario: All layers valid
- **WHEN** all group and layer items have valid sizes and supported versions
- **THEN** `Validate` SHALL NOT return a layer-related error

### Requirement: Validate checks all images parse successfully
All image items SHALL decode without error. Images with invalid dimensions, unsupported versions, or truncated data SHALL cause validation to fail.

#### Scenario: Image with invalid width
- **WHEN** an image item has width 0
- **THEN** `Validate` SHALL return a non-nil error

#### Scenario: All images valid
- **WHEN** all image items have valid dimensions and data
- **THEN** `Validate` SHALL NOT return an image-related error
