# `LVSR` — LabVIEW Save Record

**FourCC:** `LVSR`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified against 21 corpus
LVSR sections (one per `.vi` / `.ctl` / `.vit` fixture).

The `LVSR` resource — "SAVERECORD" in LabVIEW source — is the per-VI header
that stores the LabVIEW version the file was last saved with, plus a run of
flag words covering execution settings, UI preferences, and debug state.
Typical corpus payload size is 160 bytes. LabVIEW historically uses several
payload sizes for this block as new flags are added
(pylabview notes 68 / 96 / 120 / 136 / 137); the codec treats the fixed
4-byte version header as mandatory and every following byte as opaque flag
bytes, so newer layouts round-trip without special cases.

## Wire layout (big-endian)

| Offset | Size | Field     | Notes                                                           |
| -----: | ---: | --------- | --------------------------------------------------------------- |
|      0 |    4 | `Version` | Packed version word. Same BCD-ish encoding as the `vers` block. |
|      4 |  N−4 | `Raw`     | Variable-length run of flag bytes. Accessed as BE uint32 words. |

**Total size:** at least 4 bytes. The codec does not require `(N−4)` to be
a multiple of 4 — LabVIEW 14's 137-byte form includes one byte of tail that
is not part of the uint32 grid, and the codec round-trips it verbatim.

## Flag bits

Each boolean is addressed by a `(word-index, mask)` pair where *word-index*
counts 4-byte big-endian `uint32` slots starting after the `Version` header
(so word 0 is `Raw[0:4]`, word 1 is `Raw[4:8]`, etc.). A bit is considered
set when **any** bit of the mask is set — this matches pylavi's
`get_flag_set` semantics. Accessors return `false` when `Raw` is too short
to reach the addressed word.

| Accessor              | Word | Mask         | Meaning (cross-reference)                                           |
| --------------------- | :--: | :----------: | ------------------------------------------------------------------- |
| `SuspendOnRun()`      |  0   | `0x00001000` | Suspend-when-called. pylabview `VI_EXEC_FLAGS.HasSetBP`.            |
| `Locked()`            |  0   | `0x00002000` | Library containing the VI is locked. pylabview `LibProtected`.      |
| `RunOnOpen()`         |  0   | `0x00004000` | Auto-run when loaded. pylabview `VI_EXEC_FLAGS.RunOnOpen`.          |
| `SavedForPrevious()`  |  1   | `0x00000004` | Saved for an earlier LabVIEW version.                               |
| `SeparateCode()`      |  1   | `0x00000400` | Compiled code stored separately from source.                        |
| `ClearIndicators()`   |  1   | `0x01000000` | Unwired indicators are cleared on each run.                         |
| `AutoErrorHandling()` |  1   | `0x20000000` | Automatic error handling enabled.                                   |
| `HasBreakpoints()`    |  5   | `0x20000000` | VI carries stored breakpoints.                                      |
| `Debuggable()`        |  5   | `0x40000200` | Saved with debugging enabled. Either bit of the mask counts as set. |

## Breakpoint count

The integer stored at flag-word index 28 (`Raw[112:116]`) holds the total
breakpoint count across the VI. Accessible via
`(Value).BreakpointCount() (uint32, bool)`; the `bool` is `false` if the
payload is too short to reach word 28. pylavi exposes the same index via
`BREAKPOINT_COUNT_INDEX = 28`.

## Validation rules applied by `internal/codecs/lvsr`

| Severity | Code                     | Condition                                      |
| -------- | ------------------------ | ---------------------------------------------- |
| error    | `lvsr.payload.too_short` | Payload is fewer than 4 bytes (no `Version`).  |

Higher-level validation (flag-word sanity, unknown LabVIEW version gates)
is intentionally deferred — the codec is read-only and should not fail
when a corpus sample carries bits we have not classified yet.

## References

- pylavi `TypeLVSR` — the concise flag map used as the primary source:
  `references/pylavi/pylavi/resource_types.py:96–198`
- pylabview `LVSR` / `LVSRData` / `VI_EXEC_FLAGS` — confirms that the
  first flag word is `execFlags` and that `0x00002000 == LibProtected`,
  `0x00004000 == RunOnOpen`, etc.:
  `references/pylabview/pylabview/LVblock.py:3503+`,
  `references/pylabview/pylabview/LVinstrument.py:137–171`

## Open questions

- **Other flag bits.** `Raw` carries 39+ `uint32` words in the 160-byte
  corpus form but only 9 bits and one count slot are publicly documented.
  Many other LVSR bits almost certainly have meanings (pylabview's
  `VI_EXEC_FLAGS` enum has 32 names, most not yet surfaced here).
- **`SavedForPrevious` vs. `SeparateCode` interactions.** pylavi documents
  both independently, but the LabVIEW UI couples them (saving for an older
  version forces separate code in some versions).
- **Password handling.** `Locked` does not by itself imply a password;
  the stored password hash lives in `BDPW`. A codec port of `BDPW` is
  scheduled for Phase 6.3 — once it lands, `lvvi.Model` can report a
  clearer "password-protected" boolean by combining the two blocks.
- **Non-standard payload sizes.** pylabview enumerates lengths `68`, `96`,
  `120`, `136`, `137`, plus `sizeof(LVSRData)`. Only `160` is observed
  in the corpus. The codec accepts any length ≥ 4 so anomalous sizes
  round-trip cleanly; structural validators might want to flag sizes
  outside the documented set in a future strict-mode check.
