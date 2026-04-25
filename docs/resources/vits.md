# `VITS` — Virtual Instrument Tag Strings

**FourCC:** `VITS`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 21 corpus
sections totalling 33 tag entries. Variant content is preserved as opaque
bytes — interpreting it requires the LVdatafill machinery, which is out
of scope for Phase 6.3.

`VITS` carries an ordered list of (name, LVVariant) pairs that LabVIEW
uses to attach miscellaneous settings and metadata to a VI. Common
corpus tags include `NI.LV.All.SourceOnly` (separate-source-and-binary
flag) and `NI_IconEditor` (saved icon-editor state, often kilobytes long
because it embeds an `RSRC` file).

## Wire layout

```text
+---------------------+  offset 0
|  Count (u32 BE)     |
+---------------------+
|  Entry[0]           |
|    NameLen (u32 BE) |
|    Name (NameLen B) |
|    VarLen  (u32 BE) |
|    Variant (VarLen B)
+---------------------+
|  Entry[1] ...       |
+---------------------+
```

## Per-entry shape

| Offset | Size | Field     | Notes                                                                                                                                                                   |
| -----: | ---: | --------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
|      0 |    4 | `NameLen` | Big-endian byte length of the LStr name.                                                                                                                                |
|      4 |    L | `Name`    | Raw bytes of the name. Encoding is the VI's text encoding (`vi.textEncoding` in pylabview); the codec keeps the bytes opaque so callers can decode at the right moment. |
|    4+L |    4 | `VarLen`  | Big-endian byte length of the LVVariant payload.                                                                                                                        |
|    8+L |    V | `Variant` | Opaque variant bytes. Length is exactly `VarLen`.                                                                                                                       |

## What the codec does _not_ decode

LVVariant content is a tagged datafill object that depends on the VI's
consolidated type list and per-instance datafill nodes. Pulling that
into Go is part of Phase 9 (`FPHb` heap decoder), which shares the
machinery. Until then, callers that want to read individual settings
should:

1. Read `Entries[i].Name` to identify the tag.
2. Hand `Entries[i].Variant` to a future variant decoder.

The codec is round-trip-stable regardless: re-encoding a decoded VITS
payload reproduces the original bytes.

## Validation rules

| Severity | Code                     | Condition                                                                                                                      |
| -------- | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------ |
| error    | `vits.payload.malformed` | Payload could not be parsed (truncated count, name, variant length, variant bytes, or trailing data after the declared count). |

## References

- pylabview `VITS`: `LVblock.py:7015-7120` — the count + LStr name +
  variant pattern, plus the `nirviModGen` special case (variant content
  is itself an embedded `RSRC` file).
- pylabview `LVdatafill.newDataFillObject` — the variant decoder; out
  of scope for Phase 6.3.

## Open questions

- LabVIEW versions earlier than 6.5.0.2 inserted four zero bytes between
  the LStr name and the variant payload (pylabview line 7050). The codec
  rejects such payloads as malformed; if older fixtures appear in the
  corpus, the decoder will need a context-version switch.
- LabVIEW versions earlier than 6.1.0.4 encoded the count word in
  little-endian (pylabview's "endianness was wrong in some versions"
  workaround). Out of scope for now.
- Whether decoding LVVariant content unlocks any of the standard tag
  semantics (e.g. `NI.LV.All.SourceOnly` boolean). Re-evaluate in
  Phase 9.
