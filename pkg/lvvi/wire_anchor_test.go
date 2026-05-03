package lvvi

import (
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// Resolution-coverage guarantee: across the entire corpus, every wire
// endpoint must resolve to a non-negative WidgetKindTerminal heap-tree
// index via WireTerminalAnchor. If this regresses, either the model
// changed or a new fixture violates an assumption — we want to know.
func TestWireTerminalAnchorCorpusFullCoverage(t *testing.T) {
	corpusDir := filepath.Join("..", "..", "testdata", "corpus")
	matches, err := filepath.Glob(filepath.Join(corpusDir, "*.vi"))
	if err != nil {
		t.Fatalf("glob corpus: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no corpus VIs found at %s", corpusDir)
	}
	totalEndpoints := 0
	totalUnresolved := 0
	for _, path := range matches {
		t.Run(filepath.Base(path), func(t *testing.T) {
			file, err := lvrsrc.Open(path, lvrsrc.OpenOptions{})
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			model, _ := DecodeKnownResources(file)
			tree, ok := model.BlockDiagram()
			if !ok {
				t.Skip("no BD heap")
			}
			objIdx, _ := BuildHeapObjectIndex(tree)
			for i, n := range tree.Nodes {
				if n.Scope != "leaf" || n.Tag != int32(heap.FieldTagCompressedWireTable) || len(n.Content) == 0 {
					continue
				}
				ids, ok := WireTerminalIDs(tree, i)
				if !ok {
					continue
				}
				for k, id := range ids {
					totalEndpoints++
					anchor := WireTerminalAnchor(tree, objIdx, id)
					if anchor < 0 {
						totalUnresolved++
						t.Errorf("wire@%d term[%d] id=%d failed to resolve to a terminal anchor", i, k, id)
						continue
					}
					n := tree.Nodes[anchor]
					if n.Scope != "open" {
						t.Errorf("wire@%d term[%d] id=%d resolved to scope=%s, want open", i, k, id, n.Scope)
					}
					// Phase 16.4 A2: the anchor may be either a
					// WidgetKindTerminal (sdfTun, simTun) or the
					// per-endpoint canonical declaration that lives
					// inside one. Both are valid scene anchors; the
					// scene projection registers each so terminalByHeap
					// can map them back to a NodeKindTerminal.
					if WidgetKindForNode(n) != WidgetKindTerminal {
						_, hasID := HeapNodeID(tree, anchor)
						if !hasID {
							t.Errorf("wire@%d term[%d] id=%d resolved to widget=%s with no -3 ID — not a valid anchor",
								i, k, id, WidgetKindForNode(n))
						}
					}
				}
			}
		})
	}
	if totalUnresolved > 0 {
		t.Logf("corpus resolution: %d/%d endpoints unresolved (%.1f%%)",
			totalUnresolved, totalEndpoints,
			100*float64(totalUnresolved)/float64(totalEndpoints))
	}
}

// Walk-down: canonical wraps a primitive that contains a simTun.
func TestWireTerminalAnchorWalkDown(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{
		{
			Scope: "open", Tag: -6, Parent: -1, Children: []int{1},
			Attributes: []HeapAttribute{{ID: HeapAttrIndex, Value: 21}, {ID: HeapAttrObjectID, Value: 51}},
		},
		{
			Scope: "open", Tag: int32(heap.ClassTagSimTun), Parent: 0,
		},
	}}
	idx, _ := BuildHeapObjectIndex(tree)
	if got := WireTerminalAnchor(tree, idx, 51); got != 1 {
		t.Fatalf("WireTerminalAnchor(51) = %d, want 1 (the simTun)", got)
	}
}

// Walk-up: canonical is nested inside a sdfTun container. Phase 16.4
// A2 returns the canonical (per-endpoint) instead of the sdfTun
// ancestor (shared across all endpoints) so the scene can render
// distinct anchors per endpoint.
func TestWireTerminalAnchorWalkUpReturnsCanonical(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{
		// 0: sdfTun (terminal class, ancestor)
		{Scope: "open", Tag: int32(heap.ClassTagSdfTun), Parent: -1, Children: []int{1}},
		// 1: udClassDDO container (tag 268)
		{Scope: "open", Tag: 268, Parent: 0, Children: []int{2}},
		// 2: canonical declaration of ID 220 (no terminal in own subtree)
		{
			Scope: "open", Tag: -6, Parent: 1, Children: []int{3},
			Attributes: []HeapAttribute{{ID: HeapAttrIndex, Value: 21}, {ID: HeapAttrObjectID, Value: 220}},
		},
		// 3: parm leaf (not a terminal)
		{Scope: "leaf", Tag: int32(heap.ClassTagParm), Parent: 2},
	}}
	idx, _ := BuildHeapObjectIndex(tree)
	if got := WireTerminalAnchor(tree, idx, 220); got != 2 {
		t.Fatalf("WireTerminalAnchor(220) = %d, want 2 (the canonical declaration, not the sdfTun ancestor)", got)
	}
}

// sdfTun-children fallback: ID exists only as a stub leaf reference
// inside a sdfTun (no canonical declaration elsewhere).
func TestWireTerminalAnchorSdfTunChildrenFallback(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{
		// 0: sdfTun (terminal class)
		{Scope: "open", Tag: int32(heap.ClassTagSdfTun), Parent: -1, Children: []int{1}},
		// 1: stub LEAF arrayElement inside sdfTun, references ID 7777 only
		{
			Scope: "leaf", Tag: -6, Parent: 0,
			Attributes: []HeapAttribute{{ID: HeapAttrObjectID, Value: 7777}},
		},
	}}
	idx, _ := BuildHeapObjectIndex(tree)
	// Force objIdx to *not* contain 7777 (the LEAF stub got recorded —
	// drop it to simulate the "no canonical declaration" case).
	delete(idx, 7777)
	if got := WireTerminalAnchor(tree, idx, 7777); got != 0 {
		t.Fatalf("WireTerminalAnchor(7777) = %d, want 0 (the sdfTun via children scan)", got)
	}
}

// Truly missing: ID has no canonical, no walk-up terminal, no sdfTun
// child reference. Must return -1.
func TestWireTerminalAnchorReturnsMinusOneForUnknownID(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{
		{Scope: "open", Tag: -6, Parent: -1},
	}}
	idx, _ := BuildHeapObjectIndex(tree)
	if got := WireTerminalAnchor(tree, idx, 999); got != -1 {
		t.Fatalf("WireTerminalAnchor(unknown) = %d, want -1", got)
	}
}
