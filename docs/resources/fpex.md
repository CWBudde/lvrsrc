# `FPEx` — Front-Panel "Extra" Heap-Aux Block

**FourCC:** `FPEx`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 21 corpus
sections.

`FPEx` is a small auxiliary block that appears alongside the front-panel
heap (`FPHb`). pylabview does not classify it; the layout is inferred
from corpus uniformity:

- 14 of 21 sections are 4 bytes (`Count = 0`).
- 6 of 21 are 8 bytes (`Count = 1`).
- 1 section is 16 bytes (`Count = 3`).

In every observed payload the entries themselves are zero, so they look
like reserved slots LabVIEW preallocates for a future heap-aux mechanism
without committing data yet.

## Wire layout

| Offset | Size | Field     | Notes                                           |
| -----: | ---: | --------- | ----------------------------------------------- |
|      0 |    4 | `Count`   | Big-endian unsigned 32-bit entry count.         |
|      4 |  4×N | `Entries` | `Count` BE-uint32 entries (always zero so far). |

**Total size:** `4 + 4*Count` bytes.

## Validation rules

| Severity | Code                     | Condition                                                                        |
| -------- | ------------------------ | -------------------------------------------------------------------------------- |
| error    | `fpex.payload.malformed` | Payload size does not equal `4 + 4*Count`, or the count field itself is missing. |

## References

- pylabview does not contain a class for `FPEx`. The layout above is
  inferred from corpus inspection only — see the development probe in
  the Phase 6.3g task notes.

## Open questions

- The semantics of the entries. They are uniformly zero in the corpus,
  so we cannot tell whether they are pointers, indices, or flags.
- Whether `FPEx`'s count is correlated with `FPHb` content (e.g. number
  of heap fragments). Worth revisiting once Phase 9 (`FPHb` decoder)
  lands.
