# Safety Model

`lvrsrc` uses editing tiers:

1. **Tier 1 (Read-only):** inspect/dump/validate, no mutations.
2. **Tier 2 (Safe metadata edits):** targeted fields with codec-level invariants.
3. **Tier 3 (Unsafe/raw patches):** explicit opt-in for expert workflows.

Core rule: unknown sections are preserved byte-for-byte unless a mode explicitly
allows canonicalization.
