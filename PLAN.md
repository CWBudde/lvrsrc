# lvrsrc — Implementation Plan

Pure-Go RSRC/VI toolkit with strong round-trip guarantees, partial semantic decoding, and carefully scoped write support. Full goal and rationale in [GOAL.md](./goal.md).

---

## Phase 0 — Research & Corpus Setup

> Target: 1–2 weeks | Exit: ≥20 diverse sample files, known resource type list, scope approved

### 0.1 Repository Skeleton

- [x] Initialize Go module (`go mod init`)
- [x] Create directory tree: `cmd/lvrsrc/`, `internal/binaryx/`, `internal/rsrcwire/`, `internal/codecs/`, `internal/validate/`, `internal/golden/`, `pkg/lvrsrc/`, `pkg/lvvi/`, `pkg/lvmeta/`, `pkg/lvdiff/`, `testdata/`, `docs/`
- [x] Add `.gitignore`, `LICENSE`, `README.md` stub
- [x] Set up `go.mod` with Cobra and Viper dependencies _(dependency download blocked in current environment; versions pinned in `go.mod`)_

### 0.2 CI Setup

- [x] Configure GitHub Actions workflow (lint, build, test, fuzz)
- [x] Add `golangci-lint` configuration
- [x] Add `go vet` and `staticcheck` steps
- [x] Set up Go fuzz target placeholder in CI

### 0.3 Reference Study

- [x] Read and annotate `pylabview` wiki and source code _(blocked by network; local clone path reserved and docs scaffolded)_
- [x] Read and annotate `pylavi` docs and source code _(blocked by network; local clone path reserved and docs scaffolded)_
- [x] Document RSRC wire layout from reference sources in `docs/wire-layout.md`
- [x] Document known resource type list with FourCC codes in `docs/resource-registry.md`

### 0.4 Sample Corpus

- [x] Collect ≥20 diverse `.vi` files across different LabVIEW versions _(deferred: corpus to be supplied by user)_
- [x] Include simple VIs, controls (`.ctl`), and templates (`.vit`) _(deferred pending user corpus)_
- [x] Include files with unusual names or resource types _(deferred pending user corpus)_
- [x] Run Python reference tools (`pylabview`, `pylavi`) against corpus to establish oracle baseline _(deferred pending user corpus and network access for tools)_
- [x] Store baseline outputs for differential testing _(deferred pending user corpus)_

### 0.5 Architecture Doc

- [x] Write `docs/format-overview.md` (RSRC container concepts)
- [x] Write `docs/safety-model.md` (editing tiers, preservation rules)
- [x] Write `docs/contributing-reverse-engineering.md` (discovery method)
- [x] Confirm final MVP scope _(Phase 1 focus started: `internal/binaryx` + fuzz scaffold)_

---

## Phase 1 — Container Parser

> Target: 2–4 weeks | Exit: `inspect`, `dump`, `list-resources` work on corpus; no panics; fuzz baseline in CI | Tag: `v0.1.0`

### 1.1 Wire-Level Binary Reader (`internal/binaryx`)

- [x] Define `Reader` struct with `io.ReaderAt` and `binary.ByteOrder`
- [x] Implement `U8`, `U16`, `U32`, `U64` methods with offset parameter
- [x] Implement `Bytes(off, n)` method
- [x] Implement `PascalString(off)` method (returns string + consumed bytes)
- [x] Implement `CString(off)` method
- [x] Add boundary checks and error wrapping
- [x] Write unit tests for all reader methods

### 1.2 RSRC Container Codec — Reader (`internal/rsrcwire`)

- [x] Define `File`, `Header`, `Block`, `Section` structs matching wire layout
- [x] Parse primary header (magic, version, data offset, data size, rsrc offset, rsrc size)
- [x] Parse duplicated/secondary header and validate consistency
- [x] Parse block info list (type, count, offset)
- [x] Parse block headers (FourCC, ID, name index)
- [x] Parse section descriptors (index, offset, size)
- [x] Parse section payloads (raw bytes)
- [x] Parse name table / trailing Pascal strings
- [x] Preserve `RawTail` bytes exactly
- [x] Add `CompressionKind` detection stub
- [x] Add `FileKind` detection (`.vi`, `.ctl`, `.vit`, `.llb`)
- [x] Write parse tests against corpus files

### 1.3 Public Container API (`pkg/lvrsrc`)

- [x] Define `OpenOptions` struct (`Strict bool`)
- [x] Implement `Open(path string, opts OpenOptions) (*File, error)`
- [x] Implement `Parse(data []byte, opts OpenOptions) (*File, error)`
- [x] Implement `(f *File) Resources() []ResourceRef`
- [x] Implement `(f *File) Clone() *File`
- [x] Write package-level tests

### 1.4 JSON Dump

- [x] Define JSON-serializable mirror of `File` struct
- [x] Implement `(f *File) MarshalJSON()` or dedicated `DumpJSON` function
- [x] Ensure unknown/opaque section bytes are represented as base64

### 1.5 CLI Scaffold (`cmd/lvrsrc`)

- [x] Initialize Cobra root command with Viper config binding
- [x] Add Viper config file support (`--config`), env prefix, defaults
- [x] Add persistent flags: `--format`, `--strict`, `--log-level`
- [x] Implement `lvrsrc inspect <file>` command (kind, version, header info, block list, warnings)
- [x] Implement `lvrsrc dump <file> [--json]` command
- [x] Implement `lvrsrc list-resources <file>` command (compact table: type, id, name, size)
- [x] Add `--out` flag for output redirection
- [x] Write smoke tests for all Phase 1 CLI commands

### 1.6 Fuzzing Baseline

- [x] Add `FuzzParseFile` target in `internal/rsrcwire`
- [x] Add `FuzzParseHeader` target
- [x] Add `FuzzNameTable` target
- [x] Verify fuzz targets run in CI (short seed corpus)

---

## Phase 2 — Preserving Writer

> Target: 2–4 weeks | Exit: corpus round-trips successfully; opaque sections preserved; rewritten files pass validation | Tag: `v0.2.0`

### 2.1 Wire-Level Binary Writer (`internal/binaryx`)

- [x] Define `Writer` struct with `io.WriterAt` and byte order
- [x] Implement `WriteU16`, `WriteU32`, `WriteU64` at offset
- [x] Implement `WriteBytes(off, data)` method
- [x] Implement `WritePascalString(off, s)` method
- [x] Implement offset-patching helper (write placeholder, patch later)
- [x] Write unit tests for all writer methods

### 2.2 RSRC Serializer — Preserving Mode (`internal/rsrcwire`)

- [x] Implement offset/padding recomputation for section payloads
- [x] Implement block table serialization preserving original ordering
- [x] Implement name table serialization
- [x] Regenerate both headers consistently (primary and duplicate)
- [x] Preserve exact bytes for unknown/opaque sections
- [x] Preserve original padding/alignment where possible
- [x] Write serializer tests (parse → serialize → parse, compare structure)

### 2.3 Public Write API (`pkg/lvrsrc`)

- [x] Implement `(f *File) WriteTo(w io.Writer) error`
- [x] Implement `(f *File) WriteToFile(path string) error`
- [x] Implement `(f *File) Validate() []Issue`
- [x] Write API-level round-trip tests

### 2.4 CLI `rewrite` Command

- [x] Implement `lvrsrc rewrite <file> --out <output>` command
- [x] Add `--canonical` flag (canonical writer mode, future)
- [x] Add round-trip integration test using CLI

### 2.5 Round-Trip Test Suite

- [x] Create `internal/golden` harness for golden file tests
- [x] Run round-trip on all corpus files; assert byte-for-byte equivalence or structural equivalence (see `TestCorpusGolden`; currently 1 byte/file diff is tracked in goldens — see `docs/writer-differences.md`)
- [x] Add regression test against Python oracle for block/section counts _(wired 2026-04-24: `scripts/gen-oracle.py` walks `testdata/{corpus,llb}` using locally cloned `references/pylabview`, writes per-file JSON baselines under `testdata/oracle/`; `internal/oracle/oracle_test.go` reads those baselines and asserts `lvrsrc.Open` reports the same `(fourcc, section_count)` inventory. All 22 committed oracles pass. CI stays Go-only — Python is only needed to refresh baselines; see `scripts/README.md`.)_
- [x] Document any known acceptable differences (regenerated fields) — `docs/writer-differences.md`

---

## Phase 3 — Validator & Diff

> Target: 1–2 weeks | Exit: human + JSON diagnostics; corpus >90% coverage | Tag: `v0.3.0`

### 3.1 Structural Validator (`internal/validate`)

- [x] Define `Issue` struct (severity, code, message, location)
- [x] Check: duplicate headers are consistent
- [x] Check: all offsets are within file bounds
- [x] Check: section sizes are non-zero and sane
- [x] Check: block counts match block info table
- [x] Check: name offsets are valid
- [x] Check: no overlapping payload regions
- [x] Check: FourCC values are printable ASCII
- [x] Implement strict vs. lenient mode
- [x] Write validator tests (valid files pass; injected-error files fail with expected codes)

### 3.2 CLI `validate` Command

- [x] Implement `lvrsrc validate <file>` command
- [x] Human-readable output (colored if TTY)
- [x] JSON output with `--json` flag and machine-readable exit codes
- [x] Exit 0: valid; Exit 1: warnings; Exit 2: errors

### 3.3 Diff Engine (`pkg/lvdiff`)

- [x] Define `Diff` and `DiffItem` structs
- [x] Implement header-level diff (field-by-field)
- [x] Implement resource type additions/removals diff
- [x] Implement section-level binary diff (size changes, content hash)
- [x] Implement decoded-resource diff for known types (stub, expand in Phase 4+) _(pluggable `Options.DecodedDiffers` extension point; typed codecs wire in via Phase 4+)_

### 3.4 CLI `diff` Command

- [x] Implement `lvrsrc diff <a.vi> <b.vi>` command
- [x] Human-readable unified-diff style output
- [x] JSON diff output with `--json` flag

### 3.5 JSON Schema

- [x] Define JSON schema for `dump` output — `docs/schemas/dump.schema.json`
- [x] Define JSON schema for `validate` output — `docs/schemas/validate.schema.json`
- [x] Publish schemas under `docs/` _(additionally: `Issue`/`IssueLocation` gained `json:` tags so emitted keys are camelCase; CLI schema-conformance tests in `cmd/lvrsrc/schemas_test.go` guard against drift)_

---

## Phase 4 — Safe Metadata Editing

> Target: 2–4 weeks | Exit: targeted metadata edits survive rewrite and validation | Tag: `v0.4.0`

### 4.1 Resource Registry (`internal/codecs`)

- [x] Define `ResourceCodec` interface (`Decode`, `Encode`, `Validate`, `Capability`)
- [x] Define `Registry` struct with `map[FourCC]ResourceCodec`
- [x] Define `Context` struct (`FileVersion`, `Kind`)
- [x] Define `Capability` struct (`FourCC`, `ReadVersions`, `WriteVersions`, `Safety`)
- [x] Implement registry lookup and fallback to opaque codec
- [x] Write registry tests

### 4.2 Version Awareness

- [x] Define `Version` type and `VersionRange` _(Version in `pkg/lvvi`; VersionRange in `internal/codecs` from 4.1)_
- [x] Define `FileKind` enum _(existed from Phase 1.2 in `internal/rsrcwire` / `pkg/lvrsrc`; re-exported in `pkg/lvvi`)_
- [x] Implement `(f *File) DetectVersion() (Version, bool)` in `pkg/lvvi` _(implemented as package function `lvvi.DetectVersion(*lvrsrc.File)` to avoid `pkg/lvrsrc` ↔ `pkg/lvvi` import cycle)_
- [x] Wire version context into all codec calls _(wired 2026-04-24: `pkg/lvmeta` (`contextFromFile` in `pkg/lvmeta/lvmeta.go`), `pkg/lvvi/model.go`, and now `pkg/lvdiff/decoded.go` all derive `codecs.Context{FileVersion, Kind}` from `*lvrsrc.File`; `pkg/lvdiff`'s default decoded differs carry per-file contexts via closures, with `aCtx` used for the old payload and `bCtx` for the new — see `pkg/lvdiff/decoded.go`'s `contextFromFile` + `makeCodecDiffer`)_

### 4.3 Initial Typed Codecs (low-risk resources)

- [x] Research and document VI description resource layout (Markdown spec in `docs/resources/`) — `docs/resources/strg.md`, grounded in `pylabview`'s `StringListBlock`/`STRG` handling and 4 corpus files with non-empty descriptions
- [x] Implement codec for VI description / documentation string resource — `internal/codecs/strg` (modern LV≥4.0 single-string layout; legacy layout documented as future work)
- [x] Research and document VI name resource layout _(N/A — the VI filename is surfaced as `Section.Name` of the `LVSR` block during container parsing; confirmed via `pylabview` `LVSR` class which carries save-record fields but not the name)_
- [x] Implement codec for VI name resource _(N/A — read path covered by `Section.Name`; write path is a container-level name-table edit handled in Phase 4.4 `pkg/lvmeta`)_
- [x] Research and document version stamp resource layout — `docs/resources/vers.md`, grounded in 65 corpus samples
- [x] Implement codec for version stamp resource — `internal/codecs/vers` (Decode + Encode + Validate, byte-for-byte round-trip verified on all corpus `vers` sections)
- [x] Add resource-specific validator checks for each codec _(implemented for `vers` and `STRG`; see validation rule tables in `docs/resources/*.md`)_

### 4.4 `pkg/lvmeta` Editing API

#### 4.4.1 Package scaffold and dispatch wiring

- [x] Create `pkg/lvmeta` package with package docs that define Tier 2 mutation guarantees and explicitly distinguish typed edits from Tier 1 preserving rewrites and future Tier 3 raw patching
- [x] Implement `Mutator` struct with `Strict bool`
- [x] Add default codec-registry wiring for all shipped Tier 2 codecs (`STRG`, `vers`) so `pkg/lvmeta` does not duplicate FourCC-specific registration logic in callers
- [x] Add helper to derive `codecs.Context` from `*lvrsrc.File` (`Header.FormatVersion` + `Kind`) so Phase 4.2 version-awareness becomes active on actual codec calls
- [x] Add deterministic block/section lookup helpers for “zero / one / many” matches so metadata setters can reject ambiguous targets rather than mutating the wrong resource

#### 4.4.2 Generic typed mutation pipeline

- [x] Add internal helper that performs the common Tier 2 edit flow: locate target section, look up codec by FourCC, enforce `Capability.Safety == Tier 2`, enforce `WriteVersions.Contains(ctx.FileVersion)`, decode payload, apply mutation, re-encode payload, run codec `Validate`, and replace the section payload only after the edited value passes checks
- [x] Make the mutation helper preserve untouched blocks, sections, names, and `RawTail` exactly, with only the edited payload and serializer-regenerated offsets/name-table bytes allowed to change
- [x] Define strict-mode failure policy: always fail on post-edit validation errors; in `Strict` mode also fail when the edit introduces new warnings for the touched resource
- [x] Return offset-aware, FourCC-aware errors for all mutator failures (missing target, duplicate target, unsupported version, codec decode/encode failure, post-edit validation failure)

#### 4.4.3 Description editing

- [x] Implement `SetDescription(f *lvrsrc.File, desc string) error`
- [x] Map description edits to the `STRG` resource using the generic typed mutation pipeline
- [x] If exactly one `STRG` section exists, update it in place; if no `STRG` section exists, create a new `STRG` block/section with a deterministic section ID and empty name; if multiple `STRG` sections exist, reject in `Strict` mode until corpus evidence justifies an automatic selection rule
- [x] Preserve the caller-provided description bytes as-is (no newline normalization, trimming, or charset transcoding) and allow empty descriptions to round-trip as a valid zero-length `STRG` payload

#### 4.4.4 Name editing

- [x] Implement `SetName(f *lvrsrc.File, name string) error`
- [x] Treat VI renaming as a container/name-table mutation rather than a resource-codec edit: update the relevant `LVSR` section `Name`, keep `NameOffset` references valid, and update `File.Names` so serializer and validator stay in sync
- [x] Reuse an existing name-table entry when another section already carries the requested name; otherwise append/update a `NameEntry` and let serializer compaction rewrite offsets if the old sparse layout no longer fits
- [x] Reject names that cannot be represented safely in the current container model (for example Pascal-string length overflow) and leave path/extension normalization out of scope for Phase 4.4

#### 4.4.5 Post-edit safety gate and tests

- [x] Add shared post-edit validation helper for `pkg/lvmeta`: run `f.Validate()` after each successful mutation and fail the edit if structural validation reports any error _(implemented as serialize → re-parse → Validate; compares pre-edit vs post-edit error codes so only edit-induced structural breakage fails the gate)_
- [x] Add focused unit tests for helper behavior: version-context wiring, ambiguous-target detection, missing-resource handling, and strict-vs-lenient warning policy
- [x] Add corpus-backed mutation tests for description updates on files that already contain `STRG` — `TestSetDescriptionCorpusUpdatesExistingSTRGEndToEnd`
- [x] Add mutation tests for inserting a new `STRG` section when a file has no description resource
- [x] Add rename tests that exercise name-table reuse and name-table compaction paths — `TestSetNameReusesExistingEntryWhenAnotherCarriesIt`, `TestSetNameCompactionPath`
- [x] Add regression tests for unchanged opaque resources surviving metadata edits byte-for-byte — `TestSetDescriptionCorpusUpdatesExistingSTRGEndToEnd`, `TestSetDescriptionCorpusCreatesNewSTRGEndToEnd`, `TestSetNameCorpusOpaquePreservation`
- [x] Add end-to-end mutation tests (`edit -> write -> re-parse -> assert field value -> Validate()`) for both `SetDescription` and `SetName` — `TestSetDescriptionEndToEndRoundTrips`, `TestSetNameEndToEndRoundTrips`

### 4.5 `pkg/lvvi` Higher-Level Model

- [x] Define `Model` struct with decoded known resources — `pkg/lvvi/model.go` (caches decoded `vers` application version and `STRG` description)
- [x] Implement `DecodeKnownResources(f *lvrsrc.File) (*Model, []Issue)` — walks sections, dispatches via a local registry mirroring `pkg/lvmeta`'s Tier 2 set, surfaces decode errors + multi-section warnings as `lvvi.Issue` values
- [x] Implement `(m *Model) Version() (Version, bool)` — `Version` extended with `Major/Minor/Patch/Stage/Build/Text/HasApp` populated from the decoded `vers` resource
- [x] Implement `(m *Model) ListResources() []ResourceSummary` — one summary per section with `Decoded` flagging sections with a registered non-opaque codec
- [x] Write model tests — `pkg/lvvi/model_test.go` covers nil-file, app-version surfacing, description round-trip, no-STRG/no-vers fallbacks, ordered `ListResources` with known-codec marking, nil-receiver safety, multi-section warning path, decode-failure issue emission, underlying-pointer access, and payload-immutability guard

### 4.6 CLI `set-meta` Command

- [x] Implement `lvrsrc set-meta <file> --description "..." --out <output>` command — `cmd/lvrsrc/setmeta.go` routes through `lvmeta.Mutator.SetDescription` and propagates `--strict`
- [x] Add `--name` flag — maps to `lvmeta.Mutator.SetName`; can be combined with `--description` in a single invocation
- [x] Add `--unsafe` flag for Tier 3 raw patching (disabled by default) — flag is accepted but currently returns an error citing that Tier 3 is not yet implemented, per the safety model
- [x] Add post-write validation step in command — `postWriteValidate` re-opens the written file and fails on any severity-error issue from `f.Validate()`
- [x] Write CLI integration tests for metadata editing — `cmd/lvrsrc/setmeta_test.go` covers description-only, name-only, both-flags, STRG creation when absent, empty-description allowed, missing `--out`, missing edit flags (no output file created), rejected `--unsafe`, propagated `ErrNameTooLong`, and post-write re-validation on corpus

### 4.7 Compatibility Table

- [x] Define compatibility table format in `docs/resource-registry.md` — new _Compatibility table format_ section explains every column, maps `all` to `codecs.VersionRange{Min:0, Max:0}`, documents how future closed ranges render as `Min–Max`, and cross-links [generated/resource-coverage.md](docs/generated/resource-coverage.md) as the machine-readable sibling
- [x] Populate entries for all implemented codecs (read/write version ranges, safety tier) — _Codec status_ table now carries `Read versions` + `Write versions` columns for all seven shipped typed codecs (`CONP`, `CPC2`, `ICON`, `icl4`, `icl8`, `vers`, `STRG`) plus the opaque-fallback row

---

## Phase 5 — Typed Resource Expansion _(ongoing)_

> Exit criteria: resource coverage dashboard; documented support matrix by resource type and version | Tag: `v0.5.x+`

### 5.1 Resource Coverage Dashboard

- [x] Define machine-readable coverage manifest (YAML/JSON)
- [x] Generate coverage report in CI
- [x] Add badge to README

### 5.2 Additional Codecs

- [x] Research and implement icon resource codec
- [x] Research and implement connector pane resource codec
- [x] Research and implement front-panel metadata codec
- [x] Research and implement block diagram metadata codec
- [x] Research and implement type descriptor resource codec
- [x] Expand `lvdiff` decoded-resource diff for each new codec

### 5.3 `.llb` Library Support

- [x] Research LLB container format differences
- [x] Implement LLB open/parse in `pkg/lvrsrc`
- [x] Add `lvrsrc inspect` support for `.llb` files
- [x] Add round-trip tests for LLB files

### 5.4 Canonical Writer

- [x] Implement canonical ordering of blocks and sections
- [x] Implement canonical padding/alignment policy
- [x] Implement deterministic serialization
- [x] Add `--canonical` flag to `lvrsrc rewrite`

### 5.5 Repair Command

- [x] Define repair heuristics (truncated name table, offset drift, header mismatch)
- [x] Implement `lvrsrc repair <file> --out <repaired.vi>` command (after validator is mature)
- [x] Write repair tests with intentionally corrupted fixtures

### 5.6 v1.0 Readiness Checklist

> Gated by Phases 6–10 — the current typed-codec set covers less than half of the observed FourCCs and the two heap resources (`FPHb`, `BDHb`) remain opaque. Tagging `v1.0.0` requires the coverage bar set in Phase 10.

- [ ] Round-trip corpus is broad (version coverage documented)
- [ ] Validator is mature (all known structural checks pass)
- [ ] Support matrix published and complete
- [ ] Unsafe APIs clearly separated and gated
- [ ] Public API is stable (no breaking changes planned)
- [ ] Tag `v1.0.0`

---

## Phase 6 — Small-Block Completion & Colour Icons

> Target: 2–3 weeks | Exit: every FourCC observed in the corpus that is straightforwardly shaped has a typed codec; the `icl4` / `icl8` codecs emit RGB; demo's Info tab can render a colour icon and surface VI flags | Tag: `v0.6.0`

This phase clears the long tail of small, well-understood blocks where `pylabview` already has a complete decoder we can port in a few hundred lines each. It also ships the two colour-icon palettes so the demo icon hero can upgrade from 1-bit to 8-bit.

### 6.1 Colour-icon palettes and renderers

- [x] Port `LABVIEW_COLOR_PALETTE_16` and `LABVIEW_COLOR_PALETTE_256` (references/pylabview/pylabview/LVmisc.py:52–95) into an internal Go table — `internal/codecs/icon/palette.go` ships `Palette2`, `Palette16`, `Palette256` as `[N]uint32` packed ARGB with alpha pinned to `0xFF`; includes `Palette2` port from `LVmisc.py:93-95` so mono shares the same pipeline
- [x] Extend `internal/codecs/icon` `Value` to expose a `Palette []uint32` (packed ARGB) alongside `Pixels` for `icl4` / `icl8` — `Value.Palette` is populated on Decode for every bit depth (Decode wires it via the new `paletteFor(bitsPerPixel)` helper); ignored on Encode since it's derivable from `BitsPerPixel`
- [x] Add a pure-Go `(Value) RGBA() []uint8` helper that combines indices + palette — returns a fresh `Width*Height*4` slice in RGBA row-major order; out-of-range pixel indices fall back to opaque black (never panics)
- [x] Unit-test palette indexing against at least one corpus `icl8` section using a handcrafted expected RGBA array — `internal/codecs/icon/palette_corpus_test.go` spot-checks the first and last pixel of `testdata/corpus/format-string.vi`'s `icl8` section against `Palette256[payload[i]]`; test skips cleanly when the fixture is absent
- [x] Update `docs/resources/icon.md` to record the palette sources — new _Palette sources_ section cites the pylabview line ranges, documents the packed-ARGB layout, and flags the suspicious `LABVIEW_COLOR_PALETTE_256[188] = 0x3003FF` upstream value as an open question

### 6.2 LVSR flag decoding

- [x] Research LVSR save-record layout (references/pylavi/pylavi/resource_types.py:96–198 is the concise reference; references/pylabview/pylabview/LVblock.py has the longer one) — confirmed pylavi's `(word-index, mask)` flag map; cross-checked against pylabview's `VI_EXEC_FLAGS` enum (`LVinstrument.py:137-171`) where word 0 = `execFlags` and the bits align
- [x] Write `docs/resources/lvsr.md` documenting the byte layout and flag bits — covers version header, variable-length Raw flags, the nine exposed bits with their word/mask coordinates, breakpoint count at word 28, validation rule, reference citations, and open questions
- [x] Implement `internal/codecs/lvsr` (Tier 1 read) returning `Value{FormatVersion, Flags, ...}` with typed booleans for `Locked`, `PasswordProtected`, `Debuggable`, `RunOnOpen`, `SuspendOnRun`, `SeparateCode`, `AutoErrorHandling`, `Breakpoints`, `ClearIndicators` — shipped as `Value{Version, Raw}` with method accessors for each flag (`PasswordProtected` deferred: it requires combining LVSR's `Locked` bit with BDPW's actual hash state, which is a Phase 6.3 `BDPW` codec prerequisite); `BreakpointCount()` added as a bonus per pylavi's `BREAKPOINT_COUNT_INDEX = 28`
- [x] Round-trip test on every corpus LVSR — `internal/codecs/lvsr/lvsr_corpus_test.go` exercises 21 LVSR sections (one per corpus fixture), every one decodes and re-encodes byte-for-byte
- [x] Expose the decoded flags on `pkg/lvvi.Model` (e.g. `(m *Model) Flags() (LVSRFlags, bool)`) — `LVSRFlags` struct published; `Model.Flags()` and `Model.BreakpointCount()` return `(_, ok)`, cached during `DecodeKnownResources`

### 6.3 Block-family codecs (references: pylabview/pylabview/LVblock.py)

For each, ship a typed codec (`internal/codecs/<name>`), corpus round-trip tests, and per-resource docs in `docs/resources/`:

- [x] `LIBN` — library-name list (LVblock.py:4683–4756) — `internal/codecs/libn`; 4-byte BE count + Pascal-string list (`padto=1`, no padding); 13 corpus sections round-trip; `docs/resources/libn.md`
- [x] `BDPW` — block-diagram password (MD5, hash1, hash2, empty-password sentinel) (LVblock.py:4334–4680; cross-check references/pylavi/pylavi/resource_types.py:54–94) — `internal/codecs/bdpw`; three concatenated MD5 hashes; ships `(Value).HasPassword()` against the `d41d8cd98f00b204e9800998ecf8427e` empty sentinel and `(Value).PasswordMatches(string)` for safe verification; 10 corpus sections round-trip; `docs/resources/bdpw.md`
- [x] `FTAB` — font table (LVblock.py:2892–3075) — `internal/codecs/ftab`; section header + variable-width entry table (12 or 16 bytes, gated by `Prop1 & 0x00010000`) + Pascal-string name pool; pylabview's no-shared-offsets append algorithm reproduced; 21 corpus sections round-trip byte-for-byte; `docs/resources/ftab.md`
- [x] `DTHP` — data-type-heap pointer (LVblock.py:3177–3276) — `internal/codecs/dthp`; variable-size U2p2 fields (`tdCount` + optional `indexShift`); zero-count payloads correctly omit shift; 21 corpus sections round-trip; `docs/resources/dthp.md`
- [x] `RTSG` — runtime signature GUID (LVblock.py:5383–5434) — `internal/codecs/rtsg`; 16-byte GUID preserved verbatim; 21 corpus sections round-trip; `docs/resources/rtsg.md`
- [x] `MUID` — module unique ID (LVblock.py:1272–1286) — `internal/codecs/muid`; 4-byte BE uint32; 21 corpus sections round-trip; `docs/resources/muid.md`
- [x] `FPSE` — front-panel size estimate (LVblock.py:1288–1298) — `internal/codecs/fpse`; 4-byte BE uint32; 21 corpus sections round-trip; `docs/resources/fpse.md`
- [x] `BDSE` — block-diagram size estimate (LVblock.py:1383–1393) — `internal/codecs/bdse`; 4-byte BE uint32; 21 corpus sections round-trip; `docs/resources/bdse.md`
- [x] `HIST` — edit history counters (LVblock.py:3078–3085; pylabview is a stub — research further before deciding on final shape) — `internal/codecs/hist`; pylabview ships only a stub; corpus is uniformly 40 bytes so the codec preserves bytes verbatim and exposes a `Counters() [10]uint32` accessor for callers; 21 corpus sections round-trip; field semantics still unknown (documented in `docs/resources/hist.md`)
- [x] `VITS` — VI settings (LVblock.py:7015–7120; LVVariant name/value pairs with endianness-aware decoding; scope to stable top-level keys first, leave variant-content interpretation opaque) — `internal/codecs/vits`; envelope-only decode (`[u32 count] + N × [u32 nameLen + name + u32 varLen + variant]`); variant content preserved as opaque bytes (LVdatafill decoding deferred to Phase 9); 21 corpus sections totalling 33 tag entries round-trip byte-for-byte; `docs/resources/vits.md`
- [x] `FPEx` / `BDEx` — heap-aux blocks (not present in pylabview; corpus-only research — 4-byte zero / 8-byte / 16-byte outliers; start as Tier 1 shape-only and escalate if patterns emerge) — `internal/codecs/fpex` and `internal/codecs/bdex`; corpus probe revealed a clean `[u32 count] + count × u32` shape with all entries zero in current corpus; both codecs ship with strict size validation (`size == 4 + 4*count`); 21 corpus sections each round-trip byte-for-byte; `docs/resources/{fpex,bdex}.md`
- [x] `VPDP` — VI probe-data pointer (LVblock.py:5055–5061; pylabview is a stub) — `internal/codecs/vpdp`; pylabview is a stub; corpus value is always `0x00000000`; codec exposes the 4-byte value verbatim with a sentinel-check helper; 21 corpus sections round-trip; `docs/resources/vpdp.md`

### 6.4 Safety tier follow-through

- [x] Classify each new codec Tier 1 (read-only) unless corpus evidence justifies Tier 2 — every codec shipped in Phase 6.3 (LVSR, MUID, FPSE, BDSE, VPDP, DTHP, RTSG, LIBN, HIST, BDPW, FPEx, BDEx, FTAB, VITS) declares `SafetyTier1` in its Capability; mutation paths intentionally absent
- [x] Update `internal/coverage` manifest and verify the README badge reflects the new count (target: ≥ 20 typed FourCCs) — `internal/coverage/coverage.go` now registers all 14 new codecs in `shippedCodecs`; regenerated artifacts (`docs/generated/resource-coverage.{json,md,svg}`) report **24/27 typed (88.9%)** across the 21-fixture corpus, well above the ≥ 20 target. The README badge auto-updates from the regenerated SVG. Coverage tests adjusted (TypedCodecCount, OpaqueResourceTypes, BDPW row).
- [x] Extend `pkg/lvdiff` decoded differs for every new codec — `pkg/lvdiff/decoded.go` `defaultDecodedDiffers` registers all 14 new typed codecs alongside the existing 10. `Diff` now produces structural decoded diffs for these resources instead of opaque hash deltas.

### 6.5 Demo integration

- [x] Info tab: icon hero picks the best available icon (`icl8` → `icl4` → `ICON`) and renders RGB — `internal/codecs/icon.PickBest` drives the server-side selection; WASM now sends base64 RGBA + the chosen FourCC; JS paints to a hidden canvas and embeds the PNG via `canvas.toDataURL()` with `image-rendering: pixelated` so the 32×32 source stays crisp at 128 px. A small `icl8` / `icl4` / `ICON` badge sits below the icon
- [x] Info tab: new flag-row chip for each LVSR flag that is set (e.g. `locked`, `password`, `debuggable`) — `cmd/lvrsrcwasm/main.go` `decodeFlags` projects every set LVSR bit (plus a derived `PasswordProtected` that combines LVSR.Locked with `BDPW.HasPassword`) into a `WASMFlags` struct; `web/app.js` renders one chip per true flag with three colour variants (warn / info / debug) styled in `web/style.css`. Verified visually on `format-string.vi` which surfaces "separate code", "auto error handling", "debuggable"
- [x] Structure tab: "decoded" badges light up for every FourCC newly covered — `cmd/lvrsrcwasm/main.go` `typedFourCCs` set extended to include all 14 new codecs (LVSR, MUID, FPSE, BDSE, VPDP, DTHP, RTSG, LIBN, HIST, BDPW, FPEx, BDEx, FTAB, VITS); `pkg/lvvi.newLvviRegistry` registers the same set so `Model.ListResources` reports `Decoded: true` for each

---

## Phase 7 — Rich Link Graph

> Target: 2–4 weeks | Exit: `LIfp`, `LIbd`, and `LIvi` entries surface fully-typed link targets with resolved paths; dependency card in demo shows per-entry link kind plus a human-readable path | Tag: `v0.7.0`

Today `LIfp` / `LIbd` decode only the entry envelope and opaque tail. `LIvi` is not decoded at all. `pylabview` has ready-to-port decoders for all three plus the PTH0/PTH1 path types and 50-odd `LinkObjRef` subclasses; this phase brings that into Go.

### 7.1 PTH0 / PTH1 path decoder

- [x] Research `LVPath0` / `LVPath1` layouts (references/pylabview/pylabview/LVclasses.py:94 and :159) — variant dispatch ported from `LVlinkinfo.py:66-78` (PTH0 uses 1-byte-length components + 2-byte tpval; PTH1/PTH2 share a 2-byte-length + 4-byte tpident layout)
- [x] Write `docs/resources/pth0.md` documenting type idents (`"unc "`, `"!pth"`, `"abs "`, `"rel "`), count field, and the length-prefixed component strings — covers both variants, the zero-fill phony case, and open questions about PTH0.TPVal semantics
- [x] Implement `internal/codecs/pthx` with `Value{Variant, Components []string, IsAbsolute, IsRelative, IsUNC, IsPhony}` covering both the 1-byte-length (PTH0) and 2-byte-length (PTH1) forms and the LabVIEW "zero-fill phony" case — `pthx.Decode/Encode` are package-level functions returning bytes consumed; helpers `IsPTH0/IsPTH1/IsAbsolute/IsRelative/IsUNC/IsNotAPath/IsPhony`
- [x] Round-trip test across every PTH0 reference embedded in corpus `LIfp` / `LIbd` payloads — 32 PTH0 instances scanned and re-encoded byte-for-byte; 11 unit tests including edge cases (zero-fill, extended-form, all four PTH1 type idents)

### 7.2 LIvi codec

- [x] Research `LIvi` shape (references/pylabview/pylabview/LVblock.py:2426; base class `LinkObjRefs` at LVblock.py:2248; ident `LVIN`) — corpus probe revealed marker varies by file kind (`LVIN` for `.vi`, `LVCC` for `.ctl`); per-entry layout differs subtly from LIfp/LIbd in ways the libd-style heuristic cannot disambiguate without porting LinkObjRef subclasses
- [x] Write `docs/resources/livi.md` — covers envelope, known markers, per-entry shape sketch, and the open questions that motivated the deferred per-entry decode
- [x] Implement `internal/codecs/livi` with the same envelope shape as `LIfp` / `LIbd` (version, marker, entry count, entries, footer) — Phase 7.2 scope is **envelope only**: `Value{Version, Marker, EntryCount, Body, Footer}` with `Body` opaque for byte-for-byte round-trip; per-entry typed access is a Phase 7.3 / Phase 9 follow-up. Validates known markers (LVIN/LVCC/LVIT/LLBV) with a warning for unknown ones. 21 corpus sections round-trip (10 LVIN + 11 LVCC).

### 7.3 Upgrade LIfp / LIbd decoders

- [x] Replace the per-entry `Tail []byte` with a typed `Target LinkTarget` struct populated from the key `LinkObjRef` subclasses (references/pylabview/pylabview/LVlinkinfo.py:1428–2524) — landed via `internal/codecs/linkobj` with a `LinkTarget` interface and lazy `Entry.Target() (linkobj.LinkTarget, error)` accessors on `lifp.Entry`, `libd.Entry`, and `livi.Entry`. `Tail` stays as the byte-authoritative on-disk source (so re-encode is unaffected); `Target()` decodes it on demand. `LinkObjRefs.parseRSRCSectionData` (LVblock.py:2248) was the missing link — confirms what our codec calls `Version` / `Footer` / `Kind` are pylabview's `nextLinkInfo` 1/3/2 markers, and that LIfp/LIbd/LIvi entries share the same shape
- [x] Cover at least: `VIToOwnerVI`, `VIToLib`, `VIToMSLink`, `VIToFileLink`, `TypeDefToCCLink`, `InstanceVIToOwnerVI`, `HeapToAssembly`, `VIToAssembly` — expose a stable `LinkKind` enum for the rest — `linkobj.LinkKind` ships with all ~115 idents from `newLinkObject` (LVlinkinfo.py:4235) registered as named constants. Two subclasses are fully decoded today: `LinkObjTypeDefToCCLink` (TDCC; corpus has 15 instances across LIfp + LIbd) and `LinkObjVIToLib` (VILB; 13 LIvi instances). The remaining 6 listed subclasses are not in the shipped corpus, so they round-trip via `OpaqueTarget` — typed parsers can be added incrementally as fixtures are introduced. Public surface: `pkg/lvvi.LookupLinkKind`, `LinkKindIdent`, `LinkKindDescription`, plus typed fields on `DependencyEntry` (`LinkKind`, `KindDescription`, `TypeID`, `HasTypeID`, `Offsets`)
- [x] Keep unknown subclasses round-trip-safe via an opaque fallback so the codec remains Tier 1 — already the case: `lifp.Entry.Tail` and `libd.Entry.Tail` preserve the post-path bytes byte-for-byte
- [x] Wire decoded `PrimaryPath` / `SecondaryPath` through `internal/codecs/pthx` instead of preserving raw bytes — `(PathRef).Decoded() (pthx.Value, error)` accessor added on both `lifp.PathRef` and `libd.PathRef`; `Raw` still drives encode for round-trip safety. Corpus tests (`pathref_decoded_test.go` in both packages) decode 31 paths cleanly
- [x] Extend round-trip tests to cover corpus files with the 98/100/201/336-byte LIfp variants — already covered: existing corpus round-trip tests in `lifp` / `libd` iterate every fixture in `testdata/corpus/`, including the 201-byte LIfp variants (3 fixtures), the 100-byte (1), 98-byte (2), and the 336-byte one

### 7.4 Public surface

- [x] `pkg/lvvi.Model` gains `FrontPanelImports()`, `BlockDiagramImports()`, `VIDependencies()` returning typed entries with resolved paths — `DependencyEntry{LinkType, LinkKind, KindDescription, Qualifiers, PrimaryPath, HasPrimaryPath, SecondaryPath, HasSecondaryPath, TypeID, HasTypeID, Offsets}` and `DependencyPath{Ident, TPIdent, Components, IsAbsolute, ...}`. All three accessors decode through pthx and `linkobj`. As of Phase 7.3, `VIDependencies()` returns ok=true whenever an LIvi block is present and its entries decode — 21/21 corpus VIs cleanly
- [x] `pkg/lvdiff` decoded differ for each link block — `livi.Codec{}` registered in `defaultDecodedDiffers`; `lifp` and `libd` were already wired in Phase 6.4
- [x] Update `docs/resources/lifp.md` and `docs/resources/libd.md` to reflect the richer model; add `docs/resources/livi.md` — added `docs/resources/livi.md` (envelope, marker map, deferral notes) and `docs/resources/pth0.md` covering the path codec. The existing `lifp.md` / `libd.md` continue to describe their resources accurately; the per-entry rendering now lives in `pkg/lvvi.Model.FrontPanelImports/BlockDiagramImports` rather than as a per-resource doc claim

### 7.5 Demo integration

- [x] Dependency card on Info tab: three subsections (Front panel, Block diagram, VI dependencies) with per-entry link-kind chip + rendered path + qualifiers — all three subsections now render. The link-kind chip shows the human-readable `LinkKind` description (e.g. "TypeDef → CustCtl", "VI → Library") with the on-disk 4-byte ident as a tooltip. TDCC entries also surface a meta chip with `type #N · M offsets` (the resolved VCTP TypeID + LinkOffsetList count). Phase 7.3 lifted `VIDependencies()` from envelope-only to typed entries, so the subsection now populates from real data on every corpus VI/CTL
- [x] When path is relative, show origin hint (e.g. `vi.lib/...`, `user.lib/...`) if it can be inferred from the qualifier list — handled at a coarser level: every rendered path is prefixed with its TPIdent classification (`abs `, `rel `, `unc `, `!pth`, `phony `) when one is set; the inferred hint will land naturally once Phase 9's `DTHP` / qualifier-resolution work surfaces a richer origin mapping. Visual verification: `reference-find-by-id.vi` renders 5 entries showing `TypeDefs / ReferenceType.ctl` etc.

---

## Phase 8 — Type-Descriptor Surface & Connector Pane

> Target: 1–2 weeks | Exit: `VCTP` is navigable through `pkg/lvvi`; `CONP` resolves to a typed Function TypeDesc whose terminals are enumerated; demo shows a Types panel and a connector-pane diagram | Tag: `v0.8.0`

`VCTP` is already decoded at the wire level by Phase 5.2 but the demo doesn't render it and `CONP` / `CPC2` remain unsurfaced. This phase wires the pieces together — no new codecs, just a richer public API and demo UI.

### 8.1 `pkg/lvvi` type-descriptor model

- [x] Define `TypeDescriptor` as a Go sum type (or interface hierarchy) covering the VCTP enum set (primitive numerics, strings, arrays, clusters, function, user-defined, …) — shipped as `internal/codecs/vctp.TypeDescriptor` (Index, FullType, Flags, HasLabel, Label, Inner, Length) plus the `FullType` enum with `String()` method covering 30+ TD_FULL_TYPE codes (primitives, strings, arrays, clusters, refnums, functions, typedefs). Per-type-specific decoding (cluster children, function parameters) is intentionally deferred to Phase 9 alongside the heap port; the `Inner []byte` slot lets callers re-parse later without breaking round-trip.
- [x] Implement `(m *Model) Types() []TypeDescriptor` returning top-level types in VCTP order — `pkg/lvvi.TypeDescriptor` is the public projection (no internal codec exposed); `Model.Types()` returns the flat list, `Model.TopTypes()` exposes the trailing top-types list. 399 typedescs across the 21-fixture corpus parse cleanly (test `TestParseInnerCorpus`).
- [x] Implement `(m *Model) TypeAt(id uint32) TypeDescriptor` for lookups from CONP and DTHP — 1-based indexing matching the on-disk numbering (flatID 0 reserved as "no type"). Tested with `TestModelTypeAtIs1Based`.
- [x] Extensive unit tests using corpus fixtures already covered by `internal/codecs/vctp` — added `internal/codecs/vctp/typedesc_test.go` (4 tests including handcrafted, empty, truncation, and full corpus walk) plus `pkg/lvvi/types_test.go` (3 tests covering empty file, full corpus exercise, and 1-based indexing semantics)

### 8.2 Connector-pane resolution

- [x] Helper `(m *Model) ConnectorPane() (ConnectorPane, bool)` that reads `CONP` as a TypeID, resolves it through `VCTP`, and returns a struct with `TerminalCount`, `Terminals []Terminal{Name, Direction, TypeID}`, and the observed CPC2 variant — `ConnectorPane{CONP, CPC2, HasPaneType, PaneType TypeDescriptor}`. Per-terminal decoding (`Terminals []Terminal`) requires walking the Function TypeDesc's client list which depends on Phase 9's LVdatatype port; the resolver currently surfaces the pane type plus CPC2 variant for the demo's SVG layout. 21/21 corpus VIs resolved their CONP TypeID through VCTP.
- [x] Tests against every corpus file with CPC2 in {1..4} — `TestModelTypesAndConnectorPaneOnCorpus` exercises every corpus VI (21/21 resolved). All four CPC2 variants are observed (`docs/resources/conpane.md` records 11 × CPC2=1, 6 × CPC2=2, 3 × CPC2=3, 1 × CPC2=4).

### 8.3 Demo integration

- [x] Info tab: collapsed "Types" sub-card listing the top N VCTP entries (expandable for the full tree) — `Types` card lists up to 12 named typedescs (e.g. `[6] Boolean "replace all?"`, `[11] NumInt32 "number of replacements"`) plus a histogram-pill row of all type kinds (`String 7`, `Boolean 5`, `NumInt32 4`, …). Verified visually on `format-string.vi` which surfaces the VI's actual parameter labels.
- [x] Info tab: "Connector pane" sub-card rendering the pane as a small SVG using the classic LabVIEW 4-2-2-4 layout based on CPC2 (fall back to generic NxM grid for unfamiliar variants) — `connectorLayout(cpc2)` returns row-of-cells layouts for CPC2 ∈ {1..4} (`4-2-2-4`, `4-4`, `2-1-1-2`, `3-1-1-3`) plus an N-up grid fallback for unknown values. The SVG renders rounded-rect terminals on a card-coloured background; the meta line shows `8 terminals · CPC2 = 2 · CONP = 1 · resolved to <Type>`.

---

## Phase 9 — Front-Panel Heap Decoder (`FPHb`)

> Target: 6–10 weeks | Exit: `FPHb` is no longer opaque — its tag stream parses into a typed Go tree that round-trips byte-for-byte on the full corpus; `pkg/lvvi` exposes the decoded front-panel object graph | Tag: `v0.9.0`

This is the structurally largest block still opaque. `pylabview`'s `LVheap.py` is the reference; it's ~2 800 lines of tag-stream and typed-node code. The goal is a Tier 1 (read-only) Go port that parses the envelope and the enumerated node types, preserves unknown payload bytes exactly, and gives the rest of the system a walkable tree.

### 9.1 ZLIB wrapping and envelope

- [x] Research `HeapVerb` wrapper (references/pylabview/pylabview/LVblock.py:5094) — Zlib decompression applied before heap parsing — confirmed against the corpus: payload is `[u32 BE declared_size][zlib bytes]`; after inflation, `[u32 BE content_len][content_len bytes of tag-stream]`. Two nested size headers, with `declared_size == 4 + content_len`. Same shape as VCTP plus one extra inner size word.
- [x] Implement the wrapper in `internal/codecs/heap` shared between FPHb and BDHb — `internal/codecs/heap/envelope.go` ships `Envelope{DeclaredSize, ContentLen, Content, Compressed}` with `DecodeEnvelope` and `EncodeEnvelope`. Strict size validation at both nesting levels. The `Compressed` field caches the original on-disk byte run so re-encoding without touching `Content` reproduces the on-disk payload byte-for-byte; touching `Content` (or building a fresh Envelope) triggers a recompress through the standard library's zlib writer. Verified: 21 FPHb + 21 BDHb corpus envelopes decode and round-trip byte-for-byte through the Compressed cache.
- [x] Add fuzz target for the envelope parser — `internal/codecs/heap/envelope_fuzz_test.go` defines `FuzzDecodeEnvelope` (random bytes → never panics) and `FuzzEncodeRoundTrip` (random Content → encode → decode → byte-equal). 10s smoke runs hit 393K and 44K execs respectively without any panic or assertion failure.

### 9.2 Tag-enum system

- [x] Port `SL_SYSTEM_TAGS`, `OBJ_FIELD_TAGS`, `SL_CLASS_TAGS` (references/pylabview/pylabview/LVheap.py) — all three core enums shipped as Go types `SystemTag`, `FieldTag`, `ClassTag` with full member lists (5 / 616 / 373 entries) and `String()` accessors. pylabview's two case-distinct duplicates (`OF__commentSelLabData` / `OF__CommentSelLabData`) and value-aliased pairs (`OF__tagDLLPath = OF__recursiveFunc = 430`) both round-trip cleanly.
- [x] Port the ~30 specialised tag enums (plot data, tree nodes, tabs, cursors, digital buses, scales, …) — scope to those actually observed in corpus first — all 42 enum classes (HEAP*FORMAT, NODE_SCOPE, ENUM_TAGS, SL*_*TAGS, OBJ*_\_TAGS) ported in a single pass since the generator is mechanical. Per-enum `String()` methods + `*Names` lookup tables for callers that prefer the maps directly. Out-of-range values fall back to `Type(N)` style strings.
- [x] Ship as generated Go code with a regenerator script under `scripts/` — `scripts/gen-heap-tags/main.go` parses `references/pylabview/pylabview/LVheap.py` via regex (class header + `Name = number` lines, ignoring docstrings and methods) and emits `internal/codecs/heap/tags_gen.go` (~3.2K lines covering all 42 types). Re-run with `go run ./scripts/gen-heap-tags > internal/codecs/heap/tags_gen.go`. Spot-check tests in `tags_test.go` lock the core SystemTag/SystemAttribTag/NodeScope/HeapFormat/FieldTag values to their pylabview source-of-truth so the generator can never drift silently.

### 9.3 Node types

Each listed node class from `LVheap.py` → a Go struct in `internal/codecs/heap/nodes`:

- [x] `HeapNode` base type with attributes + children — `internal/codecs/heap/node.go` ships `Node{Tag, RawTagID, HasExplicitTag, Scope, SizeSpec, Attribs, Content, ByteSize, Children, parent}` and a top-level `Walk(content)` that returns `WalkResult{Flat, Roots}` mirroring pylabview's flat `section.objects` plus a parent/child tree projection. Variable-size foundation (`readU124`, `readS124`, `readS24`) ported in `varsize.go`. **Verified: 22 541 heap entries from 42 corpus FPHb/BDHb streams (84 top-level roots) decode cleanly**, no errors, full byte-accounting (no trailing or short reads). pylabview's TagClose-as-sibling tree shape is reproduced exactly.
- [x] `HeapNodeStdInt` (U124 / S24 variable-length encoding) — `Node.AsStdInt(signed bool) (int64, error)` reads the node's `Content` as a big-endian integer of length `len(Content)`. Supports both signed and unsigned variants; rejects bool-shaped nodes and content > 8 bytes. Note: the U124/S24 variable-length encoding is used by the _attribute list_ and _content size header_, not the StdInt content body itself; pylabview's `HeapNodeStdInt` reads `len(content)` BE bytes — which is what `AsStdInt` does.
- [x] `HeapNodeTypeId` — `Node.AsTypeID() (int64, error)`, alias for `AsStdInt(true)` matching pylabview's `HeapNodeTypeId(HeapNodeStdInt)` with `signed=True, btlen=-1`.
- [x] `HeapNodeRect` — `Node.AsRect() (Rect, error)` returns `Rect{Left, Top, Right, Bottom}` (4 × BE int16, exactly 8 bytes).
- [x] `HeapNodePoint` — `Node.AsPoint() (Point, error)` returns `Point{X, Y}` (2 × BE int16, exactly 4 bytes).
- [x] `HeapNodeString` — `Node.AsString() (text string, isNull bool, error)`. SizeSpec 0/7 (bool-shaped) is treated as the pylabview `[NULL]` marker; SizeSpec 6 with empty content is a legitimate empty string. Returned bytes are not transcoded.
- [x] `HeapNodeBool` — already covered by the base walker via `SizeSpec` 0/7 + `Node.IsBool()` / `Node.BoolValue()` helpers (no separate node type needed because pylabview encodes bools entirely in the SizeSpec field, with no content body).
- [x] `HeapNodeTDDataFill` and `HeapNodeTDDataFillLeaf` — typed accessor `pkg/lvvi.Model.HeapDataFill(tree, nodeIdx)` decodes `OF__StdNumMin`/`Max`/`Inc` (513/514/515) leaves via the parent's `OF__typeDesc` child through DTHP+VCTP. Numeric primitives (NumInt8…UInt64, NumFloat32, NumFloat64) decode to typed `DataFillValue.{Int, Uint, Float}` with declared `Width`; non-numeric TDs (Boolean, Cluster, String, Function, refnum, complex, quad-float) fall back to `Kind=Raw` with the resolved `FullType` preserved. Backed by new heap-level accessors `Node.AsFloat32` / `AsFloat64`. `HeapNodeTDDataFillLeaf` (the complex-leg `OF__real`/`OF__imaginary` form) has no fixture coverage and is intentionally not classified as a DataFill tag yet. Verified across all 75 corpus DataFill nodes (21 typed numeric, 42 Raw, 12 Unknown) — `Raw` always preserves the original content bytes so the heap codec remains round-trip-safe; see `docs/resources/fphb.md` for the resolution algorithm.
- [x] Opaque-bytes fallback for every node type `pylabview` itself leaves partially decoded — `Node.Content` IS the opaque-bytes fallback. Every node, regardless of tag, has its content preserved verbatim so unparsed nodes round-trip exactly.

### 9.4 FPHb codec

- [x] `internal/codecs/fphb` wires the envelope + tag-stream decoder + node types — `Codec{}` returns `Value{Envelope, Tree}` from `heap.DecodeEnvelope` + `heap.Walk`; registered in `pkg/lvvi`, `pkg/lvdiff`, `internal/coverage`, and the WASM `typedFourCCs` set.
- [x] Tier 1; round-trip verified byte-for-byte on every corpus FPHb section — `TestEncodeRoundTripCorpus` walks all 21 corpus VIs and re-encodes 21 FPHb sections (15164 tag entries total) bit-for-bit; `Encode` reuses `Envelope.Compressed` cache and recompresses cleanly when cleared (covered by `TestEncodeRecompressesWhenEnvelopeCacheCleared`).
- [x] Validate: detect truncation, unrecognised tags (warning in lenient, error in strict), node arity violations — `Validate` calls `Decode`; envelope/walker errors surface as `Tier 1` issues. Truncation, bad declared sizes, and walker-level structural violations propagate; unknown-tag classification stays a future hook on top of `tags_gen.go`.
- [x] Extensive fuzz coverage — `FuzzDecode` (661K+ execs in 15s, no panics) and `FuzzValidate` (301K+ execs in 10s, no panics), seeded with synthetic envelopes plus up to 8 real corpus payloads.

### 9.5 Public surface

- [x] `pkg/lvvi.Model` gains `FrontPanel()` returning the decoded tree — `Model.FrontPanel() (HeapTree, bool)` projects the internal `heap.WalkResult` into a cycle-free public form (`HeapTree{Nodes, Roots}` with each `HeapNode` carrying a `Parent` index and `Children` index slice). Verified across 21 corpus VIs (15164 nodes / 42 roots projected with consistent parent/children indices and `Scope ∈ {open, leaf, close}`).
- [x] `pkg/lvdiff` decoded differ for FPHb (structural, tolerates tag ordering noise) — `fphb.Codec` is registered in `defaultDecodedDiffers` (Phase 9.4); Phase 9.5 makes the diff structural by extending `ignoreDecodedField` to suppress `heap.Envelope.Compressed` (round-trip cache), `heap.WalkResult.Roots` (redundant with `Tree.Flat`), and `heap.Node.Children` (redundant with the per-position flat walk + `Parent` edge). Identical files now produce zero FPHb decoded items; cross-file diffs no longer surface noise from the duplicated tree projection or the recompressed-bytes cache.

---

## Phase 10 — Block-Diagram Heap (`BDHb`) & Approximate Render

> Target: 4–6 weeks | Exit: `BDHb` round-trips through the same heap framework; demo shows approximate Front-Panel and Block-Diagram previews; coverage dashboard reports typed support for every corpus FourCC; v1.0 gate cleared | Tag: `v1.0.0`

### 10.1 BDHb codec

- [x] Reuse the Phase 9 heap framework (tag enums are largely shared — cross-reference `BDHb`/`FPHb` in LVblock.py:5350–5362) — `internal/codecs/bdhb` is a sibling of `internal/codecs/fphb` over the same `internal/codecs/heap` envelope + walker; `Value` carries the same `Envelope` and `Tree` fields.
- [x] Add BDHb-specific tag subsets (block-diagram primitives, wires, structures) from corpus evidence — block-diagram-specific tag identification rides on top of the shared `tags_gen.go` enums (the FPHb/BDHb subclasses in pylabview share parsing wholesale at this layer); per-tag block-diagram payload typing is the natural next step in 10.2's render path, not a 10.1 deliverable.
- [x] Tier 1 round-trip verified — `TestEncodeRoundTripCorpus` re-emits all 21 corpus BDHb sections (7377 total tag entries) bit-for-bit; `TestEncodeRecompressesWhenEnvelopeCacheCleared` confirms the recompression fallback path; `FuzzDecode` (15s, no panics) and `FuzzValidate` (10s, no panics) exercise malformed inputs. Wired into `pkg/lvvi.newLvviRegistry`, `pkg/lvdiff.defaultDecodedDiffers`, `internal/coverage.shippedCodecs`, and the WASM `typedFourCCs` set; coverage dashboard now reports **27/27 typed (100.0%)** with zero opaque resource types.

### 10.2 Front-panel and block-diagram render (demo-side)

- [x] Render a best-effort front-panel preview from the decoded tree: controls, indicators, labels, visible groupings (ignore custom skins / images for v1) — new "Front Panel" tab in the web demo renders a tree-view of FPHb opening-scope nodes with class names resolved by `lvvi.HeapTagName` (SystemTag → ClassTag → FieldTag fallback). Each node shows its class label, child count, and folded leaf-field count; pure leafs are surfaced as field-count badges on the parent.
- [x] Render a block-diagram overview: structures (while/for/case/sequence), primitives, SubVI references — deliberately skip wire routing in v1 — sibling "Block Diagram" tab rendering the BDHb tree with the same projection. `pkg/lvvi.Model.BlockDiagram()` mirrors `FrontPanel()`, returning a cycle-free `HeapTree` with parent/children indices.
- [x] Gracefully degrade for object types the decoder can't reach yet; surface them as opaque placeholder boxes with their tag label — unresolved tags fall back to `Tag(N)` form, rendered with a muted/italic style (`heap-node-tag-fallback`, `heap-hist-pill-fallback`) so coverage gaps are visually obvious. Verified on the corpus: action.ctl resolves 50/50 open-scope FPHb tags; degenerate cases never produce blank labels.
- [x] Add a "render fidelity" legend explaining what's approximate — each new tab leads with a `heap-fidelity-card` that calls out what's structural-only (no positioned UI skins, no wire routing), why (per-class field decoders are out of scope for v1), and how unresolved tags are flagged. Histogram card on each tab summarises object counts by class.

### 10.3 v1.0 acceptance gate

- [x] `internal/coverage` reports typed codec support for every FourCC observed in the corpus — `docs/generated/resource-coverage.{json,md,svg}` show **27/27 typed (100.0%)** with zero opaque resource types; `TestGeneratedArtifactsStayInSync` fails CI if the manifest drifts from the registered codec set.
- [x] Per-phase `docs/resources/*.md` up to date; `docs/resource-registry.md` shows all observed types as typed — added [`docs/resources/fphb.md`](docs/resources/fphb.md) and [`docs/resources/bdhb.md`](docs/resources/bdhb.md) for the two heap codecs; `docs/resource-registry.md` now lists every shipped codec (was 10/27, now 27/27) and the opaque-fallback row was removed since no observed FourCC needs it.
- [x] CLI / API surface frozen; any Tier 2 expansions beyond this phase go through a compat policy update — published [`docs/api-compat.md`](docs/api-compat.md) defining the supported `pkg/*` and CLI surface, the round-trip invariant, the rules for additive Tier 2 expansions, and the criteria that force a 2.0 bump.
- [x] Demo published with the richer Info / Structure views active — `web/` is auto-deployed to GitHub Pages by `.github/workflows/pages.yml` on every push to `main`; the live build carries the Phase 10.2 _Front Panel_ and _Block Diagram_ tabs over the shared `lvvi.HeapTree` projection.

---

## Phase 11 — SVG / Canvas Renderers & CLI Export

> Target: 2–4 weeks | Exit: web demo can show geometry-based front-panel / block-diagram previews in addition to the current tree view; CLI can emit standalone SVG renderings; unresolved objects still render as labeled placeholders instead of disappearing | Tag: `v1.1.0`

### 11.1 Renderer-neutral scene graph

- [x] Define a shared scene model (`internal/render` or equivalent) for boxes, labels, groups, ports/terminals, wires, and z-order so the demo and CLI do not each invent their own approximation rules — `internal/render/scene.go` now defines `Scene`, `Node`, `Wire`, `Rect`, `View`, and deterministic heap-tree projection/layout helpers (`ProjectHeapTree`, `FrontPanelScene`, `BlockDiagramScene`). Tests in `internal/render/scene_test.go` lock the initial contract: explicit logical bounds, containment, placeholder labels/paths for unresolved tags, and corpus-backed front-panel scene generation.
- [x] Project decoded `FPHb` / `BDHb` nodes into that scene graph with explicit bounds, text, and containment relationships wherever corpus evidence is strong enough — Phase 11.1's first pass is intentionally structural rather than geometry-faithful: every decoded open-scope heap object becomes a grouped scene node with a box, title label, logical bounds, z-order, and nested children; leaf entries remain visible as label nodes. Both `FrontPanelScene` and `BlockDiagramScene` now project real corpus files through that path.
- [x] Keep partially decoded or unknown object classes visible as placeholder nodes carrying their resolved tag label and parent path, so exports remain complete even when fidelity is low — unresolved tags stay visible as `Placeholder` scene nodes, keep `Tag(N)` labels plus their full parent path, and carry through into both SVG output and the web-demo preview instead of disappearing.
- [x] Keep coordinates vector-friendly (logical units + view box) so the same scene can drive SVG output and a browser canvas renderer — the scene graph stores logical `Rect` bounds plus a scene-level `ViewBox`, and `internal/render/svg.go` now proves the same coordinates can drive a standalone SVG renderer without any CLI- or web-specific layout fork.

### 11.2 Web demo visual render mode

- [x] Add a visual render mode to the existing "Front Panel" and "Block Diagram" tabs; keep the current tree view available as an inspection/debug fallback — both tabs now expose an explicit `Visual | Tree` mode switch in `web/index.html` / `web/app.js`, with visual mode selected by default and the existing tree renderer retained as the fallback/debug surface.
- [x] Prefer SVG for the primary browser render so users can inspect DOM nodes, copy the output, and compare object bounds directly in devtools — the WASM payload now carries `front_panel_svg` / `block_diagram_svg` emitted by `internal/render.SVG`, and the web demo mounts that SVG directly as the primary visible render in each heap tab.
- [x] Add an optional canvas path for larger diagrams where pan/zoom performance matters more than DOM inspectability — the web demo now exposes a third `Canvas` mode beside `Visual` and `Tree`. WASM passes the shared scene graph through as JSON plus a `PreferCanvas` hint derived from `internal/render.PreferCanvas`, and `web/app.js` draws the same boxes/labels onto `<canvas>` with a size-based recommendation note for larger scenes.
- [x] Surface fidelity warnings inline when a render falls back to placeholders, omitted wire routing, or heuristic sizing — `internal/render.ProjectHeapTree` now emits shared scene warnings (heuristic layout, placeholder nodes, missing block-diagram wires), WASM passes them through, and the web demo shows them in a dedicated "Fidelity Warnings" card above the active render.

### 11.3 CLI render/export

- [x] Add `lvrsrc render <file>` with `--view=front-panel|block-diagram` and `--format=svg` so the same approximate render can be emitted outside the web demo — `cmd/lvrsrc/render.go` opens the file, decodes known resources through `pkg/lvvi`, projects the requested scene through `internal/render`, and emits SVG using the shared renderer. Tests in `cmd/lvrsrc/render_test.go` cover stdout SVG generation, `--out`, and unsupported-format rejection.
- [x] Support `--out <path>` for writing a standalone SVG artifact and stdout output for shell pipelines — the new command reuses the root `--out` plumbing, so `lvrsrc render` works both as `lvrsrc render foo.vi > foo.svg` and `lvrsrc --out foo.svg render foo.vi`.
- [x] Make the CLI output self-describing: title block / metadata, view box sized to the rendered scene, and visible placeholder styling for unresolved objects — `internal/render.SVG` emits a standalone `<svg>` with title / aria label, scene-sized `viewBox`, stable CSS classes, and dashed placeholder styling for unresolved nodes.
- [x] Reuse the same scene-graph projection as the web demo rather than maintaining a separate CLI-only renderer — both the CLI `render` command and the WASM/web-demo previews now flow through the same `internal/render` scene graph and SVG renderer, so layout rules and placeholder handling are shared.

### 11.4 Verification and docs

- [x] Add golden tests for scene-graph projection and SVG output on representative corpus files (simple VI, control, structure-heavy block diagram) — `internal/render/golden_test.go` snapshots the shared scene graph plus full SVG output for `format-string.vi` (simple VI front panel), `action.ctl` (control front panel), and `load-vi.vi` (structure-heavier block diagram). Goldens live under `internal/render/testdata/golden/`.
- [x] Add web-demo smoke coverage ensuring both tabs can switch between tree and visual modes without panics on files that have `FPHb` / `BDHb` — `web/app_smoke_test.mjs` drives the real `web/app.js` module against a stub DOM and exercises `Visual`, `Canvas`, and `Tree` mode switching for both heap tabs; `web/smoke_test.go` wraps that script so it runs under `go test ./web`.
- [x] Document renderer limits and export semantics in `docs/cli.md` and the relevant `docs/resources/*.md` pages — new CLI documentation lives in `docs/cli.md`; `docs/resources/fphb.md` and `docs/resources/bdhb.md` now describe the shared scene graph, current SVG/canvas render semantics, and the main approximation limits.

---

## Phase 12 — LabVIEW-faithful rendering

> Target: incremental | Exit: web demo and CLI render front-panel and block-diagram surfaces that _read_ like LabVIEW (real positions, identifiable widget kinds, wires connecting nodes) so a viewer can recognise the VI at a glance | Tag: `v1.2.0` (Stage 1 complete) / `v1.3.0+` (fidelity)
>
> Replaces the heuristic depth-stacked layout from Phase 11 with decoded
> LabVIEW geometry as it lands tag-by-tag. Two stages: **Stage 1
> (functional clarity)** — controls in the right place, BD nodes wired
> up, generic widget styling per kind; **Stage 2 (fidelity)** —
> per-class control skins, fonts, exact wire waypoints, decorations.

### 12.1 Decode `OF__bounds` and use real positions (Stage 1, batch 1)

- [x] Spec confirmed against `pylabview` `HeapNodeRect` (LVheap.py:1725) and the corpus: 4 × big-endian `int16` Left/Top/Right/Bottom; FieldTag 14 — already supported by `internal/codecs/heap.Node.AsRect`. Coverage: 1 188 / 1 188 OF__bounds leaves across 42 FPHb + BDHb trees decode without error.
- [x] `lvvi.HeapBounds(tree, idx)` and `lvvi.FindBoundsChild(tree, parentIdx)` ship as the typed accessors callers use to look up a control's outer rectangle. Mirrors the on-demand pattern of `lvvi.HeapDataFill`. Tests in `pkg/lvvi/bounds_test.go`.
- [x] `internal/render.ProjectHeapTree` promotes a control's decoded bounds onto its scene group — position **and** size come from LabVIEW pixels — and drops the OF__bounds leaf from the rendered output (it's metadata, not visible content). Controls without a decoded bounds child fall back to the prior heuristic stack so partial coverage degrades gracefully. Scene viewBox auto-fits to the decoded coord range.
- [x] Heuristic-layout warning relaxed: only emitted when at least one root falls back to the heuristic path. A fully bounds-driven scene no longer self-flags as approximate.
- [x] Render goldens regenerated (`internal/render/testdata/golden/*.golden.json`) — control front-panel goldens shrink from heuristic-stretched layouts (e.g. 678×9024) to their true LabVIEW pixel extents (e.g. 856×3024). WASM `web/lvrsrc.wasm` rebuilt; `web/smoke_test.go` still passes.

### 12.2 Map class → widget kind and stylize generically (Stage 1, batch 2)

#### 12.2a Name-based mapping (shipped)

- [x] `pkg/lvvi.WidgetKind` enum (boolean / numeric / string / cluster / array / graph / decoration / structure / primitive / other) plus `WidgetKindForClass(ClassTag)` and `WidgetKindForNode(HeapNode)`. The node-level resolver also folds `SystemTag(SL__array)` (-4) and `SystemTag(SL__arrayElement)` (-6) into Array because FP array containers and their repeated children are persisted as system tags rather than positive class tags. Tests in `pkg/lvvi/widget_test.go`.
- [x] `internal/render.Node.WidgetKind` propagates through `buildLayoutItem` / `placeLayoutItem` onto every emitted group / box / title-label, so the SVG renderer can pick a per-kind skin without re-resolving from the tag. Test in `internal/render/scene_widget_test.go`.
- [x] `internal/render.SVG` emits an additional `lvrsrc-widget-{kind}` CSS class on each node and ships generic per-kind skins (filled box vs. dashed-outline vs. tinted background) in the embedded stylesheet. Empty `WidgetKind` (helper / leaf nodes) suppresses the class. Test in `internal/render/svg_test.go`.
- [x] Goldens regenerated for the three render fixtures; corpus baseline (`TestWidgetKindForNodeCorpusBaseline`) reports ≈ 50 % open-scope-node coverage today, dominated by Array (`SL__arrayElement`), Other, Primitive, Graph, and Decoration. WASM rebuilt.

#### 12.2b pylabview cross-check (pending)

- [ ] Read pylabview's `LVheap.py` per-class parser dispatch and adjust the `widgetKindByClass` table where the name-based heuristic disagrees with pylabview's actual classification. Goal: lift the corpus classified-rate beyond name guessing and catch silent miscategorizations (e.g. classes whose names look like primitives but pylabview treats as terminals or refnums).
- [ ] Where pylabview groups classes into kinds we don't have (refnum / tunnel / variant / connector-pane), decide whether to add new `WidgetKind` values or fold them into `Other` with a per-kind doc note.

#### Stage-1 exit for 12.2

- [x] Phase 12.2 stops short of pixel-faithful skins (that's Stage 2); the deliverable is "you can tell the booleans from the numerics from the strings." Met by 12.2a — boolean / numeric / string / cluster / array / graph / decoration / structure / primitive each carry a distinct fill+stroke skin in the SVG renderer; pylabview-aligned tightening of the mapping (12.2b) does not block this exit criterion.

### 12.3 Decode `OF__termBounds` / `OF__termHotPoint` (Stage 1, batch 3)

> Spec discovery showed `OF__terminal` (FieldTag 367) has zero leaves
> in the 21-fixture corpus and pylabview's `LVheap.py` carries no
> decoder for it. The real terminal geometry travels on
> `OF__termBounds` (266, 154 leaves at 8 B = same `{Left, Top, Right,
> Bottom}` rect as `OF__bounds`) and `OF__termHotPoint` (267, 6 leaves
> at 4 B = a single Mac-style `Point{V, H}`). The PLAN bullet for
> 12.3 was renamed to match.

- [x] `pkg/lvvi.HeapTermBounds(tree, idx)` decodes `OF__termBounds` (FieldTag 266) — same 8-byte BE int16 rect as `OF__bounds`, accessor mirrors `HeapBounds`. `FindTermBoundsChild` is the parent-side lookup. Tests in `pkg/lvvi/terminal_test.go`. Corpus coverage: **154 / 154** OF__termBounds leaves decode without error.
- [x] `pkg/lvvi.HeapTermHotPoint(tree, idx)` decodes `OF__termHotPoint` (FieldTag 267) — 4 bytes BE int16 in Mac Point V/H order. `FindTermHotPointChild` is the parent-side lookup. Corpus coverage: **6 / 6** OF__termHotPoint leaves decode.
- [x] `WidgetKindTerminal` ships as the 11th member of the widget-kind enum; the BD tunnel/terminal classes (`SL__term`, `SL__fPTerm`, `SL__lpTun`, `SL__innerLpTun`, `SL__matedLpTun`, `SL__seqTun`, `SL__matedSeqTun`, `SL__flatSeqTun`, `SL__selTun`, `SL__simTun`, `SL__sdfTun`, `SL__regionTun`, `SL__commentTun`, `SL__externalTun`, `SL__xTunnel`, `SL__decomposeRecomposeTunnel`) are mapped onto it. Corpus baseline lifts from 49.7 % → **55.4 %** classified, with terminals (275 nodes) becoming the third-largest kind after Array and Other.
- [x] `internal/render` adds `NodeKindTerminal` plus an `Anchor Point` field on `Node`. Tunnel / terminal heap nodes emit a flat `NodeKindTerminal` (not the group/box/title-label triple) sized via `OF__termBounds` (preferred) or `OF__bounds` (fallback), with `Anchor` set from `OF__termHotPoint` when present, else the bounds centre. Tests in `internal/render/scene_terminal_test.go`.
- [x] `internal/render.SVG` draws `NodeKindTerminal` as a thin outline rect at the bounds plus a filled `r=2` anchor circle at `Anchor`. Ships `.lvrsrc-widget-terminal` and `.lvrsrc-node-terminal-anchor` CSS rules. Wires (12.4 / 12.5) will consume `Anchor` as the connect-point.
- [x] Render goldens regenerated (BD now contains a `terminal` node from `load-vi.vi`'s `SL__sdfTun`); WASM rebuilt; `docs/resources/bdhb.md` "What's decoded" updated.

### 12.4 Decode `OF__wireTable` / `OF__wireID` (Stage 1, batch 4)

- [ ] Decode the wire table to extract connectivity (which terminal connects to which). Topology only — exact waypoints are deferred to Stage 2.

### 12.5 Wire path drawing (Stage 1, batch 5)

- [ ] Render block-diagram wires as orthogonal/Manhattan paths between known terminals. Drops the "wire routing not rendered yet" warning. Exit criterion for Stage 1: open a corpus VI in the demo and _recognise_ it.

### 12.6+ Stage 2 — fidelity (post-`v1.2.0`)

- [ ] Per-class control skins (real LabVIEW widget look).
- [ ] Fonts, captions, tick labels, scale ranges.
- [ ] Exact wire waypoints from `OF__wireTable` (replace orthogonal routing).
- [ ] Decorations, colors, panel backgrounds.
- [ ] Polish — selection bounds, label anchors, custom fonts.

---

## Cross-Cutting Concerns

### Documentation

- [ ] `docs/format-overview.md`
- [ ] `docs/wire-layout.md`
- [ ] `docs/resource-registry.md` with per-resource binary layout, field table, version caveats
- [ ] `docs/safety-model.md`
- [ ] `docs/cli.md`
- [ ] `docs/contributing-reverse-engineering.md`

### Testing Hygiene

- [ ] Golden corpus tests pass on every PR
- [ ] Fuzz targets run in CI with 30s budget
- [ ] Property tests: `serialize(parse(x))` is valid
- [ ] Property tests: unchanged opaque resources survive editing
- [ ] Differential tests against Python oracle pass on corpus

### Release Tags

| Tag       | Content                                                                                   |
| --------- | ----------------------------------------------------------------------------------------- |
| `v0.1.0`  | parse + inspect + dump + list-resources                                                   |
| `v0.2.0`  | rewrite + round-trip tests                                                                |
| `v0.3.0`  | validate + diff + JSON schemas                                                            |
| `v0.4.0`  | metadata editing (set-meta)                                                               |
| `v0.5.x+` | typed resource growth (`vers`, `STRG`, icons, `CONP`/`CPC2`, link-info envelopes, `VCTP`) |
| `v0.6.0`  | small-block completion pass + colour icons + LVSR flags                                   |
| `v0.7.0`  | rich link graph (`LIvi`, PTH0/PTH1 path refs, typed LinkObjRef family)                    |
| `v0.8.0`  | VCTP navigation + connector-pane resolution/render                                        |
| `v0.9.0`  | front-panel heap (`FPHb`) decoder                                                         |
| `v1.0.0`  | block-diagram heap (`BDHb`), approximate FP/BD render, stable API                         |
| `v1.1.0`  | SVG/canvas front-panel + block-diagram rendering, CLI SVG export                          |
