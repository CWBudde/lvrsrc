# RSRC Wire Layout (Reference Notes)

This document records the working wire-level model inferred from public Python
implementations and reverse-engineering references.

## Sources reviewed

- `pylabview` (planned local clone under `references/pylabview`)
- `pylavi` (planned local clone under `references/pylavi`)

> Note: Network-restricted environment prevented cloning at scaffold time.
> Add concrete citations and offsets once repositories are available locally.

## Working layout notes

- RSRC files contain a primary header and a duplicate/secondary header.
- A block info table points to typed block headers.
- Block headers index one or more sections.
- Sections hold raw payloads; many are opaque until codec support exists.
- A trailing name table stores Pascal-style names.

These notes intentionally remain conservative until corpus-driven validation is in
place.
