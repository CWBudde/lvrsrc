# `BDSE` — Block Diagram Size Estimate

**FourCC:** `BDSE`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 21 corpus
sections.

`BDSE` carries LabVIEW's estimate of the in-memory size of the block-diagram
object graph (sibling of `FPSE`).

## Wire layout

| Offset | Size | Field      | Notes                                  |
| -----: | ---: | ---------- | -------------------------------------- |
|      0 |    4 | `Estimate` | Big-endian unsigned 32-bit byte count. |

**Total size:** 4 bytes.

## Validation rules

| Severity | Code                | Condition                       |
| -------- | ------------------- | ------------------------------- |
| error    | `bdse.payload.size` | Payload is not exactly 4 bytes. |

## References

- pylabview `BDSE` (SingleIntBlock subclass): `LVblock.py:1383-1393` —
  byteorder='big', entsize=4, signed=False.
