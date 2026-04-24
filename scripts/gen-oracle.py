#!/usr/bin/env python3
"""Generate pylabview-based oracle baselines for the lvrsrc test corpus.

For every file in testdata/corpus and testdata/llb the script records the
block/section inventory that pylabview observes and writes it to
testdata/oracle/<relative-path>.json. Those JSON files are committed to the
repository and consumed by internal/oracle's Go tests, which lets CI run
without needing a Python environment.

Usage:

    python3 scripts/gen-oracle.py

Re-run this whenever the corpus changes or pylabview is updated. Expects
references/pylabview to be present (git-cloned sibling).
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import warnings
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
PYLABVIEW_ROOT = REPO_ROOT / "references" / "pylabview"
CORPUS_DIRS = [REPO_ROOT / "testdata" / "corpus", REPO_ROOT / "testdata" / "llb"]
ORACLE_ROOT = REPO_ROOT / "testdata" / "oracle"
CORPUS_EXTS = {".vi", ".ctl", ".vit", ".llb"}


def build_options(rsrc_path: Path) -> argparse.Namespace:
    """Construct the minimal argparse.Namespace that pylabview's VI expects."""
    return argparse.Namespace(
        verbose=0,
        textcp="mac_roman",
        xml="",
        rsrc=str(rsrc_path),
        filebase=rsrc_path.stem,
        raw_connectors=False,
        print_map=None,
        keep_names=False,
        password=None,
        typedesc_list_limit=4095,
        array_data_limit=(2**28) - 1,
        store_as_data_above=4095,
    )


def load_pylabview():
    if not PYLABVIEW_ROOT.exists():
        raise SystemExit(
            f"pylabview not found at {PYLABVIEW_ROOT}; clone it first"
        )
    sys.path.insert(0, str(PYLABVIEW_ROOT))
    with warnings.catch_warnings():
        warnings.simplefilter("ignore")
        from pylabview.LVrsrcontainer import VI  # noqa: WPS433
    return VI


def oracle_for_file(VI, rsrc_path: Path) -> dict:
    po = build_options(rsrc_path)
    with warnings.catch_warnings():
        warnings.simplefilter("ignore")
        with open(rsrc_path, "rb") as fh:
            vi = VI(po, rsrc_fh=fh, text_encoding=po.textcp)

    blocks: list[dict] = []
    for _ident, block in vi.blocks.items():
        fourcc = block.ident.decode("utf-8", errors="replace")
        blocks.append({"fourcc": fourcc, "sections": len(block.sections)})

    blocks.sort(key=lambda entry: entry["fourcc"])

    return {
        "oracle": "pylabview",
        "source_path": str(rsrc_path.relative_to(REPO_ROOT)).replace("\\", "/"),
        "fmt_version": int(getattr(vi, "fmtver", 0)),
        "block_count": len(blocks),
        "blocks": blocks,
    }


def iter_corpus():
    for base in CORPUS_DIRS:
        if not base.exists():
            continue
        for path in sorted(base.rglob("*")):
            if path.is_file() and path.suffix.lower() in CORPUS_EXTS:
                yield path


def main() -> int:
    VI = load_pylabview()

    ORACLE_ROOT.mkdir(parents=True, exist_ok=True)

    written = 0
    errors: list[tuple[Path, Exception]] = []

    for rsrc_path in iter_corpus():
        rel = rsrc_path.relative_to(REPO_ROOT)
        out_path = ORACLE_ROOT / rel.relative_to("testdata").with_suffix(
            rel.suffix + ".json"
        )
        out_path.parent.mkdir(parents=True, exist_ok=True)
        try:
            oracle = oracle_for_file(VI, rsrc_path)
        except Exception as exc:  # noqa: BLE001 - record and continue
            errors.append((rsrc_path, exc))
            print(f"WARN: {rel}: {exc}", file=sys.stderr)
            continue

        out_path.write_text(
            json.dumps(oracle, indent=2, sort_keys=True) + "\n",
            encoding="utf-8",
        )
        written += 1
        print(f"wrote {out_path.relative_to(REPO_ROOT)}")

    print(f"\nTotal oracles written: {written}")
    if errors:
        print(f"Failures: {len(errors)}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
