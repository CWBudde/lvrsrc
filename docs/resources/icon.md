# Icon Resources: `ICON`, `icl4`, `icl8`

LabVIEW stores VI/control icons as fixed-size `32x32` raster resources with no
per-resource header.

## Resource Families

| FourCC | Depth | Payload size | Layout |
| ------ | ----: | -----------: | ------ |
| `ICON` | 1 bpp | 128 B        | 1024 packed bits, row-major, MSB-first within each byte |
| `icl4` | 4 bpp | 512 B        | 1024 packed nibbles, row-major, high nibble then low nibble |
| `icl8` | 8 bpp | 1024 B       | 1024 raw bytes, row-major |

All observed samples are exactly `32 * 32 = 1024` pixels. There is no leading
dimension field, palette table, or trailing checksum in the resource payload
itself.

## Decoded Model

`internal/codecs/icon` normalizes each resource to:

- `FourCC`
- `Width = 32`
- `Height = 32`
- `BitsPerPixel = 1 | 4 | 8`
- `Pixels []byte` containing one byte per pixel in row-major order

For `ICON`, decoded pixel values are `0..1`.
For `icl4`, decoded pixel values are `0..15`.
For `icl8`, decoded pixel values are `0..255`.

The codec re-packs those normalized pixels back to the original fixed-width
payload layout.

## Validation Rules

Current validator behavior is intentionally narrow and corpus-grounded:

| Severity | Code                | Condition |
| -------- | ------------------- | --------- |
| error    | `icon.payload.size` | `ICON` payload length is not exactly 128 bytes |
| error    | `icl4.payload.size` | `icl4` payload length is not exactly 512 bytes |
| error    | `icl8.payload.size` | `icl8` payload length is not exactly 1024 bytes |

No additional palette or semantic checks are implemented yet. The corpus
supports only the raw fixed-size raster interpretation so far.

## Corpus Evidence

From the current `testdata/corpus/` set:

- `ICON` appears in 21/21 fixtures
- `icl4` appears in 5/21 fixtures
- `icl8` appears in 21/21 fixtures

Every observed icon section round-trips byte-for-byte through
`internal/codecs/icon`.

## Open Questions

- Whether `icl4`/`icl8` always use LabVIEW-global palettes or can embed
  palette metadata elsewhere
- Whether older LabVIEW versions ever vary from the observed `32x32` geometry
- Whether monochrome `ICON` pixels should be interpreted as mask, image, or a
  direct black/white plane in higher-level APIs
