# `BDHb` — Block-Diagram Heap

**FourCC:** `BDHb`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified against 21 corpus
BDHb sections (one per `.vi` / `.ctl` / `.vit` fixture).

`BDHb` is LabVIEW's block-diagram object graph — the persisted form of the
diagram canvas, every primitive / structure / wire-target on it, the
sub-VI references they resolve to, and the per-class fields that describe
their layout. The block uses the same ZLIB-wrapped tag-stream envelope as
its front-panel sibling [`FPHb`](fphb.md); the codec is a thin wrapper
over the shared [`internal/codecs/heap`](../../internal/codecs/heap)
framework. Both blocks share the `tags_gen.go` enum tables, so
block-diagram-specific tags (structures, primitives, wires, sub-VI
references) ride on the same `SystemTag` / `ClassTag` / `FieldTag`
vocabulary.

## Wire layout

The on-disk payload is a heap envelope (`heap.Envelope`):

| Offset | Size | Field        | Notes                                                                   |
| -----: | ---: | ------------ | ----------------------------------------------------------------------- |
|      0 |    4 | `Header`     | `HeapFormat` (typically `BinVerB` = 4) + flag bits. See `heap` package. |
|      4 |  N−4 | `Compressed` | ZLIB stream that inflates to a tag tree; walked by `heap.Walk`.         |

The inflated tag stream is identical in shape to `FPHb`'s; only the mix
of class-tags differs in practice (block-diagram graphs lean heavier on
`SubcosmTag`, `ConnectionTag`, structure tags, …).

## Decoded shape

`bdhb.Codec.Decode` returns `bdhb.Value`:

```go
type Value struct {
    Envelope heap.Envelope   // raw envelope bytes preserved for round-trip
    Tree     heap.WalkResult // walked tag tree (cycle-free; parent/child indices)
}
```

`Envelope.Compressed` is preserved verbatim so that `Codec.Encode` can
reuse the original ZLIB stream byte-for-byte (the corpus path) or
recompress from `Envelope.Content` when the caller cleared the cache (the
recompression fallback exercised by
`TestEncodeRecompressesWhenEnvelopeCacheCleared`).

For navigation, `pkg/lvvi.Model.BlockDiagram()` projects the same tree
into a render-friendly `lvvi.HeapTree` with class names resolved through
`lvvi.HeapTagName`. The web demo's _Block Diagram_ tab is the consumer.

Phase 11 layers the shared `internal/render.Scene` graph on top of that
tree. `internal/render.BlockDiagramScene()` emits grouped scene nodes,
logical bounds, placeholder markers, warnings, and a `ViewBox` that both
the CLI (`lvrsrc render --view block-diagram`) and the web demo consume.

## Coverage

- 21/21 corpus BDHb sections round-trip bit-for-bit;
  `TestEncodeRoundTripCorpus` re-emits **7 377 total tag entries**
  unchanged.
- `FuzzDecode` (15 s, no panics) and `FuzzValidate` (10 s, no panics)
  exercise malformed envelopes and truncated tag streams; seeds drawn
  from the corpus.
- Wired into `pkg/lvvi.newLvviRegistry`, `pkg/lvdiff.defaultDecodedDiffers`,
  `internal/coverage.shippedCodecs`, and the WASM `typedFourCCs` set.

## What's decoded

The codec resolves the tag-stream **structure** — every node's enum class,
every parent/child relation, every leaf's preserved payload bytes — and
the following typed leaf payloads:

- `OF__bounds` (Phase 11.1): 4 × big-endian `int16` Left/Top/Right/Bottom
  rectangles per `pylabview`'s `HeapNodeRect` (LVheap.py:1725). Decoded
  by `lvvi.HeapBounds` and the `lvvi.FindBoundsChild` helper; consumed
  by `internal/render` so block-diagram object boxes are positioned at
  real LabVIEW pixel coordinates whenever a node carries a bounds
  child. Corpus coverage shared with FPHb: **1188 / 1188** OF__bounds
  leaves across 42 trees.
- `OF__termBounds` (Phase 12.3): same 8-byte BE int16
  Left/Top/Right/Bottom rect format as `OF__bounds`, decoded by
  `lvvi.HeapTermBounds` / `lvvi.FindTermBoundsChild`. Carries the outer
  rectangle of a tunnel / terminal class (`SL__simTun`, `SL__sdfTun`,
  `SL__seqTun`, …) and is preferred over `OF__bounds` for sizing the
  scene-graph terminal anchor. Corpus coverage: **154 / 154**
  OF__termBounds leaves decode.
- `OF__termHotPoint` (Phase 12.3): 4 bytes BE int16 in Mac Point V/H
  order, decoded by `lvvi.HeapTermHotPoint` / `lvvi.FindTermHotPointChild`
  into a `lvvi.Point{V, H}`. Becomes the connect-point on the
  `NodeKindTerminal` scene node — wires (Phase 14) will attach
  there. Corpus coverage: **6 / 6** OF__termHotPoint leaves decode;
  terminals without a hot-point fall back to the bounds centre.
- Rectangle-shaped heap fields (Phase 16.3): `lvvi.HeapRect`,
  `lvvi.HeapRectForTag`, and `lvvi.FindRectChild` decode the shared
  8-byte rectangle payload for known `OF__*Bounds` / `OF__*Rect` leaves
  including `OF__contRect`, `OF__dBounds`, `OF__growAreaBounds`,
  `OF__iconBounds`, `OF__pBounds`, `OF__sizeRect`, and
  `OF__termBounds`. The accessors expose the bytes; scene-graph
  promotion remains role-specific.
- Common scalar and color fields (Phase 16.3): `lvvi.HeapScalar`,
  `HeapScalarForTag`, and `FindScalarChild` expose observed integer /
  flag / count / id leaves such as `OF__objFlags`, `OF__howGrow`,
  `OF__partID`, `OF__masterPart`, and `OF__primIndex`; `lvvi.HeapColor`,
  `HeapColorForTag`, and `FindColorChild` expose 4-byte color-like
  leaves such as `OF__bgColor`, `OF__fgColor`, `OF__borderColor`, and
  `OF__structColor`. These are byte-shape projections only: bit names,
  enum meanings, color-space prefix semantics, and system-color
  sentinels still need controlled fixtures.

## What's still opaque

- Wire path drawing. Phase 13 is mapping the persisted-wire data. The
  first pass added a presence accessor —
  pylabview's `LVheap.py` has the enum number only, so the format
  was reverse-engineered against controlled-fixture spikes. Phase 13.3 shipped
  the typed `lvvi.HeapWire` decoder that classifies each
  `OF__compressedWireTable` chunk by mode (auto-chain `0x08`,
  manual-chain `0x04`, tree `0x00`, or other) and projects the
  payload into either an LEB128 varint stream (chain modes) or
  2-byte records (tree mode). The scene-graph projection now
  surfaces a per-mode breakdown — e.g. _"Block diagram has 4 wire
  networks (4 auto-routed, 0 manually-routed, 0 branched, 0
  other); auto-routed L-shapes and 2- and 3-branch pure Y-trees
  are typed-decoded, multi-elbow / comb and 4+ branch chunks remain
  raw (Phase 13.5)."_ Corpus coverage of `HeapWire` is **101 / 101**
  across the 40-fixture corpus (86 auto-chain, 3 manual-chain, 10 tree,
  2 other). Phase 13.4 then layered typed projections on top:
  `Wire.ChainAutoPath()` exposes `{Straight, YStep, SourceAnchorX}`
  for the most common wire shapes, and `Wire.TreeEndpoints()`
  returns `[]Point{V, H}` endpoint coordinates for pure Y-trees
  (2-branch confirmed by geometry-varied fixture; 3-branch confirmed by
  `Numeric42ThreeIndicatorsY_bottom8pxdown.vi` and independently matched by
  `reference-find-by-id.vi`). `Wire.TreeEndpointPair()` is a 2-branch
  convenience wrapper. Both projections are ground-truthed against
  controlled-fixture diffs (8/16 px y-shift, x-shift, sign flip,
  geometry-varied 2-Y/3-Y). The renderer composes the chain-auto
  projection with terminal `OF__bounds` at draw time: source +
  `SourceAnchorX` horizontally → `YStep` vertically → continue
  horizontally to sink. Multi-elbow auto-chains, manual-chains,
  and comb / 4+ branch trees stay raw until Phase 13.5 is complete.
  The current comb spike (`Numeric42ThreeIndicatorsYComb_middle8pxdown.vi`)
  shows the moved middle branch changes two adjacent records in opposite
  directions (`5b 01` → `63 01`, `57 42` → `4f 42`), so the comb payload
  carries span/junction data rather than just endpoint records.
- Terminal anchor decoding shipped as Phase 12.3 (`OF__termBounds` +
  `OF__termHotPoint`) — see the "What's decoded" section above; the
  literal `OF__terminal` (FieldTag 367) carries no payload in the
  21-fixture corpus and pylabview's `LVheap.py` has no decoder for
  it, so it remains an opaque fallback.
- Per-primitive operand metadata (selector ranges, frame counts on Case
  structures, sequence-frame ordering, …). These are domain-specific
  and still being mapped from `pylabview`'s per-primitive decoders.
- Rectangle role semantics beyond the promoted outer object and
  terminal rectangles: several known rectangle leaves now decode
  generically, but controlled fixtures still need to identify which
  tags affect rendered block-diagram geometry.
- Unresolved `Tag(N)` fallbacks: tags that don't appear in any of the
  40 enum tables in `tags_gen.go` surface with their raw numeric form
  so coverage gaps stay visible in the demo.

These are tracked as Phase 11.2+ work (post-`v1.0`) and do not block any
of the read-only inspection / validation / safe-edit flows the codec
currently powers.

## Render/export semantics

Phase 11.1 turned scene rendering into a hybrid of decoded and heuristic
layout:

- Groups whose heap node has an `OF__bounds` child are positioned and
  sized at the decoded LabVIEW pixel rectangle.
- Groups without a decoded bounds child fall back to the prior
  heuristic stack (vertical, indented by depth).
- The OF__bounds leaf itself is dropped from scene output once it has
  been promoted onto the parent.
- Unresolved classes remain visible as placeholder nodes with their
  `Tag(N)` label and parent path.
- Wire routing and terminal positions are not rendered yet; the
  renderer still emits the "wire routing not rendered" warning until
  Phase 11.5 lands.

Phase 12.2a added widget-kind classification on top of that hybrid
layout. Block-diagram nodes resolve to `structure` (loops, sequences,
case selectors, event structures, …) or `primitive` (`SL__prim`,
`SL__node`, property / invoke / call-by-ref nodes, build-array /
index / decompose primitives, formula nodes, …) via
`lvvi.WidgetKindForNode`. The shared SVG renderer emits an
`lvrsrc-widget-{kind}` CSS class alongside the existing
`lvrsrc-node-*` classes — structures get a heavier orange-brown
stroke, primitives a navy-tinted fill, decorations a dashed gray
outline.

Phase 14.1 cross-checked the table against pylabview's `LVheap.py`
class enum and per-class child-tag dispatch. Reference-bearing classes
(`SL__stdRefNum`, `SL__baseRefNum`, static/dynamic VI/control refs)
now resolve to `refnum`; `SL__stdVar` / `SL__stdLvVariant` /
`SL__oleVariant` resolve to `variant`; and `SL__conPane` resolves to
`connector-pane`. Tunnel classes stay folded into `terminal`, matching
the renderer's wire-anchor model. Unmapped classes fall back to
`other`.

## References

- pylabview `LVblock.py:5350–5362` — `FPHb` / `BDHb` sibling subclasses
  with shared parsing.
- pylabview `LVheap.py` — full enum tables, mirrored into
  [`internal/codecs/heap/tags_gen.go`](../../internal/codecs/heap/tags_gen.go)
  by `scripts/gen-heap-tags`.
- [`docs/resources/libd.md`](libd.md) — sibling `LIbd` codec for the
  small block-diagram metadata block that pairs with `BDHb`.
