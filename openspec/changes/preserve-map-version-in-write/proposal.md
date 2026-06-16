## Why

`Write` currently always serializes image items as version 1 and does not emit the TW 0.7 pixel-format field. That breaks faithful roundtrip behavior for `MapVersion07` maps and makes the writing path fall short of the audited writing spec.

## What Changes

- Preserve `Map.Version` when writing image items.
- Emit the TW 0.7 image pixel-format field for `MapVersion07` output.
- Add regression coverage proving `MapVersion06` and `MapVersion07` write distinct image-item layouts and roundtrip correctly.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `writing`: write image items using the layout that matches `Map.Version`, including TW 0.7 pixel-format metadata.

## Impact

- Affects `write.go` and roundtrip/write-focused tests.
- No intended public API changes.
- Improves fidelity and client compatibility for `MapVersion07` output.