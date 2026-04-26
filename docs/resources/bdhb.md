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

## What's still opaque

The codec resolves the tag-stream **structure** — every node's enum class,
every parent/child relation, every leaf's preserved payload bytes. It
deliberately does not yet:

- Decode wire routing (waypoint coordinates, hops, label anchors). The
  v1 demo skips wire rendering altogether; only the diagram object tree
  is shown.
- Decode per-primitive operand metadata (selector ranges, frame counts
  on Case structures, sequence-frame ordering, …). These are
  domain-specific and still being mapped from `pylabview`'s
  per-primitive decoders.
- Resolve `Tag(N)` fallbacks. Tags that don't appear in any of the 40
  enum tables in `tags_gen.go` surface with their raw numeric form so
  coverage gaps stay visible in the demo.

These are tracked as Phase 11+ work (post-`v1.0`) and do not block any of
the read-only inspection / validation / safe-edit flows the codec
currently powers.

## Render/export semantics

Current block-diagram rendering is intentionally structural:

- SVG and canvas output come from inferred scene-graph bounds, not decoded
  primitive coordinates or persisted wire geometry.
- Unresolved classes remain visible as placeholder nodes with their
  `Tag(N)` label and parent path.
- Wire routing and terminal positions are not rendered yet; the renderer
  emits explicit warnings and the web demo surfaces them inline.

## References

- pylabview `LVblock.py:5350–5362` — `FPHb` / `BDHb` sibling subclasses
  with shared parsing.
- pylabview `LVheap.py` — full enum tables, mirrored into
  [`internal/codecs/heap/tags_gen.go`](../../internal/codecs/heap/tags_gen.go)
  by `scripts/gen-heap-tags`.
- [`docs/resources/libd.md`](libd.md) — sibling `LIbd` codec for the
  small block-diagram metadata block that pairs with `BDHb`.
