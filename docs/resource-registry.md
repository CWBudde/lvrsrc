# Resource Registry

This registry tracks known resource types (FourCC), decode status, safety
tier, and the Go package that implements the codec.

## Observed in corpus

These FourCCs appear in the 21-file `testdata/corpus/` set (as of 2026-04-24):

| FourCC                                     | Count        | Typical size | Notes                                                                 |
| ------------------------------------------ | ------------ | ------------ | --------------------------------------------------------------------- |
| `LVSR`                                     | ≥1 per file  | 160 B        | LabVIEW Save Record. Carries the VI's display name in `Section.Name`. |
| `vers`                                     | 1–5 per file | 12–14 B      | Version stamp — see [vers.md](resources/vers.md).                     |
| `LIBN`                                     | 1 per file   | 16–27 B      | Library-name list (Pascal-string list of `.lvlib` membership).        |
| `LIvi`                                     | 1 per file   | 51–176 B     | LabVIEW Info: VI dependencies (library imports, `PTH0` paths).        |
| `LIfp`                                     | 1 per file   | 12–201 B     | LabVIEW Info: Front Panel imports — see [lifp.md](resources/lifp.md). |
| `LIbd`                                     | 1 per file   | 12–201 B     | LabVIEW Info: Block Diagram imports — see [libd.md](resources/libd.md). |
| `BDPW`                                     | 1 per file   | 48 B         | Block-diagram password hash (lockout info).                           |
| `ICON`                                     | 1 per file   | 128 B        | 1-bit VI icon — see [icon.md](resources/icon.md).                     |
| `icl4`                                     | 0–1 per file | 512 B        | 4-bit color icon — see [icon.md](resources/icon.md).                  |
| `icl8`                                     | 1 per file   | 1024 B       | 8-bit color icon — see [icon.md](resources/icon.md).                  |
| `FPHb`                                     | 1 per file   | variable     | Front panel heap.                                                     |
| `BDHb`                                     | 1 per file   | variable     | Block diagram heap.                                                   |
| `VCTP`                                     | 1 per file   | variable     | Type descriptor pool.                                                 |
| `HIST`                                     | 1 per file   | 40 B         | Edit history counters.                                                |
| `VITS`                                     | 1 per file   | variable     | VI settings / misc.                                                   |
| `CONP`                                     | 1 per file   | 2 B          | Connector pane selector/pointer — see [conpane.md](resources/conpane.md). |
| `CPC2`                                     | 1 per file   | 2 B          | Connector pane count/variant — see [conpane.md](resources/conpane.md). |
| `RTSG`                                     | 1 per file   | 16 B         | Runtime signature.                                                    |
| `FTAB`                                     | 1 per file   | ~100 B       | Font table.                                                           |
| `MUID`                                     | 1 per file   | 4 B          | Module unique ID.                                                     |
| `DTHP`                                     | 1 per file   | 4 B          | Default data heap pointer.                                            |
| `FPEx` / `BDEx` / `FPSE` / `BDSE` / `VPDP` | 1 per file   | 4–8 B        | Small heap auxiliary blocks.                                          |

## Compatibility table format

The *Codec status* table below is the human-readable compatibility table for
every shipped typed codec. Columns:

- **FourCC** — 4-character resource type (see *Observed in corpus* above
  for where each one appears).
- **Decode / Encode / Validate** — ✅ means the codec implements
  `codecs.ResourceCodec.Decode` / `Encode` / `Validate`; `—` means opaque
  (bytes preserved verbatim, no semantic interpretation).
- **Safety** — which safety tier the codec declares in its `Capability()`:
  Tier 1 = read-only, Tier 2 = safe edits via `pkg/lvmeta`, Tier 3 = raw
  patching (none shipped yet). See [safety-model.md](safety-model.md).
- **Read versions / Write versions** — file-format versions the codec
  advertises support for via `Capability().ReadVersions` /
  `WriteVersions` (`codecs.VersionRange`). `all` means
  `VersionRange{Min: 0, Max: 0}`, which `VersionRange.Contains` treats as
  unbounded (every observed LabVIEW RSRC revision). A future tiered codec
  would express a closed range inclusively as e.g. `8–10` (read `Min..Max`
  where `Max != 0`).
- **Package** — Go package path of the implementation.

For the machine-readable view that also counts corpus occurrences per
FourCC and is regenerated by CI, see
[generated/resource-coverage.md](generated/resource-coverage.md). That
artifact is the source of truth for the `internal/coverage` package and
the coverage badge.

## Codec status

| FourCC     | Decode | Encode | Validate | Safety | Read versions | Write versions | Package                      |
| ---------- | :----: | :----: | :------: | ------ | ------------- | -------------- | ---------------------------- |
| `CONP`     |   ✅   |   ✅   |    ✅    | Tier 2 | all           | all            | `internal/codecs/conpane`    |
| `CPC2`     |   ✅   |   ✅   |    ✅    | Tier 2 | all           | all            | `internal/codecs/conpane`    |
| `ICON`     |   ✅   |   ✅   |    ✅    | Tier 2 | all           | all            | `internal/codecs/icon`       |
| `LIbd`     |   ✅   |   ✅   |    ✅    | Tier 1 | all           | all            | `internal/codecs/libd`       |
| `LIfp`     |   ✅   |   ✅   |    ✅    | Tier 1 | all           | all            | `internal/codecs/lifp`       |
| `icl4`     |   ✅   |   ✅   |    ✅    | Tier 2 | all           | all            | `internal/codecs/icon`       |
| `icl8`     |   ✅   |   ✅   |    ✅    | Tier 2 | all           | all            | `internal/codecs/icon`       |
| `vers`     |   ✅   |   ✅   |    ✅    | Tier 2 | all           | all            | `internal/codecs/vers`       |
| `STRG`     |   ✅   |   ✅   |    ✅    | Tier 2 | all           | all            | `internal/codecs/strg`       |
| all others |   —    |   —    |    —     | Opaque | —             | —              | `internal/codecs` (fallback) |

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
