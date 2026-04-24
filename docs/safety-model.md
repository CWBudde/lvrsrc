# Safety Model

`lvrsrc` uses editing tiers:

1. **Tier 1 (Read-only):** inspect/dump/validate, no mutations.
2. **Tier 2 (Safe metadata edits):** targeted fields with codec-level invariants.
3. **Tier 3 (Unsafe/raw patches):** explicit opt-in for expert workflows.

Core rule: unknown sections are preserved byte-for-byte unless a mode explicitly
allows canonicalization.

Current canonicalization scope normalizes both layout and ordering
deterministically: corpus-guided block ordering, stable section ordering within
each block, compact referenced-name tables, and 4-byte zero padding. Opaque
`RawTail` bytes are still preserved for now rather than discarded.

Repair is conservative rather than forensic. It only accepts files that already
parse in lenient mode, only applies narrow structural fixes, and refuses any
case that would require guessing missing names or payload bytes.
