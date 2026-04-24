# `RTSG` — Runtime Signature GUID

**FourCC:** `RTSG`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 21 corpus
sections.

`RTSG` stores a 16-byte GUID identifying the LabVIEW runtime contract this
VI was compiled against.

## Wire layout

| Offset | Size | Field  | Notes                                           |
| -----: | ---: | ------ | ----------------------------------------------- |
|      0 |   16 | `GUID` | Raw bytes, preserved verbatim. No byte-order interpretation. |

**Total size:** 16 bytes.

## Validation rules

| Severity | Code               | Condition                          |
| -------- | ------------------ | ---------------------------------- |
| error    | `rtsg.payload.size` | Payload is not exactly 16 bytes.  |

## References

- pylabview `RTSG`: `LVblock.py:5383-5434` — reads exactly 16 bytes;
  exports as a hex string in XML; no byte reordering.

## Open questions

- Whether the GUID is stored in the Microsoft "mixed-endian" layout
  (first three groups little-endian, last two big-endian) or as a flat
  big-endian byte run. The codec preserves bytes verbatim either way;
  callers that need a UI display string should pick the convention that
  matches their corpus tooling.
