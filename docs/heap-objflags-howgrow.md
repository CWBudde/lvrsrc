# Heap fields `OF__objFlags` and `OF__howGrow`

This note records the semantics of two common heap part fields, derived
**empirically from the checked-in corpus** (no new fixtures) and
cross-checked against the pylabview reference. It distinguishes
**confirmed** facts (authoritative source agreement and/or strong corpus
evidence) from **inferred** hypotheses that are documented but not yet
asserted in code.

Typed accessors live in `pkg/lvvi/scalar.go`
(`HeapObjFlags`/`FindObjFlagsChild`, `HeapHowGrow`/`FindHowGrowChild`).
Both preserve the full raw word; only authoritative meanings are surfaced
as named fields. Coverage status for both tags stays `partial` — most
bits remain unnamed in the published format.

## Method

For every FPHb/BDHb heap tree in `testdata/corpus/`, each part node (a
node owning an `OF__objFlags` child) was joined with its sibling
`OF__partID`, `OF__howGrow`, and `OF__masterPart` leaves. This yielded
**2630 part tuples** (2630 `objFlags`, 1774 `howGrow` leaves) across 130
heap trees in 65 fixtures. Bit-set frequencies were computed overall and
conditioned on `partID`; `howGrow` values were tabulated per `partID`.

## `OF__objFlags` (tag 172)

A 32-bit flag word. pylabview's `LVparts.py` `OBJ_FLAGS` enum names only
**two** bits; the other 30 are annotated `unknown`.

### Confirmed (named in code)

| Bit | Name          | Authoritative meaning           | Corpus evidence                                                                                                                                                                                        |
| --: | ------------- | ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
|   0 | `IsIndicator` | "indicator, input is disabled"  | Set on 100% of non-interactive structural sub-parts (FRAME 227/227, CONTENT_AREA 117/117, EXTRA_FRAME_PART 94/94); clear on interactive parts (X/Y scrollbars 0/94). Consistent with "input disabled". |
|   3 | `IsHidden`    | "part is not visible on screen" | Set on 160/166 (96%) of RADIX parts and ~70% of increment/decrement sub-parts (partID 2/3) — matching LabVIEW defaults where the radix and stepper arrows are hidden.                                  |

### Observed but unnamed (left as raw bits, reachable via `ObjFlagsValue.Bit`)

Per-bit set frequency across 2630 parts (bits never set are omitted):

```
bit1  65.4%   bit2  21.8%   bit4  52.6%   bit5  44.4%   bit6  40.3%
bit8  61.0%   bit11 48.4%   bit16 19.3%   bit17 32.4%   bit18 31.7%
bit20 17.0%   (bit7,9,10,12-15,19,21-26,28 are sparse <12%)
```

The **high byte (bits 16–23) correlates with the part family**: NAME_LABEL
parts cluster on values `0x0017xxxx` (e.g. `0x0017014a`), matching the
modRSRC.py anchor `NAME_LABEL objFlags = 0x17114a`. These bits are _not_
named because pylabview does not document them and corpus correlation
alone is insufficient to assign individual meanings.

## `OF__howGrow` (tag 106)

A resize/anchor bitfield controlling how a part grows when its parent
resizes. pylabview has **no** enum for it. No bits are named in code; the
raw word is preserved and individual bits are reachable via
`HowGrowValue.Bit`.

### Confirmed (corpus + modRSRC.py anchors)

`howGrow` is **deterministic per structural partID**, and the observed
values match modRSRC.py exactly:

| partID | Part         | howGrow (corpus)      | modRSRC.py |
| -----: | ------------ | --------------------- | ---------- |
|     38 | X_SCROLLBAR  | `0x38` (56), 94/94    | 56 ✓       |
|     39 | Y_SCROLLBAR  | `0xC2` (194), 94/94   | 194 ✓      |
|     28 | CONTENT_AREA | `0x78` (120) dominant | 120 ✓      |

### Inferred (documented, not asserted)

The low 8 bits look like two grow-direction groups. X_SCROLLBAR=`0x38`
(bits 3,4,5) and Y_SCROLLBAR=`0xC2` (bits 1,6,7) are near-transposes, and
CONTENT_AREA=`0x78` (bits 3,4,5,6) combines them — consistent with a
horizontal group (bits ~3–5) and a vertical group (bits ~1,6,7). The most
common value overall is `0xF0` (240, bits 4–7; 585 parts), the apparent
"grow freely" default for cosmetic/text parts. Upper bits also appear
(`0x1000`/`0x3000` common on labels, `0x0F00` on a few parts), so the
field is wider than 8 bits. Exact per-bit edge/anchor assignments remain
a hypothesis pending controlled resize fixtures.

## Next steps

- Name the remaining `objFlags` bits once authoritative references or
  controlled toggle fixtures are available.
- Confirm the `howGrow` grow/anchor bit layout with controlled
  parent-resize fixtures, then name the bits.
