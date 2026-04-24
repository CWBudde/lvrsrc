# scripts/

Developer scripts that live outside the Go build. Do not import from these.

## `gen-oracle.py`

Generates pylabview-based oracle baselines for the lvrsrc test corpus, one
JSON file per corpus entry under `testdata/oracle/`. The committed JSON is
consumed by `internal/oracle`'s Go tests so CI stays hermetic (Go-only) and
does not need Python.

Prerequisites:

- Python 3.10+ (script uses PEP 604 union type syntax).
- `Pillow` installed (`pip install Pillow`) — required indirectly by
  `pylabview`.
- `references/pylabview/` must be populated; the script imports directly
  from that path without installing the package.

Refresh flow:

```sh
python3 scripts/gen-oracle.py
```

Re-run whenever the corpus in `testdata/corpus/` or `testdata/llb/` changes,
or after updating `references/pylabview`. Review the resulting JSON diff
before committing — divergence is the signal this oracle is designed to
catch.

What the oracle records (intentionally minimal):

- Top-level: `oracle` tag (`"pylabview"`), `source_path`, `fmt_version`,
  `block_count`.
- `blocks[]`: for each block pylabview surfaced, the FourCC and the count
  of sections inside it. Blocks are sorted by FourCC so diffs stay stable.

What it does *not* record: decoded payloads, byte offsets, or anything
downstream of pylabview's section parsing (pylabview emits warnings on some
VITS sections; those warnings do not affect the block/section inventory).

## Troubleshooting

- `ImportError: pylabview` — clone submodule/reference: `git clone
  https://github.com/mefistotelis/pylabview references/pylabview`.
- `ModuleNotFoundError: PIL` — `pip install Pillow`.
- Spurious `VITS` warnings on newer LV versions are expected and do not
  invalidate the generated oracle.
