# `HIST` — Changes History

**FourCC:** `HIST`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 21 corpus
sections. Field semantics not yet confirmed — see open questions.

`HIST` records edit history counters for the VI. pylabview ships only a
stub `Block` subclass with no parser, but the corpus shows every `HIST`
payload is exactly 40 bytes. This codec preserves those bytes verbatim
and exposes them as ten big-endian `uint32` slots via `(Value).Counters()`
so callers can compare counter values across files without committing
to specific slot semantics.

## Wire layout

| Offset | Size | Field | Notes                                              |
| -----: | ---: | ----- | -------------------------------------------------- |
|      0 |   40 | `Raw` | Ten consecutive big-endian uint32 counter slots.   |

**Total size:** 40 bytes (uniform across the corpus).

## Validation rules

| Severity | Code               | Condition                          |
| -------- | ------------------ | ---------------------------------- |
| error    | `hist.payload.size` | Payload is not exactly 40 bytes.  |

## References

- pylabview `HIST` (stub `Block` subclass): `LVblock.py:3078-3085`. The
  upstream class has no `parseRSRCSectionData` override, so the byte
  layout is inferred from corpus uniformity rather than from a published
  spec.

## Open questions

- The semantics of the ten counter slots. Likely candidates:
  total-edits-since-last-save, total-edits-ever, runs-since-last-save,
  etc. — but none have been confirmed against documented LabVIEW
  behaviour.
- Whether older or newer LabVIEW versions ever ship `HIST` at a size
  other than 40 bytes. The codec rejects anything else as invalid; if
  variant sizes appear, the decoder will need a length-aware shape.
