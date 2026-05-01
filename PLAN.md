# lvrsrc — Implementation Plan

Pure-Go RSRC/VI toolkit with strong round-trip guarantees, partial semantic decoding, and carefully scoped write support. Full goal and rationale in [GOAL.md](./goal.md).

---

## Phase 0 — Research & Corpus Setup

> Target: 1-2 weeks | Exit: corpus baseline, resource inventory, and MVP scope approved
- Go module, directory skeleton, README/LICENSE/gitignore, and dependency pins were established.
- CI, lint, vet/staticcheck, build/test, and fuzz placeholders were wired.
- Reference docs captured RSRC layout, resource registry, safety model, and reverse-engineering workflow.
- Corpus/oracle setup was documented and later backed by committed fixtures and baselines.

---

## Phase 1 — Container Parser

> Tag: v0.1.0 | Exit: inspect, dump, and list-resources work on corpus without panics
- Added bounded binary reader helpers for integers, byte ranges, Pascal strings, and C strings.
- Parsed RSRC headers, block info, block headers, sections, payloads, names, raw tails, file kind, and compression stubs.
- Published lvrsrc Open/Parse/Resources/Clone APIs plus JSON dump support with opaque bytes as base64.
- Shipped Cobra/Viper inspect, dump, and list-resources commands with smoke tests and parser fuzz targets.

---

## Phase 2 — Preserving Writer

> Tag: v0.2.0 | Exit: corpus round-trips while opaque data is preserved
- Added offset-aware binary writer helpers and Pascal-string writing tests.
- Implemented preserving RSRC serialization: offsets, padding, block/name tables, duplicate headers, raw tails, and opaque sections.
- Published WriteTo, WriteToFile, and Validate APIs plus the lvrsrc rewrite command.
- Added golden round-trip harness, CLI integration coverage, Python-oracle inventory regression tests, and writer-difference docs.

---

## Phase 3 — Validator & Diff

> Tag: v0.3.0 | Exit: human and JSON diagnostics plus resource diffs
- Added structural validator issues for headers, bounds, sizes, counts, names, payload overlap, FourCCs, and strict/lenient mode.
- Shipped lvrsrc validate with human/JSON output and machine-readable exit codes.
- Implemented lvdiff header/resource/section diffs plus decoded-resource extension hooks.
- Added lvrsrc diff and dump/validate JSON schemas with schema-conformance tests.

---

## Phase 4 — Safe Metadata Editing

> Tag: v0.4.0 | Exit: targeted metadata edits survive rewrite and validation
- Added codec registry, version/file-kind context, compatibility table, and Tier 2 STRG/vers codecs.
- Implemented lvmeta mutation pipeline with safety checks, strict-mode warning policy, and offset/FourCC-aware errors.
- Added SetDescription and SetName, including STRG insertion, name-table reuse/compaction, and post-edit validation.
- Exposed lvvi decoded model data and shipped lvrsrc set-meta with integration and corpus preservation tests.

---

## Phase 5 — Typed Resource Expansion _(ongoing)_

> Exit criteria: resource coverage dashboard; documented support matrix by resource type and version | Tag: `v0.5.x+`

### 5.1 Resource Coverage Dashboard

- [x] Define machine-readable coverage manifest (YAML/JSON)
- [x] Generate coverage report in CI
- [x] Add badge to README

### 5.2 Additional Codecs

- [x] Research and implement icon resource codec
- [x] Research and implement connector pane resource codec
- [x] Research and implement front-panel metadata codec
- [x] Research and implement block diagram metadata codec
- [x] Research and implement type descriptor resource codec
- [x] Expand `lvdiff` decoded-resource diff for each new codec

### 5.3 `.llb` Library Support

- [x] Research LLB container format differences
- [x] Implement LLB open/parse in `pkg/lvrsrc`
- [x] Add `lvrsrc inspect` support for `.llb` files
- [x] Add round-trip tests for LLB files

### 5.4 Canonical Writer

- [x] Implement canonical ordering of blocks and sections
- [x] Implement canonical padding/alignment policy
- [x] Implement deterministic serialization
- [x] Add `--canonical` flag to `lvrsrc rewrite`

### 5.5 Repair Command

- [x] Define repair heuristics (truncated name table, offset drift, header mismatch)
- [x] Implement `lvrsrc repair <file> --out <repaired.vi>` command (after validator is mature)
- [x] Write repair tests with intentionally corrupted fixtures

### 5.6 v1.0 Readiness Checklist

> Gated by Phases 6–10 — the current typed-codec set covers less than half of the observed FourCCs and the two heap resources (`FPHb`, `BDHb`) remain opaque. Tagging `v1.0.0` requires the coverage bar set in Phase 10.

- [ ] Round-trip corpus is broad (version coverage documented)
- [ ] Validator is mature (all known structural checks pass)
- [ ] Support matrix published and complete
- [ ] Unsafe APIs clearly separated and gated
- [ ] Public API is stable (no breaking changes planned)
- [ ] Tag `v1.0.0`

---

## Phase 6 — Small-Block Completion & Colour Icons

> Tag: v0.6.0 | Exit: small observed FourCCs typed, colour icons rendered, LVSR flags surfaced
- Ported LabVIEW icon palettes and RGBA rendering for ICON/icl4/icl8.
- Added LVSR flag decoding and public lvvi flag/breakpoint accessors.
- Added Tier 1 codecs/docs/tests for LIBN, BDPW, FTAB, DTHP, RTSG, MUID, FPSE, BDSE, HIST, VITS, FPEx, BDEx, and VPDP.
- Updated coverage to 24/27 typed FourCCs, extended decoded diffs, and lit up icon/flag/decoded badges in the demo.

---

## Phase 7 — Rich Link Graph

> Tag: v0.7.0 | Exit: LIfp, LIbd, and LIvi entries expose typed link targets and paths
- Implemented PTH0/PTH1 path decoding and LIvi envelope parsing with round-trip preservation.
- Upgraded LIfp/LIbd/LIvi entries with lazy typed LinkObjRef targets, LinkKind metadata, and opaque fallback.
- Exposed FrontPanelImports, BlockDiagramImports, and VIDependencies through lvvi, with decoded diffs for link resources.
- Documented path/link resources and added dependency-card rendering in the web demo.

---

## Phase 8 — Type-Descriptor Surface & Connector Pane

> Tag: v0.8.0 | Exit: VCTP navigation and connector-pane rendering
- Exposed public TypeDescriptor projections, top-type lists, and 1-based TypeAt lookups over VCTP.
- Resolved CONP through VCTP and surfaced CPC2 connector-pane variants across the corpus.
- Added codec/model tests for typedesc parsing, corpus lookup, and connector-pane resolution.
- Rendered Types and Connector Pane cards in the Info tab.

---

## Phase 9 — Front-Panel Heap Decoder (FPHb)

> Tag: v0.9.0 | Exit: FPHb parses to a typed tree and round-trips byte-for-byte
- Implemented shared heap zlib envelope with cached-byte preservation, recompression fallback, and fuzz targets.
- Generated/ported heap tag enums from pylabview and added typed node accessors for ints, type IDs, rects, points, strings, bools, and data fills.
- Added FPHb Tier 1 codec, validation, corpus round-trip tests, and extensive fuzz coverage.
- Exposed lvvi FrontPanel trees and structural decoded diffs for FPHb.

---

## Phase 10 — Block-Diagram Heap (BDHb) & Approximate Render

> Tag: v1.0.0 | Exit: BDHb typed, approximate FP/BD render, 100% corpus FourCC coverage
- Reused the heap framework for BDHb with Tier 1 round-trip validation and fuzz coverage.
- Added FrontPanel and BlockDiagram tree projections plus approximate demo render tabs with fidelity notes.
- Brought coverage to 27/27 typed observed FourCCs and updated generated coverage/resource-registry docs.
- Published API compatibility policy and deployed the richer web demo flow.

---

## Phase 11 — SVG / Canvas Renderers & CLI Export

> Tag: v1.1.0 | Exit: shared scene graph, web visual modes, and CLI SVG export
- Added renderer-neutral scene graph with bounds, labels, containment, placeholders, wires, view boxes, and z-order.
- Projected FPHb/BDHb trees into shared SVG/canvas render paths used by both WASM and CLI.
- Shipped lvrsrc render with front-panel/block-diagram SVG output and --out/stdout support.
- Added scene/SVG goldens, web visual/canvas/tree smoke coverage, and renderer limit docs.

---

## Phase 12 — LabVIEW Geometry & Widget Foundations

> Target: Stage 1 foundation | Exit: controls have real bounds, generic widget kinds, and terminal anchors for later wire rendering | Tag: `v1.2.0-pre`
>
> Replaces the Phase 11 heuristic depth-stacked layout with decoded
> LabVIEW geometry wherever the heap exposes it. This phase is complete:
> it does not attempt full LabVIEW skins or wire routing.

### 12.1 `OF__bounds` real positions (completed)

- Spec confirmed against `pylabview` `HeapNodeRect` (LVheap.py:1725): FieldTag 14 stores 4 big-endian `int16` values `{Left, Top, Right, Bottom}`. Corpus coverage: 1 188 / 1 188 OF__bounds leaves decode across 42 FPHb + BDHb trees.
- `lvvi.HeapBounds(tree, idx)` and `lvvi.FindBoundsChild(tree, parentIdx)` expose bounds with tests in `pkg/lvvi/bounds_test.go`.
- `internal/render.ProjectHeapTree` promotes decoded bounds to scene groups, drops the metadata leaf from visible output, keeps heuristic fallback for controls without bounds, and auto-fits the scene viewBox.
- Heuristic-layout warnings now appear only when at least one root falls back to heuristic placement. Render goldens were regenerated; WASM rebuilt; `web/smoke_test.go` still passes.

### 12.2 Generic widget-kind styling (completed)

- `pkg/lvvi.WidgetKind` covers boolean, numeric, string, cluster, array, graph, decoration, structure, primitive, terminal, and other; name-based class mapping also folds `SL__array` and `SL__arrayElement` into Array.
- `internal/render.Node.WidgetKind` flows through layout and SVG output, emitting `lvrsrc-widget-{kind}` CSS classes with distinct generic fills/strokes. Empty helper/leaf kinds suppress the class.
- Corpus baseline (`TestWidgetKindForNodeCorpusBaseline`) reports roughly 50% open-scope-node coverage before terminal classes, dominated by Array, Other, Primitive, Graph, and Decoration.
- Stage 1 deliberately stops short of pixel-faithful skins; the shipped goal is that booleans, numerics, strings, clusters, arrays, graphs, decorations, structures, primitives, and terminals are visually distinguishable.

### 12.3 Terminal geometry and anchors (completed)

> Spec discovery showed `OF__terminal` (FieldTag 367) has zero leaves
> in the 21-fixture corpus and pylabview has no decoder for it. Real
> terminal geometry travels via `OF__termBounds` and `OF__termHotPoint`.

- `pkg/lvvi.HeapTermBounds` decodes FieldTag 266 as the same 8-byte rect shape as `OF__bounds`; corpus coverage is 154 / 154 leaves. `FindTermBoundsChild` supports parent-side lookup.
- `pkg/lvvi.HeapTermHotPoint` decodes FieldTag 267 as a 4-byte Mac-style `Point{V, H}`; corpus coverage is 6 / 6 leaves. `FindTermHotPointChild` mirrors the bounds lookup helper.
- `WidgetKindTerminal` maps BD tunnel/terminal classes including `SL__term`, `SL__fPTerm`, loop/sequence/case tunnels, region/comment/external tunnels, `SL__xTunnel`, and `SL__decomposeRecomposeTunnel`. Corpus classified coverage rose from 49.7% to 55.4%.
- `internal/render` emits tunnel/terminal heap nodes as flat `NodeKindTerminal` nodes with bounds from termBounds/bounds and `Anchor` from termHotPoint or the bounds centre.
- SVG output draws terminal bounds as thin outlines plus a filled `r=2` anchor circle. Goldens were regenerated, WASM rebuilt, and `docs/resources/bdhb.md` documents the decoded terminal surface.

---

## Phase 13 — Compressed Wire Table Decoding

> Target: Stage 1 wire semantics | Exit: compressed wire-table chunks are typed enough for reliable block-diagram wire paths | Tag: `v1.2.0-pre`
>
> This phase covers `OF__compressedWireTable` reverse engineering. The
> presence and basic accessor layers are shipped; comb topology,
> multi-elbow chain geometry, and manual-chain semantics remain open.

### 13.1 Wire-table source discovery (completed)

- The literal `OF__wireTable` (296), `OF__wireID` (295), `OF__wireGlyphID` (294), `OF__signalList` (232/233 naming context), and `OF__signalIndex` (232) have 0 relevant leaves across the original 21-fixture corpus.
- Actual wire connectivity lives in `OF__compressedWireTable` (FieldTag 456): initially 80 leaves, children of `SL__arrayElement`, with variable payload sizes of 2, 4, 6, 8, 10, 12, 14, and 20 bytes.
- Pylabview carries the enum number but no decoder, so connectivity decoding is corpus- and controlled-fixture-driven.
- `pkg/lvvi.HeapCompressedWireTable` returns raw payload bytes; `CountCompressedWireTables` reports chunk counts; render warnings now surface compressed-wire presence instead of silently dropping wire data.

### 13.2 Signal-list correction and first controlled-fixture spike

The original "primitive-internal metadata only" conclusion was wrong.
Tag 233 collides between `ClassTag.SL__baseTableControl` and
`FieldTag.OF__signalList`; the heap resolver prefers ClassTag, so
debug output mislabeled the container. In BD context, parented under
`SL__eventDataNode`, `SL__sdfTun`, or `SL__concatDCO`, it is a signal
list, and each `arrayElement` child holds one wire/signal entry:

```text
SL__rootObject
  └── SL__eventDataNode (or other primitive)
       └── OF__signalList
            └── SL__arrayElement
                 └── OF__compressedWireTable
```

The first spike authored 12 deliberately varied VIs:
`blank.vi`, `Numeric42.vi`, `Numeric4Dot2.vi`, `BoolToLED.vi`,
`Add17Plus25.vi`, `Numeric42Far.vi`, `Numeric42Bend.vi`,
`Numeric42_8px_down.vi`, `Numeric42_16px_down.vi`,
`Numeric42_8px_down_8px_further_right.vi`,
`Numeric42TwoIndicatorsY.vi`, and `Numeric42ThreeIndicatorsY.vi`.

Findings:

1. Chunks are wire-networks, not edges: a Y-shaped fan-out emits one chunk.
2. `byte0` is the waypoint count: endpoints plus internal corners. One auto-bend can add two corners because LabVIEW renders an offset path as a Z-shape.
3. `byte1` is the mode flag: `0x08` auto-routed chain, `0x04` manual chain, `0x00` tree, other values unknown.
4. Chain-mode trailing payload uses LEB128 varints for deltas over geometry recoverable from terminal positions.
5. Tree-mode payload is fixed-width 2-byte records: `(byte0, byte1)` followed by `byte0 - 1` records for branch/topology geometry.

### 13.3 Typed wire accessor (completed)

- `pkg/lvvi.WireMode` ships `WireModeAutoChain`, `WireModeManualChain`, `WireModeTree`, and `WireModeOther`, with `String()` for diagnostics.
- `pkg/lvvi.HeapWire(tree, idx)` returns `{Mode, Waypoints, ChainGeometry []uint64, TreeRecords [][2]byte, Raw []byte}`. Chain mode decodes LEB128 varints; tree mode splits fixed 2-byte records; all modes preserve `Raw`.
- `pkg/lvvi.CountWireMix(tree)` returns per-mode counts and drives scene warnings such as "N wire networks (X auto-routed, Y manual, Z branched, W other)".
- Tests cover all modes with controlled-fixture payloads plus a corpus sweep. Coverage on the 33-fixture corpus is 93 / 93 chunks: 83 auto-chain, 3 manual-chain, 5 tree, 2 other.

### 13.4 Per-record semantics shipped so far

The second spike added `Numeric42_8px_up.vi`,
`Numeric42TwoNetworks.vi`,
`Numeric42TwoIndicatorsY_top7right_bottom11down.vi`, and
`Numeric42FourIndicatorsY_single.vi`.

Findings:

- Sign is stored separately, not zigzag. Moving the indicator 8 px above the source changed payload[0] from `0x00` to `0x01`; the y-step magnitude byte stayed `0x08`.
- Network boundaries were confirmed: two disconnected const-to-indicator pairs emit exactly two `0208` chunks.
- 2-Y tree endpoints were ground-truthed: geometry changes moved record #4 H by +7 and record #5 V by +10, so records #4/#5 are endpoint coordinates for the two branches.
- 3+ branch tree mode is topology-dependent. The 4-indicator comb fixture emits 10 records, not the linear 8 that a single-junction fan-out model would predict.

Shipped accessors:

- `Wire.ChainAutoPath()` returns `ChainAutoPath{Straight, YStep, SourceAnchorX}` for the `0208` sentinel and 4-varint L-shape payload (`[direction, 0, source-anchor-x, y-step-mag]`). Multi-elbow payloads return `ok=false` when magnitudes become implausible (>4096).
- `Wire.TreeEndpointPair()` returns two `Point{V, H}` endpoints for 2-fan-out tree networks (`byte0 == 6`).
- Scene warnings now state that auto-routed L-shapes and 2-branch trees are typed-decoded while multi-elbow and larger tree chunks remain raw.

### 13.5 Remaining tree-mode and chain semantics (partial)

- 3-branch pure Y-tree endpoints are now ground-truthed: `Wire.TreeEndpoints()` generalizes `TreeEndpointPair()` for `byte0` in `{6, 7}`. `Numeric42ThreeIndicatorsY_bottom8pxdown.vi` changes exactly one endpoint record (`44 2d` → `44 35`, +8 in the second coordinate), confirming the "last N = byte0 - 4 endpoint records" rule for pure 3-Y chunks. The independent corpus `reference-find-by-id.vi` chunk still matches the same shape.

- [ ] **Comb and 4+ branch topology.** Comb chunks (`byte0=10`, `rec[2][0]=6`) have records with `H=1` that are clearly flags or topology markers, not pixel coordinates. `Numeric42ThreeIndicatorsYComb_middle8pxdown.vi` shows the middle-branch edit changes two adjacent records in opposite directions (`5b 01` → `63 01`, `57 42` → `4f 42`), so the comb payload carries span/junction data around the moved branch rather than a simple trailing endpoint list. Other `byte0=10` chunks (`ndjson-parser.vi`, `reference-find-by-id.vi`) make the "last 4" rule suspicious (`H=4/10/16`). All `byte0 >= 8` shapes return nil/false from `TreeEndpoints()` until further ground-truthed.

- [ ] **Multi-elbow chain decoding.** Multi-elbow auto-chain payloads carry more than 4 varints. The synthetic test case in `TestChainAutoPathDoesNotMakeUpMultiElbowGeometry` (`[0, 0, 255, 9 456]`) shows byte 3 is implausibly large for a pixel y-step, indicating routing-index or per-segment delta encoding. Needs a controlled fixture with deliberate 2- or 3-elbow auto-routing.

- [ ] **Manual-chain decoding.** The 3 manual-chain chunks (`byte1 == 0x04`) decode as long varint streams, but per-position semantics are unmapped beyond "user-placed waypoints with explicit deltas." Needs controlled manual-bend fixtures with known-position waypoints.

Remaining fixtures needed:

1. **One more comb-geometry variation** — move a different branch or change actual vertical placement in `Numeric42ThreeIndicatorsYComb.vi` to separate endpoint coordinates from junction/span records.
2. **`Numeric42TwoElbows.vi`** — force a single wire to auto-route with 2 elbows, then compare with `Numeric42_8px_down.vi` to map per-elbow varint roles.

---

## Phase 14 — Wire Rendering & Stage 1 Exit

> Target: Stage 1 complete | Exit: a corpus VI is recognizable in the demo from positioned controls plus rendered block-diagram wires | Tag: `v1.2.0`

### 14.1 Pylabview class cross-check

- [x] Read pylabview's `LVheap.py` per-class parser dispatch and adjust `widgetKindByClass` where the name-based heuristic disagrees with pylabview's classification. Done: cross-checked `SL_CLASS_TAGS` plus `CLASS_EN_TO_TAG_LIST_MAPPING`; added explicit `refnum`, `variant`, and `connector-pane` widget kinds for pylabview-modeled classes that were previously `other`.
- [x] Where pylabview groups classes into kinds not currently modeled (refnum, tunnel, variant, connector-pane), decide whether to add new `WidgetKind` values or fold them into `Other` with a per-kind doc note. Done: refnum, variant, and connector-pane are first-class `WidgetKind` values; tunnels stay folded into `terminal` because they are rendered as wire anchors.

### 14.2 Wire path drawing

- [x] Render block-diagram wires as orthogonal/Manhattan paths between known terminals using terminal anchors plus typed wire-network data. Done: `internal/render.ProjectHeapTree` now populates `Scene.Wires` for recognized block-diagram wire chunks, and SVG output draws them as polylines.
- [x] Use `Wire.ChainAutoPath`, `Wire.TreeEndpointPair`, and `Wire.TreeEndpoints` where semantics are known; retain explicit warnings/placeholders for comb, 4+ branch, multi-elbow, manual-chain, and other unknown chunks. Done: auto-chain paths use `Wire.ChainAutoPath`; pure 2/3-branch tree networks use `Wire.TreeEndpoints` (which subsumes `TreeEndpointPair` for 2-branch cases); unknown/manual/multi-elbow/comb chunks remain unrendered with the wire summary warning.
- [x] Drop the broad "wire routing not rendered yet" warning once known single-edge and 2/3-branch networks render visibly. Done: scenes with at least one rendered recognized wire report the rendered-network count instead of the broad not-rendered warning.
- [ ] Exit criterion: open a corpus VI in the demo and recognize it from real control positions, generic widget styling, and connected block-diagram wire paths.

---

## Phase 15 — LabVIEW Fidelity Stage 2

> Target: incremental fidelity | Exit: renders look substantially closer to native LabVIEW beyond Stage 1 functional clarity | Tag: `v1.3.0+`

- [ ] Per-class control skins approximating the real LabVIEW widget look.
- [ ] Fonts, captions, tick labels, scale ranges, and label anchors.
- [ ] Exact wire waypoints once `OF__wireTable` / compressed-wire semantics are fully decoded, replacing heuristic orthogonal routing.
- [ ] Decorations, colors, panel backgrounds, and custom style attributes.
- [ ] Polish for selection bounds, resize handles, and export-quality SVG/canvas output.

---

## Phase 16 — Format Closure & Semantic Completeness

> Target: close the remaining reverse-engineering gap | Exit: every observed byte is either semantically named, explicitly reserved/padding, or documented opaque with corpus evidence and a path to decode | Tag: `v1.4.0+`
>
> "Fully understand the format" is treated as a practical, evidence-backed
> goal: complete for the supported LabVIEW versions and corpus families, with
> version gates where older/newer files diverge. Unknown data should never be
> hidden behind a typed codec name without a byte-level disposition.

### 16.1 Coverage Accounting

- [x] Add a byte-disposition report per resource: semantic fields, reserved/padding, checksums/compressed bytes, and opaque spans. Started in `internal/coverage`: the manifest now records corpus sections/bytes plus semantic/reserved/compressed/opaque/next fields for every observed FourCC.
- [x] Extend `docs/generated/resource-coverage.md` beyond FourCC-level typed coverage so it reports semantic byte coverage for `FPHb`, `BDHb`, `VCTP`, `LIfp`, `LIbd`, `LIvi`, and LVSR tails. The generated JSON/Markdown artifacts now include corpus breadth, corpus sections/bytes, and byte-disposition status/details for every observed FourCC.
- [x] Add tests that fail when a codec silently introduces a new opaque span without documenting it in the resource note. `internal/coverage` now fails the manifest test if any observed resource has an `undocumented` byte disposition or no semantic entry.
- [x] Track corpus breadth by LabVIEW major version, platform, file kind (`.vi`, `.ctl`, `.vit`, `.llb`), password/protection state, compiled-code setting, and localized text encoding. The manifest now reports file kinds/extensions, RSRC format versions, decoded LabVIEW `vers` labels, password/BDPW state, LVSR locked flag, LVSR separate-compiled-code flag, and explicit `unknown` buckets for platform/text encoding until those fields are decoded.

### 16.2 Corpus & Oracle Expansion

- [x] Build a controlled VI matrix that changes one feature at a time: labels, captions, fonts, colors, scales, decorations, clusters, arrays, graphs, structures, refnums, variants, subVI calls, event structures, cases, loops, and disabled/conditional diagrams. `docs/corpus-matrix.json` now records every current `.vi` / `.ctl` fixture, its focus, its working hypothesis, the missing one-feature fixture names, and the oracle targets. `internal/corpus` tests keep the matrix in sync with the scanned corpus.
- [ ] Author the missing one-feature fixtures listed in `docs/corpus-matrix.json`, then move their gap entries into fixture entries with measured deltas.
- [ ] Add version-spanning fixtures from old and current LabVIEW releases, with expected version gates for fields that move or change encoding.
- [x] Keep an external-oracle comparison harness against `pylabview`, `pylavi`, and native LabVIEW save-as/read-back behavior where available. `internal/oracle` already verifies checked-in pylabview baselines; `docs/oracle-targets.json` now tracks automated pylabview coverage plus planned pylavi and manual native-LabVIEW artifact targets, with tests enforcing the registry shape.
- [x] Store reverse-engineering deltas as fixture pairs with a short hypothesis note, not just as final decoded structs. `docs/reverse-deltas.json` now records controlled fixture pairs, topics, hypotheses, evidence notes, and status; `internal/corpus` tests ensure the referenced fixtures exist and key topics remain covered.

### 16.3 Heap Field Semantics

- [x] Add a generic typed accessor for rectangle-like heap fields beyond `OF__bounds` / `OF__termBounds`: `OF__contRect`, `OF__dBounds`, `OF__pBounds`, `OF__iconBounds`, `OF__growAreaBounds`, `OF__sizeRect`, and related panel/diagram rectangles. `pkg/lvvi.HeapRect`, `HeapRectForTag`, `FindRectChild`, and `IsHeapRectTag` now share the `HeapNodeRect` decoder, while the existing `HeapBounds` / `HeapTermBounds` APIs stay source-compatible.
- [ ] Promote the newly decoded rectangle roles into scene-graph layout only where controlled fixtures show that they affect visible geometry.
- [x] Add generic typed accessors for common scalar and color heap fields. `pkg/lvvi.HeapScalar`, `HeapScalarForTag`, `FindScalarChild`, `HeapColor`, `HeapColorForTag`, `FindColorChild`, `IsHeapScalarTag`, and `IsHeapColorTag` now expose common integer / flag / id / count leaves and 4-byte color-like leaves while leaving role-specific bit names and color-prefix semantics open.
- [ ] Decode front-panel visual fields: label and caption anchors, font refs, text runs, colors, booleans, numerics, strings, arrays, clusters, graphs, paths, rings/enums, refnums, decorations, and custom-control style records.
- [ ] Decode block-diagram semantic fields: primitive operand metadata, case selector values, frame ordering, loop terminals, tunnels, shift registers, event data nodes, sequence frames, formula nodes, property/invoke nodes, and subVI call-site metadata.
- [ ] Replace broad widget-kind heuristics with per-class decoders where the heap class has known required/optional fields.
- [x] Document unresolved heap tags in a generated report sorted by frequency and fixture provenance. `docs/generated/heap-tag-gaps.json` and `.md` now list unresolved/partial heap tags by corpus frequency with status buckets, FPHb/BDHb counts, fixture provenance, parent contexts, and next decode action; `internal/coverage` tests keep the artifacts current.

### 16.4 Wire Topology

- [ ] Finish `OF__compressedWireTable` semantics for comb and 4+ branch trees with controlled branch-move fixtures.
- [ ] Decode multi-elbow auto-chain payloads into exact waypoint lists rather than heuristic Manhattan reconstruction.
- [ ] Decode manual-chain payloads from known user-placed waypoint fixtures.
- [ ] Map wire glyph IDs, signal indices, and terminal associations so each rendered wire is linked to source/sink heap nodes without positional inference.
- [ ] Add round-trip and render tests that compare decoded wire waypoints against fixture screenshots or LabVIEW-exported geometry.

### 16.5 Type System

- [ ] Complete VCTP type-descriptor grammar for arrays, clusters, functions, typedefs, polymorphic VIs, refnums, variants, paths, strings, pictures, and complex/extended numeric types.
- [ ] Decode type-specific `Inner` payloads instead of leaving them as raw bytes once a grammar is confirmed.
- [ ] Connect DTHP/VCTP type references to every heap data-fill site, including complex data-fill leaves (`OF__real` / `OF__imaginary`).
- [ ] Add semantic diffs for type changes that report field-level changes instead of compressed-pool byte changes.

### 16.6 Link Info & Metadata Tails

- [ ] Decode opaque `Tail` spans inside `LIfp`, `LIbd`, and `LIvi` entries into named subrecords, version fields, flags, path variants, and secondary references.
- [ ] Expand LinkObjRef target decoding beyond currently typed subclasses; keep unknown subclasses as explicit enum gaps with fixture examples.
- [ ] Finish LVSR tail semantics: all known execution flags, breakpoint/debug fields, protection bits, save flags, and version-specific flag words.
- [ ] Resolve small-block field meanings for `HIST`, `VPDP`, `FPEx`, `BDEx`, `FPSE`, `BDSE`, `VITS`, `RTSG`, `MUID`, `DTHP`, and `BDPW` beyond current structural preservation.

### 16.7 Mutation Safety

- [ ] Promote fields from Tier 1 to Tier 2 only after encode/decode round-trip, LabVIEW open/save validation, and invariant tests over all affected corpus versions.
- [ ] Add mutation tests for every newly editable semantic field, including negative tests that reject inconsistent cross-resource updates.
- [ ] Define cross-resource consistency checks: heap type refs vs VCTP/DTHP, connector pane vs terminal refs, link info vs subVI nodes, and wire terminals vs diagram objects.
- [ ] Keep raw-patch/Tier 3 APIs quarantined from normal rewrite and metadata-edit flows.

### 16.8 Documentation Exit Criteria

- [ ] Every resource doc has a field table with offsets, sizes, encoding, version gates, confidence level, fixture evidence, and open questions.
- [ ] `docs/wire-layout.md` is upgraded from scaffold notes to a full RSRC container specification with examples.
- [ ] `docs/format-overview.md` clearly separates container guarantees, semantic guarantees, unsupported version ranges, and known lossy render approximations.
- [ ] The demo exposes unknown/opaque byte counts so users can see which parts of a file remain unmapped.

---

## Cross-Cutting Concerns

### Documentation

- [ ] `docs/format-overview.md`
- [ ] `docs/wire-layout.md`
- [ ] `docs/resource-registry.md` with per-resource binary layout, field table, version caveats
- [ ] `docs/safety-model.md`
- [ ] `docs/cli.md`
- [ ] `docs/contributing-reverse-engineering.md`

### Testing Hygiene

- [ ] Golden corpus tests pass on every PR
- [ ] Fuzz targets run in CI with 30s budget
- [ ] Property tests: `serialize(parse(x))` is valid
- [ ] Property tests: unchanged opaque resources survive editing
- [ ] Differential tests against Python oracle pass on corpus

### Release Tags

| Tag       | Content                                                                                   |
| --------- | ----------------------------------------------------------------------------------------- |
| `v0.1.0`  | parse + inspect + dump + list-resources                                                   |
| `v0.2.0`  | rewrite + round-trip tests                                                                |
| `v0.3.0`  | validate + diff + JSON schemas                                                            |
| `v0.4.0`  | metadata editing (set-meta)                                                               |
| `v0.5.x+` | typed resource growth (`vers`, `STRG`, icons, `CONP`/`CPC2`, link-info envelopes, `VCTP`) |
| `v0.6.0`  | small-block completion pass + colour icons + LVSR flags                                   |
| `v0.7.0`  | rich link graph (`LIvi`, PTH0/PTH1 path refs, typed LinkObjRef family)                    |
| `v0.8.0`  | VCTP navigation + connector-pane resolution/render                                        |
| `v0.9.0`  | front-panel heap (`FPHb`) decoder                                                         |
| `v1.0.0`  | block-diagram heap (`BDHb`), approximate FP/BD render, stable API                         |
| `v1.1.0`  | SVG/canvas front-panel + block-diagram rendering, CLI SVG export                          |
| `v1.2.0`  | LabVIEW geometry foundations, compressed wire semantics, and recognizable wire rendering   |
| `v1.3.0+` | LabVIEW-fidelity skins, text, colors, decorations, and exact wire waypoint rendering       |
| `v1.4.0+` | semantic byte coverage, heap/type/link closure, and evidence-backed format specification   |
