package lvvi

import "github.com/CWBudde/lvrsrc/internal/codecs/heap"

// WidgetKind groups the ~370 LabVIEW SL__ object classes into a small
// set of rendering categories. The web demo and SVG renderer use it to
// pick a generic skin (filled box vs. outlined box vs. ring vs. label)
// so a viewer can tell booleans from numerics from strings without
// matching every individual ClassTag.
//
// The table began as a name-based heuristic in Phase 12.2a and was
// cross-checked against pylabview's LVheap.py class enum / per-class
// child-tag dispatch in Phase 14.1. pylabview does not expose a visual
// widget-kind enum; the explicit refnum / variant / connector-pane
// classes below are carried as separate kinds because they are stable
// heap classes rather than renderer guesses.
type WidgetKind string

// Members of WidgetKind. The set is intentionally small; classes that
// don't fit a category fall back to WidgetKindOther so the renderer
// keeps emitting placeholder boxes for them rather than crashing.
const (
	WidgetKindBoolean    WidgetKind = "boolean"
	WidgetKindNumeric    WidgetKind = "numeric"
	WidgetKindString     WidgetKind = "string"
	WidgetKindCluster    WidgetKind = "cluster"
	WidgetKindArray      WidgetKind = "array"
	WidgetKindGraph      WidgetKind = "graph"
	WidgetKindDecoration WidgetKind = "decoration"
	WidgetKindStructure  WidgetKind = "structure"
	WidgetKindPrimitive  WidgetKind = "primitive"
	WidgetKindTerminal   WidgetKind = "terminal"
	WidgetKindRefnum     WidgetKind = "refnum"
	WidgetKindVariant    WidgetKind = "variant"
	WidgetKindConPane    WidgetKind = "connector-pane"
	WidgetKindOther      WidgetKind = "other"
)

// WidgetKindForClass maps a positive ClassTag to its widget kind.
// Unmapped classes return WidgetKindOther.
func WidgetKindForClass(c heap.ClassTag) WidgetKind {
	if k, ok := widgetKindByClass[c]; ok {
		return k
	}
	return WidgetKindOther
}

// WidgetKindForNode classifies a heap node by its tag. Positive tags
// resolve through ClassTag; system tags fall back to WidgetKindOther
// except for the array machinery (SL__array, SL__arrayElement), which
// map to WidgetKindArray because FP array containers and their
// repeated children are persisted as system tags rather than positive
// class tags.
func WidgetKindForNode(n HeapNode) WidgetKind {
	if n.Tag < 0 {
		switch heap.SystemTag(n.Tag) {
		case heap.SystemTagArray, heap.SystemTagArrayElement:
			return WidgetKindArray
		}
		return WidgetKindOther
	}
	return WidgetKindForClass(heap.ClassTag(n.Tag))
}

// widgetKindByClass is the curated lookup table. Entries are ordered
// roughly by category (matching the WidgetKind constant order) so a
// reviewer can scan the FP-side controls before BD-side primitives.
var widgetKindByClass = map[heap.ClassTag]WidgetKind{
	// Boolean controls.
	heap.ClassTagStdBool: WidgetKindBoolean,

	// Numeric controls — value is a single number, possibly with units.
	heap.ClassTagStdNum:         WidgetKindNumeric,
	heap.ClassTagStdColorNum:    WidgetKindNumeric,
	heap.ClassTagStdSlide:       WidgetKindNumeric,
	heap.ClassTagStdKnob:        WidgetKindNumeric,
	heap.ClassTagStdRing:        WidgetKindNumeric,
	heap.ClassTagStdRamp:        WidgetKindNumeric,
	heap.ClassTagStdMeasureData: WidgetKindNumeric,

	// String / path / combo controls.
	heap.ClassTagStdString:   WidgetKindString,
	heap.ClassTagStdPath:     WidgetKindString,
	heap.ClassTagStdComboBox: WidgetKindString,
	heap.ClassTagStdTag:      WidgetKindString,

	// Refnum controls / constants / data objects. pylabview's
	// LVheap.py exposes these as explicit SL__*Ref* / SL__baseRefNum
	// classes; keep them distinct from generic primitives so callers
	// can style and audit reference-bearing nodes separately.
	heap.ClassTagStdRefNum:    WidgetKindRefnum,
	heap.ClassTagStdHandle:    WidgetKindRefnum,
	heap.ClassTagGRef:         WidgetKindRefnum,
	heap.ClassTagGRefDCO:      WidgetKindRefnum,
	heap.ClassTagCtlRefConst:  WidgetKindRefnum,
	heap.ClassTagCtlRefDCO:    WidgetKindRefnum,
	heap.ClassTagOldStatVIRef: WidgetKindRefnum,
	heap.ClassTagStatVIRef:    WidgetKindRefnum,
	heap.ClassTagDynLink:      WidgetKindRefnum,
	heap.ClassTagBaseRefNum:   WidgetKindRefnum,

	// Variant controls / data objects.
	heap.ClassTagStdVar:       WidgetKindVariant,
	heap.ClassTagOleVariant:   WidgetKindVariant,
	heap.ClassTagStdLvVariant: WidgetKindVariant,

	// Connector pane object. It has its own pylabview class
	// (SL__conPane) and a dedicated child-tag mapping in
	// CLASS_EN_TO_TAG_LIST_MAPPING, so it is not just a generic box.
	heap.ClassTagConPane: WidgetKindConPane,

	// Cluster controls.
	heap.ClassTagStdClust:   WidgetKindCluster,
	heap.ClassTagRadioClust: WidgetKindCluster,

	// Array containers (positive ClassTag forms; SystemTag(-4) is
	// handled at the node-level resolver).
	heap.ClassTagIndArr:   WidgetKindArray,
	heap.ClassTagTabArray: WidgetKindArray,

	// Graph / table / picture / list — complex visualizations whose
	// outer rect we can render but whose contents are deferred.
	heap.ClassTagStdGraph:          WidgetKindGraph,
	heap.ClassTagStdTable:          WidgetKindGraph,
	heap.ClassTagStdListbox:        WidgetKindGraph,
	heap.ClassTagStdPict:           WidgetKindGraph,
	heap.ClassTagStdPixMap:         WidgetKindGraph,
	heap.ClassTagDigitalTable:      WidgetKindGraph,
	heap.ClassTagTreeControl:       WidgetKindGraph,
	heap.ClassTagTableControl:      WidgetKindGraph,
	heap.ClassTagBaseListbox:       WidgetKindGraph,
	heap.ClassTagBaseTableControl:  WidgetKindGraph,
	heap.ClassTagScenegraphdisplay: WidgetKindGraph,

	// Decorations — purely visual elements (labels, ornaments, fonts).
	heap.ClassTagCosm:          WidgetKindDecoration,
	heap.ClassTagMultiCosm:     WidgetKindDecoration,
	heap.ClassTagBigMultiCosm:  WidgetKindDecoration,
	heap.ClassTagSubCosm:       WidgetKindDecoration,
	heap.ClassTagLabel:         WidgetKindDecoration,
	heap.ClassTagMultiLabel:    WidgetKindDecoration,
	heap.ClassTagBigMultiLabel: WidgetKindDecoration,
	heap.ClassTagNumLabel:      WidgetKindDecoration,
	heap.ClassTagSelLabel:      WidgetKindDecoration,
	heap.ClassTagSubLabel:      WidgetKindDecoration,
	heap.ClassTagTextHair:      WidgetKindDecoration,
	heap.ClassTagFontRun:       WidgetKindDecoration,
	heap.ClassTagAttachment:    WidgetKindDecoration,

	// Structures — block-diagram containers that house a sub-diagram.
	heap.ClassTagForLoop:                     WidgetKindStructure,
	heap.ClassTagWhileLoop:                   WidgetKindStructure,
	heap.ClassTagTimeLoop:                    WidgetKindStructure,
	heap.ClassTagSequence:                    WidgetKindStructure,
	heap.ClassTagFlatSequence:                WidgetKindStructure,
	heap.ClassTagTimeSequence:                WidgetKindStructure,
	heap.ClassTagTimeFlatSequence:            WidgetKindStructure,
	heap.ClassTagSequenceFrame:               WidgetKindStructure,
	heap.ClassTagTimeFlatSequenceFrame:       WidgetKindStructure,
	heap.ClassTagCaseSel:                     WidgetKindStructure,
	heap.ClassTagEventStruct:                 WidgetKindStructure,
	heap.ClassTagSimDiag:                     WidgetKindStructure,
	heap.ClassTagXStructure:                  WidgetKindStructure,
	heap.ClassTagRegionNode:                  WidgetKindStructure,
	heap.ClassTagDecomposeRecomposeStructure: WidgetKindStructure,
	heap.ClassTagStateNode:                   WidgetKindStructure,
	heap.ClassTagStateDiagWiz:                WidgetKindStructure,
	heap.ClassTagTransition:                  WidgetKindStructure,
	heap.ClassTagSelect:                      WidgetKindStructure,

	// Primitives — single-step block-diagram operations.
	heap.ClassTagPrim:                       WidgetKindPrimitive,
	heap.ClassTagNode:                       WidgetKindPrimitive,
	heap.ClassTagSNode:                      WidgetKindPrimitive,
	heap.ClassTagGrowableNode:               WidgetKindPrimitive,
	heap.ClassTagPropNode:                   WidgetKindPrimitive,
	heap.ClassTagInvokeNode:                 WidgetKindPrimitive,
	heap.ClassTagCallByRefNode:              WidgetKindPrimitive,
	heap.ClassTagMathScriptNode:             WidgetKindPrimitive,
	heap.ClassTagMathScriptCallByRefNode:    WidgetKindPrimitive,
	heap.ClassTagExtFunc:                    WidgetKindPrimitive,
	heap.ClassTagIUse:                       WidgetKindPrimitive,
	heap.ClassTagDynIUse:                    WidgetKindPrimitive,
	heap.ClassTagPolyIUse:                   WidgetKindPrimitive,
	heap.ClassTagDynPolyIUse:                WidgetKindPrimitive,
	heap.ClassTagGenIUse:                    WidgetKindPrimitive,
	heap.ClassTagConcat:                     WidgetKindPrimitive,
	heap.ClassTagSubset:                     WidgetKindPrimitive,
	heap.ClassTagMergeSignal:                WidgetKindPrimitive,
	heap.ClassTagSplitSignal:                WidgetKindPrimitive,
	heap.ClassTagInterLeave:                 WidgetKindPrimitive,
	heap.ClassTagDecimate:                   WidgetKindPrimitive,
	heap.ClassTagGrowArrayNode:              WidgetKindPrimitive,
	heap.ClassTagSharedGrowArrayNode:        WidgetKindPrimitive,
	heap.ClassTagCpdArith:                   WidgetKindPrimitive,
	heap.ClassTagMux:                        WidgetKindPrimitive,
	heap.ClassTagDemux:                      WidgetKindPrimitive,
	heap.ClassTagNMux:                       WidgetKindPrimitive,
	heap.ClassTagDecomposeArrayNode:         WidgetKindPrimitive,
	heap.ClassTagDecomposeClusterNode:       WidgetKindPrimitive,
	heap.ClassTagDecomposeMatchNode:         WidgetKindPrimitive,
	heap.ClassTagDecomposeVariantNode:       WidgetKindPrimitive,
	heap.ClassTagDecomposeDataValRefNode:    WidgetKindPrimitive,
	heap.ClassTagDecomposeArraySplitNode:    WidgetKindPrimitive,
	heap.ClassTagSimNode:                    WidgetKindPrimitive,
	heap.ClassTagSdfNode:                    WidgetKindPrimitive,
	heap.ClassTagEventDataNode:              WidgetKindPrimitive,
	heap.ClassTagTimeDataNode:               WidgetKindPrimitive,
	heap.ClassTagXNode:                      WidgetKindPrimitive,
	heap.ClassTagXDataNode:                  WidgetKindPrimitive,
	heap.ClassTagForkNode:                   WidgetKindPrimitive,
	heap.ClassTagJoinNode:                   WidgetKindPrimitive,
	heap.ClassTagJunctionNode:               WidgetKindPrimitive,
	heap.ClassTagPlaceholderNode:            WidgetKindPrimitive,
	heap.ClassTagCommentNode:                WidgetKindPrimitive,
	heap.ClassTagTextNode:                   WidgetKindPrimitive,
	heap.ClassTagExprNode:                   WidgetKindPrimitive,
	heap.ClassTagScriptNode:                 WidgetKindPrimitive,
	heap.ClassTagConstructorNode:            WidgetKindPrimitive,
	heap.ClassTagEventRegNode:               WidgetKindPrimitive,
	heap.ClassTagEventRegCallback:           WidgetKindPrimitive,
	heap.ClassTagSlaveFBInputNode:           WidgetKindPrimitive,
	heap.ClassTagHiddenFBNode:               WidgetKindPrimitive,
	heap.ClassTagDexChannelCreateNode:       WidgetKindPrimitive,
	heap.ClassTagDexChannelShutdownNode:     WidgetKindPrimitive,
	heap.ClassTagParForWorkers:              WidgetKindPrimitive,
	heap.ClassTagMergeErrors:                WidgetKindPrimitive,
	heap.ClassTagPolySelector:               WidgetKindPrimitive,
	heap.ClassTagFxpUnbundle:                WidgetKindPrimitive,
	heap.ClassTagSharedVariable:             WidgetKindPrimitive,
	heap.ClassTagSharedVariableDynamicOpen:  WidgetKindPrimitive,
	heap.ClassTagSharedVariableDynamicRead:  WidgetKindPrimitive,
	heap.ClassTagSharedVariableDynamicWrite: WidgetKindPrimitive,
	heap.ClassTagFBox:                       WidgetKindPrimitive,
	heap.ClassTagABuild:                     WidgetKindPrimitive,
	heap.ClassTagCABuild:                    WidgetKindPrimitive,
	heap.ClassTagAIndx:                      WidgetKindPrimitive,
	heap.ClassTagADelete:                    WidgetKindPrimitive,
	heap.ClassTagAInit:                      WidgetKindPrimitive,
	heap.ClassTagAInsert:                    WidgetKindPrimitive,
	heap.ClassTagAReplace:                   WidgetKindPrimitive,
	heap.ClassTagAReshape:                   WidgetKindPrimitive,

	// Terminals and tunnels — anchor points where wires connect to a
	// node, including structure tunnels (inputs/outputs that pierce a
	// loop or sequence boundary) and FP-side terminals that link a
	// front-panel control to its block-diagram representation.
	heap.ClassTagTerm:                     WidgetKindTerminal,
	heap.ClassTagFPTerm:                   WidgetKindTerminal,
	heap.ClassTagLpTun:                    WidgetKindTerminal,
	heap.ClassTagInnerLpTun:               WidgetKindTerminal,
	heap.ClassTagMatedLpTun:               WidgetKindTerminal,
	heap.ClassTagSeqTun:                   WidgetKindTerminal,
	heap.ClassTagMatedSeqTun:              WidgetKindTerminal,
	heap.ClassTagFlatSeqTun:               WidgetKindTerminal,
	heap.ClassTagSelTun:                   WidgetKindTerminal,
	heap.ClassTagSimTun:                   WidgetKindTerminal,
	heap.ClassTagSdfTun:                   WidgetKindTerminal,
	heap.ClassTagRegionTun:                WidgetKindTerminal,
	heap.ClassTagCommentTun:               WidgetKindTerminal,
	heap.ClassTagExternalTun:              WidgetKindTerminal,
	heap.ClassTagXTunnel:                  WidgetKindTerminal,
	heap.ClassTagDecomposeRecomposeTunnel: WidgetKindTerminal,
}
