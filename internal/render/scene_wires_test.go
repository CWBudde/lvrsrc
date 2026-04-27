package render

import (
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

// A BD heap tree that contains compressed wire-table chunks should
// surface a scene warning telling the demo how many were recorded.
// This is the 12.4a presence signal — connectivity is still opaque.
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
		if strings.Contains(w, "compressed wire-table") && strings.Contains(w, "2") {
			saw = true
			break
		}
	}
	if !saw {
		t.Errorf("expected compressed-wire-table warning with count 2, got %v", scene.Warnings)
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
		if strings.Contains(w, "compressed wire-table") {
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
		if strings.Contains(w, "compressed wire-table") {
			t.Errorf("BD scene without wire-table chunks emitted count warning: %q", w)
		}
	}
}
