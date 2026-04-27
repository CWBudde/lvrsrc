package render

import (
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

// A BD heap tree that contains wire-table chunks should surface a
// scene warning enumerating the wire-network count and the per-mode
// breakdown — Phase 12.4b₁ replaced the older "compressed
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
