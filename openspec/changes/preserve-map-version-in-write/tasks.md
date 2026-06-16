## 1. Version-Aware Image Writing

- [ ] 1.1 Update `writeImages` to choose the image item version from `Map.Version`
- [ ] 1.2 Emit the TW 0.7 pixel-format field for `MapVersion07` image items while preserving current `MapVersion06` layout

## 2. Regression Coverage

- [ ] 2.1 Add a write/re-parse test covering `MapVersion06` image item layout
- [ ] 2.2 Add a write/re-parse test covering `MapVersion07` image item layout and pixel-format preservation

## 3. Verification

- [ ] 3.1 Run the write/roundtrip-focused test suite and confirm both map-version paths pass