# Type Descriptor Codec: `VCTP`

This note captures the current implementation state for the Phase 5.2 type
descriptor resource codec.

## Status

`VCTP` is now a shipped typed codec. It models the stable outer envelope the
corpus supports:

- `u32` declared uncompressed size
- zlib-compressed descriptor-pool bytes

The codec inflates the pool and exposes the uncompressed bytes as a typed blob,
but does not yet claim to understand individual type records inside that pool.

Safety tier: Tier 1.

Implementation package: `internal/codecs/vctp`.

## Corpus Evidence

From the current `testdata/corpus/` set:

- `VCTP` appears in 21/21 fixtures
- every observed payload begins with a 4-byte big-endian size prefix
- the remaining bytes form a valid zlib stream in all current samples
- the inflated byte count matches the declared size in all current samples

Representative samples in the corpus start with:

- `00 00 00 84 78 9c ...`
- `00 00 01 be 78 9c ...`
- `00 00 02 40 78 9c ...`

That envelope is stable enough for a narrow codec even though the descriptor
records inside the inflated pool are not yet modeled semantically.

## Current Decoded Shape

The shipped model exposes:

- `DeclaredSize uint32`
- `Inflated []byte`
- `Compressed []byte`

`Compressed` is preserved so decode followed by encode can round-trip the exact
original bytes. `lvdiff` intentionally ignores that field and compares the
inflated descriptor blob instead, which keeps decoded diffs focused on semantic
pool content rather than zlib encoding artifacts.

## Validation Scope

The codec validates only invariants supported by the corpus:

- minimum payload length
- valid zlib-compressed payload
- inflated byte count matches the declared size

Descriptor-level interpretation remains out of scope for Tier 1.

## Remaining Work

The obvious follow-on work is semantic decoding of the inflated descriptor
records inside the pool. That likely belongs in a later phase once the type
record grammar is documented well enough to avoid over-claiming field meaning.
