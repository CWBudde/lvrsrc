package lvvi

import (
	"encoding/binary"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
)

// PointValue is a heap-node coordinate/size pair: pylabview's
// HeapNodePoint (LVheap.py:1765). The payload is 2 x big-endian int16
// in binary X, Y order.
//
// This is intentionally separate from Point, whose V/H field names are
// used by the terminal and wire helpers that model Mac-style points.
type PointValue struct {
	Tag  int32
	X, Y int16
}

// IsHeapPointTag reports whether tag is a known heap field whose leaf
// payload is a HeapNodePoint-style 4-byte coordinate/size pair.
func IsHeapPointTag(tag int32) bool {
	switch tag {
	case int32(heap.FieldTagMaxPaneSize),
		int32(heap.FieldTagMaxPanelSize),
		int32(heap.FieldTagMinButSize),
		int32(heap.FieldTagMinPaneSize),
		int32(heap.FieldTagMinPanelSize),
		int32(heap.FieldTagOrigin):
		return true
	default:
		return false
	}
}

// HeapPoint decodes any known point-shaped heap leaf at
// tree.Nodes[nodeIdx]. Returns ok=false when the index is out of range,
// the tag is not registered as a point tag, or the payload length is not
// exactly 4 bytes.
func HeapPoint(tree HeapTree, nodeIdx int) (PointValue, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return PointValue{}, false
	}
	n := tree.Nodes[nodeIdx]
	if !IsHeapPointTag(n.Tag) {
		return PointValue{}, false
	}
	return decodePointContent(n.Tag, n.Content)
}

// HeapPointForTag decodes a point leaf only when tree.Nodes[nodeIdx]
// has the requested point tag.
func HeapPointForTag(tree HeapTree, nodeIdx int, tag int32) (PointValue, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return PointValue{}, false
	}
	n := tree.Nodes[nodeIdx]
	if n.Tag != tag || !IsHeapPointTag(tag) {
		return PointValue{}, false
	}
	return decodePointContent(tag, n.Content)
}

// FindPointChild walks the children of tree.Nodes[parentIdx] and
// returns the first decodable child carrying the requested point tag.
func FindPointChild(tree HeapTree, parentIdx int, tag int32) (PointValue, bool) {
	if parentIdx < 0 || parentIdx >= len(tree.Nodes) || !IsHeapPointTag(tag) {
		return PointValue{}, false
	}
	parent := tree.Nodes[parentIdx]
	for _, ci := range parent.Children {
		if ci < 0 || ci >= len(tree.Nodes) || tree.Nodes[ci].Tag != tag {
			continue
		}
		if p, ok := HeapPointForTag(tree, ci, tag); ok {
			return p, true
		}
	}
	return PointValue{}, false
}

func decodePointContent(tag int32, content []byte) (PointValue, bool) {
	if len(content) != 4 {
		return PointValue{}, false
	}
	return PointValue{
		Tag: tag,
		X:   int16(binary.BigEndian.Uint16(content[0:2])),
		Y:   int16(binary.BigEndian.Uint16(content[2:4])),
	}, true
}
