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
- [ ] Add `FuzzParseHeader` target
- [ ] Add `FuzzNameTable` target
- [ ] Verify fuzz targets run in CI (short seed corpus)

---

## Phase 2 — Preserving Writer

> Target: 2–4 weeks | Exit: corpus round-trips successfully; opaque sections preserved; rewritten files pass validation | Tag: `v0.2.0`

### 2.1 Wire-Level Binary Writer (`internal/binaryx`)

- [ ] Define `Writer` struct with `io.WriterAt` and byte order
- [ ] Implement `WriteU16`, `WriteU32`, `WriteU64` at offset
- [ ] Implement `WriteBytes(off, data)` method
- [ ] Implement `WritePascalString(off, s)` method
- [ ] Implement offset-patching helper (write placeholder, patch later)
- [ ] Write unit tests for all writer methods

### 2.2 RSRC Serializer — Preserving Mode (`internal/rsrcwire`)

- [ ] Implement offset/padding recomputation for section payloads
- [ ] Implement block table serialization preserving original ordering
- [ ] Implement name table serialization
- [ ] Regenerate both headers consistently (primary and duplicate)
- [ ] Preserve exact bytes for unknown/opaque sections
- [ ] Preserve original padding/alignment where possible
- [ ] Write serializer tests (parse → serialize → parse, compare structure)

### 2.3 Public Write API (`pkg/lvrsrc`)

- [ ] Implement `(f *File) WriteTo(w io.Writer) error`
- [ ] Implement `(f *File) WriteToFile(path string) error`
- [ ] Implement `(f *File) Validate() []Issue`
- [ ] Write API-level round-trip tests

### 2.4 CLI `rewrite` Command

- [ ] Implement `lvrsrc rewrite <file> --out <output>` command
- [ ] Add `--canonical` flag (canonical writer mode, future)
- [ ] Add round-trip integration test using CLI

### 2.5 Round-Trip Test Suite

- [ ] Create `internal/golden` harness for golden file tests
- [ ] Run round-trip on all corpus files; assert byte-for-byte equivalence or structural equivalence
- [ ] Add regression test against Python oracle for block/section counts
- [ ] Document any known acceptable differences (regenerated fields)

---

## Phase 3 — Validator & Diff

> Target: 1–2 weeks | Exit: human + JSON diagnostics; corpus >90% coverage | Tag: `v0.3.0`

### 3.1 Structural Validator (`internal/validate`)

- [ ] Define `Issue` struct (severity, code, message, location)
- [ ] Check: duplicate headers are consistent
- [ ] Check: all offsets are within file bounds
- [ ] Check: section sizes are non-zero and sane
- [ ] Check: block counts match block info table
- [ ] Check: name offsets are valid
- [ ] Check: no overlapping payload regions
- [ ] Check: FourCC values are printable ASCII
- [ ] Implement strict vs. lenient mode
- [ ] Write validator tests (valid files pass; injected-error files fail with expected codes)

### 3.2 CLI `validate` Command

- [ ] Implement `lvrsrc validate <file>` command
- [ ] Human-readable output (colored if TTY)
- [ ] JSON output with `--json` flag and machine-readable exit codes
- [ ] Exit 0: valid; Exit 1: warnings; Exit 2: errors

### 3.3 Diff Engine (`pkg/lvdiff`)

- [ ] Define `Diff` and `DiffItem` structs
- [ ] Implement header-level diff (field-by-field)
- [ ] Implement resource type additions/removals diff
- [ ] Implement section-level binary diff (size changes, content hash)
- [ ] Implement decoded-resource diff for known types (stub, expand in Phase 4+)

### 3.4 CLI `diff` Command

- [ ] Implement `lvrsrc diff <a.vi> <b.vi>` command
- [ ] Human-readable unified-diff style output
- [ ] JSON diff output with `--json` flag

### 3.5 JSON Schema

- [ ] Define JSON schema for `dump` output
- [ ] Define JSON schema for `validate` output
- [ ] Publish schemas under `docs/`

---

## Phase 4 — Safe Metadata Editing

> Target: 2–4 weeks | Exit: targeted metadata edits survive rewrite and validation | Tag: `v0.4.0`

### 4.1 Resource Registry (`internal/codecs`)

- [ ] Define `ResourceCodec` interface (`Decode`, `Encode`, `Validate`, `Capability`)
- [ ] Define `Registry` struct with `map[FourCC]ResourceCodec`
- [ ] Define `Context` struct (`FileVersion`, `Kind`)
- [ ] Define `Capability` struct (`FourCC`, `ReadVersions`, `WriteVersions`, `Safety`)
- [ ] Implement registry lookup and fallback to opaque codec
- [ ] Write registry tests

### 4.2 Version Awareness

- [ ] Define `Version` type and `VersionRange`
- [ ] Define `FileKind` enum
- [ ] Implement `(f *File) DetectVersion() (Version, bool)` in `pkg/lvvi`
- [ ] Wire version context into all codec calls

### 4.3 Initial Typed Codecs (low-risk resources)

- [ ] Research and document VI description resource layout (Markdown spec in `docs/resources/`)
- [ ] Implement codec for VI description / documentation string resource
- [ ] Research and document VI name resource layout
- [ ] Implement codec for VI name resource
- [ ] Research and document version stamp resource layout
- [ ] Implement codec for version stamp resource
- [ ] Add resource-specific validator checks for each codec

### 4.4 `pkg/lvmeta` Editing API

- [ ] Implement `Mutator` struct with `Strict bool`
- [ ] Implement `SetDescription(f *lvrsrc.File, desc string) error`
- [ ] Implement `SetName(f *lvrsrc.File, name string) error`
- [ ] Implement post-edit validation (Tier 2 safety gate)
- [ ] Write mutation tests (edit → write → re-parse → assert field value)

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

- [ ] Define machine-readable coverage manifest (YAML/JSON)
- [ ] Generate coverage report in CI
- [ ] Add badge to README

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
