# Contributing Reverse Engineering

When adding new resource knowledge:

1. Start from corpus evidence (multiple versions if possible).
2. Document hypotheses in markdown before implementing encoders.
3. Add parser-only support first; avoid writes until invariants are known.
4. Include regression fixtures and validator checks.
5. Record safety tier and version range in the registry.
