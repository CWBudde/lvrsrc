# `LIvi` — VI / Control Link References

**FourCC:** `LIvi`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across 21 corpus
sections (10 with marker `LVIN` from `.vi` fixtures, 11 with marker `LVCC`
from `.ctl` fixtures). The codec exposes the outer envelope; per-entry
parsing is deferred — see *Open questions* below.

`LIvi` records dependencies between this VI/CTL and other VIs, classes,
and project libraries. Sister to `LIfp` (front-panel imports) and `LIbd`
(block-diagram imports).

## Wire layout

| Offset | Size | Field        | Notes                                                                |
| -----: | ---: | ------------ | -------------------------------------------------------------------- |
|      0 |    2 | `Version`    | Big-endian uint16. Corpus is uniformly `0x0001`.                     |
|      2 |    4 | `Marker`     | File-kind FourCC (`LVIN` for `.vi`, `LVCC` for `.ctl`, etc.).        |
|      6 |    4 | `EntryCount` | Big-endian uint32 number of dependency entries.                      |
|     10 |  ... | `Body`       | `EntryCount` entries in their on-disk form. Layout described below.  |
|   N-2  |    2 | `Footer`     | Big-endian uint16 trailing value (corpus: always `0x0003`).          |

The `Marker` field mirrors the file's content type. `Validate` warns on
an unknown marker but Decode accepts any 4-byte value so future kinds
round-trip cleanly.

## Per-entry layout (informational)

Each entry begins with `[u16 kind][4-byte LinkType][u32 qualifier_count]`
followed by the qualifier strings, an embedded `PTH0` reference, and
optional tail/secondary-path bytes — analogous to `LIfp` / `LIbd` but
with subtle structural differences this codec does not yet decode.

The Phase 7.2 scope is the **envelope only**: `Body` is preserved as
opaque bytes to guarantee byte-for-byte round-trip. Per-entry typed
access lands in Phase 7.3 (LIfp/LIbd refit) and Phase 9 (LinkObjRef
family port).

## Validation rules

| Severity | Code                  | Condition                                                       |
| -------- | --------------------- | --------------------------------------------------------------- |
| error    | `livi.payload.short`  | Payload is shorter than the 12-byte minimum (header + footer).  |
| warning  | `livi.marker.unknown` | Marker is not in the corpus-observed set (LVIN/LVCC/LVIT/LLBV). |

## References

- pylabview `LIvi`: `LVblock.py:2426-2434` — confirms ident `LVIN`.
- pylabview `LinkObjRefs` base class: `LVblock.py:2248+`.
- pylabview `LVlinkinfo.py:1428-2524` — the LinkObjRef subclass family
  that decodes per-entry semantics. Port deferred to Phase 7.3 / 9.

## Open questions

- The exact per-entry layout for `LVCC` (control) markers vs `LVIN`
  (VI). Hex inspection shows both variants reach the same `PTH0`
  payload but the bytes between `qualifier_count` and `PTH0` differ in
  ways the libd-style heuristic cannot disambiguate without porting
  the LinkObjRef subclass family.
- Whether the trailing `Footer` carries semantic meaning. Corpus is
  uniformly `0x0003`.
- Markers `LVIT` (template) and `LLBV` (library) are accepted on the
  basis of pylabview's class hierarchy but are not present in the
  shipped corpus, so their per-entry shape is not yet validated.
