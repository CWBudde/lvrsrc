package render

import (
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

// A heap node carrying a known control class should propagate its
// widget kind onto the rendered group and the inner box, so the SVG
// renderer can pick a generic skin per kind.
func TestProjectHeapTreeStampsWidgetKindOnGroupAndBox(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{
				Tag:      int32(heap.ClassTagStdBool),
				Scope:    "open",
				Parent:   -1,
				Children: []int{1},
			},
			{
				Tag:     int32(heap.FieldTagBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: boundsContent(0, 0, 60, 30),
			},
		},
		Roots: []int{0},
	}

	scene := ProjectHeapTree(tree, ViewFrontPanel)

	if len(scene.Roots) != 1 {
		t.Fatalf("len(scene.Roots) = %d, want 1", len(scene.Roots))
	}
	root := scene.Nodes[scene.Roots[0]]
	if root.Kind != NodeKindGroup {
		t.Fatalf("root.Kind = %q, want %q", root.Kind, NodeKindGroup)
	}
	if root.WidgetKind != lvvi.WidgetKindBoolean {
		t.Errorf("root.WidgetKind = %q, want %q", root.WidgetKind, lvvi.WidgetKindBoolean)
	}

	var saw bool
	for _, ci := range root.Children {
		c := scene.Nodes[ci]
		if c.Kind == NodeKindBox {
			saw = true
			if c.WidgetKind != lvvi.WidgetKindBoolean {
				t.Errorf("box.WidgetKind = %q, want %q", c.WidgetKind, lvvi.WidgetKindBoolean)
			}
		}
	}
	if !saw {
		t.Fatal("group emitted no NodeKindBox child")
	}
}

// Unmapped classes (e.g. SL__dataObj) should still produce a node in
// the scene, but the WidgetKind should be Other so the renderer can
// distinguish them from classified widgets.
func TestProjectHeapTreeUnmappedClassUsesOtherKind(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{
				Tag:      int32(heap.ClassTagDataObj),
				Scope:    "open",
				Parent:   -1,
				Children: []int{1},
			},
			{
				Tag:     int32(heap.FieldTagBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: boundsContent(0, 0, 80, 40),
			},
		},
		Roots: []int{0},
	}

	scene := ProjectHeapTree(tree, ViewFrontPanel)
	root := scene.Nodes[scene.Roots[0]]
	if root.WidgetKind != lvvi.WidgetKindOther {
		t.Errorf("root.WidgetKind = %q, want %q", root.WidgetKind, lvvi.WidgetKindOther)
	}
}
