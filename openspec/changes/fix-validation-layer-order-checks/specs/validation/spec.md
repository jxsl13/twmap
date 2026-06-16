## MODIFIED Requirements

### Requirement: Validate requires all DDNet entity layers in the same group
All DDNet entity layers (game, front, tele, speedup, switch, tune) SHALL be located in the same group. An entity layer in a different group than the game layer is invalid, regardless of whether the game layer is encountered before or after the special layer during traversal.

#### Scenario: DDNet entity layers in game group
- **WHEN** game, front, tele, speedup, switch, and tune layers are all in the same group
- **THEN** `Validate` SHALL NOT return a group-related error

#### Scenario: DDNet entity layer in separate group
- **WHEN** a tele layer is in a different group than the game layer
- **THEN** `Validate` SHALL return `ErrTooManyGameGroups`

#### Scenario: Special layer precedes game layer in traversal
- **WHEN** a special layer is encountered before the game layer during traversal and belongs to a different group than the eventual game layer
- **THEN** `Validate` SHALL still return `ErrTooManyGameGroups`

### Requirement: Validate requires special layer dimensions to match game layer
Every DDNet special layer (front, tele, speedup, switch, tune) SHALL have the same `Width` and `Height` as the game layer. A dimension mismatch is invalid regardless of whether the game layer is encountered before or after the special layer during traversal.

#### Scenario: Matching dimensions
- **WHEN** the game layer is 50×50 and the tele layer is 50×50
- **THEN** `Validate` SHALL NOT return a dimension error

#### Scenario: Width mismatch
- **WHEN** the game layer is 50×50 and the speedup layer is 100×50
- **THEN** `Validate` SHALL return `ErrInconsistentGameLayerDimensions`

#### Scenario: Special layer precedes game layer with mismatched dimensions
- **WHEN** a special layer is encountered before the game layer during traversal and its dimensions do not match the eventual game layer
- **THEN** `Validate` SHALL still return `ErrInconsistentGameLayerDimensions`