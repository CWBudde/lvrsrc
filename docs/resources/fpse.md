# `FPSE` — Front Panel Size Estimate

**FourCC:** `FPSE`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 21 corpus
sections.

`FPSE` carries LabVIEW's estimate of the in-memory size of the front-panel
object graph (used as a sizing hint when re-loading the VI).

## Wire layout

| Offset | Size | Field      | Notes                                  |
| -----: | ---: | ---------- | -------------------------------------- |
|      0 |    4 | `Estimate` | Big-endian unsigned 32-bit byte count. |

**Total size:** 4 bytes.

## Validation rules

| Severity | Code                | Condition                       |
| -------- | ------------------- | ------------------------------- |
| error    | `fpse.payload.size` | Payload is not exactly 4 bytes. |

## References

- pylabview `FPSE` (SingleIntBlock subclass): `LVblock.py:1288-1298` —
  byteorder='big', entsize=4, signed=False.
