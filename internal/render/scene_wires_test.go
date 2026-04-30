package render

import (
	"reflect"
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
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{
				Tag:      int32(heap.ClassTagSimTun),
				Scope:    "open",
				Parent:   -1,
				Children: []int{1, 2},
			},
			{
				Tag:     int32(heap.FieldTagTermBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: termBoundsContent(10, 20, 18, 28),
			},
			{
				Tag:     int32(heap.FieldTagTermHotPoint),
				Scope:   "leaf",
				Parent:  0,
				Content: termHotPointContent(4, 8),
			},
			{
				Tag:      int32(heap.ClassTagSimTun),
				Scope:    "open",
				Parent:   -1,
				Children: []int{4, 5},
			},
			{
				Tag:     int32(heap.FieldTagTermBounds),
				Scope:   "leaf",
				Parent:  3,
				Content: termBoundsContent(100, 28, 108, 36),
			},
			{
				Tag:     int32(heap.FieldTagTermHotPoint),
				Scope:   "leaf",
				Parent:  3,
				Content: termHotPointContent(4, 0),
			},
			{
				Tag:     int32(heap.FieldTagCompressedWireTable),
				Scope:   "leaf",
				Parent:  -1,
				Content: []byte{0x04, 0x08, 0x00, 0x00, 0x20, 0x08},
			},
		},
		Roots: []int{0, 3, 6},
	}

	scene := ProjectHeapTree(tree, ViewBlockDiagram)
	if len(scene.Wires) != 1 {
		t.Fatalf("len(scene.Wires) = %d, want 1", len(scene.Wires))
	}
	want := []Point{
		{X: 42, Y: 48},
		{X: 74, Y: 48},
		{X: 74, Y: 56},
		{X: 124, Y: 56},
	}
	if !reflect.DeepEqual(scene.Wires[0].Points, want) {
		t.Fatalf("wire points = %+v, want %+v", scene.Wires[0].Points, want)
	}
	for _, w := range scene.Warnings {
		if strings.Contains(w, "not yet rendered") {
			t.Fatalf("rendered auto-chain scene kept broad no-wire warning: %q", w)
		}
	}
}

func TestProjectHeapTreeRendersPureTreeWireEndpoints(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{
				Tag:      int32(heap.ClassTagSimTun),
				Scope:    "open",
				Parent:   -1,
				Children: []int{1},
			},
			{
				Tag:     int32(heap.FieldTagTermBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: termBoundsContent(0, 0, 8, 8),
			},
			{
				Tag:      int32(heap.ClassTagSimTun),
				Scope:    "open",
				Parent:   -1,
				Children: []int{3},
			},
			{
				Tag:     int32(heap.FieldTagTermBounds),
				Scope:   "leaf",
				Parent:  2,
				Content: termBoundsContent(100, 0, 108, 8),
			},
			{
				Tag:     int32(heap.FieldTagCompressedWireTable),
				Scope:   "leaf",
				Parent:  -1,
				Content: []byte{0x06, 0x00, 0x08, 0x07, 0x00, 0x03, 0x00, 0x41, 0x31, 0x44, 0x2d, 0x42},
			},
		},
		Roots: []int{0, 2, 4},
	}

	scene := ProjectHeapTree(tree, ViewBlockDiagram)
	if len(scene.Wires) != 2 {
		t.Fatalf("len(scene.Wires) = %d, want one branch per endpoint", len(scene.Wires))
	}
	wantA := []Point{{X: 90, Y: 71}, {X: 92, Y: 71}, {X: 92, Y: 73}}
	wantB := []Point{{X: 90, Y: 71}, {X: 90, Y: 69}}
	if !reflect.DeepEqual(scene.Wires[0].Points, wantA) {
		t.Fatalf("branch A = %+v, want %+v", scene.Wires[0].Points, wantA)
	}
	if !reflect.DeepEqual(scene.Wires[1].Points, wantB) {
		t.Fatalf("branch B = %+v, want %+v", scene.Wires[1].Points, wantB)
	}
}
