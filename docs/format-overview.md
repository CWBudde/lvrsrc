# Format Overview

`lvrsrc` treats LabVIEW RSRC containers as a structured envelope around typed and
opaque data. The parser prioritizes:

- exact byte-preserving reads,
- explicit structural metadata,
- future-safe extension points for typed codecs.

The architecture separates wire parsing (`internal/rsrcwire`) from public API
surfaces (`pkg/lvrsrc`, `pkg/lvvi`, `pkg/lvmeta`, `pkg/lvdiff`).
