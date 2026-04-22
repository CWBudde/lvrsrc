## 1. Product goal

Build a repository where:

* the **main deliverable is a Go library** for parsing, editing, validating, and writing `.vi` files,
* the **CLI is secondary** and uses **Cobra** for commands and **Viper** for config and environment integration,
* the implementation is **fully standalone at runtime** and **pure Go**,
* Python projects are used only as **spec/reference material**, not as dependencies. Cobra is a standard Go CLI framework, and Viper is explicitly designed to pair well with Cobra. ([GitHub][2])

## 2. Scope decision: what “write VI files” should mean

This is the most important design choice.

A LabVIEW VI is not just “a binary blob with metadata.” The file contains many internal typed resources, version-sensitive structures, and relationships between sections. NI has even documented security fixes related to incomplete RSRC validation in VI files, which is a strong signal that malformed writes can crash or at least break consumers. ([NI][3])

So define writing in **three levels**:

### Level 1 — byte-preserving structural round-trip

Open a `.vi`, parse the RSRC container, and write it back **bit-for-bit equivalent where possible** or semantically identical where regenerated fields differ. This is the first milestone and the foundation for trustworthiness. Public reverse-engineering references describe the global RSRC layout, including duplicated headers, block info, block headers, section info, and trailing names. ([GitHub][4])

### Level 2 — safe metadata edits

Allow edits to low-risk, well-understood resources first:

* names
* descriptions
* documentation strings
* selected flags
* version stamps
* non-executable metadata

NI’s docs confirm that VIs contain documentation fields and descriptive metadata users interact with. ([NI][5])

### Level 3 — semantic VI mutation

Editing front panel, block diagram, compiled code, type maps, connector pane, links, library membership, etc.

This is the hardest part and should be treated as a **long-term track**, not part of MVP. The reverse-engineered Python projects suggest the ecosystem is still incomplete and specialized rather than “fully solved.” ([GitHub][1])

## 3. Recommended MVP

Your MVP should be:

* **Read** `.vi`, `.ctl`, optionally `.vit` and `.llb` later
* **Parse and expose** RSRC container structure
* **List and inspect** resources and sections
* **Validate** structural integrity
* **Write back** unchanged files
* **Support selective metadata edits**
* **Emit JSON** views for inspection/debugging
* **Provide diff and repair-ish tooling** at the structural level

Do **not** promise full semantic editing of arbitrary VI internals in v1.

## 4. Architecture

Repository shape:

```text
labview-go/
  cmd/lvrsrc/
  internal/
    binaryx/
    rsrcwire/
    codecs/
    validate/
    golden/
  pkg/
    lvrsrc/        // public API for container-level read/write
    lvvi/          // public API for VI-oriented abstractions
    lvmeta/        // public API for common metadata editing
    lvdiff/        // diff helpers
  testdata/
  docs/
```

### Public library packages

#### `pkg/lvrsrc`

Low-level container API.

Responsibilities:

* detect file kind
* parse file header(s)
* parse resource/block tables
* parse section descriptors
* preserve unknown bytes exactly
* write container back out

Core types:

```go
type File struct {
    Header1 Header
    Header2 Header
    Blocks  []Block
    Names   []string
    RawTail []byte
}

type Header struct {
    // exact fields based on reverse-engineered wire layout
}

type Block struct {
    Type       FourCC
    ID         int32
    Name       string
    Sections   []Section
    Attributes BlockAttributes
}

type Section struct {
    Index       int
    Offset      uint32
    Size        uint32
    Data        []byte
    Compression CompressionKind
}
```

#### `pkg/lvvi`

VI-oriented higher-level model.

Responsibilities:

* identify known VI resource types
* decode known resources into typed Go structs
* expose convenience methods like `Version()`, `SetDescription()`, `ListResources()`

#### `pkg/lvmeta`

Focused editing helpers for low-risk changes.

Examples:

* set VI description
* set icon label text if resource is understood
* update custom metadata tags
* rename resource names where supported

#### `pkg/lvdiff`

Structural diffing between two VIs:

* header differences
* resource type additions/removals
* section-level binary diffs
* decoded known-resource diffs

## 5. Internal implementation layers

### Layer A — wire-level binary reader/writer

Create a disciplined binary package:

```go
type Reader struct {
    r   io.ReaderAt
    ord binary.ByteOrder
}

func (r *Reader) U16(off int64) (uint16, error)
func (r *Reader) U32(off int64) (uint32, error)
func (r *Reader) Bytes(off int64, n int) ([]byte, error)
func (r *Reader) PascalString(off int64) (string, int, error)
```

Same for a writer with offset patching. The reason is that the format is offset-heavy and version-sensitive. Both `pylavi` and `pylabview` indicate this is fundamentally a structured resource container, so offset correctness is central. ([GitHub][1])

### Layer B — RSRC container codec

Implement exact parsing and reserialization of:

* primary header
* duplicated header
* block info list
* block headers
* block section info
* section payloads
* name tables / trailing Pascal strings

Start from the RSRC structure descriptions in the `pylabview` wiki and the conceptual file model in `pylavi`. ([GitHub][4])

### Layer C — resource registry

Define a registry for known block/resource types:

```go
type ResourceCodec interface {
    Decode([]byte, Context) (any, error)
    Encode(any, Context) ([]byte, error)
    Validate(any, Context) []Issue
}

type Registry struct {
    codecs map[FourCC]ResourceCodec
}
```

This lets you:

* preserve unknown resources losslessly,
* decode known resources selectively,
* evolve coverage incrementally.

### Layer D — typed resource decoders

Implement decoders only for resources you truly understand. Everything else stays opaque.

This is the only sustainable way to keep the writer safe.

### Layer E — validator

Validator should check:

* duplicate headers consistent
* offsets inside bounds
* section sizes sane
* block counts match tables
* name offsets valid
* no overlapping payload regions unless explicitly allowed
* resource-specific invariants for known types

Given NI’s published note about incomplete RSRC validation leading to crashes, strong validation is not optional. ([NI][3])

## 6. File format strategy

Use a **two-model approach**:

### Structural model

Represents the file exactly as an RSRC container.

This is the source of truth for reading and writing.

### Semantic model

Derived views for known resource types and VI concepts.

This is optional and partial.

That separation is important because full semantic knowledge of VIs is incomplete in public sources, while the container shape is much better understood. `pylavi` explicitly frames the format as a solid file-format API first, then richer resource-specific work later. ([PyPI][6])

## 7. Writing strategy

There are really two writers you need.

### Writer 1 — preserving writer

For unchanged and partially changed files.

Rules:

* preserve exact section bytes for unknown resources
* preserve original ordering
* preserve original padding/alignment where possible
* rewrite only tables/offsets impacted by edits
* regenerate both headers consistently

This enables round-trip stability.

### Writer 2 — canonical writer

For files built or normalized by your library.

Rules:

* canonical ordering of blocks and sections
* canonical padding/alignment policy
* recomputed offsets
* deterministic serialization

This is useful for tests, generated fixtures, and future repair tooling.

## 8. Editing policy

Support edits using explicit safety tiers.

### Tier 0 — read-only

No changes; parse/validate/export only.

### Tier 1 — opaque-preserving edits

Update container metadata without touching unknown section bytes.

### Tier 2 — typed edits on well-understood resources

Only allowed if a typed codec exists and passes post-encode validation.

### Tier 3 — unsafe/raw patching

Expose only as an advanced API, off by default.

This prevents the library from pretending it can safely mutate every VI concept before it truly can.

## 9. CLI plan with Cobra/Viper

Cobra for command structure, Viper for config file, env vars, and defaults is the right stack here. Both projects explicitly describe working well together. ([GitHub][2])

CLI name example: `lvrsrc`

### Commands

#### `lvrsrc inspect file.vi`

Print summary:

* file kind
* version if detected
* header info
* blocks/resources
* section counts
* warnings

#### `lvrsrc dump file.vi --json`

Emit structural JSON.

#### `lvrsrc list-resources file.vi`

Compact list of resource types, ids, names, sizes.

#### `lvrsrc extract file.vi --type XXXX --id 12 --out data.bin`

Extract raw section/resource bytes.

#### `lvrsrc validate file.vi`

Structural and typed validation with machine-readable exit codes.

#### `lvrsrc rewrite file.vi --out rewritten.vi`

Round-trip the file through the library.

#### `lvrsrc set-meta file.vi --description "..." --out out.vi`

Safe metadata mutations only.

#### `lvrsrc diff a.vi b.vi`

Structural and typed diff.

#### `lvrsrc repair file.vi --out repaired.vi`

Later-phase command; only after validation and rewrite are mature.

### Viper config

Support:

* config file path
* default output format
* strict validation mode
* unsafe editing toggle
* logging level
* golden fixture directory

Example:

```yaml
format: json
strict: true
unsafe: false
log_level: info
```

## 10. API sketch

```go
package lvrsrc

type OpenOptions struct {
    Strict bool
}

func Open(path string, opts OpenOptions) (*File, error)
func Parse(data []byte, opts OpenOptions) (*File, error)
func (f *File) Validate() []Issue
func (f *File) WriteTo(w io.Writer) error
func (f *File) Clone() *File
func (f *File) Resources() []ResourceRef
```

```go
package lvmeta

type Mutator struct {
    Strict bool
}

func (m Mutator) SetDescription(f *lvrsrc.File, desc string) error
func (m Mutator) SetName(f *lvrsrc.File, name string) error
```

```go
package lvvi

func DetectVersion(f *lvrsrc.File) (Version, bool)
func DecodeKnownResources(f *lvrsrc.File) (*Model, []Issue)
```

## 11. Testing strategy

This project will live or die by its tests.

### Golden corpus

Collect a wide sample set:

* different LabVIEW versions
* simple VIs
* controls
* templates
* passworded/corrupt examples if legally/operationally acceptable
* files with unusual names/resources

Sources should include your own generated sample files plus public examples where licensing allows.

### Test categories

#### Parse tests

Known files parse without panic and produce expected counts/structures.

#### Round-trip tests

`read -> write -> read` must preserve structure and, when possible, bytes.

#### Differential tests against Python references

Use `pylabview` and `pylavi` out of band during development to compare:

* detected resource types
* table counts
* selected decoded fields

Those projects should be used as **development or CI oracle references**, not runtime deps. That still satisfies your “standalone pure Go” requirement.

#### Fuzzing

Use Go fuzzing on:

* header parser
* section table parser
* name table parser
* whole-file parser with size guards

NI’s security note is a strong reason to budget for this early. ([NI][3])

#### Property tests

* serialize(parse(x)) is valid
* parse(serialize(model)) is equivalent
* unchanged opaque resources survive editing untouched

## 12. Reverse-engineering workflow

Since public documentation is incomplete, create a formal process.

### Spec source hierarchy

1. observed binary corpus
2. `pylabview` wiki and code
3. `pylavi` docs and code
4. LabVIEW behavior on open/save/repair
5. NI public docs only for high-level validation and VI concepts

That is important because NI does not publish a full VI binary spec publicly, while the Python projects do publish reverse-engineered structure information. ([GitHub][1])

### Discovery method

For each unknown resource type:

* identify it across multiple VI samples
* cluster by version
* diff before/after a single LabVIEW GUI edit
* infer field meanings from controlled mutations
* document as Markdown spec in-repo before implementing codec

That keeps the codebase from turning into folklore.

## 13. Milestone plan

## Phase 0 — research and corpus

Duration: 1–2 weeks

Deliverables:

* corpus of sample `.vi` files
* normalized notes from `pylabview` and `pylavi`
* repo skeleton
* architecture doc
* CI setup

Exit criteria:

* at least 20 diverse sample files
* list of known resource types observed
* exact MVP scope approved

## Phase 1 — container parser

Duration: 2–4 weeks

Deliverables:

* binary reader
* header parser
* block table parser
* section parser
* name parser
* JSON dump command

Exit criteria:

* `inspect`, `dump`, `list-resources` work on corpus
* no panics
* fuzz baseline in CI

## Phase 2 — preserving writer

Duration: 2–4 weeks

Deliverables:

* serializer
* offset/padding recomputation
* rewrite command
* round-trip tests

Exit criteria:

* corpus round-trips successfully
* unchanged opaque sections preserved
* rewritten files reopen in parser and pass validation

## Phase 3 — validator and diff

Duration: 1–2 weeks

Deliverables:

* structural validator
* diff engine
* CLI commands `validate` and `diff`

Exit criteria:

* human-readable diagnostics
* machine-readable JSON diagnostics
* corpus coverage >90% of common paths

## Phase 4 — safe metadata editing

Duration: 2–4 weeks

Deliverables:

* initial typed codecs for low-risk resources
* `set-meta` command
* post-edit validation

Exit criteria:

* targeted metadata edits survive rewrite
* edited files remain structurally valid
* verified against sample-open behavior where possible

## Phase 5 — typed resource expansion

Duration: ongoing

Deliverables:

* more decoders/encoders
* better VI semantic model
* eventually repair/normalize features

Exit criteria:

* resource coverage dashboard
* documented support matrix by resource type and LabVIEW version

## 14. Versioning and compatibility

You should assume that VI internals vary by LabVIEW version. `pylavi` and other reverse-engineered projects are oriented around handling the resource format first because higher-level semantics are version-sensitive and incomplete. ([PyPI][6])

So add version-awareness from day one:

```go
type Context struct {
    FileVersion Version
    Kind        FileKind
}
```

Every typed codec should declare:

```go
type Capability struct {
    FourCC       FourCC
    ReadVersions []VersionRange
    WriteVersions []VersionRange
    Safety       SafetyTier
}
```

Then publish a compatibility table like:

| Resource | Read      | Write    | Notes       |
| -------- | --------- | -------- | ----------- |
| XXXX     | 8.6–2020  | 8.6–2019 | stable      |
| YYYY     | 2019–2024 | none     | decode only |

## 15. Documentation plan

Add these docs early:

* `docs/format-overview.md`
* `docs/wire-layout.md`
* `docs/resource-registry.md`
* `docs/safety-model.md`
* `docs/cli.md`
* `docs/contributing-reverse-engineering.md`

For every known resource type:

* binary layout
* field table
* examples
* version caveats
* whether write support exists

## 16. Risks

### Biggest risk: “write” means more than the public reverse-engineering currently supports

Mitigation:

* ship preserving writer first
* only enable typed writes for proven resources
* make support matrix explicit

### Biggest engineering risk: silent corruption

Mitigation:

* opaque preservation
* layered validation
* round-trip and fuzz testing
* deterministic serializer

### Biggest product risk: users expect full VI editing

Mitigation:

* market the project as:

  * structural RSRC toolkit first
  * safe metadata editor second
  * semantic VI editor only where implemented

## 17. Recommended repo roadmap

Initial tags:

* `v0.1.0`: parse + inspect
* `v0.2.0`: rewrite + validate
* `v0.3.0`: diff + JSON schema
* `v0.4.0`: metadata editing
* `v0.5.x+`: typed resource growth

Do not call `v1.0.0` until:

* round-trip corpus is broad,
* validator is mature,
* support matrix is published,
* unsafe APIs are clearly separated.

## 18. Practical recommendation

If this were my project, I would explicitly design it as:

**“A pure-Go RSRC/VI toolkit with strong round-trip guarantees, partial semantic decoding, and carefully scoped write support.”**

That is ambitious but credible.

I would **not** position v1 as “a full pure-Go LabVIEW VI editor,” because the public state of knowledge from `pylabview` and `pylavi` supports a strong container-level implementation, but not a blanket claim of safe arbitrary semantic VI mutation. ([GitHub][1])

[1]: https://github.com/mefistotelis/pylabview?utm_source=chatgpt.com "GitHub - mefistotelis/pylabview: Python reader of LabVIEW RSRC files ..."
[2]: https://github.com/spf13/viper?utm_source=chatgpt.com "GitHub - spf13/viper: Go configuration with fangs"
[3]: https://www.ni.com/en/support/documentation/supplemental/17/incomplete-rsrc-validation-in-labview.html?utm_source=chatgpt.com "Incomplete RSRC Validation in LabVIEW - NI"
[4]: https://github.com/marcpage/pylavi/blob/main/docs/file.md?utm_source=chatgpt.com "pylavi/docs/file.md at main · marcpage/pylavi · GitHub"
[5]: https://www.ni.com/docs/en-US/bundle/labview/page/documenting-vis.html?utm_source=chatgpt.com "Documenting VIs - NI"
[6]: https://pypi.org/project/pylavi/?utm_source=chatgpt.com "pylavi · PyPI"
