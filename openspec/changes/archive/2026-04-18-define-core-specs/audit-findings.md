# Audit Findings

## Parsing Audit

- Confirmed: `Parse` accepts datafile v3 and v4 containers via `parseDatafile` and rejects bad magic / unsupported container versions.
- Confirmed: map version auto-detection is driven by image item versions in `parseImages`.
- Confirmed: `Parse` fully decodes info, images, envelopes, groups, layers, sounds, and DDNet special layer tile slices.
- Mismatch: `Parse` does not enforce the full structural rules currently stated in the parsing spec. `parseMap` never calls `validateMapStructure`, and `parseGroups` only guarantees that at least one game layer exists.
- Impact: `Parse` currently guarantees `ErrNoGameLayer`, but not `ErrTooManyGameLayers`, `ErrTooManyGameGroups`, or `ErrInconsistentGameLayerDimensions`.
- Evidence: `Parse`, `parseMap`, `parseGroups`, and `parseTilemapLayer` in `map.go`; `ParseInfo` / `checkMapVersion` / `parseInfo` in `map.go`; parser smoke tests in `map_test.go`.

## Writing Audit

- Confirmed: `(*Map).Write` always builds a datafile v4 container and compresses data blocks with zlib.
- Confirmed: DDNet special layer encoders use the expected per-tile byte sizes: tele=2, speedup=6, switch=4, tune=2.
- Confirmed: string-backed data blocks are written as individual NUL-terminated data blocks.
- Mismatch: `writeImages` always writes image item version `1` and never emits the TW 0.7 pixel-format field, so `MapVersion07` output parity is not implemented.
- Mismatch: `datafileBuilder.buildDatafile` groups items by type ID before writing, so the binary item order is type-grouped, not the prose order currently stated in the writing spec.
- Mismatch: fixed-point source-of-truth is still `float64` in `Point`, and `writePointBytes` / `writeTexCoordBytes` quantize floats back to `int32`. Raw fixed-point preservation and explicit overflow checks are not implemented yet.
- Limitation: the existing roundtrip tests prove current fixture compatibility, but they do not prove full spec-level symmetry for TW 0.7 image items or raw fixed-point preservation.
- Evidence: `Write`, `writeImages`, `writeGroupsAndLayers`, `writePointBytes`, `writeTexCoordBytes`, `encode*Tiles` in `write.go`; `datafile.writeTo` in `datafile.go`; roundtrip tests in `map_test.go`.

## Validation Audit

- Confirmed: `Validate` is implemented as `Parse` plus `validateMapStructure`.
- Confirmed: duplicate game layers and duplicate DDNet special layers are rejected by `validateMapStructure`.
- Confirmed: `WithRequireInfo`, version checks, and lower-level parse failures are passed through from `Parse`.
- Mismatch: same-group and dimension checks are order-dependent. If a DDNet special layer appears before the game layer during traversal, `validateMapStructure` skips the group/dimension checks for that earlier layer because `gameGroupIdx` and `gameWidth`/`gameHeight` are not known yet.
- Impact: `Validate` does not currently guarantee the same-group and same-dimensions rules for every valid traversal order.
- Evidence: `Validate` and `validateMapStructure` in `validate.go`; parser and validation tests in `map_test.go`.

## Rendering Audit

- Confirmed: the render pipeline order matches the spec in practice: collect design layers, collect DDNet entity layers, compute crop, choose tile resolution, prepare assets, fill checkerboard, render design, render particles, render game-skin entities, render DDNet entity layers, then scale output for `WithMaxSize`.
- Confirmed: checkerboard colors, native tile resolution fallback, aspect-ratio-preserving `WithMaxSize`, parallax filtering, group clipping, detail-layer exclusion, base-layer filtering, tile flag transforms, quad barycentric rasterization, layer color modulation, tileset fallback, game-skin entity rendering, death-border rendering, overlay alpha semantics, invalid speedup diagnostics, and bitmap-font text overlays all match the current implementation.
- Mismatch: crop semantics are broader than the current rendering spec wording. `renderMap` includes game/front source layers in the bounds pass when entity sprites or particle markers are enabled, even if the corresponding DDNet entity overlay layer is not enabled for final rendering.
- Impact: the current implementation can produce a non-1x1 checkerboard image for entity-/particle-only renders because those passes still need source-layer bounds.
- Ambiguity: invalid speedup diagnostics intentionally render numeric text even without a registered speedup-arrow asset when `WithInvalidTiles(true)` is enabled. The current spec wording for speedup labels is too strict if this diagnostic path is considered intentional.
- Clarification: overlay layer compositing order is stable in the implementation: game, front, tele, speedup, switch, tune.
- Evidence: `renderMap`, `collectRenderSteps`, `collectOverlayRenderLayers`, `cropToNonAir`, `prepareTilesets`, `renderSingleTileLayer`, `transformTileCoord`, `renderSingleQuadLayer`, `rasterizeTriangle`, `renderEntities`, `renderOverlayLayers`, `renderParticles`, and overlay text helpers in `rendering.go`; rendering coverage in `rendering_layers_test.go` and `rendering_transform_test.go`.

## Mismatch Classification

- Spec-fix: parsing spec currently overstates structural validation inside `Parse`. The current implementation only guarantees `ErrNoGameLayer`; the stricter same-group and dimension rules belong to `Validate`.
- Code-fix: validation same-group and dimension checks are order-dependent and should not depend on whether the game layer is encountered before special layers.
- Code-fix: `Write` does not yet preserve `Map.Version` for image item versions and does not emit the TW 0.7 pixel-format field.
- Spec-fix: writing spec currently overstates binary item ordering. The implementation intentionally groups items by type ID in the final datafile.
- Code-fix: raw fixed-point preservation and explicit `int32` overflow checks are not implemented yet; this is the tracked Option C migration.
- Spec-fix: rendering crop wording is too narrow; the implementation intentionally includes source game/front layers needed by enabled entity-sprite or particle passes.
- Spec-fix: rendering wording for speedup labels is too strict for diagnostic mode; invalid speedup diagnostics intentionally allow text without an arrow asset when `WithInvalidTiles(true)` is enabled.