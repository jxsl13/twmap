## Why

`Validate` is supposed to reject DDNet special layers that live in the wrong group or have dimensions that do not match the game layer. The current implementation performs those checks during a single pass that depends on seeing the game layer first, which means invalid maps can slip through when special layers are encountered earlier.

## What Changes

- Make validation of DDNet entity-layer group placement independent of layer traversal order.
- Make validation of DDNet entity-layer dimensions independent of layer traversal order.
- Add regression coverage for maps where special layers appear before the game layer.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `validation`: enforce same-group and same-dimension rules for DDNet entity layers regardless of layer order.

## Impact

- Affects `validate.go` and validation-focused tests.
- No public API changes.
- Tightens correctness of `Validate` so it matches the audited validation spec.