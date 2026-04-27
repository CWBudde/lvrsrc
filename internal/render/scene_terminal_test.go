package render

import (
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

func termBoundsContent(left, top, right, bottom int16) []byte {
	return boundsContent(left, top, right, bottom)
}

func termHotPointContent(v, h int16) []byte {
	out := make([]byte, 4)
	out[0] = byte(uint16(v) >> 8)
	out[1] = byte(uint16(v))
	out[2] = byte(uint16(h) >> 8)
	out[3] = byte(uint16(h))
	return out
}

// A tunnel class (e.g. SL__simTun) with an OF__termBounds child should
// emit a single NodeKindTerminal with Bounds taken from termBounds and
// the WidgetKind=Terminal stamp — not the usual Group / Box /
// title-label triple.
func TestProjectHeapTreeTerminalEmitsFlatTerminalNode(t *testing.T) {
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
				Content: termBoundsContent(40, 60, 48, 68),
			},
		},
		Roots: []int{0},
	}

	scene := ProjectHeapTree(tree, ViewBlockDiagram)

	if len(scene.Roots) != 1 {
		t.Fatalf("len(scene.Roots) = %d, want 1", len(scene.Roots))
	}
	root := scene.Nodes[scene.Roots[0]]
	if root.Kind != NodeKindTerminal {
		t.Fatalf("root.Kind = %q, want %q", root.Kind, NodeKindTerminal)
	}
	if root.WidgetKind != lvvi.WidgetKindTerminal {
		t.Errorf("root.WidgetKind = %q, want %q", root.WidgetKind, lvvi.WidgetKindTerminal)
	}
	wantX := 40.0 + sceneMarginX
	wantY := 60.0 + sceneMarginY
	if root.Bounds.X != wantX || root.Bounds.Y != wantY {
		t.Errorf("root.Bounds origin = (%g,%g), want (%g,%g)",
			root.Bounds.X, root.Bounds.Y, wantX, wantY)
	}
	if root.Bounds.Width != 8 || root.Bounds.Height != 8 {
		t.Errorf("root.Bounds size = (%g x %g), want (8 x 8)",
			root.Bounds.Width, root.Bounds.Height)
	}

	// A flat terminal node has no group/box/title-label children.
	if len(root.Children) != 0 {
		t.Errorf("root.Children = %v, want empty for terminal", root.Children)
	}
}

// When OF__termHotPoint is present, the terminal's Anchor should be
// set to (bounds.Left + H, bounds.Top + V) in scene coords. pylabview
// stores the hot-point as an offset from the terminal's origin in
// Mac Point V/H order.
func TestProjectHeapTreeTerminalUsesHotPointForAnchor(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{
				Tag:      int32(heap.ClassTagSeqTun),
				Scope:    "open",
				Parent:   -1,
				Children: []int{1, 2},
			},
			{
				Tag:     int32(heap.FieldTagTermBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: termBoundsContent(100, 50, 116, 66),
			},
			{
				Tag:     int32(heap.FieldTagTermHotPoint),
				Scope:   "leaf",
				Parent:  0,
				Content: termHotPointContent(8, 0), // V=8, H=0
			},
		},
		Roots: []int{0},
	}

	scene := ProjectHeapTree(tree, ViewBlockDiagram)
	root := scene.Nodes[scene.Roots[0]]
	wantAnchorX := 100.0 + 0.0 + sceneMarginX
	wantAnchorY := 50.0 + 8.0 + sceneMarginY
	if root.Anchor.X != wantAnchorX || root.Anchor.Y != wantAnchorY {
		t.Errorf("root.Anchor = (%g,%g), want (%g,%g)",
			root.Anchor.X, root.Anchor.Y, wantAnchorX, wantAnchorY)
	}
}

// Without an OF__termHotPoint child, the terminal anchor falls back
// to the centre of its bounds, so wires (12.5) always have a connect
// point even on terminals that didn't record a hot-point.
func TestProjectHeapTreeTerminalAnchorDefaultsToBoundsCenter(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{
				Tag:      int32(heap.ClassTagSdfTun),
				Scope:    "open",
				Parent:   -1,
				Children: []int{1},
			},
			{
				Tag:     int32(heap.FieldTagTermBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: termBoundsContent(0, 0, 10, 6),
			},
		},
		Roots: []int{0},
	}
	scene := ProjectHeapTree(tree, ViewBlockDiagram)
	root := scene.Nodes[scene.Roots[0]]
	wantX := 5.0 + sceneMarginX
	wantY := 3.0 + sceneMarginY
	if root.Anchor.X != wantX || root.Anchor.Y != wantY {
		t.Errorf("root.Anchor = (%g,%g), want (%g,%g)",
			root.Anchor.X, root.Anchor.Y, wantX, wantY)
	}
}

// OF__termBounds leaves carry the terminal's geometry only; once the
// terminal has been positioned from them, the leaf must not also
// emit a separate label node.
func TestProjectHeapTreeTerminalDropsTermBoundsLeaf(t *testing.T) {
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
				Content: termBoundsContent(0, 0, 8, 8),
			},
			{
				Tag:     int32(heap.FieldTagTermHotPoint),
				Scope:   "leaf",
				Parent:  0,
				Content: termHotPointContent(0, 0),
			},
		},
		Roots: []int{0},
	}
	scene := ProjectHeapTree(tree, ViewBlockDiagram)
	for _, n := range scene.Nodes {
		if n.HeapIndex == 1 || n.HeapIndex == 2 {
			if n.Kind == NodeKindLabel {
				t.Errorf("scene emitted label for promoted terminal leaf: %+v", n)
			}
		}
	}
}
