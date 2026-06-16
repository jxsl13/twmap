## Context

twmap is a Go library with four core responsibilities: parsing, writing, validating, and rendering Teeworlds/DDNet map files. The existing behavioral documentation lives in three places:

- `README.md` — user-facing feature list and API overview
- `docs/architecture.md` — internal design notes and decision log
- `docs/open_issues.md` — known gaps and follow-up work

None of these are normative specifications. This design describes how to derive formal specs from them and how to structure the audit process that compares specs against the current implementation.

## Goals / Non-Goals

**Goals:**
- Define one spec file per capability (`parsing`, `writing`, `validation`, `rendering`) under `openspec/specs/`.
- Each spec states only the currently guaranteed behavior — not aspirational features.
- Identify mismatches between spec and implementation as concrete tasks (fix-code or adjust-spec).
- The rendering spec is explicitly allowed to reference existing code structure for context but must state requirements independently of implementation details.

**Non-Goals:**
- Pixel-identical DDNet editor fidelity is not a goal of this spec round. Known approximations (bitmap font vs DDNet FreeType, camera-space export) are documented as accepted limitations.
- Specifying internal helper functions, unexported types, or test utilities.
- Defining an asset spec. The asset registry is a supporting mechanism described within the rendering spec's asset-requirement sections, not a standalone capability.

## Decisions

### 1. Four capabilities, not one monolith

The proposal defines `parsing`, `writing`, `validation`, and `rendering` as separate specs.

**Rationale:** Each capability has different stability levels and change frequencies. Parsing and writing are stable and tightly coupled (roundtrip symmetry). Validation is a thin layer on top of parsing. Rendering is large, evolving, and DDNet-version-sensitive. Keeping them separate lets future changes touch only the relevant spec.

**Alternative considered:** A single `twmap` spec covering everything. Rejected because the rendering section alone would dominate the document and make reviewing unrelated parsing changes unnecessarily heavy.

### 2. Spec describes current guarantees, not aspirations

Each spec captures what the code currently guarantees, verified by the audit step. Known gaps from `docs/open_issues.md` are listed as accepted limitations or out-of-scope items, not as requirements.

**Rationale:** A spec that claims behavior the code doesn't deliver is worse than no spec. Future changes can extend specs when the implementation catches up.

**Alternative considered:** Writing the spec as the desired target and treating all mismatches as implementation bugs. Rejected because some open issues (e.g., pixel-identical text rendering) are deliberate trade-offs, not bugs.

### 3. Rendering spec uses a layered structure

The rendering spec is organized into these sections:
1. Pipeline ordering (which passes run and in what order)
2. Option semantics (what each RenderOption guarantees)
3. Asset requirements (which options depend on which registered assets)
4. Overlay architecture (DDNet-specific layer rules)
5. Crop, region, camera behavior
6. Accepted limitations

**Rationale:** Rendering is the most complex capability. A flat requirement list would be unnavigable. The layered structure mirrors the actual render pipeline, making it easy to trace a requirement to its implementation location.

### 4. Step-by-step audit process

After writing each spec, the corresponding implementation files are audited against it. Mismatches produce one of:
- A spec adjustment (the spec was too strict or too loose)
- An implementation fix task (the code should change)
- An accepted-limitation note (the gap is known and intentional)

**Rationale:** Writing specs without auditing is theater. The audit is what makes the specs trustworthy.

### 5. Rendering spec is written last

Parsing, writing, and validation specs are created and audited first. The rendering spec is written afterward with the benefit of the established spec style and any lessons learned.

**Rationale:** The rendering capability is the largest and most ambiguous. Starting with the simpler, more stable capabilities establishes patterns and calibrates the right level of detail before tackling the complex one.

### 6. Option C for fixed-point domain modeling

The fixed-point coordinate domain SHALL move from plain `float64` fields to plain raw integer fields of type `int`, representing the original datafile `int32` fixed-point values directly.

Target shape:
- Store raw fixed-point values as `int` source-of-truth (position: 17.15 raw, texcoord: 22.10 raw).
- Keep conversion at call sites explicit (for example `float64(raw)/32768.0`) instead of embedding behavior in dedicated value objects.
- `Write` uses these integer raw values directly (after range-safe cast to `int32`).
- Migration scope: `Point.X` and `Point.Y` change from `float64` to `int`, which directly changes all public surfaces that embed `Point`, namely `Quad.Points`, `Quad.TexCoords`, and `SoundSource.Position`.
- Migration mode: hard break. No compatibility bridge or dual raw/float representation will be kept in the public model.

**Rationale:** This preserves semantic roundtrip guarantees by construction while keeping the representation simple and avoiding custom wrapper types.

**Hard-break rationale:** a compatibility bridge would create two competing sources of truth (raw integers and derived floats) during write and render paths. Because this change is specifically about making the raw integers authoritative, keeping both forms in the public model would add complexity exactly where the migration is trying to simplify the domain.

**Alternatives considered:**
- Keep plain `float64` (Option B). Rejected because correctness depends on implicit precision assumptions.
- Use dedicated fixed-point value objects (previous Option C draft). Rejected because the added type abstraction is not needed for the current scope.

## Risks / Trade-offs

- **[Over-specification of rendering]** → Mitigated by explicitly listing accepted limitations and by scoping requirements to current behavior only.
- **[Spec rot after initial creation]** → Mitigated by making future OpenSpec changes require spec updates when behavior changes. The decision log in `docs/architecture.md` continues to capture rationale.
- **[Audit finds too many mismatches]** → Expected for rendering, less likely for parsing/writing/validation. Each mismatch is triaged individually; no blanket "fix everything" commitment.
- **[Roundtrip symmetry is hard to spec precisely]** → The writing spec defines symmetry as: `Parse(Write(m))` produces a Map structurally equal to `m` for all fields that survive the datafile format. Fields that are Go-only (e.g., computed caches) are excluded.
- **[Option C implies API changes]** -> Mitigated by taking a single hard-break migration to `int` raw values and updating parse/write/render call sites in the same change.
- **[int width differs by architecture]** -> Mitigated by defining `int` values as logical carriers of datafile `int32` values and requiring range-safe cast checks when writing.
