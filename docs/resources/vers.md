# `vers` — Version Stamp Resource

**FourCC:** `vers`
**Safety tier:** 2 (safe metadata edit)
**Status:** decode + encode + validate, grounded in corpus evidence across 20 `.vi`/`.ctl` files (59 `vers` sections total).

The `vers` resource records a LabVIEW version stamp — either the version that
last saved the file or a historical save-version marker. Files commonly carry
multiple `vers` sections (observed IDs 4, 7, 8, 9, 10) that appear to
represent different save-version lineages.

## Wire layout (big-endian)

| Offset | Size | Field        | Notes                                                         |
| -----: | ---: | ------------ | ------------------------------------------------------------- |
|      0 |    1 | `Major`      | BCD-packed major version. `0x25` = LabVIEW 25.                |
|      1 |    1 | `MinorPatch` | Upper nibble = minor, lower nibble = patch. `0x12` = `.1.2`.  |
|      2 |    1 | `Stage`      | Release stage. All observed samples: `0x80` (release/final).  |
|      3 |    1 | `Build`      | Build counter. In corpus, matches `patch` for all 59 samples. |
|      4 |    2 | `Reserved`   | Always `0x0000` in corpus.                                    |
|      6 |    1 | `TextLen`    | Pascal-string length (`N`).                                   |
|      7 |    N | `Text`       | ASCII version label (e.g. `"25.1.2"` or `"25.0"`).            |
|  7 + N |    1 | `Trailer`    | Always `0x00` in corpus — single trailing NUL byte.           |

**Total size:** `8 + N` bytes. Corpus `N ∈ {4, 6}` → payload sizes `12` and
`14` bytes.

## Decoded example

Raw bytes (hex): `25 12 80 02 00 00 06 32 35 2e 31 2e 32 00`

```text
Major      = 0x25  → 25 (BCD)
MinorPatch = 0x12  → minor=1, patch=2
Stage      = 0x80  → release
Build      = 0x02
Reserved   = 0x0000
TextLen    = 6
Text       = "25.1.2"
Trailer    = 0x00
```

## Validation rules applied by `internal/codecs/vers`

| Severity | Code                         | Condition                                                                                              |
| -------- | ---------------------------- | ------------------------------------------------------------------------------------------------------ |
| error    | `vers.payload.short`         | Payload is fewer than 8 bytes (cannot contain fixed header + empty text + trailer).                    |
| error    | `vers.text.overruns_payload` | `TextLen` would read past the end of the payload.                                                      |
| error    | `vers.trailer.missing`       | No trailing NUL byte at the expected offset.                                                           |
| warning  | `vers.reserved.nonzero`      | Bytes 4–5 are non-zero (corpus always 0; an unexpected value may indicate a version we have not seen). |
| warning  | `vers.stage.unknown`         | `Stage` is not `0x80` (observed release marker).                                                       |
| warning  | `vers.text.inconsistent`     | Decoded `major.minor[.patch]` does not match the ASCII `Text` field.                                   |
| warning  | `vers.text.nonascii`         | `Text` contains non-printable or non-ASCII bytes.                                                      |

## Round-trip guarantee

The codec preserves every observed byte on round-trip, including the trailing
NUL. Re-encoding a decoded value reproduces the original payload byte-for-byte
for all 59 corpus samples (verified by test).

## Open questions / confidence

- **BCD interpretation of `Major`.** Corpus only carries `0x25` (LabVIEW 25).
  Interpretation as BCD (`2×10+5 = 25`) matches the ASCII text. Other major
  versions are not yet in the corpus; the encoding should be revisited when
  older or newer samples arrive.
- **`Stage` semantics.** Only `0x80` (release) has been observed. Other
  LabVIEW tools historically use `0x40` (beta), `0x60` (release candidate),
  etc., but we cannot confirm mapping from corpus alone.
- **`Build` field meaning.** In every corpus sample `Build == patch`. It is
  possible this is coincidental and the field is actually a build counter
  independent of patch.
- **Multiple `vers` sections per file.** Sections with IDs 4, 7, 8, 9, 10
  appear together. The exact semantic of each ID is not yet documented;
  callers should treat them all as version stamps.
