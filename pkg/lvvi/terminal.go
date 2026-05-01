package lvvi

import (
	"encoding/binary"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
)

// Point is the lvvi-level projection of an OF__termHotPoint heap leaf:
// a Mac-style Point with vertical-before-horizontal byte order. The
// 4-byte payload is decoded as 2 × big-endian int16 in the order V, H.
//
// pylabview's `LVheap.py` does not carry a typed decoder for this tag;
// the V/H ordering follows the classic Mac toolbox Point convention
// shared by other LabVIEW geometry payloads we've inspected
// (OF__bounds, OF__termBounds — both Mac Rect = top, left, bottom,
// right). A single sample from the corpus (`0000fffc`) decodes to
// `{V: 0, H: -4}`, which matches the typical "anchor offset from the
// terminal's outer rect origin" interpretation.
type Point struct {
	V, H int16
}

// HeapTermBounds decodes an OF__termBounds heap leaf at
// tree.Nodes[nodeIdx]. The 8-byte payload is identical in shape to
// OF__bounds (4 × big-endian int16, Left/Top/Right/Bottom), so the
// decoded value reuses the existing Bounds type.
//
// Returns ok=false on out-of-range index, wrong tag, or wrong byte
// length — same contract as HeapBounds.
func HeapTermBounds(tree HeapTree, nodeIdx int) (Bounds, bool) {
	return HeapRectForTag(tree, nodeIdx, int32(heap.FieldTagTermBounds))
}

// HeapTermHotPoint decodes an OF__termHotPoint heap leaf at
// tree.Nodes[nodeIdx]. The 4-byte payload is 2 × big-endian int16 in
// V, H order (Mac Point convention).
func HeapTermHotPoint(tree HeapTree, nodeIdx int) (Point, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return Point{}, false
	}
	n := tree.Nodes[nodeIdx]
	if n.Tag != int32(heap.FieldTagTermHotPoint) {
		return Point{}, false
	}
	if len(n.Content) != 4 {
		return Point{}, false
	}
	c := n.Content
	return Point{
		V: int16(binary.BigEndian.Uint16(c[0:2])),
		H: int16(binary.BigEndian.Uint16(c[2:4])),
	}, true
}

// FindTermBoundsChild walks the children of tree.Nodes[parentIdx] and
// returns the first decoded OF__termBounds leaf. Mirrors
// FindBoundsChild for the terminal-rect case; tunnel / terminal class
// nodes carry their geometry as a sibling field tag.
func FindTermBoundsChild(tree HeapTree, parentIdx int) (Bounds, bool) {
	return FindRectChild(tree, parentIdx, int32(heap.FieldTagTermBounds))
}

// FindTermHotPointChild walks the children of tree.Nodes[parentIdx]
// and returns the first decoded OF__termHotPoint leaf.
func FindTermHotPointChild(tree HeapTree, parentIdx int) (Point, bool) {
	if parentIdx < 0 || parentIdx >= len(tree.Nodes) {
		return Point{}, false
	}
	parent := tree.Nodes[parentIdx]
	for _, ci := range parent.Children {
		if ci < 0 || ci >= len(tree.Nodes) {
			continue
		}
		if tree.Nodes[ci].Tag != int32(heap.FieldTagTermHotPoint) {
			continue
		}
		if p, ok := HeapTermHotPoint(tree, ci); ok {
			return p, true
		}
	}
	return Point{}, false
}
