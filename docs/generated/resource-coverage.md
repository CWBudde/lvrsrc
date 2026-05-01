# Resource Coverage

Typed coverage: 27/27 resource types (100.0%) across 40 corpus fixtures.

## Corpus Breadth

- File kinds: ctl=11, vi=29
- File extensions: .ctl=11, .vi=29
- RSRC format versions: 3=40
- LabVIEW versions: 25.1.1=3, 25.1.2=10, 25.3.2=27
- Platforms: unknown=40
- Text encodings: unknown=40
- Password protection: empty-password=29, no-bdpw=11
- LVSR locked flag: false=40
- Separate compiled code: true=40

## Resource Table

| FourCC | Corpus fixtures | Sections | Bytes | Typed decode | Typed encode | Typed validate | Byte disposition  | Safety | Package                   | Read versions | Write versions |
| ------ | --------------: | -------: | ----: | :----------: | :----------: | :------------: | ----------------- | ------ | ------------------------- | ------------- | -------------- |
| `BDEx` |              40 |       40 |   188 |     yes      |     yes      |      yes       | opaque-preserving | Tier 1 | `internal/codecs/bdex`    | all           | all            |
| `BDHb` |              40 |       40 | 27075 |     yes      |     yes      |      yes       | partial           | Tier 1 | `internal/codecs/bdhb`    | all           | all            |
| `BDPW` |              29 |       29 |  1392 |     yes      |     yes      |      yes       | structural        | Tier 1 | `internal/codecs/bdpw`    | all           | all            |
| `BDSE` |              40 |       40 |   160 |     yes      |     yes      |      yes       | opaque-preserving | Tier 1 | `internal/codecs/bdse`    | all           | all            |
| `CONP` |              40 |       40 |    80 |     yes      |     yes      |      yes       | partial           | Tier 2 | `internal/codecs/conpane` | all           | all            |
| `CPC2` |              40 |       40 |    80 |     yes      |     yes      |      yes       | partial           | Tier 2 | `internal/codecs/conpane` | all           | all            |
| `DTHP` |              40 |       40 |   158 |     yes      |     yes      |      yes       | partial           | Tier 1 | `internal/codecs/dthp`    | all           | all            |
| `FPEx` |              40 |       40 |   196 |     yes      |     yes      |      yes       | opaque-preserving | Tier 1 | `internal/codecs/fpex`    | all           | all            |
| `FPHb` |              40 |       40 | 43035 |     yes      |     yes      |      yes       | partial           | Tier 1 | `internal/codecs/fphb`    | all           | all            |
| `FPSE` |              40 |       40 |   160 |     yes      |     yes      |      yes       | opaque-preserving | Tier 1 | `internal/codecs/fpse`    | all           | all            |
| `FTAB` |              40 |       40 |  4139 |     yes      |     yes      |      yes       | partial           | Tier 1 | `internal/codecs/ftab`    | all           | all            |
| `HIST` |              40 |       40 |  1600 |     yes      |     yes      |      yes       | structural        | Tier 1 | `internal/codecs/hist`    | all           | all            |
| `ICON` |              40 |       40 |  5120 |     yes      |     yes      |      yes       | full-observed     | Tier 2 | `internal/codecs/icon`    | all           | all            |
| `LIBN` |              13 |       13 |   263 |     yes      |     yes      |      yes       | partial           | Tier 1 | `internal/codecs/libn`    | all           | all            |
| `LIbd` |              40 |       40 |  1466 |     yes      |     yes      |      yes       | partial           | Tier 1 | `internal/codecs/libd`    | all           | all            |
| `LIfp` |              40 |       40 |  1631 |     yes      |     yes      |      yes       | partial           | Tier 1 | `internal/codecs/lifp`    | all           | all            |
| `LIvi` |              40 |       40 |  1604 |     yes      |     yes      |      yes       | partial           | Tier 1 | `internal/codecs/livi`    | all           | all            |
| `LVSR` |              40 |       40 |  6400 |     yes      |     yes      |      yes       | partial           | Tier 1 | `internal/codecs/lvsr`    | all           | all            |
| `MUID` |              40 |       40 |   160 |     yes      |     yes      |      yes       | partial           | Tier 1 | `internal/codecs/muid`    | all           | all            |
| `RTSG` |              40 |       40 |   640 |     yes      |     yes      |      yes       | structural        | Tier 1 | `internal/codecs/rtsg`    | all           | all            |
| `STRG` |               4 |        4 |   577 |     yes      |     yes      |      yes       | full-observed     | Tier 2 | `internal/codecs/strg`    | all           | all            |
| `VCTP` |              40 |       40 |  7640 |     yes      |     yes      |      yes       | partial           | Tier 1 | `internal/codecs/vctp`    | all           | all            |
| `VITS` |              40 |       40 | 63362 |     yes      |     yes      |      yes       | partial           | Tier 1 | `internal/codecs/vits`    | all           | all            |
| `VPDP` |              40 |       40 |   160 |     yes      |     yes      |      yes       | opaque-preserving | Tier 1 | `internal/codecs/vpdp`    | all           | all            |
| `icl4` |               5 |        5 |  2560 |     yes      |     yes      |      yes       | full-observed     | Tier 2 | `internal/codecs/icon`    | all           | all            |
| `icl8` |              40 |       40 | 40960 |     yes      |     yes      |      yes       | full-observed     | Tier 2 | `internal/codecs/icon`    | all           | all            |
| `vers` |              40 |       84 |  1132 |     yes      |     yes      |      yes       | partial           | Tier 2 | `internal/codecs/vers`    | all           | all            |

## Byte Disposition

Status values are semantic byte-coverage claims, not codec availability. `structural` means the stable envelope is decoded but important inner bytes remain raw; `partial` means selected fields have semantic projections; `opaque-preserving` means payload bytes are intentionally retained without field meanings; `undocumented` is a failing coverage gap.

### `BDEx`

- Status: opaque-preserving
- Semantic: small block-diagram auxiliary envelope size and round-trip invariants
- Opaque: entry meanings and correlation with BDHb remain unmapped
- Next: correlate non-zero samples with block-diagram heap changes

### `BDHb`

- Status: partial
- Semantic: heap envelope header; tag-stream node structure; tag enum names; OF__bounds; OF__termBounds; OF__termHotPoint; selected compressed-wire projections; rectangle-like heap fields; point/size-pair heap fields; common scalar heap fields; common color heap fields; structural heap container fields
- Compressed/checksum: zlib-compressed heap stream preserved byte-for-byte when possible
- Opaque: per-class primitive fields; structure metadata; multi-elbow/manual/comb wire records; point/size coordinate origin and UI role semantics; container child ordering/member role semantics; scalar bit/enum roles; color prefix/system-color semantics; unknown tags surfaced as Tag(N)
- Next: decode per-class BD fields and finish compressed-wire topology

### `BDPW`

- Status: structural
- Semantic: fixed-size block-diagram password/protection payload shape
- Opaque: hash/salt and lockout field meanings are not mutation-safe
- Next: separate protection flags from hash material with controlled password fixtures

### `BDSE`

- Status: opaque-preserving
- Semantic: small block-diagram settings payload shape and round-trip invariants
- Opaque: bit and counter meanings remain unmapped
- Next: vary diagram settings one at a time

### `CONP`

- Status: partial
- Semantic: connector-pane pointer/selector; links to VCTP top types where present
- Opaque: version-specific selector semantics beyond observed CPC2 forms
- Next: expand connector-pane fixture variants and version gates

### `CPC2`

- Status: partial
- Semantic: connector-pane count/variant value
- Opaque: full pane-pattern catalog and version-specific meanings
- Next: map every LabVIEW connector pattern against terminal type refs

### `DTHP`

- Status: partial
- Semantic: default data heap index shift used to resolve heap TypeIDs into VCTP descriptors
- Opaque: broader version behavior when DTHP is absent or multi-section
- Next: cross-check older fixtures and every heap data-fill site

### `FPEx`

- Status: opaque-preserving
- Semantic: small front-panel auxiliary envelope size and round-trip invariants
- Opaque: entry meanings and correlation with FPHb remain unmapped
- Next: correlate non-zero samples with front-panel heap changes

### `FPHb`

- Status: partial
- Semantic: heap envelope header; tag-stream node structure; tag enum names; OF__bounds; selected numeric data fills; rectangle-like heap fields; point/size-pair heap fields; common scalar heap fields; common color heap fields; structural heap container fields
- Compressed/checksum: zlib-compressed heap stream preserved byte-for-byte when possible
- Opaque: per-class visual fields; label/caption/font/style records; rectangle role semantics; point/size coordinate origin and UI role semantics; container child ordering/member role semantics; scalar bit/enum roles; color prefix/system-color semantics; custom-control state; unknown tags surfaced as Tag(N)
- Next: decode per-class FP fields and promote additional geometry tags

### `FPSE`

- Status: opaque-preserving
- Semantic: small front-panel settings payload shape and round-trip invariants
- Opaque: bit and counter meanings remain unmapped
- Next: vary panel settings one at a time

### `FTAB`

- Status: partial
- Semantic: font table entry envelope and names
- Opaque: platform-specific font attributes not fully classified
- Next: add font variation fixtures across platforms

### `HIST`

- Status: structural
- Semantic: fixed array of edit-history counters
- Opaque: individual counter meanings are not confirmed
- Next: diff save/edit operations to name each slot

### `ICON`

- Status: full-observed
- Semantic: 32x32 1-bit icon pixels and palette mapping
- Next: keep older-version icon geometry as a version-gated check

### `LIBN`

- Status: partial
- Semantic: library-name list envelope and Pascal-style names
- Opaque: multi-library membership behavior and text encoding edge cases
- Next: add multi-library and localized-name fixtures

### `LIbd`

- Status: partial
- Semantic: link-info header; BDHP marker; entry count; qualifiers; primary/secondary PTH path refs; typed LinkObjRef targets where ported
- Opaque: Tail bytes between path refs; unported LinkObjRef subclasses
- Next: decode Tail subrecords and expand LinkObjRef target families

### `LIfp`

- Status: partial
- Semantic: link-info header; FPHP marker; entry count; qualifiers; primary/secondary PTH path refs; typed LinkObjRef targets where ported
- Opaque: Tail bytes between path refs; unported LinkObjRef subclasses
- Next: decode Tail subrecords and expand LinkObjRef target families

### `LIvi`

- Status: partial
- Semantic: VI link-info header; file-kind marker; entry count; qualifiers; primary/secondary PTH path refs; typed LinkObjRef targets where ported
- Opaque: Tail bytes between path refs; unported LinkObjRef subclasses; future file-kind markers
- Next: decode Tail subrecords and broaden dependency fixture shapes

### `LVSR`

- Status: partial
- Semantic: version word; selected execution/debug/protection flag projections
- Opaque: unsurfaced flag words and version-specific tail fields
- Next: name every observed flag word and add version gates

### `MUID`

- Status: partial
- Semantic: maximum object UID value observed at save time
- Opaque: allocation scope and lifecycle semantics
- Next: diff object creation/deletion sequences

### `RTSG`

- Status: structural
- Semantic: fixed-size runtime signature payload
- Opaque: signature field roles and validation algorithm
- Next: vary runtime/signature-affecting settings

### `STRG`

- Status: full-observed
- Semantic: modern LabVIEW >= 4 string-list description payload
- Opaque: legacy LabVIEW < 4 count-prefixed layout is documented but untested
- Next: add legacy fixtures before claiming all-version semantic coverage

### `VCTP`

- Status: partial
- Semantic: outer size prefix; zlib descriptor pool; flat descriptor headers; flags; FullType codes; labels; top-type list
- Compressed/checksum: compressed descriptor-pool bytes preserved; semantic diffs compare inflated pool
- Opaque: type-specific Inner payloads for arrays, clusters, functions, refnums, variants, typedefs, and complex types
- Next: decode each type-specific grammar and report field-level diffs

### `VITS`

- Status: partial
- Semantic: tag entry envelope and names
- Opaque: variant content bytes and per-tag meanings
- Next: decode known VITS tag payloads with setting-specific fixtures

### `VPDP`

- Status: opaque-preserving
- Semantic: observed all-zero 4-byte payload shape
- Opaque: VI primitive dependency flag meanings
- Next: create primitive-dependency fixtures that produce non-zero payloads

### `icl4`

- Status: full-observed
- Semantic: 32x32 4-bit icon pixels and LabVIEW palette mapping
- Next: verify whether any version embeds alternate palettes

### `icl8`

- Status: full-observed
- Semantic: 32x32 8-bit icon pixels and LabVIEW palette mapping
- Next: verify palette index 188 and older-version palette behavior

### `vers`

- Status: partial
- Semantic: LabVIEW major/minor/patch/stage version stamp and text
- Opaque: exact meaning of multiple version stamp roles in one file
- Next: map version resource IDs to producer/save/load roles

