# Writer Differences — Known Round-Trip Gaps

This document tracks how `rsrcwire.Serialize(rsrcwire.Parse(x))` compares to the
original bytes `x`, and what remains before the PLAN.md Phase 2 goal of a
byte-exact preserving writer is met.

## Current state

Measured across the 21-file corpus under `testdata/corpus/` (see
`internal/rsrcwire/testdata/golden/*.golden.json`):

| File kind                    | Files | Size-match | Byte-diffs per file |
| ---------------------------- | ----: | ---------- | ------------------: |
| `.vi` (LVIN)                 |    11 | 100%       |       **exactly 1** |
| `.ctl` (LVCC)                |    10 | 100%       |       **exactly 1** |
| synthetic (crafted in tests) |     1 | 100%       |                   0 |

Every production file round-trips to the **same length** but differs in
**exactly one byte**, and that byte is always at the same logical location:
the low-order byte of `BlockInfoList.BlockInfoSize` inside the info-section
list header (absolute offset `header.InfoOffset + 51`).

## Diagnosis

`BlockInfoSize` is a wire-format uint32 that declares the size of the
block-info region following the list header. The parser reads it, the
serializer recomputes it from the blocks it emitted, and the recomputed value
is consistently **smaller** than the original.

Sample deltas (original − serialized):

| File                       |    In |   Out | Delta (bytes) |
| -------------------------- | ----: | ----: | ------------: |
| `action.ctl`               | 0x368 | 0x33f |            41 |
| `is-int.vi`                | 0x358 | 0x32e |            42 |
| `format-string.vi`         | 0x378 | 0x355 |            35 |
| `module-data--cluster.ctl` | 0x3c8 | 0x3ad |            27 |

The delta varies per file, suggesting the original LabVIEW writer reserves
trailing space inside the declared block-info region (possibly for alignment,
compressor scratch, or historical padding) that our serializer tightens out.
Because the overall file length is still correct, offsets into
`header.InfoOffset + BlockInfoSize` also land correctly — the declared size is
simply less conservative than the original declaration. No parse error, no
data loss; but not byte-exact.

## Acceptable for now

- **Structural round-trip is preserved.** `Parse(Serialize(Parse(x)))` yields
  a `*File` whose `Header`, block-section tree, name table, and payload bytes
  are equal to the original parse.
- **Size is preserved.** Downstream consumers that rely on `len(output) ==
len(input)` are unaffected.
- **The validator accepts both.** `internal/validate` does not flag the
  reduced `BlockInfoSize` because the name table and name-offsets still fall
  within the declared region.

## Not yet acceptable

- **Byte-exact round-trip.** PLAN.md Phase 2 calls this out as the goal for
  `v0.2.0`. We are one byte away across the whole corpus, which is both
  tantalising and a clear single-point fix.
- **Producer attribution.** Without byte-exact output, we cannot yet claim
  "this file was only read by `lvrsrc`" in audit scenarios.

## Path to closure

1. Capture the original `BlockInfoSize` value when parsing into a new
   `BlockInfoList.OriginalBlockInfoSize uint32` field (preserved across
   round-trip by the serializer when present).
2. In `Serialize`, prefer that preserved value over a recomputed one **when
   it is ≥ the recomputed minimum**. If a caller mutates blocks in a way that
   needs more space, fall back to the recomputed minimum.
3. Add a regression row to `TestCorpusGolden` goldens so diffs drop from 1 →
   0 across the corpus, and any future regression is immediately caught.

## How to verify

```sh
go test ./internal/rsrcwire -run TestCorpusGolden
```

Fails on byte-diff changes. To accept intentional changes:

```sh
UPDATE_GOLDEN=1 go test ./internal/rsrcwire -run TestCorpusGolden
git diff internal/rsrcwire/testdata/golden/
```

Review the diff before committing — especially the `round_trip.byte_diffs`
field, which should trend toward 0 as the writer approaches byte-exactness.
