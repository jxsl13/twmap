# Open Issues

This file tracks known gaps that still need work after the current renderer and
model refactor.

## Rendering

- Parallax/detail rendering for very large full-map exports is still inherently
  camera-dependent. A single exported image of the entire map cannot perfectly
  match every DDNet editor viewport at once. The current renderer is consistent,
  but a dedicated design-space vs camera-space export mode is still missing.
- Tele/switch/tune/speedup text overlays now render, but they still use an
  internal bitmap font rather than DDNet's own text renderer and atlas logic.
  Visual parity is close enough for inspection, but not pixel-identical.
- Speedup rendering uses a procedural arrow instead of DDNet's dedicated
  speed_arrow_array.png asset. This fixes visibility, but visual parity is not
  exact yet.
- The tutorial render utility now skips categories that do not affect the map,
  but it still writes full native-resolution exports. On very wide maps that is
  expensive and can take a long time.

## Verification And Tooling

- More visual regression tests are needed against DDNet editor reference output,
  especially for maps with heavy speedup/tune/switch usage.
- The renderer has unit coverage for ordering and selection logic, but not yet a
  golden-image style suite for multi-group parallax maps.

## API And Documentation

- README-level API documentation has not yet been expanded for every new public
  helper and type alias introduced in the current refactor.
- The new architecture decision log exists, but future changes still need a
  disciplined process so decisions do not drift into code-only history.
