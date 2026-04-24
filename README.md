# lvrsrc

[![Resource coverage](docs/generated/resource-coverage-badge.svg)](docs/generated/resource-coverage.md)

`lvrsrc` is a pure-Go toolkit for inspecting, validating, diffing, rewriting, and safely editing LabVIEW RSRC/VI files.

The library is the primary deliverable. The CLI is built on top of it and provides the same container-aware workflows for day-to-day use, automation, and corpus analysis.

Want a fast browser preview instead? [Explore the live RSRC demo](https://cwbudde.github.io/lvrsrc/).

## Highlights

- Pure Go parser, validator, writer, and editing stack with no Python runtime dependency
- Support for `.vi`, `.ctl`, `.vit`, and `.llb` RSRC containers
- Structural parsing of headers, block tables, section descriptors, name tables, and opaque tails
- Preserving rewrite pipeline for round-trip-safe edits and normalization
- Typed resource decoding for known metadata resources with version-aware safety checks
- Structural diffing, JSON export, machine-readable validation output, and repair-oriented workflows
- Corpus-backed tests, fuzzing targets, and differential checks against reference tooling during development

## What `lvrsrc` Does

`lvrsrc` is designed around two complementary models:

- A structural RSRC container model that preserves unknown bytes and drives safe read/write behavior
- A semantic model for known LabVIEW resources that enables higher-level inspection and targeted metadata edits

That split keeps the toolkit useful across partially understood file variants without pretending every resource can be mutated safely.

## CLI

The CLI uses Cobra for commands and Viper for config, environment variables, and defaults.

### Common commands

```bash
lvrsrc inspect example.vi
lvrsrc dump example.vi --json
lvrsrc list-resources example.vi
lvrsrc extract example.vi --type BDPW --id 12 --out block.bin
lvrsrc validate example.vi
lvrsrc rewrite example.vi --out rewritten.vi
lvrsrc rewrite example.vi --canonical --out canonical.vi
lvrsrc set-meta example.vi --description "Updated description" --out edited.vi
lvrsrc diff before.vi after.vi
lvrsrc repair damaged.vi --out repaired.vi
```

### Command summary

- `inspect`: file kind, detected version, header summary, block/resource counts, warnings
- `dump`: full structural dump in text or JSON form
- `list-resources`: compact resource listing by type, id, name, and size
- `extract`: raw resource or section extraction for reverse-engineering and analysis
- `validate`: structural and typed validation with human-readable or JSON diagnostics
- `rewrite`: preserving round-trip by default, with optional deterministic canonical layout via `--canonical`
- `set-meta`: safe metadata edits such as name, description, version stamps, and other supported low-risk fields
- `diff`: structural and typed diffs between two files
- `repair`: conservative structural repair for files that already parse leniently, followed by strict post-write validation

### Configuration

`lvrsrc` supports config files, environment variables, and flags for common defaults such as output format, strict validation, logging, unsafe editing, and fixture/corpus locations.

Example config:

```yaml
format: json
strict: true
unsafe: false
log_level: info
golden_fixture_dir: testdata
```

## Go API

Container-level access lives in `pkg/lvrsrc`. Higher-level helpers are split into focused packages:

- `pkg/lvrsrc`: parse, inspect, validate, clone, diff-friendly resource access, write
- `pkg/lvvi`: version detection and typed known-resource decoding
- `pkg/lvmeta`: safe metadata editing helpers
- `pkg/lvdiff`: structural and typed diffs

Example:

```go
package main

import (
	"log"

	"github.com/CWBudde/lvrsrc/pkg/lvmeta"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

func main() {
	f, err := lvrsrc.Open("example.vi", lvrsrc.OpenOptions{Strict: true})
	if err != nil {
		log.Fatal(err)
	}

	if issues := f.Validate(); len(issues) != 0 {
		log.Printf("validation reported %d issue(s)", len(issues))
	}

	model, issues := lvvi.DecodeKnownResources(f)
	_ = model
	_ = issues

	mut := lvmeta.Mutator{Strict: true}
	if err := mut.SetDescription(f, "Updated by lvrsrc"); err != nil {
		log.Fatal(err)
	}

	if err := f.WriteToFile("edited.vi"); err != nil {
		log.Fatal(err)
	}
}
```

## Safety Model

`lvrsrc` separates editing into explicit safety tiers:

- Tier 0: read-only inspection, export, diff, and validation
- Tier 1: opaque-preserving rewrites that keep unknown resource bytes intact
- Tier 2: typed edits on well-understood resources with post-encode validation
- Tier 3: explicit unsafe/raw patching for advanced users and experiments

The default workflow is preserving and conservative. Unknown resources are retained exactly, original ordering is preserved where possible, and rewrites recompute only the container structures that must change. Canonical rewrite is available as an explicit mode for deterministic layout and ordering normalization; it now applies a corpus-guided block order, canonical section ordering within each block, and still preserves opaque `RawTail` bytes for safety.

Repair is narrower than rewrite: it only operates on files that already parse in lenient mode, it does not guess missing names or payloads, and it only succeeds if the written output re-parses strictly with zero validation errors.

## Build And Test

Build and verification follow the same checks used in CI:

```bash
go build ./...
go test ./...
go test ./internal/rsrcwire -run='^$' -fuzz=FuzzParseFile -fuzztime=10s
go test ./internal/rsrcwire -run='^$' -fuzz=FuzzParseHeader -fuzztime=10s
go test ./internal/rsrcwire -run='^$' -fuzz=FuzzNameTable -fuzztime=10s
go vet ./...
golangci-lint run
```

Format code with:

```bash
gofmt -w .
goimports -w .
```

## Project Layout

```text
cmd/lvrsrc/        CLI entrypoint
internal/binaryx/  offset-aware binary reader/writer helpers
internal/rsrcwire/ RSRC container parser and serializer
internal/codecs/   typed resource codecs and registry
internal/validate/ structural and typed validation
internal/golden/   golden corpus and round-trip test harness
pkg/lvrsrc/        public container API
pkg/lvvi/          higher-level VI model and version-aware decoding
pkg/lvmeta/        safe metadata editing helpers
pkg/lvdiff/        diff helpers
docs/              format notes, safety model, resource registry, schemas
testdata/          fixtures and regression corpus
```

## Documentation

- [docs/generated/resource-coverage.md](docs/generated/resource-coverage.md)
- [docs/format-overview.md](docs/format-overview.md)
- [docs/formats/llb.md](docs/formats/llb.md)
- [docs/wire-layout.md](docs/wire-layout.md)
- [docs/resource-registry.md](docs/resource-registry.md)
- [docs/safety-model.md](docs/safety-model.md)
- [docs/contributing-reverse-engineering.md](docs/contributing-reverse-engineering.md)

## Scope

`lvrsrc` is intended to be trustworthy at the container level and selective at the semantic level. It can fully inspect, validate, diff, and rewrite supported RSRC containers, and it can safely edit known low-risk metadata resources. It does not claim universal support for arbitrary semantic mutation of every LabVIEW internal resource.
