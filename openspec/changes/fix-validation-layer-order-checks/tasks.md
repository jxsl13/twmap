## 1. Validation Refactor

- [ ] 1.1 Refactor `validateMapStructure` to collect game-layer and special-layer facts independently of traversal order
- [ ] 1.2 Preserve duplicate-layer detection and existing validation error values while removing order dependence

## 2. Regression Coverage

- [ ] 2.1 Add a validation test where a special layer appears before the game layer in a different group and must return `ErrTooManyGameGroups`
- [ ] 2.2 Add a validation test where a special layer appears before the game layer with mismatched dimensions and must return `ErrInconsistentGameLayerDimensions`

## 3. Verification

- [ ] 3.1 Run the validation-focused test suite and confirm the new regression cases pass