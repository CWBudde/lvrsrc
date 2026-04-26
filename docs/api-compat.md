# API & CLI Compatibility Policy

This document records what is considered the public, supported surface of
`lvrsrc` and the rules for changing it. It is the policy referenced by
PLAN.md Phase 10.3 ("CLI / API surface frozen; any Tier 2 expansions beyond
this phase go through a compat policy update") and Phase 5.6 ("Public API
is stable").

## Scope

The following are the **supported public surface** and are subject to the
compatibility rules below:

- **Go API**
  - `pkg/lvrsrc` — file open / parse / write, header and block models
  - `pkg/lvmeta` — Tier 2 metadata mutators (`STRG`, `vers`, name table)
  - `pkg/lvdiff` — structural and decoded diffing
  - `pkg/lvvi` — high-level VI / control model
- **Codec contracts**
  - `internal/codecs.ResourceCodec`, `Capability`, `Context`,
    `VersionRange`, `Registry`, `FourCC`, the `Safety` enum, and the
    `validate.Issue` / `validate.Severity` types these codecs return.
    Although the package lives under `internal/`, the contract is
    consumed by `pkg/*` and is therefore part of the supported surface
    for downstream embedders that vendor the codecs through one of the
    `pkg/*` entry points.
- **CLI**
  - The `lvrsrc` binary's command set, flag names, flag semantics, exit
    codes (see `cmd/lvrsrc/exit_code.go`), and `--json` output schemas
    (see [`docs/schemas/`](schemas)).
- **Generated artifacts**
  - The shape of `docs/generated/resource-coverage.json` (consumed by the
    coverage badge and dashboards).
- **Codec round-trip invariants**
  - For every typed codec listed in
    [`resource-registry.md`](resource-registry.md): `Encode(Decode(p))`
    must equal `p` byte-for-byte for every payload `p` shipped in
    `testdata/corpus/`. New codec versions may relax this only by
    bumping `Capability().WriteVersions` and documenting the divergence.

The following are **not** part of the supported surface and may change
without a major-version bump:

- Anything in `internal/` that is not transitively re-exported through
  `pkg/*`.
- The wire format of intermediate representations
  (`heap.WalkResult`, `heap.Envelope`, codec `Value` types' unexported
  fields).
- Test helpers, fixtures, fuzz seeds, and the `testdata/` layout.
- The web demo (`web/`) and its WASM glue.
- The `lvrsrcwasm` binary's exported function names.

## What "frozen" means at v1.0

After tagging `v1.0.0` (Phase 5.6 / 10.3 item 5):

- **No removals.** Public types, functions, methods, fields, CLI commands,
  flags, exit codes, and JSON keys may not be removed in a 1.x release.
- **No semantic changes.** Existing methods and CLI flags must keep their
  observable behavior; bug fixes that change observable output count as
  semantic changes and require either a feature flag or a 2.0 bump.
- **Additions are allowed.** New types, new optional fields, new CLI
  subcommands, new optional flags, and new typed codecs may be added in
  a minor (1.y.0) release as long as they don't change defaults for
  existing callers.
- **Tier 2 expansions** (a codec moves from `SafetyTier1` to
  `SafetyTier2`, or a new mutator lands in `pkg/lvmeta`) are
  feature-additive but require:
  1. A round-trip corpus test passing on every shipped fixture that
     uses the codec.
  2. A post-edit `Validate()` gate (no new error- or warning-severity
     issues introduced by the edit).
  3. A `docs/resources/<fourcc>.md` update describing what fields the
     mutator may touch and what stays opaque.
- **JSON-schema additions** must be additive and registered under
  `docs/schemas/`. Existing JSON keys may not change types.

Bug fixes that close round-trip gaps (a codec that was emitting wrong
bytes) count as bug fixes, not breaking changes, even when they alter
output — provided the corpus round-trip suite continues to pass.

## What requires a 2.0 bump

- Removing or renaming a `pkg/*` exported identifier.
- Changing a CLI subcommand, flag name, flag default, exit code, or JSON
  key.
- Tightening a `ResourceCodec.Capability().Safety` tier downward (e.g.
  Tier 2 → Tier 1).
- Removing a typed codec or returning the opaque fallback for a FourCC
  that was previously typed.
- Changing the `internal/codecs.ResourceCodec` interface in a way that
  forces existing implementers to change.

## How changes are proposed

1. Open a PR that includes the plan-document update (PLAN.md or
   `docs/api-compat.md` itself if the policy needs to evolve) alongside
   the implementation.
2. The corpus round-trip suite, fuzzers, and `internal/coverage` tests
   must stay green.
3. If a CLI flag or JSON key is added, also update the matching
   `docs/schemas/*.json` and the generated badge JSON.
4. The PR description must call out which bullet of this document it
   is exercising (additive feature, bug fix, breaking change for 2.0).

## Pre-1.0 caveat

Until the `v1.0.0` tag actually lands, this policy describes the **target
contract** for v1, not a guarantee that already applies to every commit
on `main`. The `v0.x` series may still tighten or rename pre-release
surface area; once `v1.0.0` is tagged the rules above become binding.
