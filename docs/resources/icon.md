# Icon Resources: `ICON`, `icl4`, `icl8`

LabVIEW stores VI/control icons as fixed-size `32x32` raster resources with no
per-resource header.

## Resource Families

| FourCC | Depth | Payload size | Layout                                                      |
| ------ | ----: | -----------: | ----------------------------------------------------------- |
| `ICON` | 1 bpp |        128 B | 1024 packed bits, row-major, MSB-first within each byte     |
| `icl4` | 4 bpp |        512 B | 1024 packed nibbles, row-major, high nibble then low nibble |
| `icl8` | 8 bpp |       1024 B | 1024 raw bytes, row-major                                   |

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
- `Palette []uint32` in packed ARGB layout (bits 24..31 alpha, 16..23 red,
  8..15 green, 0..7 blue), with alpha always `0xFF`

For `ICON`, decoded pixel values are `0..1`.
For `icl4`, decoded pixel values are `0..15`.
For `icl8`, decoded pixel values are `0..255`.

The codec re-packs those normalized pixels back to the original fixed-width
payload layout. `Palette` is a read-side convenience; it is ignored on
encode because its contents are determined entirely by `BitsPerPixel`.

### Palette sources

The palettes are ported verbatim from `pylabview`:

- `ICON` → `Palette2` (index 0 = white, 1 = black), from
  `references/pylabview/pylabview/LVmisc.py:93-95`
  (`LABVIEW_COLOR_PALETTE_2`)
- `icl4` → `Palette16`, from
  `references/pylabview/pylabview/LVmisc.py:88-91`
  (`LABVIEW_COLOR_PALETTE_16`)
- `icl8` → `Palette256`, from
  `references/pylabview/pylabview/LVmisc.py:52-85`
  (`LABVIEW_COLOR_PALETTE_256`)

The pylabview `LABVIEW_COLOR_PALETTE_256` contains one entry that looks like
a typo in the upstream source: index 188 is `0x3003FF` where the
surrounding blue-ramp pattern would suggest `0x0033FF`. The Go port
reproduces this verbatim so behaviour matches pylabview; corpus evidence
for a pixel actually using index 188 is not yet available to confirm or
refute.

### RGBA expansion

`(Value).RGBA() []byte` returns a fresh `Width*Height*4` byte slice in
row-major R,G,B,A order, suitable for handing to a browser canvas, Go's
`image` package, or any other renderer. Out-of-range `Pixels` entries
(should not occur from `Decode` but are allowed in hand-crafted `Value`s)
map to opaque black, so the helper is panic-free.

## Validation Rules

Current validator behavior is intentionally narrow and corpus-grounded:

| Severity | Code                | Condition                                       |
| -------- | ------------------- | ----------------------------------------------- |
| error    | `icon.payload.size` | `ICON` payload length is not exactly 128 bytes  |
| error    | `icl4.payload.size` | `icl4` payload length is not exactly 512 bytes  |
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
- Whether `LABVIEW_COLOR_PALETTE_256[188]` is really `0x3003FF` or a
  long-standing pylabview typo for `0x0033FF` (see _Palette sources_ above)
