# Repair Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a conservative `lvrsrc repair` command that only operates on files that already parse in lenient mode and repairs a narrow allowlist of structural issues.

**Architecture:** Introduce an internal repair package that analyzes validator issues on a leniently parsed `pkg/lvrsrc.File`, applies only safe structural fixes, then relies on the preserving writer to recompute headers and offsets. The CLI command will write the repaired file and require a strict re-parse plus zero validation errors before claiming success.

**Tech Stack:** Go, `internal/repair`, `pkg/lvrsrc`, Cobra CLI tests.

---

### Task 1: Add failing tests for repair heuristics and CLI behavior

**Files:**
- Create: `internal/repair/repair_test.go`
- Modify: `cmd/lvrsrc/main_test.go`

- [ ] Add unit tests for repairable `header.mismatch`, repairable payload-overlap/offset-drift, and unrepairable truncated-name-table cases.
- [ ] Add CLI tests for `lvrsrc repair <file> --out <file>` on a repairable corrupted fixture.
- [ ] Add CLI refusal coverage for a truncated-name-table fixture that still parses leniently.
- [ ] Run focused tests and confirm they fail before implementation.

### Task 2: Implement conservative repair package

**Files:**
- Create: `internal/repair/repair.go`

- [ ] Define the allowlist of repairable issue codes and explicit refusal rules.
- [ ] Implement name-table rebuilding from referenced `Section.Name` values only; refuse when an invalid name offset has no resolved name.
- [ ] Leave header mismatch and offset drift to the preserving serializer after heuristic checks pass.

### Task 3: Wire the CLI command and post-write safety gate

**Files:**
- Modify: `cmd/lvrsrc/main.go`

- [ ] Add `repair` to the root command set.
- [ ] Open input with `Strict: false`, run the repair package, write the output in preserving mode, and require strict re-parse + zero validation errors.
- [ ] Return clear refusal errors for unsupported or unrepairable cases.

### Task 4: Document and mark Phase 5.5 progress

**Files:**
- Modify: `README.md`
- Modify: `docs/safety-model.md`
- Modify: `PLAN.md`

- [ ] Document the conservative scope: lenient-parse only, preserving writer, no guessed names/payloads.
- [ ] Mark the three Phase 5.5 checklist items complete once tests and CLI ship.

### Task 5: Verify end-to-end

**Files:**
- None

- [ ] Run `gofmt -w` on touched Go files.
- [ ] Run focused tests for `internal/repair` and `cmd/lvrsrc`.
- [ ] Run `go test ./...` as the completion gate.
