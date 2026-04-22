# Repository Guidelines

## Project Structure & Module Organization

`lvrsrc` is a Go module for parsing and validating LabVIEW RSRC/VI data. Keep executable entrypoints under `cmd/lvrsrc/` as that CLI fills out. Put implementation details in `internal/` packages such as `internal/binaryx` and `internal/rsrcwire`; use `pkg/` only for APIs intended for external consumers (`pkg/lvrsrc`, `pkg/lvmeta`, `pkg/lvdiff`, `pkg/lvvi`). Store sample inputs and regression fixtures in `testdata/`. Repository documentation and reverse-engineering notes live in `docs/`.

## Build, Test, and Development Commands

Use the same checks locally that CI runs in `.github/workflows/ci.yml`.

- `go build ./...` builds all current packages.
- `go test ./...` runs the unit test suite.
- `go test ./internal/rsrcwire -run='^$' -fuzz=FuzzParseFile -fuzztime=10s` runs the fuzz smoke target used in CI.
- `go vet ./...` catches suspicious Go usage.
- `gofmt -w .` and `goimports -w <file>` apply formatting and import cleanup.
- `golangci-lint run` runs the configured linters (`govet`, `staticcheck`, `errcheck`, `ineffassign`, `gofmt`, `goimports`).

## Coding Style & Naming Conventions

Follow standard Go formatting: tabs for indentation, `gofmt` layout, and `goimports` import grouping. Keep package names short and lowercase (`binaryx`, `rsrcwire`). Exported identifiers use `CamelCase`; unexported helpers use `camelCase`. Prefer small parsing helpers with explicit bounds checks and offset-aware error messages, matching `internal/binaryx/reader.go`.

## Testing Guidelines

Write table-driven tests in `*_test.go` files next to the code they cover. Name tests by behavior, for example `TestReaderBounds` or `FuzzParseFile`. Add regression fixtures under `testdata/` when a parser bug depends on real bytes. There is no stated coverage gate yet, but new parsing logic should land with unit tests and, when appropriate, fuzz seeds or validator checks.

## Commit & Pull Request Guidelines

Recent history mixes plain imperative subjects and conventional prefixes (`feat: added PLAN.md`). Prefer short, imperative commit titles and keep each commit focused. Pull requests should summarize the affected format area, list validation run locally, and link any issue or planning doc. When behavior changes depend on corpus evidence, update the relevant docs in `docs/` and note any new fixtures or safety assumptions for reviewers.
