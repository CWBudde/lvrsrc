# Connector Pane Resources: `CONP`, `CPC2`

The current corpus supports a minimal, fixed-width interpretation for the
connector-pane resources:

| FourCC | Payload size | Decoded shape       |
| ------ | -----------: | ------------------- |
| `CONP` |          2 B | big-endian `uint16` |
| `CPC2` |          2 B | big-endian `uint16` |

Neither resource carries an internal header, checksum, or trailing metadata in
the observed samples.

## Decoded Model

`internal/codecs/conpane` exposes both resources as:

- `FourCC`
- `Value uint16`

The codec preserves the raw two-byte big-endian layout on re-encode.

## Validation Rules

Current validation is intentionally narrow and corpus-grounded:

| Severity | Code                | Condition                                    |
| -------- | ------------------- | -------------------------------------------- |
| error    | `conp.payload.size` | `CONP` payload length is not exactly 2 bytes |
| error    | `cpc2.payload.size` | `CPC2` payload length is not exactly 2 bytes |

No additional semantic range checks are enforced yet.

## Corpus Evidence

From the current `testdata/corpus/` set:

- `CONP` appears in 21/21 fixtures and is always `0x0001`
- `CPC2` appears in 21/21 fixtures with observed values `0x0001`, `0x0002`,
  `0x0003`, and `0x0004`

Observed `CPC2` value distribution:

| Value | Fixtures |
| ----: | -------: |
|     1 |       11 |
|     2 |        6 |
|     3 |        3 |
|     4 |        1 |

Every observed `CONP` and `CPC2` section round-trips byte-for-byte through
`internal/codecs/conpane`.

## Open Questions

- Whether `CONP` is a connector-pane template selector, a pointer into another
  heap/resource, or just an enum with the currently observed fixed value `1`
- Whether `CPC2` is the connector-terminal count, a pane variant code, or a
  second-level lookup key
- Whether older LabVIEW versions or unusual connector panes use values outside
  the currently observed `1..4` range
