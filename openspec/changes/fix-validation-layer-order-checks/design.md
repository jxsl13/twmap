## Context

`Validate` is implemented as `Parse` plus `validateMapStructure`. The current `validateMapStructure` logic discovers the game layer and validates special-layer placement and dimensions in a single traversal. That makes same-group and dimension checks dependent on seeing the game layer before the special layers, which is not a safe assumption.

## Goals / Non-Goals

**Goals:**
- Make same-group validation independent of layer traversal order.
- Make special-layer dimension validation independent of layer traversal order.
- Preserve existing error types and public `Validate` API.
- Add regression coverage for previously missed invalid layouts.

**Non-Goals:**
- Changing `Parse` semantics.
- Changing the duplicate-layer rules or error taxonomy.
- Introducing new validation options or configuration.

## Decisions

- Collect validation facts first, then evaluate constraints after traversal.
  Rationale: this removes order dependence cleanly and keeps the logic easy to reason about.
  Alternative considered: multiple targeted re-scans for each special layer kind. Rejected because it is more repetitive and less maintainable.

- Keep duplicate-layer detection during traversal.
  Rationale: duplicate kinds are independent of game-layer discovery order and can still fail fast.

- Add regression tests with special layers placed before the game layer.
  Rationale: the bug is specifically traversal-order dependent, so the tests should encode that failure mode directly.

## Risks / Trade-offs

- [Refactor changes validation control flow] → Keep the implementation local to `validateMapStructure` and preserve current error values.
- [Synthetic test fixtures may miss real-world layouts] → Cover both group mismatch and dimension mismatch in explicit targeted tests.