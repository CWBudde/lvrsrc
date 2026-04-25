# `MUID` — Map Unique Identifier

**FourCC:** `MUID`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 21 corpus
sections.

`MUID` stores the maximum object UID used by the VI's LoadRefMap at save
time. LabVIEW assigns each object inside the VI a fresh UID whenever it
changes, so this counter effectively tracks total edits to the file.

## Wire layout

| Offset | Size | Field | Notes                               |
| -----: | ---: | ----- | ----------------------------------- |
|      0 |    4 | `UID` | Big-endian unsigned 32-bit integer. |

**Total size:** 4 bytes.

## Validation rules

| Severity | Code                | Condition                       |
| -------- | ------------------- | ------------------------------- |
| error    | `muid.payload.size` | Payload is not exactly 4 bytes. |

## References

- pylabview `MUID` (SingleIntBlock subclass): `LVblock.py:1272-1286` —
  byteorder='big', entsize=4, signed=False.

## Open questions

- Whether the UID space is global per VI or scoped per heap. pylabview
  documents it as a per-VI maximum across the LoadRefMap.
