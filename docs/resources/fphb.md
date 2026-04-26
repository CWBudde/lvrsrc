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

## Coverage

- 21/21 corpus FPHb sections round-trip bit-for-bit.
- `FuzzDecode` (15 s, no panics) and `FuzzValidate` (10 s, no panics)
  exercise malformed envelopes and truncated tag streams; seeds drawn
  from the corpus.
- Wired into `pkg/lvvi.newLvviRegistry`, `pkg/lvdiff.defaultDecodedDiffers`,
  `internal/coverage.shippedCodecs`, and the WASM `typedFourCCs` set.

## What's still opaque

The codec resolves the tag-stream **structure** — every node's enum class,
every parent/child relation, every leaf's preserved payload bytes. It
deliberately does not yet:

- Decode per-class field payloads (control geometry, label fonts, scale
  ticks, …). Those carry domain-specific binary formats that vary by
  control type and are still being mapped from `pylabview`.
- Resolve `Tag(N)` fallbacks to higher-level categories. Tags that don't
  appear in any of the 40 enum tables in `tags_gen.go` are surfaced
  with their raw numeric form so coverage gaps stay visible in the
  demo.

These are tracked as Phase 11+ work (post-`v1.0`) and do not block any of
the read-only inspection / validation / safe-edit flows the codec
currently powers.

## References

- pylabview `LVblock.py:5350–5362` — `FPHb` / `BDHb` sibling subclasses
  with shared parsing.
- pylabview `LVheap.py` — full enum tables, mirrored into
  [`internal/codecs/heap/tags_gen.go`](../../internal/codecs/heap/tags_gen.go)
  by `scripts/gen-heap-tags`.
- [`docs/resources/lifp.md`](lifp.md) — sibling `LIfp` codec for the
  small front-panel metadata block that pairs with `FPHb`.
