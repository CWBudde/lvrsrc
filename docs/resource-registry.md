# Resource Registry

This registry tracks known resource types (FourCC), decode status, safety
tier, and the Go package that implements the codec.

## Observed in corpus

These FourCCs appear in the 20-file `testdata/corpus/` set (as of 2026-04-22):

| FourCC                                     | Count        | Typical size | Notes                                                                 |
| ------------------------------------------ | ------------ | ------------ | --------------------------------------------------------------------- |
| `LVSR`                                     | ≥1 per file  | 160 B        | LabVIEW Save Record. Carries the VI's display name in `Section.Name`. |
| `vers`                                     | 1–5 per file | 12–14 B      | Version stamp — see [vers.md](resources/vers.md).                     |
| `LIBN`                                     | 1 per file   | 16–27 B      | Library-name list (Pascal-string list of `.lvlib` membership).        |
| `LIvi`                                     | 1 per file   | 51–176 B     | LabVIEW Info: VI dependencies (library imports, `PTH0` paths).        |
| `LIfp`                                     | 1 per file   | 12–201 B     | LabVIEW Info: Front Panel imports.                                    |
| `LIbd`                                     | 1 per file   | 12–201 B     | LabVIEW Info: Block Diagram imports.                                  |
| `BDPW`                                     | 1 per file   | 48 B         | Block-diagram password hash (lockout info).                           |
| `ICON`                                     | 1 per file   | 128 B        | 1-bit VI icon.                                                        |
| `icl4`                                     | 0–1 per file | 512 B        | 4-bit color icon.                                                     |
| `icl8`                                     | 1 per file   | 1024 B       | 8-bit color icon.                                                     |
| `FPHb`                                     | 1 per file   | variable     | Front panel heap.                                                     |
| `BDHb`                                     | 1 per file   | variable     | Block diagram heap.                                                   |
| `VCTP`                                     | 1 per file   | variable     | Type descriptor pool.                                                 |
| `HIST`                                     | 1 per file   | 40 B         | Edit history counters.                                                |
| `VITS`                                     | 1 per file   | variable     | VI settings / misc.                                                   |
| `CONP`                                     | 1 per file   | 2 B          | Connector pane pointer.                                               |
| `CPC2`                                     | 1 per file   | 2 B          | Connector pane counter.                                               |
| `RTSG`                                     | 1 per file   | 16 B         | Runtime signature.                                                    |
| `FTAB`                                     | 1 per file   | ~100 B       | Font table.                                                           |
| `MUID`                                     | 1 per file   | 4 B          | Module unique ID.                                                     |
| `DTHP`                                     | 1 per file   | 4 B          | Default data heap pointer.                                            |
| `FPEx` / `BDEx` / `FPSE` / `BDSE` / `VPDP` | 1 per file   | 4–8 B        | Small heap auxiliary blocks.                                          |

## Codec status

| FourCC     | Decode | Encode | Validate | Safety | Package                      |
| ---------- | :----: | :----: | :------: | ------ | ---------------------------- |
| `vers`     |   ✅   |   ✅   |    ✅    | Tier 2 | `internal/codecs/vers`       |
| `STRG`     |   ✅   |   ✅   |    ✅    | Tier 2 | `internal/codecs/strg`       |
| all others |   —    |   —    |    —     | Opaque | `internal/codecs` (fallback) |

The opaque fallback preserves payload bytes exactly on round-trip; it is used
by `Registry.Lookup` for any FourCC without a registered codec.

## Phase 4.3 disposition

| Intent               | Disposition                                                                                                                                                                                                                                                            |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| VI name codec        | **N/A** — the VI filename is already exposed on `Section.Name` of the `LVSR` block (e.g. `get-vi-description.vi` → `"get vi description.vi"`). No payload codec is needed to read it; writing is a container-level name-table edit, handled in Phase 4.4 `pkg/lvmeta`. |
| VI description codec | **Shipped** as `STRG` — see [strg.md](resources/strg.md). Grounded in `pylabview`'s `StringListBlock`/`STRG` handling and verified against 4 corpus files that carry non-empty descriptions.                                                                           |
| Version stamp codec  | **Shipped** as `vers` — see [vers.md](resources/vers.md). Verified against 65 corpus `vers` sections.                                                                                                                                                                  |

## Method

1. Extract the resource-type set from the corpus with `lvrsrc list-resources`.
2. Dump candidate payloads via `lvrsrc dump --json` and pattern-match against
   published references (pylabview / pylavi) and hypothesis-driven bitfield
   probes.
3. Verify any hypothesis by decoding every corpus sample and round-tripping
   byte-for-byte. A codec is only marked ✅ once this test passes.
4. Record findings and open questions per-resource under `docs/resources/`.
