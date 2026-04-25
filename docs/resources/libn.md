# `LIBN` — Library Names

**FourCC:** `LIBN`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 13 corpus
sections (8 corpus fixtures do not carry `LIBN` because they are not
members of any `.lvlib`).

`LIBN` lists the names of the LabVIEW project libraries (`*.lvlib` files)
that this VI is a member of. The names are stored as raw bytes in
LabVIEW's text encoding; the codec preserves them verbatim, leaving any
charset interpretation to the caller.

## Wire layout

| Offset | Size | Field   | Notes                                          |
| -----: | ---: | ------- | ---------------------------------------------- |
|      0 |    4 | `Count` | Big-endian unsigned 32-bit number of names.    |
|      4 |    1 | `Len`   | First name's Pascal length byte.               |
|      5 |  Len | `Bytes` | First name's raw bytes (no NUL terminator).    |
|    ... |  ... | ...     | Remaining name pairs, no padding between them. |

Names use pylabview's `padto=1` Pascal-string variant, which means **no
alignment padding** between entries.

## Validation rules

| Severity | Code                     | Condition                            |
| -------- | ------------------------ | ------------------------------------ |
| error    | `libn.payload.malformed` | Payload could not be parsed cleanly. |

The decoder rejects payloads with truncated counts, truncated name bytes,
and trailing data after the last name.

## References

- pylabview `LIBN`: `LVblock.py:4683-4756` — confirms the count + Pascal-
  string layout and the round-trip path.
- pylabview `readPStr` / `preparePStr` with `padto=1`: `LVmisc.py:516-532`.

## Open questions

- Text encoding. pylabview defers encoding to the file's
  `vi.textEncoding`, which can vary by platform. The codec keeps the
  bytes opaque so callers (`pkg/lvvi`) can decode at the right moment.
- Whether multi-library membership (count > 1) ever occurs in the wild.
  All corpus samples carry exactly one entry; the decoder supports
  arbitrary counts so this is not a blocker.
