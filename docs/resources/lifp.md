# Front-Panel Metadata Codec: `LIfp`

This note captures the current implementation state for the Phase 5.2
front-panel metadata codec.

## Status

`LIfp` is now the shipped front-panel metadata codec. It remains intentionally
narrow: it decodes the stable entry envelope observed in the corpus and
preserves the still-unknown inner bytes exactly on re-encode.

Safety tier: Tier 1.

Implementation package: `internal/codecs/lifp`.

## Recommendation

`LIfp` was the right first target, not `FPHb`.

`LIfp` is small, repeated, and structurally consistent across the current
corpus. `FPHb` is the actual front-panel heap, but it is a much larger tagged
object graph and is not yet narrow enough for a first safe typed codec.

## Corpus Evidence

From the current `testdata/corpus/` set:

- `LIfp` appears in 21/21 fixtures
- payload sizes observed: `12` (14 fixtures), `98` (2), `100` (1), `201` (3),
  `336` (1)
- the dominant 12-byte form is exactly:
  `00 01 46 50 48 50 00 00 00 00 00 03`
- larger forms contain the same leading pattern plus embedded `PTH0` path
  records and Pascal-style names such as `.ctl` / `.lvlib`

Observed examples in this corpus strongly suggest that `LIfp` is a
front-panel link/import list:

- a leading big-endian count-like field (`0x0001` in all current samples)
- an `FPHP` marker
- a repeated path/name structure in larger payloads

By contrast, the neighboring front-panel resources show much weaker codec
readiness:

- `FPHb`: 20 distinct sizes in 21 fixtures, hundreds to thousands of bytes,
  clearly a heap/object graph
- `FPEx`: mostly 4-byte zero payloads, with a few 8-byte and 16-byte outliers
- `FPSE`: always 4 bytes, but many different values and unclear semantics
- `VPDP`: always `0x00000000` in this corpus, too little information to justify
  a standalone semantic codec

## Upstream Reverse-Engineering Evidence

The `pylabview` source separates front-panel concerns into two different areas:

- `LVheap.py` documents the front-panel heap family (`FPHT`, `FPHX`, `FPHB`,
  `FPHb`, `FPHc`) as heap/object-graph storage
- `LVlinkinfo.py` implements generic link-save structures built from qualified
  names, `PTH0`/`PTH1` path refs, and version-aware link metadata

That split matches the local corpus:

- `FPHb` behaves like a heap payload
- `LIfp` behaves like link metadata, not heap state

This is the main reason `LIfp` is the safer Phase 5.2 target.

## Codec Scope

The current `LIfp` codec stays narrow:

1. Parse the common header fields that are stable in the corpus.
2. Detect and expose the `FPHP` marker.
3. Decode embedded `PTH0` references and attached names when present.
4. Preserve unknown trailing/sub-record bytes exactly on re-encode.
5. Validate only the invariants the corpus actually supports.

The codec should not attempt to decode `FPHb` heap internals or reconstruct
front-panel object graphs.

## Current Decoded Shape

The shipped model exposes:

- `Version uint16`
- `Marker string` (`"FPHP"`)
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

`Tail` is deliberately opaque. The corpus shows that the bytes between the
primary and secondary path refs are not yet stable enough to claim semantic
field meanings across all fixtures, so the codec preserves them byte-for-byte.

## Blockers / Open Questions

- Whether the leading `uint16` is always an entry count or a versioned tag
- The exact sub-record grammar after the `FPHP` marker in the 98/100/201/336 B
  forms
- Whether any non-`PTH0` path class appears in a broader corpus
- Whether `LIfp` and `LIbd` share a single generic link-info codec with
  resource-specific markers

## Remaining Work

Recommended order for the remaining Phase 5.2 work:

1. Reuse the same parsing strategy for `LIbd` if the block-diagram payloads
   match the same link-info pattern.
2. Leave `FPHb` for a later heap/object-graph phase.
