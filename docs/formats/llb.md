# `.llb` LabVIEW Library Containers

Phase 5.3 treats `.llb` support as a container-level compatibility milestone,
not as a new embedded-file extraction feature.

## Research Summary

- `.llb` files still use the standard RSRC framing: the header magic is
  `RSRC\r\n`, the creator is typically `LBVW`, and the distinguishing type is
  `LVAR`.
- In other words, the top-level parser does not need a second container format.
  The main difference is the resource mix inside the file, which behaves like a
  LabVIEW library directory and asset bundle rather than a single VI or control.
- A small real-world sample shows blocks such as `ADir`, `PALM`, `PLM2`,
  `CPST`, `ICON`, `icl4`, `icl8`, and `STR `.

## Current Scope

- `pkg/lvrsrc.Open` and `pkg/lvrsrc.Parse` accept `.llb` files by recognizing
  `LVAR` as `FileKindLibrary`.
- `lvrsrc inspect` reports `.llb` files as `Kind: Library` and lists their
  contained RSRC blocks like any other parsed container.
- `lvrsrc rewrite` round-trips `.llb` files structurally. Byte-exact output is
  not required here; the regression target is parse -> serialize -> parse
  equivalence with no validation issues.

## Out Of Scope

- decoding the `ADir` library directory semantically
- extracting embedded member files as standalone `.vi`/`.ctl` artifacts
- reconstructing LabVIEW's original member/name ordering rules exactly
