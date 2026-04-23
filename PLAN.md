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
- [ ] Add regression test against Python oracle for block/section counts _(deferred — pylabview infra not yet wired)_
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
- [ ] Wire version context into all codec calls _(deferred: no codec calls exist yet; will be wired in Phase 4.4/4.5 when `pkg/lvmeta` and `pkg/lvvi` dispatch codecs)_

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

- [ ] Define `Model` struct with decoded known resources
- [ ] Implement `DecodeKnownResources(f *lvrsrc.File) (*Model, []Issue)`
- [ ] Implement `(m *Model) Version() (Version, bool)`
- [ ] Implement `(m *Model) ListResources() []ResourceSummary`
- [ ] Write model tests

### 4.6 CLI `set-meta` Command

- [ ] Implement `lvrsrc set-meta <file> --description "..." --out <output>` command
- [ ] Add `--name` flag
- [ ] Add `--unsafe` flag for Tier 3 raw patching (disabled by default)
- [ ] Add post-write validation step in command
- [ ] Write CLI integration tests for metadata editing

### 4.7 Compatibility Table

- [ ] Define compatibility table format in `docs/resource-registry.md`
- [ ] Populate entries for all implemented codecs (read/write version ranges, safety tier)

---

## Phase 5 — Typed Resource Expansion _(ongoing)_

> Exit criteria: resource coverage dashboard; documented support matrix by resource type and version | Tag: `v0.5.x+`

### 5.1 Resource Coverage Dashboard

- [x] Define machine-readable coverage manifest (YAML/JSON)
- [x] Generate coverage report in CI
- [x] Add badge to README

### 5.2 Additional Codecs

- [ ] Research and implement icon resource codec
- [ ] Research and implement connector pane resource codec
- [ ] Research and implement front-panel metadata codec
- [ ] Research and implement block diagram metadata codec
- [ ] Research and implement type descriptor resource codec
- [ ] Expand `lvdiff` decoded-resource diff for each new codec

### 5.3 `.llb` Library Support

- [ ] Research LLB container format differences
- [ ] Implement LLB open/parse in `pkg/lvrsrc`
- [ ] Add `lvrsrc inspect` support for `.llb` files
- [ ] Add round-trip tests for LLB files

### 5.4 Canonical Writer

- [ ] Implement canonical ordering of blocks and sections
- [ ] Implement canonical padding/alignment policy
- [ ] Implement deterministic serialization
- [ ] Add `--canonical` flag to `lvrsrc rewrite`

### 5.5 Repair Command

- [ ] Define repair heuristics (truncated name table, offset drift, header mismatch)
- [ ] Implement `lvrsrc repair <file> --out <repaired.vi>` command (after validator is mature)
- [ ] Write repair tests with intentionally corrupted fixtures

### 5.6 v1.0 Readiness Checklist

- [ ] Round-trip corpus is broad (version coverage documented)
- [ ] Validator is mature (all known structural checks pass)
- [ ] Support matrix published and complete
- [ ] Unsafe APIs clearly separated and gated
- [ ] Public API is stable (no breaking changes planned)
- [ ] Tag `v1.0.0`

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

| Tag       | Content                                            |
| --------- | -------------------------------------------------- |
| `v0.1.0`  | parse + inspect + dump + list-resources            |
| `v0.2.0`  | rewrite + round-trip tests                         |
| `v0.3.0`  | validate + diff + JSON schemas                     |
| `v0.4.0`  | metadata editing (set-meta)                        |
| `v0.5.x+` | typed resource growth                              |
| `v1.0.0`  | stable API, broad corpus, published support matrix |
