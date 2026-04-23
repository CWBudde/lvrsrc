# Block-Diagram Metadata Codec: `LIbd`

This note captures the current implementation state for the Phase 5.2
block-diagram metadata codec.

## Status

`LIbd` is now a shipped typed codec. It mirrors the narrow `LIfp` strategy:
decode the stable entry envelope, require the `BDHP` marker, and preserve the
unknown bytes between the primary and secondary path refs exactly.

Safety tier: Tier 1.

Implementation package: `internal/codecs/libd`.

## Corpus Evidence

From the current `testdata/corpus/` set:

- `LIbd` appears in 21/21 fixtures
- payload sizes observed: `12`, `98`, `201`, and `336`
- the dominant 12-byte form is exactly:
  `00 01 42 44 48 50 00 00 00 00 00 03`
- larger forms contain `PTH0` path refs plus Pascal-style names such as
  `.ctl`, `.vi`, and `.lvlib`

The shared outer structure with `LIfp` is clear in the corpus:

- `u16` version, observed as `1`
- 4-byte marker `BDHP`
- `u32` entry count
- repeated entries
- `u16` footer, observed as `3`

## Current Decoded Shape

The shipped model exposes:

- `Version uint16`
- `Marker string` (`"BDHP"`)
- `EntryCount uint32`
- `Entries []Entry`
- `Footer uint16`

Each `Entry` exposes:

- `Kind uint16`
- `LinkType string`
- `QualifierCount uint32`
- `Qualifiers []string`
- `PrimaryPath PathRef`
- `Tail []byte`
- optional `SecondaryPath *PathRef`

`Tail` remains opaque by design. The corpus supports the envelope and path
boundaries, but not a stable semantic interpretation of the bytes in between.

## Validation Scope

The codec validates only invariants supported by the corpus:

- minimum payload length
- required `BDHP` marker
- successful parse of the repeated entry envelope

Deeper semantic validation of the opaque tail bytes is intentionally out of
scope for Tier 1.

## Relationship To `LIfp`

`LIbd` and `LIfp` clearly share the same family of link-info structure, with
resource-specific markers:

- `LIfp` uses `FPHP`
- `LIbd` uses `BDHP`

The current implementation keeps them as separate codecs with the same narrow
strategy. If more link-info resources appear later, that shared parser logic is
the obvious consolidation point.
