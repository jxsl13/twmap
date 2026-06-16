## 1. Parsing Spec Audit

- [x] 1.1 Audit Parse against specs/parsing/spec.md: verify datafile v3 and v4 acceptance, magic/version error paths
- [x] 1.2 Audit auto-detection logic: verify MapVersion06 vs MapVersion07 assignment from image item versions
- [x] 1.3 Audit full structure decode: verify all Map fields are populated (Info, Images, Envelopes, Groups with Layers, Sounds)
- [x] 1.4 Audit DDNet special tile decode: verify TeleTile/SpeedupTile/SwitchTile/TuneTile slice population
- [x] 1.5 Audit structural validation inside Parse: ErrNoGameLayer, ErrTooManyGameLayers, ErrTooManyGameGroups, ErrInconsistentGameLayerDimensions
- [x] 1.6 Audit ParseInfo: verify metadata-only parse, ErrMissingVersion, ErrInvalidVersion error paths
- [x] 1.7 Audit WithRequireInfo: verify ErrMissingInfo when true, empty Info when false
- [x] 1.8 Audit Go-native type mapping: time.Duration for EnvPoint.Time, SoundSource.Delay, etc.; EnvelopeChannels and ShapeType aliases
- [x] 1.9 Document any spec-vs-implementation mismatches found in parsing audit

## 2. Writing Spec Audit

- [x] 2.1 Audit Write output format: verify it always produces datafile v4 with zlib compression
- [x] 2.2 Audit structure serialization order: version, info, images, envelopes, envpoints, sounds, groups, layers
- [x] 2.3 Audit roundtrip symmetry: run existing Parse/Write roundtrip tests, verify all fields listed in spec survive
- [x] 2.4 Audit fixed-point raw semantics: verify int-based raw position (17.15) and texcoord (22.10) values are preserved through write/re-parse
- [x] 2.5 Audit DDNet special tile encoding sizes: TeleTile=2, SpeedupTile=6, SwitchTile=4, TuneTile=2
- [x] 2.6 Audit string encoding: NUL-termination, data block placement
- [x] 2.7 Document any spec-vs-implementation mismatches found in writing audit

## 3. Validation Spec Audit

- [x] 3.1 Audit Validate against specs/validation/spec.md: verify it runs all Parse checks plus validateMapStructure
- [x] 3.2 Audit game-layer uniqueness: exactly one game layer required
- [x] 3.3 Audit same-group constraint: all DDNet entity layers must share the game layer's group
- [x] 3.4 Audit dimension consistency: special layer dimensions must match game layer
- [x] 3.5 Audit WithRequireInfo pass-through: verify ErrMissingInfo when true, no error when false
- [x] 3.6 Audit version check: ErrMissingVersion, ErrInvalidVersion
- [x] 3.7 Audit group/layer/image parse checks: invalid dimensions, unsupported versions
- [x] 3.8 Document any spec-vs-implementation mismatches found in validation audit

## 4. Rendering Spec Audit

- [x] 4.1 Audit pipeline ordering against specs/rendering/spec.md: verify the 11-step sequence
- [x] 4.2 Audit checkerboard background: RGB(186,186,186) / RGB(153,153,153) colors
- [x] 4.3 Audit crop behavior: auto-crop to non-air bounding box, WithRegion override
- [x] 4.4 Audit WithMaxSize scaling: aspect-ratio-preserving fit, native resolution without it
- [x] 4.5 Audit parallax filtering: only 100/100 without camera, all groups with camera
- [x] 4.6 Audit group clipping: clip rect in game-pixel coords, pixels outside suppressed
- [x] 4.7 Audit detail layer exclusion/inclusion via WithDetail
- [x] 4.8 Audit WithoutBaseLayerKinds filtering
- [x] 4.9 Audit tile flag transforms: VFlip, HFlip, Rotate, combinations
- [x] 4.10 Audit quad rasterization: barycentric interpolation, vertex colors, bilinear texture sampling
- [x] 4.11 Audit layer color modulation and alpha composition
- [x] 4.12 Audit missing tileset fallback: solid white
- [x] 4.13 Audit WithEntities: game skin present → sprites drawn; absent → nothing drawn; spawns excluded
- [x] 4.14 Audit WithParticles: particle image present → markers; absent → skip
- [x] 4.15 Audit WithGameLayer: entities sheet overlay + death border
- [x] 4.16 Audit WithFrontLayer, WithTeleLayer, WithSwitchLayer, WithTuneLayer overlay rendering
- [x] 4.17 Audit WithSpeedupLayer: arrow asset required, numeric labels, rotation
- [x] 4.18 Audit WithOverlayEntities: alpha semantics, auto-enable of all DDNet entity layers, val=100 skips design
- [x] 4.19 Audit WithInvalidTiles: diagnostic rendering of broken speedup tiles
- [x] 4.20 Audit overlay text labels: bitmap font rendering, minimum tile size threshold
- [x] 4.21 Document any spec-vs-implementation mismatches found in rendering audit

## 5. Spec Adjustments and Follow-ups

- [x] 5.1 For each documented mismatch: classify as spec-fix, code-fix, or accepted-limitation
- [x] 5.2 Apply spec adjustments where the implementation behavior is intentional and the spec was too strict
- [x] 5.3 Create follow-up change proposals for code fixes where the spec correctly identifies a bug
- [x] 5.4 Archive the define-core-specs change after all audits and adjustments are complete

## 6. Option C Migration (int Raw Fixed-Point)

- [x] 6.1 Define API migration scope for `Point` and texture coordinates: identify all public fields changing from `float64` to `int`
- [x] 6.2 Decide migration mode (hard break or compatibility bridge) and record it in design.md
- [x] 6.3 Update parse path to store raw fixed-point values directly as `int`
- [x] 6.4 Update write path to cast raw `int` to `int32` with explicit range checks and error on overflow
- [x] 6.5 Update rendering path to derive float-space values from raw integers at use sites
- [x] 6.6 Add tests proving raw-value preservation for parsed maps after write/re-parse
- [x] 6.7 Add tests proving write fails on out-of-range fixed-point raw integers
