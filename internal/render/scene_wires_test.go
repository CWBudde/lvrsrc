package render

import (
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

// A BD heap tree that contains wire-table chunks should surface a
// scene warning enumerating the wire-network count and the per-mode
// breakdown — Phase 13 replaced the older "compressed
// wire-table chunks" wording with the proper "wire networks"
// terminology after the spike confirmed that one chunk encodes a
// whole network (not an edge).
func TestProjectHeapTreeAddsWireTableWarning(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{Tag: -3, Scope: "open", Parent: -1, Children: []int{1, 2}},
			{
				Tag:     int32(heap.FieldTagCompressedWireTable),
				Scope:   "leaf",
				Parent:  0,
				Content: []byte{0x02, 0x08},
			},
			{
				Tag:     int32(heap.FieldTagCompressedWireTable),
				Scope:   "leaf",
				Parent:  0,
				Content: []byte{0x03, 0x08, 0x00, 0x1b},
			},
		},
		Roots: []int{0},
	}
	scene := ProjectHeapTree(tree, ViewBlockDiagram)
	saw := false
	for _, w := range scene.Warnings {
		if strings.Contains(w, "wire networks") &&
			strings.Contains(w, "2 auto-routed") {
			saw = true
			break
		}
	}
	if !saw {
		t.Errorf("expected wire-networks warning with auto-routed count 2, got %v", scene.Warnings)
	}
}

// Front-panel scenes don't carry wire data, so they must not emit
// the BD-only wire warnings even if (synthetically) a wire-table
// leaf were present.
func TestProjectHeapTreeFrontPanelSuppressesWireWarning(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{Tag: -3, Scope: "open", Parent: -1},
		},
		Roots: []int{0},
	}
	scene := ProjectHeapTree(tree, ViewFrontPanel)
	for _, w := range scene.Warnings {
		if strings.Contains(w, "wire networks") {
			t.Errorf("front-panel scene emitted wire warning: %q", w)
		}
		if strings.Contains(w, "wire routing") || strings.Contains(w, "terminal positions") {
			t.Errorf("front-panel scene emitted BD-only warning: %q", w)
		}
	}
}

// A BD scene without wire-table chunks should still get the
// "wires not rendered yet" warning, but not the count-bearing one.
func TestProjectHeapTreeBlockDiagramWithoutWireTablesOmitsCountWarning(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{Tag: -3, Scope: "open", Parent: -1},
		},
		Roots: []int{0},
	}
	scene := ProjectHeapTree(tree, ViewBlockDiagram)
	for _, w := range scene.Warnings {
		if strings.Contains(w, "wire networks") {
			t.Errorf("BD scene without wire-table chunks emitted count warning: %q", w)
		}
	}
}

func TestProjectHeapTreeRendersAutoChainWire(t *testing.T) {
	// The wire references terminals by heap-object ID through a
	// termList container, mirroring the pattern verified across the
	// real corpus. Each SimTun terminal is wrapped in a canonical
	// OPEN arrayElement that declares its ID via attribute -3.
	tree := buildBDWireTree(bdWireSpec{
		terminals: []terminalSpec{
			{id: 51, bounds: rectBounds{10, 20, 18, 28}, hotPoint: hotPoint{4, 8}},
			{id: 95, bounds: rectBounds{100, 28, 108, 36}, hotPoint: hotPoint{4, 0}},
		},
		wireIDs:     []lvvi.HeapObjectID{51, 95},
		wireContent: []byte{0x04, 0x08, 0x00, 0x00, 0x20, 0x08},
	})

	scene := ProjectHeapTree(tree, ViewBlockDiagram)
	if len(scene.Wires) != 1 {
		t.Fatalf("len(scene.Wires) = %d, want 1, warnings=%v", len(scene.Wires), scene.Warnings)
	}
	// Expected scene anchors come from the wrapped synthetic layout
	// (each terminal sits inside its own arrayElement wrapper plus a
	// SimTun container, so absolute pixel positions differ from a bare
	// SimTun tree). The key property checked here is that the wire's
	// endpoints are the source/sink terminal anchors and the elbow
	// path follows ChainAutoPath{SourceAnchorX:32, YStep:8} from the
	// wireContent.
	w := scene.Wires[0]
	if len(w.Points) < 2 {
		t.Fatalf("wire has %d points, want >=2", len(w.Points))
	}
	if w.From == w.To {
		t.Fatalf("wire from == to (%d) — terminals collapsed onto same scene node", w.From)
	}
	if scene.Nodes[w.From].HeapIndex == scene.Nodes[w.To].HeapIndex {
		t.Fatalf("wire endpoints share heap index — terminals not distinct")
	}
	if w.Points[0] != scene.Nodes[w.From].Anchor {
		t.Fatalf("wire start %+v != source anchor %+v", w.Points[0], scene.Nodes[w.From].Anchor)
	}
	if w.Points[len(w.Points)-1] != scene.Nodes[w.To].Anchor {
		t.Fatalf("wire end %+v != sink anchor %+v", w.Points[len(w.Points)-1], scene.Nodes[w.To].Anchor)
	}
	for _, w := range scene.Warnings {
		if strings.Contains(w, "not yet rendered") {
			t.Fatalf("rendered auto-chain scene kept broad no-wire warning: %q", w)
		}
	}
}

func TestProjectHeapTreeRendersPureTreeWireEndpoints(t *testing.T) {
	tree := buildBDWireTree(bdWireSpec{
		terminals: []terminalSpec{
			{id: 51, bounds: rectBounds{0, 0, 8, 8}},
			{id: 95, bounds: rectBounds{100, 0, 108, 8}},
		},
		wireIDs:     []lvvi.HeapObjectID{51, 95},
		wireContent: []byte{0x06, 0x00, 0x08, 0x07, 0x00, 0x03, 0x00, 0x41, 0x31, 0x44, 0x2d, 0x42},
	})

	scene := ProjectHeapTree(tree, ViewBlockDiagram)
	if len(scene.Wires) != 2 {
		t.Fatalf("len(scene.Wires) = %d, want one branch per endpoint, warnings=%v", len(scene.Wires), scene.Warnings)
	}
	// Both branches must originate at the same junction point (the
	// shared start of every Wire emitted by tree-mode rendering); the
	// two branch endpoints must differ.
	if scene.Wires[0].Points[0] != scene.Wires[1].Points[0] {
		t.Fatalf("tree branches do not share a junction: %+v vs %+v",
			scene.Wires[0].Points[0], scene.Wires[1].Points[0])
	}
	endA := scene.Wires[0].Points[len(scene.Wires[0].Points)-1]
	endB := scene.Wires[1].Points[len(scene.Wires[1].Points)-1]
	if endA == endB {
		t.Fatalf("tree branch endpoints collapsed onto same point: %+v", endA)
	}
}

// A wire whose termList ID has no canonical declaration anywhere in
// the BD tree must NOT render against an arbitrary terminal — that was
// the failure mode that pushed the project into Phase 16.4 work.
func TestProjectHeapTreeSkipsWireWithUnresolvedTerminals(t *testing.T) {
	// Terminal IDs in the canonical declarations: 51, 95. The wire
	// references 51 and 7777 — the latter is dangling.
	tree := buildBDWireTree(bdWireSpec{
		terminals: []terminalSpec{
			{id: 51, bounds: rectBounds{10, 20, 18, 28}, hotPoint: hotPoint{4, 8}},
			{id: 95, bounds: rectBounds{100, 28, 108, 36}, hotPoint: hotPoint{4, 0}},
		},
		wireIDs:     []lvvi.HeapObjectID{51, 7777},
		wireContent: []byte{0x04, 0x08, 0x00, 0x00, 0x20, 0x08},
	})
	scene := ProjectHeapTree(tree, ViewBlockDiagram)
	if len(scene.Wires) != 0 {
		t.Fatalf("len(scene.Wires) = %d, want 0 for unresolvable wire", len(scene.Wires))
	}
	saw := false
	for _, w := range scene.Warnings {
		if strings.Contains(w, "skipped") {
			saw = true
			break
		}
	}
	if !saw {
		t.Fatalf("expected skip warning, got %v", scene.Warnings)
	}
}

type rectBounds struct{ left, top, right, bottom int16 }
type hotPoint struct{ h, v int16 }

type terminalSpec struct {
	id       lvvi.HeapObjectID
	bounds   rectBounds
	hotPoint hotPoint
}

type bdWireSpec struct {
	terminals   []terminalSpec
	wireIDs     []lvvi.HeapObjectID
	wireContent []byte
}

// buildBDWireTree constructs a HeapTree mirroring the per-wire shape
// observed in real corpus BD heaps:
//
//	arrayElement open {-2 N, -3 ID}              <- canonical terminal wrappers
//	  SL__simTun open
//	    OF__termBounds leaf
//	    OF__termHotPoint leaf  (optional)
//	... (one wrapper per terminal) ...
//	arrayElement open                            <- wire-bearing parent
//	  termList open (tag 268) {-5 N}
//	    arrayElement leaf {-3 ID}                <- one per wire endpoint ID
//	    ...
//	  compressedWireTable leaf
func buildBDWireTree(spec bdWireSpec) lvvi.HeapTree {
	var nodes []lvvi.HeapNode
	add := func(n lvvi.HeapNode) int {
		nodes = append(nodes, n)
		return len(nodes) - 1
	}

	for _, term := range spec.terminals {
		wrapperIdx := add(lvvi.HeapNode{
			Tag:    -6,
			Scope:  "open",
			Parent: -1,
			Attributes: []lvvi.HeapAttribute{
				{ID: lvvi.HeapAttrIndex, Value: 21},
				{ID: lvvi.HeapAttrObjectID, Value: int32(term.id)},
			},
		})
		simTunIdx := add(lvvi.HeapNode{
			Tag:    int32(heap.ClassTagSimTun),
			Scope:  "open",
			Parent: wrapperIdx,
		})
		boundsIdx := add(lvvi.HeapNode{
			Tag:     int32(heap.FieldTagTermBounds),
			Scope:   "leaf",
			Parent:  simTunIdx,
			Content: termBoundsContent(term.bounds.left, term.bounds.top, term.bounds.right, term.bounds.bottom),
		})
		simTunChildren := []int{boundsIdx}
		if term.hotPoint != (hotPoint{}) {
			hpIdx := add(lvvi.HeapNode{
				Tag:     int32(heap.FieldTagTermHotPoint),
				Scope:   "leaf",
				Parent:  simTunIdx,
				Content: termHotPointContent(term.hotPoint.h, term.hotPoint.v),
			})
			simTunChildren = append(simTunChildren, hpIdx)
		}
		nodes[simTunIdx].Children = simTunChildren
		nodes[wrapperIdx].Children = []int{simTunIdx}
	}

	wireParentIdx := add(lvvi.HeapNode{
		Tag:    -6,
		Scope:  "open",
		Parent: -1,
	})
	termListIdx := add(lvvi.HeapNode{
		Tag:        int32(heap.FieldTagTermList),
		Scope:      "open",
		Parent:     wireParentIdx,
		Attributes: []lvvi.HeapAttribute{{ID: lvvi.HeapAttrChildCount, Value: int32(len(spec.wireIDs))}},
	})
	leafChildren := make([]int, 0, len(spec.wireIDs))
	for _, id := range spec.wireIDs {
		leafIdx := add(lvvi.HeapNode{
			Tag:        -6,
			Scope:      "leaf",
			Parent:     termListIdx,
			Attributes: []lvvi.HeapAttribute{{ID: lvvi.HeapAttrObjectID, Value: int32(id)}},
		})
		leafChildren = append(leafChildren, leafIdx)
	}
	nodes[termListIdx].Children = leafChildren
	wireIdx := add(lvvi.HeapNode{
		Tag:     int32(heap.FieldTagCompressedWireTable),
		Scope:   "leaf",
		Parent:  wireParentIdx,
		Content: spec.wireContent,
	})
	nodes[wireParentIdx].Children = []int{termListIdx, wireIdx}

	var roots []int
	for i, n := range nodes {
		if n.Parent == -1 {
			roots = append(roots, i)
		}
	}
	return lvvi.HeapTree{Nodes: nodes, Roots: roots}
}
