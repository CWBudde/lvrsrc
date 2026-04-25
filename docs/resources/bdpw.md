# `BDPW` — Block Diagram Password

**FourCC:** `BDPW`
**Safety tier:** 1 (read-only)
**Status:** decode + encode + validate, round-trip verified across the
corpus BDPW sections (every shipped fixture is unprotected, so all carry
the empty-password MD5 sentinel).

`BDPW` stores three concatenated 16-byte MD5 hashes that LabVIEW writes
when saving a VI:

1. `PasswordMD5` — the MD5 of the user's password (or the MD5 of the
   empty string for unprotected VIs).
2. `Hash1` — a derived verification hash LabVIEW recomputes from the VI's
   contents on save. Used as a tamper-evidence signal.
3. `Hash2` — a second derived hash added in LabVIEW 8.0 (older LV
   versions ship without it for a 32-byte payload). The current corpus is
   uniformly 48 bytes, so this codec only accepts the modern form.

## Wire layout

| Offset | Size | Field         | Notes                                                                             |
| -----: | ---: | ------------- | --------------------------------------------------------------------------------- |
|      0 |   16 | `PasswordMD5` | MD5 of the password. Empty-password sentinel: `d41d8cd98f00b204e9800998ecf8427e`. |
|     16 |   16 | `Hash1`       | LabVIEW-derived verification hash.                                                |
|     32 |   16 | `Hash2`       | Second verification hash (LV >= 8.0).                                             |

**Total size:** 48 bytes.

## Helpers

The `Value` type exposes:

- `HasPassword() bool` — `true` when `PasswordMD5` is not the empty-
  password sentinel. Cheap and side-effect-free.
- `PasswordMatches(password string) bool` — checks a candidate password
  by MD5-hashing it and comparing against `PasswordMD5`. Verifying a
  candidate is safe (no derivation secrets are stored client-side); the
  codec deliberately does not provide a way to _change_ the stored
  password because `Hash1` and `Hash2` derivation has not been validated
  against arbitrary VIs.
- `EmptyPasswordHex()` — the canonical hex string of the empty-password
  MD5 (`d41d8cd98f00b204e9800998ecf8427e`), useful for UI displays.

## Validation rules

| Severity | Code                | Condition                        |
| -------- | ------------------- | -------------------------------- |
| error    | `bdpw.payload.size` | Payload is not exactly 48 bytes. |

## References

- pylabview `BDPW`: `LVblock.py:4334-4680` — the full implementation
  including `recalculateHash1`, `recalculateHash2`, salt detection, and
  password recognition.
- pylavi `TypeBDPW`: `pylavi/resource_types.py:54-94` — the concise
  reference for the empty-password sentinel and the `password_matches`
  helper.

## Open questions

- Older LabVIEW versions (< 8.0) ship BDPW as 32 bytes (no `Hash2`). The
  codec rejects such payloads as malformed; if older fixtures appear,
  the decoder will need a length-aware shape (likely a context-version
  switch).
- Setting a non-empty password requires reproducing pylabview's `Hash1`
  / `Hash2` derivation, which depends on salt detection from `CPC2` and
  the type pool. That is out of scope for Phase 6.3 — re-evaluate when
  Phase 8 (`VCTP` + connector pane) lands.
