package lvvi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestWidgetKindForClassBoolean(t *testing.T) {
	if got := WidgetKindForClass(heap.ClassTagStdBool); got != WidgetKindBoolean {
		t.Errorf("ClassTagStdBool = %q, want %q", got, WidgetKindBoolean)
	}
}

func TestWidgetKindForClassNumeric(t *testing.T) {
	cases := []heap.ClassTag{
		heap.ClassTagStdNum,
		heap.ClassTagStdColorNum,
		heap.ClassTagStdSlide,
		heap.ClassTagStdKnob,
		heap.ClassTagStdRing,
		heap.ClassTagStdRamp,
		heap.ClassTagStdMeasureData,
	}
	for _, c := range cases {
		if got := WidgetKindForClass(c); got != WidgetKindNumeric {
			t.Errorf("%s = %q, want %q", c, got, WidgetKindNumeric)
		}
	}
}

func TestWidgetKindForClassString(t *testing.T) {
	cases := []heap.ClassTag{
		heap.ClassTagStdString,
		heap.ClassTagStdPath,
		heap.ClassTagStdComboBox,
		heap.ClassTagStdTag,
	}
	for _, c := range cases {
		if got := WidgetKindForClass(c); got != WidgetKindString {
			t.Errorf("%s = %q, want %q", c, got, WidgetKindString)
		}
	}
}

func TestWidgetKindForClassRefnum(t *testing.T) {
	cases := []heap.ClassTag{
		heap.ClassTagStdRefNum,
		heap.ClassTagStdHandle,
		heap.ClassTagGRef,
		heap.ClassTagGRefDCO,
		heap.ClassTagCtlRefConst,
		heap.ClassTagCtlRefDCO,
		heap.ClassTagOldStatVIRef,
		heap.ClassTagStatVIRef,
		heap.ClassTagDynLink,
		heap.ClassTagBaseRefNum,
	}
	for _, c := range cases {
		if got := WidgetKindForClass(c); got != WidgetKindRefnum {
			t.Errorf("%s = %q, want %q", c, got, WidgetKindRefnum)
		}
	}
}

func TestWidgetKindForClassVariant(t *testing.T) {
	cases := []heap.ClassTag{
		heap.ClassTagStdVar,
		heap.ClassTagOleVariant,
		heap.ClassTagStdLvVariant,
	}
	for _, c := range cases {
		if got := WidgetKindForClass(c); got != WidgetKindVariant {
			t.Errorf("%s = %q, want %q", c, got, WidgetKindVariant)
		}
	}
}

func TestWidgetKindForClassConnectorPane(t *testing.T) {
	if got := WidgetKindForClass(heap.ClassTagConPane); got != WidgetKindConPane {
		t.Errorf("%s = %q, want %q", heap.ClassTagConPane, got, WidgetKindConPane)
	}
}

func TestWidgetKindForClassCluster(t *testing.T) {
	cases := []heap.ClassTag{
		heap.ClassTagStdClust,
		heap.ClassTagRadioClust,
	}
	for _, c := range cases {
		if got := WidgetKindForClass(c); got != WidgetKindCluster {
			t.Errorf("%s = %q, want %q", c, got, WidgetKindCluster)
		}
	}
}

func TestWidgetKindForClassArray(t *testing.T) {
	cases := []heap.ClassTag{
		heap.ClassTagIndArr,
		heap.ClassTagTabArray,
	}
	for _, c := range cases {
		if got := WidgetKindForClass(c); got != WidgetKindArray {
			t.Errorf("%s = %q, want %q", c, got, WidgetKindArray)
		}
	}
}

func TestWidgetKindForClassGraph(t *testing.T) {
	cases := []heap.ClassTag{
		heap.ClassTagStdGraph,
		heap.ClassTagStdTable,
		heap.ClassTagStdListbox,
		heap.ClassTagStdPict,
		heap.ClassTagStdPixMap,
		heap.ClassTagDigitalTable,
		heap.ClassTagTreeControl,
		heap.ClassTagTableControl,
		heap.ClassTagBaseListbox,
		heap.ClassTagBaseTableControl,
		heap.ClassTagScenegraphdisplay,
	}
	for _, c := range cases {
		if got := WidgetKindForClass(c); got != WidgetKindGraph {
			t.Errorf("%s = %q, want %q", c, got, WidgetKindGraph)
		}
	}
}

func TestWidgetKindForClassDecoration(t *testing.T) {
	cases := []heap.ClassTag{
		heap.ClassTagCosm,
		heap.ClassTagMultiCosm,
		heap.ClassTagBigMultiCosm,
		heap.ClassTagSubCosm,
		heap.ClassTagLabel,
		heap.ClassTagMultiLabel,
		heap.ClassTagBigMultiLabel,
		heap.ClassTagNumLabel,
		heap.ClassTagSelLabel,
		heap.ClassTagSubLabel,
		heap.ClassTagTextHair,
		heap.ClassTagFontRun,
		heap.ClassTagAttachment,
	}
	for _, c := range cases {
		if got := WidgetKindForClass(c); got != WidgetKindDecoration {
			t.Errorf("%s = %q, want %q", c, got, WidgetKindDecoration)
		}
	}
}

func TestWidgetKindForClassStructure(t *testing.T) {
	cases := []heap.ClassTag{
		heap.ClassTagForLoop,
		heap.ClassTagWhileLoop,
		heap.ClassTagTimeLoop,
		heap.ClassTagSequence,
		heap.ClassTagFlatSequence,
		heap.ClassTagTimeSequence,
		heap.ClassTagTimeFlatSequence,
		heap.ClassTagSequenceFrame,
		heap.ClassTagTimeFlatSequenceFrame,
		heap.ClassTagCaseSel,
		heap.ClassTagEventStruct,
		heap.ClassTagSimDiag,
		heap.ClassTagXStructure,
		heap.ClassTagRegionNode,
		heap.ClassTagDecomposeRecomposeStructure,
		heap.ClassTagStateNode,
		heap.ClassTagStateDiagWiz,
		heap.ClassTagTransition,
		heap.ClassTagSelect,
	}
	for _, c := range cases {
		if got := WidgetKindForClass(c); got != WidgetKindStructure {
			t.Errorf("%s = %q, want %q", c, got, WidgetKindStructure)
		}
	}
}

func TestWidgetKindForClassPrimitive(t *testing.T) {
	cases := []heap.ClassTag{
		heap.ClassTagPrim,
		heap.ClassTagNode,
		heap.ClassTagSNode,
		heap.ClassTagGrowableNode,
		heap.ClassTagPropNode,
		heap.ClassTagInvokeNode,
		heap.ClassTagCallByRefNode,
		heap.ClassTagMathScriptNode,
		heap.ClassTagMathScriptCallByRefNode,
		heap.ClassTagExtFunc,
		heap.ClassTagIUse,
		heap.ClassTagDynIUse,
		heap.ClassTagPolyIUse,
		heap.ClassTagDynPolyIUse,
		heap.ClassTagGenIUse,
		heap.ClassTagConcat,
		heap.ClassTagSubset,
		heap.ClassTagMergeSignal,
		heap.ClassTagSplitSignal,
		heap.ClassTagInterLeave,
		heap.ClassTagDecimate,
		heap.ClassTagGrowArrayNode,
		heap.ClassTagSharedGrowArrayNode,
		heap.ClassTagCpdArith,
		heap.ClassTagMux,
		heap.ClassTagDemux,
		heap.ClassTagNMux,
		heap.ClassTagDecomposeArrayNode,
		heap.ClassTagDecomposeClusterNode,
		heap.ClassTagDecomposeMatchNode,
		heap.ClassTagDecomposeVariantNode,
		heap.ClassTagDecomposeDataValRefNode,
		heap.ClassTagDecomposeArraySplitNode,
		heap.ClassTagSimNode,
		heap.ClassTagSdfNode,
		heap.ClassTagEventDataNode,
		heap.ClassTagTimeDataNode,
		heap.ClassTagXNode,
		heap.ClassTagXDataNode,
		heap.ClassTagForkNode,
		heap.ClassTagJoinNode,
		heap.ClassTagJunctionNode,
		heap.ClassTagPlaceholderNode,
		heap.ClassTagCommentNode,
		heap.ClassTagTextNode,
		heap.ClassTagExprNode,
		heap.ClassTagScriptNode,
		heap.ClassTagConstructorNode,
		heap.ClassTagEventRegNode,
		heap.ClassTagEventRegCallback,
		heap.ClassTagSlaveFBInputNode,
		heap.ClassTagHiddenFBNode,
		heap.ClassTagDexChannelCreateNode,
		heap.ClassTagDexChannelShutdownNode,
		heap.ClassTagParForWorkers,
		heap.ClassTagMergeErrors,
		heap.ClassTagPolySelector,
		heap.ClassTagFxpUnbundle,
		heap.ClassTagSharedVariable,
		heap.ClassTagSharedVariableDynamicOpen,
		heap.ClassTagSharedVariableDynamicRead,
		heap.ClassTagSharedVariableDynamicWrite,
		heap.ClassTagFBox,
		heap.ClassTagABuild,
		heap.ClassTagCABuild,
		heap.ClassTagAIndx,
		heap.ClassTagADelete,
		heap.ClassTagAInit,
		heap.ClassTagAInsert,
		heap.ClassTagAReplace,
		heap.ClassTagAReshape,
	}
	for _, c := range cases {
		if got := WidgetKindForClass(c); got != WidgetKindPrimitive {
			t.Errorf("%s = %q, want %q", c, got, WidgetKindPrimitive)
		}
	}
}

func TestWidgetKindForClassTerminal(t *testing.T) {
	cases := []heap.ClassTag{
		heap.ClassTagTerm,
		heap.ClassTagFPTerm,
		heap.ClassTagLpTun,
		heap.ClassTagInnerLpTun,
		heap.ClassTagMatedLpTun,
		heap.ClassTagSeqTun,
		heap.ClassTagMatedSeqTun,
		heap.ClassTagFlatSeqTun,
		heap.ClassTagSelTun,
		heap.ClassTagSimTun,
		heap.ClassTagSdfTun,
		heap.ClassTagRegionTun,
		heap.ClassTagCommentTun,
		heap.ClassTagExternalTun,
		heap.ClassTagXTunnel,
		heap.ClassTagDecomposeRecomposeTunnel,
	}
	for _, c := range cases {
		if got := WidgetKindForClass(c); got != WidgetKindTerminal {
			t.Errorf("%s = %q, want %q", c, got, WidgetKindTerminal)
		}
	}
}

func TestWidgetKindForClassUnmappedReturnsOther(t *testing.T) {
	// FontRun is mapped (Decoration); pick something that should fall
	// through — a low-level helper class, plus an entirely synthetic tag
	// number that no enum entry uses.
	cases := []heap.ClassTag{
		heap.ClassTagDataObj, // helper, not a control/structure/primitive
		heap.ClassTag(99999), // numeric fallback
	}
	for _, c := range cases {
		if got := WidgetKindForClass(c); got != WidgetKindOther {
			t.Errorf("%s = %q, want %q", c, got, WidgetKindOther)
		}
	}
}

// FP-side array containers are persisted as a SystemTag(SL__array)
// (-4), not a positive ClassTag. The node-level resolver should still
// tag them as Array so visualizations can group them with sized
// arrays. The same applies to SL__arrayElement (-6), the per-element
// child container that dominates corpus heaps with repeated children.
func TestWidgetKindForNodeArrayContainer(t *testing.T) {
	for _, st := range []heap.SystemTag{heap.SystemTagArray, heap.SystemTagArrayElement} {
		n := HeapNode{Tag: int32(st), Scope: "open"}
		if got := WidgetKindForNode(n); got != WidgetKindArray {
			t.Errorf("WidgetKindForNode(%s) = %q, want %q", st, got, WidgetKindArray)
		}
	}
}

// A positive-tag node should resolve through ClassTag like the
// class-level resolver does.
func TestWidgetKindForNodeUsesClassTag(t *testing.T) {
	n := HeapNode{Tag: int32(heap.ClassTagStdBool), Scope: "open"}
	if got := WidgetKindForNode(n); got != WidgetKindBoolean {
		t.Errorf("WidgetKindForNode(StdBool) = %q, want %q", got, WidgetKindBoolean)
	}
}

// Negative system tags other than SL__array carry no widget meaning.
// They should fall back to Other so the renderer treats them as
// unclassified.
func TestWidgetKindForNodeUnknownSystemTagReturnsOther(t *testing.T) {
	n := HeapNode{Tag: int32(heap.SystemTagObject), Scope: "open"}
	if got := WidgetKindForNode(n); got != WidgetKindOther {
		t.Errorf("WidgetKindForNode(SystemTagObject) = %q, want %q", got, WidgetKindOther)
	}
}

// Sweep every FPHb/BDHb tree in the corpus. Every classified node
// should resolve to one of the named WidgetKinds; the unmapped
// "Other" rate gives a baseline coverage metric for follow-up
// pylabview-aligned work in 12.2b.
func TestWidgetKindForNodeCorpusBaseline(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totals := map[WidgetKind]int{}
	exercised := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".vi" && ext != ".ctl" && ext != ".vit" {
			continue
		}
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		if err != nil {
			continue
		}
		m, _ := DecodeKnownResources(f)
		for _, getter := range []func() (HeapTree, bool){m.FrontPanel, m.BlockDiagram} {
			tree, ok := getter()
			if !ok {
				continue
			}
			exercised++
			for _, n := range tree.Nodes {
				if n.Scope != "open" {
					continue
				}
				totals[WidgetKindForNode(n)]++
			}
		}
	}
	if exercised == 0 {
		t.Skip("no corpus VI yielded an FPHb or BDHb tree")
	}
	total := 0
	for _, c := range totals {
		total += c
	}
	if total == 0 {
		t.Fatal("found 0 classifiable nodes across the corpus")
	}
	classified := total - totals[WidgetKindOther]
	t.Logf("WidgetKindForNode: %d/%d open-scope nodes classified (%.1f%%) across %d trees; per-kind: %v",
		classified, total, 100*float64(classified)/float64(total), exercised, totals)
	if classified == 0 {
		t.Fatalf("no nodes classified — mapping table is empty or wrong")
	}
}
