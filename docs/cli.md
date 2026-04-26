# `lvrsrc` CLI

`lvrsrc` exposes the same parser, validator, rewrite pipeline, and typed
resource decoding that the Go library uses, but packages them into
container-aware commands for day-to-day inspection and automation.

## Global flags

Every command inherits the root flags from `cmd/lvrsrc/main.go`:

- `--config <path>`: read defaults from a Viper config file
- `--format <text|json>`: select output formatting where the command
  supports multiple forms
- `--strict`: open and validate with strict parsing rules
- `--log-level <level>`: reserved verbosity control
- `--out <path>`: write command output to a file instead of stdout

Environment variables use the `LVRSRC_` prefix with dashes rewritten to
underscores, for example `LVRSRC_FORMAT=json`.

## Inspection and validation

### `inspect <file>`

Print the detected file kind, header summary, and block inventory.

```bash
lvrsrc inspect example.vi
```

### `dump <file>`

Emit the parsed RSRC container in text form or JSON.

```bash
lvrsrc dump example.vi
lvrsrc dump example.vi --json
```

### `list-resources <file>`

List every resource section with its FourCC, id, name, and payload size.

```bash
lvrsrc list-resources example.vi
```

### `validate <file>`

Run structural validation and return machine-readable exit codes:

- `0`: valid
- `1`: warnings only
- `2`: at least one error

```bash
lvrsrc validate example.vi
lvrsrc validate example.vi --json
```

## Rewrite, repair, and metadata edits

### `rewrite <file>`

Write the file back out through the preserving serializer. `--out` is
required.

```bash
lvrsrc rewrite example.vi --out rewritten.vi
lvrsrc rewrite example.vi --canonical --out canonical.vi
```

`--canonical` enables deterministic canonical layout; without it, the
default preserving path keeps parsed ordering and opaque bytes wherever
possible.

### `repair <file>`

Open leniently, apply the narrow structural repair allowlist, write the
result, then require a strict re-parse plus zero validation errors.

```bash
lvrsrc repair damaged.vi --out repaired.vi
```

### `set-meta <file>`

Apply Tier 2 safe metadata edits such as the VI name or description.
`--out` is required.

```bash
lvrsrc set-meta example.vi --description "Updated by lvrsrc" --out edited.vi
lvrsrc set-meta example.vi --name renamed.vi --out renamed.vi
```

## Diff and render

### `diff <a> <b>`

Compare two RSRC files structurally and, where codecs exist, semantically.

```bash
lvrsrc diff before.vi after.vi
lvrsrc diff before.vi after.vi --json
```

### `render <file>`

Emit an approximate render of the decoded front panel or block diagram as
standalone SVG.

```bash
lvrsrc render example.vi --view front-panel > panel.svg
lvrsrc render example.vi --view block-diagram --out diagram.svg
```

Supported render flags:

- `--view front-panel|block-diagram`
- `--format svg`

Current render semantics:

- The output is driven by the shared `internal/render` scene graph used by
  the web demo; CLI and browser renders use the same logical bounds,
  placeholder handling, and warning model.
- The SVG is structural and heuristic, not geometry-faithful. Bounds are
  inferred from heap structure, not decoded LabVIEW coordinates.
- Unresolved object classes stay visible as placeholder nodes with dashed
  styling instead of being omitted.
- Block-diagram wire routing and terminal placement are not rendered yet.

## Verification notes

The render and validation commands are covered by repository tests:

- `go test ./cmd/lvrsrc`
- `go test ./internal/render`
- `go test ./web`

The web smoke test in `web/app_smoke_test.mjs` exercises the heap-tab mode
switches (`Visual`, `Canvas`, `Tree`) against a synthetic parse payload to
ensure the demo wiring does not panic when both `FPHb` and `BDHb` views are
present.
