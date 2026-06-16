## MODIFIED Requirements

### Requirement: Write produces a datafile matching the map version
`(*Map).Write` SHALL serialize the map into a Teeworlds datafile version 4 container with zlib-compressed data blocks, written to the provided `io.Writer`. The image item version written for each image SHALL correspond to `Map.Version`: version 1 for `MapVersion06`, version 2 for `MapVersion07`. TW 0.7 image items SHALL additionally include the pixel-format field. The output SHALL be loadable by the TW/DDNet client that matches the map version.

#### Scenario: Write a MapVersion06 map
- **WHEN** `Write` is called on a `Map` with `Version = MapVersion06`
- **THEN** all image items SHALL use image-item version 1 and the output SHALL be loadable by TW 0.6/DDNet clients

#### Scenario: Write a MapVersion07 map
- **WHEN** `Write` is called on a `Map` with `Version = MapVersion07`
- **THEN** all image items SHALL use image-item version 2 with a pixel-format field and the output SHALL be loadable by TW 0.7 clients

#### Scenario: Image item layout follows map version
- **WHEN** `Write` serializes image items for maps of different `Map.Version` values
- **THEN** the image item layout SHALL follow the version-specific field count and SHALL NOT be hardcoded to the 0.6/DDNet shape