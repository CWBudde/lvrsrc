# `BDEx` — Block-Diagram "Extra" Heap-Aux Block

**FourCC:** `BDEx`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 21 corpus
sections.

`BDEx` is the block-diagram sibling of `FPEx`. Same wire shape, same
"reserved slots that are zero in the corpus" pattern. pylabview does
not classify it.

Corpus distribution: 15 sections at 4 bytes (`Count = 0`), 5 at 8 bytes
(`Count = 1`), 1 at 12 bytes (`Count = 2`).

## Wire layout

| Offset | Size | Field     | Notes                                   |
| -----: | ---: | --------- | --------------------------------------- |
|      0 |    4 | `Count`   | Big-endian unsigned 32-bit entry count. |
|      4 |  4×N | `Entries` | `Count` BE-uint32 entries.              |

**Total size:** `4 + 4*Count` bytes.

## Validation rules

| Severity | Code                     | Condition                                                                        |
| -------- | ------------------------ | -------------------------------------------------------------------------------- |
| error    | `bdex.payload.malformed` | Payload size does not equal `4 + 4*Count`, or the count field itself is missing. |

## References

- See `docs/resources/fpex.md` for the reverse-engineering notes; `BDEx`
  shares the inferred structure.

## Open questions

- Same as FPEx: entry semantics unknown, correlation with `BDHb` unclear.
