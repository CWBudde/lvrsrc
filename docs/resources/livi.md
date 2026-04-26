# `LIvi` — VI / Control Link References

**FourCC:** `LIvi`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate + per-entry typed access. Round-trip
verified across 21 corpus sections (10 with marker `LVIN` from `.vi`
fixtures, 11 with marker `LVCC` from `.ctl` fixtures); Phase 7.3 lifted
the envelope-only Phase 7.2 codec to a typed entry list.

`LIvi` records dependencies between this VI/CTL and other VIs, classes,
and project libraries. Sister to `LIfp` (front-panel imports) and `LIbd`
(block-diagram imports).

## Wire layout

| Offset | Size | Field        | Notes                                                                         |
| -----: | ---: | ------------ | ----------------------------------------------------------------------------- |
|      0 |    2 | `Version`    | Big-endian uint16. Corpus is uniformly `0x0001`. (Pylabview reads this as the `nextLinkInfo=1` list-start marker.) |
|      2 |    4 | `Marker`     | File-kind FourCC (`LVIN` for `.vi`, `LVCC` for `.ctl`, etc.).                 |
|      6 |    4 | `EntryCount` | Big-endian uint32 number of dependency entries.                               |
|     10 |  ... | `Entries`    | `EntryCount` typed `Entry` records (layout below).                            |
|    N-2 |    2 | `Footer`     | Big-endian uint16 trailing value. Corpus is uniformly `0x0003` — pylabview's `nextLinkInfo=3` list-end marker. |

The `Marker` field mirrors the file's content type. `Validate` warns on
an unknown marker but Decode accepts any 4-byte value so future kinds
round-trip cleanly.

## Per-entry layout

Each `Entry` follows the same shape as `LIfp` / `LIbd`:

| Offset | Size | Field            | Notes                                                                                      |
| -----: | ---: | ---------------- | ------------------------------------------------------------------------------------------ |
|      0 |    2 | `Kind`           | Big-endian uint16. Always `0x0002` — pylabview's `nextLinkInfo=2` continuation marker.     |
|      2 |    4 | `LinkType`       | 4-byte LinkObjRef wire ident (`VILB`, `VICC`, `TDCC`, …); see `internal/codecs/linkobj`.   |
|      6 |  0–3 | `prefixPad`      | Zero-fill alignment bytes pylabview's `parseBasicLinkSaveInfo` inserts so the qualifier count starts on a 4-byte section boundary. Present whenever the previous entry's encoded size was not a multiple of four. |
|    ... |    4 | `QualifierCount` | Big-endian uint32 number of Pascal-string qualifiers.                                      |
|    ... |  ... | `Qualifiers`     | `QualifierCount` consecutive `[u8 length][bytes]` Pascal strings.                          |
|    ... |  0–3 | `qualifierPad`   | Zero-fill alignment bytes before the primary path (pylabview's 2-byte align, plus slack).  |
|    ... |  ... | `PrimaryPath`    | Embedded `PTH0` reference (see `docs/resources/pth0.md`). Decoded lazily through `internal/codecs/pthx`. |
|    ... |  ... | `Tail`           | Post-path bytes — the `LinkObj`-specific payload. Decoded on demand through `Entry.Target() (linkobj.LinkTarget, error)`. |
|    ... |  ... | `SecondaryPath`  | Optional `PTH0` reference (the `viLSPathRef` from `HeapToVILinkSaveInfo`). Present only for some `LinkType`s.            |

`Entry.Target()` returns one of the typed `linkobj.LinkTarget`
implementations (`TypeDefToCCLink`, `VIToLib`, …) or `OpaqueTarget` when
the LinkObjRef subclass hasn't been ported. The opaque fallback
preserves `Tail` + `SecondaryPath` byte-for-byte so unknown subclasses
still round-trip.

## Validation rules

| Severity | Code                  | Condition                                                       |
| -------- | --------------------- | --------------------------------------------------------------- |
| error    | `livi.payload.short`  | Payload is shorter than the 12-byte minimum (header + footer).  |
| error    | `livi.decode.invalid` | Per-entry decode failed (boundary heuristic could not match).   |
| warning  | `livi.marker.unknown` | Marker is not in the corpus-observed set (LVIN/LVCC/LVIT/LLBV). |

## References

- pylabview `LIvi`: `LVblock.py:2426-2434` — confirms ident `LVIN`.
- pylabview `LinkObjRefs.parseRSRCSectionData`: `LVblock.py:2259+` —
  defines the outer envelope (the `nextLinkInfo` 1/2/3 markers we read
  as `Version` / `Kind` / `Footer`).
- pylabview `LVlinkinfo.py:4235` (`newLinkObject`) — the dispatch table
  that maps each 4-byte LinkType to a LinkObjRef subclass.
- `internal/codecs/linkobj` — typed `LinkKind` enum + per-class
  decoders.

## Open questions

- Markers `LVIT` (template) and `LLBV` (library) are accepted on the
  basis of pylabview's class hierarchy but are not present in the
  shipped corpus, so their per-entry shape is not yet validated.
- ~50 LinkObjRef subclasses still surface as `OpaqueTarget` because no
  fixture exercises them. New typed parsers can be added incrementally
  without breaking round-trip safety.
