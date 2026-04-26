package render

import (
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

// boundsContent is a helper that encodes a Bounds rect as the
// 8-byte big-endian {Left, Top, Right, Bottom} payload pylabview
// writes to OF__bounds leaves (LVheap.py:1735).
func boundsContent(left, top, right, bottom int16) []byte {
	out := make([]byte, 8)
	put16 := func(off int, v int16) {
		out[off] = byte(uint16(v) >> 8)
		out[off+1] = byte(uint16(v))
	}
	put16(0, left)
	put16(2, top)
	put16(4, right)
	put16(6, bottom)
	return out
}

// A control whose OF__bounds child decodes successfully should be
// positioned and sized using the decoded rectangle, not the heuristic
// depth-based layout.
func TestProjectHeapTreeUsesDecodedBoundsForGroupNode(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{Tag: -3, Scope: "open", Parent: -1, Children: []int{1}},
			{
				Tag:     int32(heap.FieldTagBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: boundsContent(50, 30, 200, 100),
			},
		},
		Roots: []int{0},
	}

	scene := ProjectHeapTree(tree, ViewFrontPanel)

	root := scene.Nodes[scene.Roots[0]]
	if root.Kind != NodeKindGroup {
		t.Fatalf("root.Kind = %q, want %q", root.Kind, NodeKindGroup)
	}
	// scene viewBox margins offset everything by sceneMarginX / sceneMarginY,
	// so the decoded (50,30) lands at (50+24, 30+24) in scene coords.
	wantX, wantY := 50.0+sceneMarginX, 30.0+sceneMarginY
	wantW, wantH := 150.0, 70.0
	if root.Bounds.X != wantX || root.Bounds.Y != wantY {
		t.Errorf("root.Bounds origin = (%g,%g), want (%g,%g)",
			root.Bounds.X, root.Bounds.Y, wantX, wantY)
	}
	if root.Bounds.Width != wantW || root.Bounds.Height != wantH {
		t.Errorf("root.Bounds size = (%g x %g), want (%g x %g)",
			root.Bounds.Width, root.Bounds.Height, wantW, wantH)
	}
}

// The OF__bounds leaf itself carries metadata, not visible content.
// Once the parent has been positioned from it, the leaf should not
// also emit a separate label node.
func TestProjectHeapTreeOmitsOFBoundsLeafFromScene(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{Tag: -3, Scope: "open", Parent: -1, Children: []int{1}},
			{
				Tag:     int32(heap.FieldTagBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: boundsContent(0, 0, 100, 50),
			},
		},
		Roots: []int{0},
	}

	scene := ProjectHeapTree(tree, ViewFrontPanel)

	for _, n := range scene.Nodes {
		if n.HeapIndex == 1 && n.Kind == NodeKindLabel {
			t.Fatalf("scene emitted label node for OF__bounds leaf: %+v", n)
		}
	}
}

// Nodes without an OF__bounds child should still receive a heuristic
// layout, so partial bounds coverage gracefully degrades.
func TestProjectHeapTreeFallsBackToHeuristicWhenNoBoundsChild(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{Tag: -3, Scope: "open", Parent: -1, Children: []int{1}},
			{Tag: int32(heap.FieldTagBgColor), Scope: "leaf", Parent: 0},
		},
		Roots: []int{0},
	}

	scene := ProjectHeapTree(tree, ViewFrontPanel)

	root := scene.Nodes[scene.Roots[0]]
	if root.Bounds.Width <= 0 || root.Bounds.Height <= 0 {
		t.Fatalf("root.Bounds = %+v, want positive heuristic size", root.Bounds)
	}
	// Heuristic layout starts at the scene margin.
	if root.Bounds.X != sceneMarginX || root.Bounds.Y != sceneMarginY {
		t.Errorf("heuristic root origin = (%g,%g), want (%g,%g)",
			root.Bounds.X, root.Bounds.Y, sceneMarginX, sceneMarginY)
	}
}

// When at least one node lands on decoded bounds, the heuristic-layout
// warning should be relaxed so the demo no longer claims the entire
// scene is unreliable.
func TestProjectHeapTreeDropsHeuristicWarningWhenAllRootsHaveBounds(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{Tag: -3, Scope: "open", Parent: -1, Children: []int{1}},
			{
				Tag:     int32(heap.FieldTagBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: boundsContent(0, 0, 200, 100),
			},
		},
		Roots: []int{0},
	}

	scene := ProjectHeapTree(tree, ViewFrontPanel)
	for _, w := range scene.Warnings {
		if containsString([]string{w}, "Layout is heuristic") {
			t.Errorf("scene still warns %q despite all roots having decoded bounds", w)
		}
	}
}

// ViewBox should encompass decoded coordinate ranges so SVG/canvas
// rendering shows the real layout extents rather than clipping.
func TestProjectHeapTreeViewBoxEncompassesDecodedBounds(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{Tag: -3, Scope: "open", Parent: -1, Children: []int{1}},
			{
				Tag:     int32(heap.FieldTagBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: boundsContent(0, 0, 800, 600),
			},
		},
		Roots: []int{0},
	}

	scene := ProjectHeapTree(tree, ViewFrontPanel)
	wantW := 800.0 + 2*sceneMarginX
	wantH := 600.0 + 2*sceneMarginY
	if scene.ViewBox.Width < wantW {
		t.Errorf("ViewBox.Width = %g, want >= %g", scene.ViewBox.Width, wantW)
	}
	if scene.ViewBox.Height < wantH {
		t.Errorf("ViewBox.Height = %g, want >= %g", scene.ViewBox.Height, wantH)
	}
}
