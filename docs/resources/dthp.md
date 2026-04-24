# `DTHP` — Data Types for Heap

**FourCC:** `DTHP`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 21 corpus
sections.

`DTHP` describes the slice of the VCTP type list that the front-panel and
block-diagram heaps reference. From LabVIEW 8.0 onward, the slice is always
at the tail of `VCTP`, so `IndexShift + TDCount` equals the total VCTP entry
count.

## Wire layout

DTHP uses pylabview's "U2 plus 2" variable-size encoding for both fields.
A 16-bit big-endian word represents the value directly when its high bit
(`0x8000`) is clear; when set, the low 15 bits are shifted up by 16 and
combined with the next 16-bit word to form a 31-bit value. See
`references/pylabview/pylabview/LVmisc.py:336` (`readVariableSizeFieldU2p2`).

| Offset | Size | Field        | Notes                                                                        |
| -----: | ---: | ------------ | ---------------------------------------------------------------------------- |
|      0 |  2/4 | `TDCount`    | Variable-size U2p2: number of TypeDescs in the heap-used slice.              |
|        |  2/4 | `IndexShift` | Variable-size U2p2: starting index into VCTP. **Omitted when TDCount = 0.**  |

**Total size:** 2 bytes when `TDCount` is zero (no IndexShift field).
Otherwise the sum of both fields' encoded widths — typically 4 bytes in
the corpus (both values fit in 15 bits).

## Validation rules

| Severity | Code                  | Condition                                    |
| -------- | --------------------- | -------------------------------------------- |
| error    | `dthp.payload.malformed` | Payload could not be parsed (short/trailing). |

## References

- pylabview `DTHP`: `LVblock.py:3177-3278` — confirms the LV8.0+ shape and
  the "no shift when count is zero" rule.
- pylabview `readVariableSizeFieldU2p2` / `prepareVariableSizeFieldU2p2`:
  `LVmisc.py:336-355` — the codec's serialisation helper.

## Open questions

- LabVIEW 7.1 and earlier use a different layout (per pylabview's
  `NotImplementedError`). This codec rejects such payloads as malformed;
  if older fixtures appear in the corpus, the legacy form will need its
  own decode path.
- Whether `IndexShift + TDCount` should be cross-validated against the
  actual VCTP entry count. That belongs to a higher-level validator
  (e.g. `pkg/lvvi`) once VCTP is surfaced through `Model.Types()` in
  Phase 8.
