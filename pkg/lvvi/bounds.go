package lvvi

import (
	"encoding/binary"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
)

// Bounds is the lvvi-level projection of an OF__bounds heap leaf:
// pylabview's HeapNodeRect (LVheap.py:1725). Bytes on disk are 4 ×
// big-endian int16 in the order Left, Top, Right, Bottom — matching
// internal/codecs/heap.Rect.
//
// All four fields are signed: corpus VIs do contain negative
// coordinates (e.g. controls scrolled off-pane).
type Bounds struct {
	Left, Top, Right, Bottom int16
}

// Width is Right - Left. Returns a non-negative value for well-formed
// rects; pylabview does not enforce ordering, so callers that need
// absolute extents should clamp at the call site.
func (b Bounds) Width() int16 { return b.Right - b.Left }

// Height is Bottom - Top.
func (b Bounds) Height() int16 { return b.Bottom - b.Top }

// HeapBounds decodes an OF__bounds heap leaf at tree.Nodes[nodeIdx].
//
// Returns ok=false when:
//   - nodeIdx is out of range,
//   - the node's tag is not FieldTagBounds (14), or
//   - the node's content is not exactly 8 bytes.
//
// On success the returned Bounds carries the four signed int16 corners
// in pylabview's Left/Top/Right/Bottom order.
func HeapBounds(tree HeapTree, nodeIdx int) (Bounds, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return Bounds{}, false
	}
	n := tree.Nodes[nodeIdx]
	if n.Tag != int32(heap.FieldTagBounds) {
		return Bounds{}, false
	}
	if len(n.Content) != 8 {
		return Bounds{}, false
	}
	c := n.Content
	return Bounds{
		Left:   int16(binary.BigEndian.Uint16(c[0:2])),
		Top:    int16(binary.BigEndian.Uint16(c[2:4])),
		Right:  int16(binary.BigEndian.Uint16(c[4:6])),
		Bottom: int16(binary.BigEndian.Uint16(c[6:8])),
	}, true
}

// FindBoundsChild walks the children of tree.Nodes[parentIdx] looking
// for an OF__bounds leaf and returns its decoded Bounds. Returns
// ok=false when the parent has no decodable OF__bounds child.
//
// Heap controls carry their layout rectangle as a sibling tag of their
// other field children — this helper is the natural lookup for callers
// (scene projection, demo) that need to position a control.
func FindBoundsChild(tree HeapTree, parentIdx int) (Bounds, bool) {
	if parentIdx < 0 || parentIdx >= len(tree.Nodes) {
		return Bounds{}, false
	}
	parent := tree.Nodes[parentIdx]
	for _, ci := range parent.Children {
		if ci < 0 || ci >= len(tree.Nodes) {
			continue
		}
		if tree.Nodes[ci].Tag != int32(heap.FieldTagBounds) {
			continue
		}
		if b, ok := HeapBounds(tree, ci); ok {
			return b, true
		}
	}
	return Bounds{}, false
}
