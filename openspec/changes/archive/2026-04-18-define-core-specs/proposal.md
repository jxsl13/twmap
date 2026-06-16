## Why

The twmap project has grown organically with parsing, writing, validation, and rendering capabilities that are documented only in README prose and an informal architecture doc. There are no normative specs that define what the library guarantees. This makes it hard to verify whether the implementation matches its intended behavior, to identify where specs need to change vs. where the code is simply wrong, and to onboard future changes with clear acceptance criteria.

Defining core specs now establishes a verifiable baseline before adding more features.

## What Changes

- Introduce a `parsing` spec that formally defines what Parse/ParseInfo must accept, produce, and reject.
- Introduce a `writing` spec that defines Write behavior and the roundtrip symmetry guarantee.
- Introduce a `validation` spec that defines the structural checks Validate must perform.
- Introduce a `rendering` spec that defines the render pipeline contract, option semantics, asset requirements, and overlay behavior — using existing code as reference but stating requirements clearly.
- Audit the current implementation against each spec and capture mismatches as either spec adjustments or implementation fix tasks.

## Capabilities

### New Capabilities
- `parsing`: Covers Parse, ParseInfo, ParseOption behavior, format auto-detection (0.6/DDNet vs 0.7), public Map model as decode target, and error semantics.
- `writing`: Covers Map.Write, datafile v4 serialization, and the roundtrip symmetry invariant with Parse.
- `validation`: Covers Validate, structural integrity checks, game-layer uniqueness, DDNet special-layer dimension rules, and configurable info-item requirement.
- `rendering`: Covers RenderMap/Render, all RenderOption semantics, asset registry requirements, render pipeline ordering, overlay architecture, crop/region/camera behavior, tile flags, quad rasterization, and DDNet-specific overlay rules.

### Modified Capabilities

(none — no existing specs to modify)

## Impact

- No code changes in this proposal itself; the specs are documentation artifacts.
- Subsequent tasks will identify implementation gaps where code does not match spec or where the spec should be relaxed to match intentional behavior.
- The rendering spec may surface requirements that lead to follow-up implementation changes (e.g., known open issues around pixel-identical text overlays, speedup arrow fidelity, and camera-space export modes).
- All four specs become the normative reference for future changes, replacing the informal architecture and README prose as the source of truth for behavior guarantees.
