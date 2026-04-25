# `VPDP` — VI Probe-Data Pointer

**FourCC:** `VPDP`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 21 corpus
sections (all observed values are `0x00000000`).

`VPDP` is labeled "VI Primitive Dependency Flags" in pylabview but the
class is a stub (no parsing). The shipped corpus carries it as a uniform
4-byte payload of all zeros, so this codec exposes it as an opaque
big-endian uint32 — round-trip preserves the value exactly and callers
can verify it stayed at the expected sentinel.

## Wire layout

| Offset | Size | Field   | Notes                             |
| -----: | ---: | ------- | --------------------------------- |
|      0 |    4 | `Flags` | Big-endian unsigned 32-bit value. |

**Total size:** 4 bytes.

## Validation rules

| Severity | Code                | Condition                       |
| -------- | ------------------- | ------------------------------- |
| error    | `vpdp.payload.size` | Payload is not exactly 4 bytes. |

## References

- pylabview `VPDP` (`Block` stub): `LVblock.py:5055-5061`.

## Open questions

- The bit semantics. The "Primitive Dependency Flags" label suggests this
  is a bitfield, but every corpus sample is zero so we cannot infer
  individual bit meanings.
- Whether non-zero VPDP values exist in older or unusual LabVIEW saves.
  If they appear in a broader corpus, this codec should be revisited.
