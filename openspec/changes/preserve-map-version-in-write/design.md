## Context

The parsed `Map` model distinguishes `MapVersion06` and `MapVersion07`, but the current write path always emits image items as version 1 and never writes the TW 0.7 pixel-format field. That means `Write` cannot faithfully preserve 0.7 image-item layout today even though the public model already exposes the version distinction.

## Goals / Non-Goals

**Goals:**
- Make image item serialization follow `Map.Version`.
- Emit the TW 0.7 pixel-format field for `MapVersion07` image items.
- Preserve current `MapVersion06` behavior.
- Add regression tests for both 0.6/DDNet and 0.7 image-item layouts.

**Non-Goals:**
- Changing the outer datafile container version.
- Redesigning the public image model.
- Solving the separate raw fixed-point migration tracked in the core-spec change.

## Decisions

- Select image item layout from `Map.Version` inside `writeImages`.
  Rationale: the map version is already the public source of truth for container semantics.
  Alternative considered: always emit version 1 for compatibility. Rejected because it breaks semantic roundtrip for `MapVersion07` maps.

- Use TW 0.7 image item version 2 with an explicit pixel-format field.
  Rationale: this matches the parsed model and the current writing spec.

- Add targeted write/re-parse tests using synthetic maps.
  Rationale: existing fixtures appear biased toward current 0.6/DDNet coverage, so regression tests should assert both layouts directly.

## Risks / Trade-offs

- [Wrong pixel-format default could produce unreadable output] → Keep the initial implementation conservative and align it with the current embedded RGBA representation.
- [Version-sensitive write logic could regress 0.6 behavior] → Cover both versions in focused tests.