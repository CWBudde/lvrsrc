# `FPHb` — Front-Panel Heap

**FourCC:** `FPHb`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified against 21 corpus
FPHb sections (one per `.vi` / `.ctl` / `.vit` fixture).

`FPHb` is LabVIEW's front-panel object graph — the persisted form of the
panel canvas, every control and indicator on it, their captions and labels,
and the per-class fields that drive their on-screen appearance. The block
is a ZLIB-compressed envelope wrapping a tag-stream tree; the codec
preserves the envelope layout exactly and decodes the tag tree to a typed
node graph via the shared [`internal/codecs/heap`](../../internal/codecs/heap)
framework that `BDHb` also uses.

## Wire layout

The on-disk payload is a heap envelope (`heap.Envelope`):

| Offset | Size | Field        | Notes                                                                   |
| -----: | ---: | ------------ | ----------------------------------------------------------------------- |
|      0 |    4 | `Header`     | `HeapFormat` (typically `BinVerB` = 4) + flag bits. See `heap` package. |
|      4 |  N−4 | `Compressed` | ZLIB stream that inflates to a tag tree; walked by `heap.Walk`.         |

The inflated tag stream is a sequence of `(tag, scope, payload)` records
described in [`docs/resources/`](../resources) as the tag-stream model
used jointly by `FPHb` / `BDHb`. Tag numbers are typed against
`internal/codecs/heap/tags_gen.go`'s enum tables (`SystemTag`,
`ClassTag`, `FieldTag`, …) generated from `LVheap.py`.

## Decoded shape

`fphb.Codec.Decode` returns `fphb.Value`:

```go
type Value struct {
    Envelope heap.Envelope   // raw envelope bytes preserved for round-trip
    Tree     heap.WalkResult // walked tag tree (cycle-free; parent/child indices)
}
```

`Envelope.Compressed` is preserved verbatim so that re-encoding can either
reuse the original ZLIB stream byte-for-byte (the corpus path) or
recompress from `Envelope.Content` when the caller cleared the cache (the
recompression fallback exercised by
`TestEncodeRecompressesWhenEnvelopeCacheCleared`).

For navigation, `pkg/lvvi.Model.FrontPanel()` projects the same tree into
a render-friendly `lvvi.HeapTree` with class names resolved through
`lvvi.HeapTagName`. The web demo's _Front Panel_ tab is the consumer.

Phase 11 adds a second projection on top of that tree: the shared
`internal/render.Scene` graph. `internal/render.FrontPanelScene()` turns
the decoded heap into grouped scene nodes with logical bounds, labels,
placeholder markers, and a scene-level `ViewBox`. Both the CLI
(`lvrsrc render --view front-panel`) and the web demo's `Visual` / `Canvas`
modes use that shared projection.

## Coverage

- 21/21 corpus FPHb sections round-trip bit-for-bit.
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
  by `internal/render` so the scene graph places groups at real
  LabVIEW pixel positions whenever a control carries a bounds child.
  Corpus coverage: **1188 / 1188** OF__bounds leaves across 42 FPHb +
  BDHb trees.

## What's still opaque

- Per-class field payloads other than `OF__bounds` (label fonts, scale
  ticks, button geometry beyond the outer rect, custom controls, …).
  Those carry domain-specific binary formats that vary by control type
  and are still being mapped from `pylabview`.
- Wire routing (`OF__wireTable`, `OF__wireID`, `OF__wireGlyphID`) and
  terminal positions (`OF__terminal`) — recognised as tags but content
  bytes left raw. Tracked as Phase 11.3–11.5.
- Other rectangle-shaped tags (`OF__contRect`, `OF__dBounds`,
  `OF__pBounds`, `OF__iconBounds`, …): the binary format is identical
  to `OF__bounds` but they are not yet promoted onto scene-graph
  geometry; only the outer `OF__bounds` rectangle is consumed today.
- Unresolved `Tag(N)` fallbacks: tags that don't appear in any of the
  40 enum tables in `tags_gen.go` are surfaced with their raw numeric
  form so coverage gaps stay visible in the demo.

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
  been promoted onto the parent — its content is metadata, not visible.
- Unresolved classes remain visible as placeholder nodes with their
  `Tag(N)` label and parent path.
- The web demo's "Layout is heuristic" warning now reads as "_some_
  positions and sizes are heuristic" and only appears when at least one
  root falls back, so a fully bounds-driven scene is no longer flagged
  as approximate.

Phase 12.2a layered a generic widget-kind classification on top of that
hybrid layout. Each scene group / box carries an `lvvi.WidgetKind`
(`boolean` / `numeric` / `string` / `cluster` / `array` / `graph` /
`decoration` / `structure` / `primitive` / `other`) resolved from the
heap node's class tag via `lvvi.WidgetKindForNode`. The shared SVG
renderer (`internal/render.SVG`) emits an `lvrsrc-widget-{kind}` CSS
class alongside the existing `lvrsrc-node-*` classes and ships
distinct per-kind skins — booleans get a green-tinted fill, numerics
blue, strings purple, structures a heavier orange-brown stroke, and so
on. The mapping in `widgetKindByClass` covers the obvious-by-name
classes (`SL__stdBool`, `SL__stdNum`, `SL__forLoop`, `SL__prim`,
…); unmapped classes fall back to `other` so unknown widgets still
render as plain placeholder boxes. A pylabview-aligned cross-check
pass (Phase 12.2b) is pending and will tighten the table where the
name heuristic disagrees with pylabview's actual per-class parser
dispatch.

## Typed data fills (`OF__StdNumMin` / `Max` / `Inc`)

pylabview's `HeapNodeTDDataFill` (LVheap.py:1911) interprets the
content bytes of `OF__StdNumMin` (FieldTag 513), `OF__StdNumMax` (514),
and `OF__StdNumInc` (515) leaves as a numeric value of the surrounding
object's TypeDesc — the min / max / increment range stored on a
numeric control or indicator.

`pkg/lvvi.Model.HeapDataFill(tree, nodeIdx)` exposes this as a typed
projection:

1. Locate the node's parent (the surrounding `TagOpen`).
2. Find the parent's `OF__typeDesc` child (FieldTag 283); its content
   is the heap-local TypeID (a signed BE integer ≥ 1).
3. Resolve via `VCTP.descriptors[DTHP.IndexShift + heapTypeID − 1]`.
4. Switch on the resolved descriptor's `FullType` to pick a typed
   decoder.

Supported numeric kinds (`DataFillKind`): `Int` (NumInt8/16/32/64),
`UInt` (NumUInt8/16/32/64 — sign-extended, then masked to width),
`Float32` (NumFloat32), `Float64` (NumFloat64). Anything else
(Boolean, Cluster, String, Function, refnum, complex, quad-float, …)
falls back to `Kind = Raw` with the resolved `FullType` recorded so
callers can still display the type name.

Content lengths reflect pylabview's `shrinkRepeatedBits` truncation
(LVheap.py:1942): an Int32 value of 0 may be encoded as a single
`0x00` byte. The decoder sign-extends `len(Content)` bytes into the
declared width.

`DataFillValue.Raw` always holds the original content bytes regardless
of `Kind`, so the heap codec stays round-trip-safe — typed access is
strictly a projection.

Coverage: across the 21-fixture corpus, **75** DataFill nodes were
swept; **21** resolved to typed numeric kinds (12 `Int` from
`load-vi.vi`-style `NumInt32` ranges, 9 `UInt` from
`action.ctl` / `datatypes.ctl` `NumUInt16` ranges) and the remainder
fell to `Raw` on non-numeric TDs or `Unknown` when no DTHP / parent
typeDesc could be located. The complex-leg form
(`HeapNodeTDDataFillLeaf` for `OF__real` / `OF__imaginary`) has no
fixture coverage and therefore no typed decoder yet — it would
currently return `ok = false` since those tags are not in the
DataFill tag set.

## References

- pylabview `LVblock.py:5350–5362` — `FPHb` / `BDHb` sibling subclasses
  with shared parsing.
- pylabview `LVheap.py` — full enum tables, mirrored into
  [`internal/codecs/heap/tags_gen.go`](../../internal/codecs/heap/tags_gen.go)
  by `scripts/gen-heap-tags`.
- pylabview `LVheap.py:1911-2295` — `HeapNodeTDDataFill` /
  `HeapNodeTDDataFillLeaf` reference for the typed-fill projection.
- pylabview `LVblock.py:3280-3292` — `Block.getHeapTD` (DTHP
  IndexShift + VCTP top-type lookup).
- [`docs/resources/lifp.md`](lifp.md) — sibling `LIfp` codec for the
  small front-panel metadata block that pairs with `FPHb`.
