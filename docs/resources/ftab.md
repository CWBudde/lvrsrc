# `FTAB` — Font Table

**FourCC:** `FTAB`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 21 corpus
sections.

`FTAB` is the VI's font registry. It carries a section-level header, a
fixed-width entry table whose width depends on the header flags, and a
trailing pool of Pascal-string font names referenced by per-entry
offsets. Bit `0x00010000` of the header `Prop1` selects between the
12-byte ("narrow") entry shape and the 16-byte ("wide") shape; every
shipped corpus sample uses the wide shape.

## Wire layout

```text
+-------------------+  offset 0
|  Prop1 (u32 BE)   |
|  Prop3 (u16 BE)   |
|  Count (u16 BE)   |  count <= 127
+-------------------+  offset 8
|  Entry table      |  Count × {12 | 16} bytes
+-------------------+  offset 8 + Count×entrySize
|  Name pool        |  contiguous Pascal-strings (length byte + bytes)
+-------------------+
```

### Per-entry layout

| Offset | Size | Field      | Notes                                          |
| -----: | ---: | ---------- | ---------------------------------------------- |
|      0 |    4 | `NameOffs` | Absolute offset into the payload to the name. `0` means no name. |
|      4 |    2 | `Prop2`    | Big-endian u16, semantics not documented.      |
|      6 |    2 | `Prop3`    | Big-endian u16.                                |
|      8 |    2 | `Prop4`    | Big-endian u16.                                |

Then either:

- **Narrow** (`Prop1 & 0x00010000 == 0`, 12-byte entries):
  - offset 10, size 2: `Prop5` (BE u16).
- **Wide** (`Prop1 & 0x00010000 != 0`, 16-byte entries):
  - offset 10, size 2: `Prop6` (BE u16).
  - offset 12, size 2: `Prop7` (BE u16).
  - offset 14, size 2: `Prop8` (BE u16).

## Round-trip notes

pylabview's encoder appends a fresh Pascal-string per entry rather than
sharing offsets across duplicate names. This Go implementation matches
that algorithm exactly, so re-encoding the decoded form reproduces the
original payload byte-for-byte across the full corpus — including the
common case where the same font name (e.g. `"Segoe UI"`) appears under
distinct offsets in different entries.

## Validation rules

| Severity | Code                  | Condition                                           |
| -------- | --------------------- | --------------------------------------------------- |
| error    | `ftab.payload.malformed` | Payload could not be parsed (header truncated, oversized count, truncated entry table, or out-of-bounds / overrunning name reference). |

## References

- pylabview `FTAB`: `LVblock.py:2892-3075` — confirms the entry-width
  flag, the 127-entry limit, and the per-entry append-order name pool.
- pylabview `readPStr` / `preparePStr` with `padto=1`: `LVmisc.py:516-532`
  — names use the padto=1 variant (no padding between names).

## Open questions

- The semantics of `Prop1` bits other than `0x00010000`. Three values
  observed in the corpus: `0x00010002`, `0x00010003`, `0x00010005`.
- The semantics of `Prop2`/`Prop3`/`Prop4`/`Prop5`/`Prop6`/`Prop7`/`Prop8`.
  pylabview names them generically and does not document their meaning.
- Whether the implicit "name pool follows entry table contiguously"
  layout is universal or whether other LabVIEW versions interleave or
  pad.
