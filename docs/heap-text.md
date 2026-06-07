# Heap text: labels, captions, and item lists

This note records how LabVIEW stores human-readable text inside the
front-panel (FPHb) and block-diagram (BDHb) heaps, derived **empirically
from the checked-in corpus** (no new fixtures) and cross-checked against
the pylabview reference (`references/pylabview/pylabview/LVheap.py`,
`LVparts.py`).

Typed accessors live in `pkg/lvvi/text.go`
(`HeapStringAt`, `HeapLabelListAt`, and the `HeapTexts` enumerator, plus
`Model.FrontPanelTexts`/`Model.BlockDiagramTexts`). Part roles are named by
`pkg/lvvi/partid.go` (`PartID`, mirroring pylabview's `PARTID` enum).

## Two storage mechanisms

LabVIEW persists heap text two different ways. Both are decoded.

### 1. Raw strings — `OF__text` and friends

A node whose tag resolves (in its parent-class context) to one of
pylabview's `NODE_STRING_TAGS_LIST` members stores its text **directly as
the node content**, with no length prefix:

| Resolved tag       | Meaning                                              |
| ------------------ | ---------------------------------------------------- |
| `OF__text`         | control / pane / terminal **name label** text        |
| `OF__format`       | numeric / string display format (`"%#_g"`, `"%.0f"`) |
| `OF__nodeName`     | block-diagram node name                              |
| `OF__methName`     | block-diagram method name                            |
| `OF__tagDLLName`   | Call Library node DLL name                           |
| `OF__PropItemName` | property-node item name                              |

Detection keys on the **context-resolved** name rather than the bare tag
id, because the ids collide across classes. The important case:
`OF__text` is tag `3`, which means `OF__activePlot` everywhere _except_
inside an `SL__textHair` object. `ParentTopClass` + `ResolveTagName`
disambiguate exactly the way pylabview's `tagIdToEnum` does.

### 2. P-string lists — `OF__buf` under `SL__multiLabel`

Multi-item text (ring/enum item lists, boolean per-state text) is stored
in an `OF__buf` node nested in an `SL__multiLabel` object. The content is a
flat sequence of **one-byte-length-prefixed strings**:

```
[len0][bytes0][len1][bytes1]…
```

`decodePStrList` splits this; `HeapLabelListAt` returns the lines.
`SL__bigMultiLabel` (a wider-length variant) is not present in the corpus
and is not decoded, matching pylabview, which dispatches its `PStrList`
reader for `SL__multiLabel` only.

## Role and owner attribution

`HeapTexts` attaches two pieces of context to every decoded element:

- **Role** (`PartID`): the `OF__partID` of the nearest enclosing part. A
  name label resolves to `NAME_LABEL` (16), ring item text to `RING_TEXT`
  (12), and boolean state text to `BOOLEAN_TEXT` (22). `OF__text` is the
  content of _whatever_ label/caption part encloses it, so its role varies
  with context (also seen: `DIAGRAM_IDENTIFIER`, or `NO_PARTID` for free
  text).
- **Owner class** (`heap.ClassTag`): the nearest enclosing object class
  that is not a text wrapper (`SL__textHair`, `SL__fontRun`,
  `SL__multiLabel` are skipped) — e.g. `SL__label` for a name label or
  `SL__stdRing` for ring item text.

Linking a _name label_ back to the specific control it titles requires the
`OF__masterPart` cross-reference and is left as future work; `OwnerClass`
reports the class that directly contains the text.

## Corpus evidence

Across the corpus FPHb/BDHb heaps:

- **311 `OF__text`** strings (303 with role `NAME_LABEL`) — control names
  (`"Numeric"`, `"Boolean"`), terminal names (`"Input A"`, `"Output Y"`),
  and pane names (`"Pane"`).
- **156 `OF__format`** strings (`"%#_g"`, `"%.0f"`).
- **20 `SL__multiLabel`** P-string lists — e.g. `["VI", "Project"]`,
  `["Wait For Event", "Update UI", "Update Server", "Load Project"]`, and
  the full type-name ring in `is-*.vi`.

All 20 P-string buffers round-trip byte-for-byte
(`TestHeapLabelListRoundTrip`).

## Status

This is an additive projection layer; it does not change byte disposition.
`OF__nodeName`/`OF__methName`/`OF__tagDLLName`/`OF__PropItemName` are
decoded but not exercised by the current corpus (no Call Library / property
nodes present). Promoting the `OF__text`/`OF__buf` coverage status and
wiring the decoded text into the SVG/scene renderer are natural follow-ups.
