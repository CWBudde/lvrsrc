package lvvi

import (
	"encoding/binary"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
)

// Bounds is the lvvi-level projection of a LabVIEW heap rectangle leaf:
// pylabview's HeapNodeRect (LVheap.py:1725). Bytes on disk are 4 x
// big-endian int16 in the order Left, Top, Right, Bottom, matching
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

// IsHeapRectTag reports whether tag is a known heap field whose leaf
// payload is a HeapNodeRect-style 8-byte rectangle.
func IsHeapRectTag(tag int32) bool {
	switch tag {
	case int32(heap.FieldTagBounds),
		int32(heap.FieldTagCallerGlyphBounds),
		int32(heap.FieldTagContRect),
		int32(heap.FieldTagDBounds),
		int32(heap.FieldTagDocBounds),
		int32(heap.FieldTagDynBounds),
		int32(heap.FieldTagGrowAreaBounds),
		int32(heap.FieldTagHoodBounds),
		int32(heap.FieldTagIconBounds),
		int32(heap.FieldTagPBounds),
		int32(heap.FieldTagSizeRect),
		int32(heap.FieldTagSubVIGlyphBounds),
		int32(heap.FieldTagTermBounds),
		int32(heap.FieldTagTotalBounds),
		int32(heap.FieldTagStateBounds),
		int32(heap.FieldTagIntensityGraphBounds):
		return true
	default:
		return false
	}
}

// HeapRect decodes any known rectangle-shaped heap leaf at
// tree.Nodes[nodeIdx]. Returns ok=false when the index is out of range,
// the tag is not registered as a rectangle tag, or the payload length is
// not exactly 8 bytes.
func HeapRect(tree HeapTree, nodeIdx int) (Bounds, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return Bounds{}, false
	}
	n := tree.Nodes[nodeIdx]
	if !IsHeapRectTag(n.Tag) {
		return Bounds{}, false
	}
	return decodeBoundsContent(n.Content)
}

// HeapRectForTag decodes a rectangle leaf only when tree.Nodes[nodeIdx]
// has the requested rectangle tag. Use this when a caller needs a
// specific rectangle role rather than "any known rectangle".
func HeapRectForTag(tree HeapTree, nodeIdx int, tag int32) (Bounds, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return Bounds{}, false
	}
	n := tree.Nodes[nodeIdx]
	if n.Tag != tag || !IsHeapRectTag(tag) {
		return Bounds{}, false
	}
	return decodeBoundsContent(n.Content)
}

// FindRectChild walks the children of tree.Nodes[parentIdx] and returns
// the first decodable child carrying the requested rectangle tag.
func FindRectChild(tree HeapTree, parentIdx int, tag int32) (Bounds, bool) {
	if parentIdx < 0 || parentIdx >= len(tree.Nodes) || !IsHeapRectTag(tag) {
		return Bounds{}, false
	}
	parent := tree.Nodes[parentIdx]
	for _, ci := range parent.Children {
		if ci < 0 || ci >= len(tree.Nodes) {
			continue
		}
		if tree.Nodes[ci].Tag != tag {
			continue
		}
		if b, ok := HeapRectForTag(tree, ci, tag); ok {
			return b, true
		}
	}
	return Bounds{}, false
}

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
	return HeapRectForTag(tree, nodeIdx, int32(heap.FieldTagBounds))
}

// FindBoundsChild walks the children of tree.Nodes[parentIdx] looking
// for an OF__bounds leaf and returns its decoded Bounds. Returns
// ok=false when the parent has no decodable OF__bounds child.
//
// Heap controls carry their layout rectangle as a sibling tag of their
// other field children — this helper is the natural lookup for callers
// (scene projection, demo) that need to position a control.
func FindBoundsChild(tree HeapTree, parentIdx int) (Bounds, bool) {
	return FindRectChild(tree, parentIdx, int32(heap.FieldTagBounds))
}

func decodeBoundsContent(c []byte) (Bounds, bool) {
	if len(c) != 8 {
		return Bounds{}, false
	}
	return Bounds{
		Left:   int16(binary.BigEndian.Uint16(c[0:2])),
		Top:    int16(binary.BigEndian.Uint16(c[2:4])),
		Right:  int16(binary.BigEndian.Uint16(c[4:6])),
		Bottom: int16(binary.BigEndian.Uint16(c[6:8])),
	}, true
}
