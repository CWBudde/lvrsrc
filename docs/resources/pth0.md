# Path References: `PTH0`, `PTH1`, `PTH2`

**Package:** `internal/codecs/pthx`
**Safety tier:** Helper (not a top-level resource codec)
**Status:** decode + encode, round-trip verified across 32 PTH0 instances
embedded in corpus `LIfp` / `LIbd` payloads. PTH1 / PTH2 are not present
in the shipped corpus but the codec is implemented from the pylabview
spec for forward compatibility.

`PTH0` / `PTH1` / `PTH2` are LabVIEW's path-reference data structures.
They are not registered as top-level RSRC resources; they appear
embedded inside link-info blocks (`LIfp`, `LIbd`, `LIvi`) and inside
data fills generally. This package decodes them as a unit so the
enclosing codecs do not have to redo the path-walk logic.

## Variants

LabVIEW selects layout by the leading FourCC `Ident`:

- **`PTH0`** — the original (`LVPath0` in pylabview). 1-byte length
  prefix per component plus a 2-byte `TPVal` whose meaning is
  undocumented (only values `0` and `1` observed; range is `0..3`).
- **`PTH1` / `PTH2`** — the newer form (`LVPath1`). 2-byte length
  prefix per component plus a 4-byte `TPIdent` chosen from
  `"abs "`, `"rel "`, `"unc "`, `"!pth"`. PTH2 is treated as a PTH1
  variant — pylabview's `parsePathRef` accepts both interchangeably.

## Wire layout

```text
+--------------------+  offset 0
|  Ident (4 bytes)   |
|  TotLen (u32 BE)   |  byte length of the body that follows
+--------------------+  offset 8
|        body        |  layout depends on Ident
+--------------------+
```

### PTH0 body

| Offset (within body) | Size | Field      | Notes                                   |
| -------------------: | ---: | ---------- | --------------------------------------- |
|                    0 |    2 | `TPVal`    | u16 BE, observed values `0` and `1`.    |
|                    2 |    2 | `Count`    | u16 BE, number of components.           |
|                    4 |    1 | `Len[0]`   | Pascal length byte for component `[0]`. |
|                  ... |  ... | ...        | Components packed back-to-back.         |

### PTH1 / PTH2 body

| Offset (within body) | Size | Field           | Notes                                                 |
| -------------------: | ---: | --------------- | ----------------------------------------------------- |
|                    0 |    4 | `TPIdent`       | One of `"abs "`, `"rel "`, `"unc "`, `"!pth"`.        |
|                    4 |    2 | `Len[0]` (u16)  | Length of component `[0]`.                            |
|                    6 |  ... | `Bytes[0]`      | Raw component bytes.                                  |
|                  ... |  ... | ...             | Components packed until the body is consumed.         |

### Zero-fill phony PTH0

LabVIEW occasionally writes a "phony" 8-byte PTH0: just the FourCC
ident followed by four zero bytes (`TotLen == 0`, no body at all).
pylabview calls this `canZeroFill`. The codec preserves it via the
`Value.ZeroFill` flag and `(Value).IsPhony()` reports it.

## Public API

```go
type Value struct {
    Ident      string   // "PTH0", "PTH1", "PTH2"
    TPVal      uint16   // PTH0 only
    TPIdent    string   // PTH1/PTH2 only — 4 chars
    Components [][]byte
    ZeroFill   bool     // PTH0 phony 8-byte form
}

func Decode(buf []byte) (Value, int, error) // returns Value + bytes consumed
func Encode(v Value) ([]byte, error)
```

Helpers on `Value`:

- `IsPTH0()`, `IsPTH1()` — variant predicates.
- `IsAbsolute()`, `IsRelative()`, `IsUNC()`, `IsNotAPath()` — `TPIdent`
  conveniences for PTH1/PTH2.
- `IsPhony()` — `ZeroFill && IsPTH0()`.

## References

- pylabview `LVPath0`: `LVclasses.py:159-238`.
- pylabview `LVPath1`: `LVclasses.py:94-156`.
- pylabview `parsePathRef` (variant dispatch): `LVlinkinfo.py:66-78`.
- pylabview `readPStr` / `preparePStr`: `LVmisc.py:516-532`.

## Open questions

- The semantics of `PTH0.TPVal`. pylabview observes `0` and `1` and
  documents the range as `0..3`, but nothing about what each value
  means. May correlate with `TPIdent` in PTH1 (`abs ` / `rel ` /
  `unc ` / `!pth`) but corpus evidence is too thin to confirm.
- Text encoding of components. pylabview uses `vi.textEncoding`
  which varies by platform; this codec keeps the bytes opaque and
  defers decoding to the caller.
- Whether PTH2 ever differs from PTH1 in any meaningful way beyond
  the FourCC. pylabview treats them identically; the codec follows.
