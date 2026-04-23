# Front-Panel Metadata Candidate: `LIfp`

This note captures the current research state for Phase 5.2 front-panel
metadata work.

## Recommendation

The next front-panel metadata codec should target `LIfp`, not `FPHb`.

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

## Proposed Scope For The Codec

The first `LIfp` codec should stay narrow:

1. Parse the common header fields that are stable in the corpus.
2. Detect and expose the `FPHP` marker.
3. Decode embedded `PTH0` references and attached names when present.
4. Preserve unknown trailing/sub-record bytes exactly on re-encode.
5. Validate only the invariants the corpus actually supports.

The codec should not attempt to decode `FPHb` heap internals or reconstruct
front-panel object graphs.

## Suggested Decoded Shape

A practical first decoded model would look like:

- `EntryCount uint16`
- `Marker string` (`"FPHP"`)
- `Entries []Entry`

Each `Entry` would likely need:

- optional class/kind marker
- optional name bytes / decoded text
- optional `PTH0` path reference
- opaque remainder bytes for fields not yet understood

That model is intentionally conservative: structure where the corpus is clear,
opaque preservation where it is not.

## Blockers / Open Questions

- Whether the leading `uint16` is always an entry count or a versioned tag
- The exact sub-record grammar after the `FPHP` marker in the 98/100/201/336 B
  forms
- Whether any non-`PTH0` path class appears in a broader corpus
- Whether `LIfp` and `LIbd` share a single generic link-info codec with
  resource-specific markers

## Implementation Order

Recommended order for the remaining Phase 5.2 work:

1. Implement `LIfp` as the first front-panel metadata codec.
2. Reuse the same parsing strategy for `LIbd` if the block-diagram payloads
   match the same link-info pattern.
3. Leave `FPHb` for a later heap/object-graph phase.
