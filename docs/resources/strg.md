# `STRG` — String Description Resource

**FourCC:** `STRG`
**Safety tier:** 2 (safe metadata edit)
**Status:** decode + encode + validate for LabVIEW ≥ 4.0 modern layout, grounded in corpus evidence across 4 `.vi`/`.ctl` files carrying user-entered descriptions. Legacy LabVIEW < 4.0 layout is documented but not yet implemented (no corpus samples).
**Reference:** `references/pylabview/pylabview/LVblock.py` → `class STRG(StringListBlock)` and `StringListBlock.parseRSRCStringList`.

The `STRG` resource carries a string description — the free-form text a user
enters in the _VI Properties → Documentation → VI Description_ field. A VI
with no description simply does not emit a `STRG` block; 16 of 20 corpus
files fall into that category. This codec handles the 4 that do.

Per the `pylabview` reference, `STRG` is a `StringListBlock` whose storage
mode varies with LabVIEW version:

| LabVIEW range | count_len | size_len | padding_len | Strings stored |
| ------------- | --------: | -------: | ----------: | -------------- |
| ≥ 4.0         |         0 |        4 |           1 | exactly 1      |
| < 4.0 (LV2.5) |         4 |        1 |           1 | count-prefixed |

Our corpus is entirely LabVIEW 25, so only the modern single-string layout is
implemented. The legacy layout is described here for completeness and will
be added when an older corpus sample arrives.

## Wire layout (modern, LabVIEW ≥ 4.0, big-endian)

| Offset | Size | Field  | Notes                                                          |
| -----: | ---: | ------ | -------------------------------------------------------------- |
|      0 |    4 | `Size` | uint32 BE — length of `Text` in bytes.                         |
|      4 | Size | `Text` | Raw string content. May contain `CR`/`LF`/`CRLF` line endings. |

**Total size:** `4 + Size` bytes. No count prefix, no padding, no trailer.

## Decoded example

Raw bytes (from `format-string.vi`, first 40 bytes of 118-byte payload):

```text
00 00 00 72  r e p l a c e s _ a l l _ s p a c e s ...
└─size=114┘  └────── text (114 bytes) ──────────────...
```

Decoded `Text`: `"replaces all spaces with \"_\" and turns all to lower case ..."`

## Validation rules applied by `internal/codecs/strg`

| Severity | Code                         | Condition                                                                                 |
| -------- | ---------------------------- | ----------------------------------------------------------------------------------------- |
| error    | `strg.payload.short`         | Payload is fewer than 4 bytes (cannot contain the `Size` field).                          |
| error    | `strg.size.overruns_payload` | `Size` is greater than `len(payload) - 4` (would read past end).                          |
| warning  | `strg.size.trailing_bytes`   | Payload contains bytes after `Size + Text` (unexpected — corpus always has an exact fit). |
| warning  | `strg.text.control_chars`    | `Text` contains control bytes other than `CR` (0x0D) and `LF` (0x0A).                     |

## Round-trip guarantee

The codec preserves every observed byte on round-trip. Re-encoding a decoded
`Value` reproduces the original payload byte-for-byte for all 4 corpus STRG
samples (verified by `TestCorpusRoundTrip`).

## Open questions

- **Text encoding.** `pylabview` decodes `Text` via the VI's declared
  `textEncoding` (typically MBCS on Windows, UTF-8 on modern files). This
  codec returns raw bytes as a Go `string` and leaves encoding interpretation
  to callers. Add an encoding argument to `Decode` when a concrete
  non-UTF-8 corpus sample forces the issue.
- **Legacy layout.** LabVIEW ≤ 2.x files use count-prefixed storage with
  multiple strings per block (`count_len=4, size_len=1, padding_len=1`).
  Not implemented — will need a pre-4.0 corpus sample to verify.
- **Line endings.** Pylabview normalizes `CR`/`LF`/`CRLF` into chunks during
  export; we preserve the raw bytes, including mixed line endings if
  present.
