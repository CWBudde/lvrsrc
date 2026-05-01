package lvvi

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
)

// ScalarValue is a round-trip-safe numeric projection of common scalar
// heap fields. Raw always preserves the original payload bytes; Signed
// and Unsigned are big-endian interpretations of the same bytes.
type ScalarValue struct {
	Tag      int32
	Width    int
	Signed   int64
	Unsigned uint64
	Raw      []byte
}

// ColorValue is a 4-byte LabVIEW heap color payload. Raw preserves the
// full big-endian word; Prefix is the high byte, while R/G/B are the low
// three bytes used by observed RGB color fields.
type ColorValue struct {
	Tag    int32
	Raw    uint32
	Prefix uint8
	R      uint8
	G      uint8
	B      uint8
}

// IsHeapScalarTag reports whether tag is a common heap field currently
// treated as a scalar integer/flag/count/id payload.
func IsHeapScalarTag(tag int32) bool {
	switch tag {
	case int32(heap.FieldTagActiveDiag),
		int32(heap.FieldTagActiveMarker),
		int32(heap.FieldTagActiveXScale),
		int32(heap.FieldTagActiveYScale),
		int32(heap.FieldTagClumpNum),
		int32(heap.FieldTagConId),
		int32(heap.FieldTagConNum),
		int32(heap.FieldTagDsw),
		int32(heap.FieldTagFirstNodeIdx),
		int32(heap.FieldTagFormat),
		int32(heap.FieldTagFrontRow),
		int32(heap.FieldTagHowGrow),
		int32(heap.FieldTagHGrowNodeListLength),
		int32(heap.FieldTagInplace),
		int32(heap.FieldTagInstrStyle),
		int32(heap.FieldTagLastSignalKind),
		int32(heap.FieldTagMasterPart),
		int32(heap.FieldTagObjFlags),
		int32(heap.FieldTagParmIndex),
		int32(heap.FieldTagPartID),
		int32(heap.FieldTagPrimIndex),
		int32(heap.FieldTagPrimResID),
		int32(heap.FieldTagRefListLength),
		int32(heap.FieldTagRsrcID),
		int32(heap.FieldTagShortCount),
		int32(heap.FieldTagState),
		int32(heap.FieldTagTermListLength):
		return true
	default:
		return IsHeapColorTag(tag)
	}
}

// IsHeapColorTag reports whether tag is a heap field whose observed
// 4-byte payload is color-like.
func IsHeapColorTag(tag int32) bool {
	switch tag {
	case int32(heap.FieldTagBgColor),
		int32(heap.FieldTagBorderColor),
		int32(heap.FieldTagColor),
		int32(heap.FieldTagColorDSO),
		int32(heap.FieldTagColorTDO),
		int32(heap.FieldTagFgColor),
		int32(heap.FieldTagHierarchyColor),
		int32(heap.FieldTagScaleHiColor),
		int32(heap.FieldTagScaleLoColor),
		int32(heap.FieldTagSelectionColor),
		int32(heap.FieldTagStructColor):
		return true
	default:
		return false
	}
}

// HeapScalar decodes a known scalar heap leaf at tree.Nodes[nodeIdx].
// Bool-shaped scalar nodes decode as width 0 with value 0 or 1. Returns
// ok=false for out-of-range indices, non-scalar tags, or payloads wider
// than 8 bytes.
func HeapScalar(tree HeapTree, nodeIdx int) (ScalarValue, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return ScalarValue{}, false
	}
	n := tree.Nodes[nodeIdx]
	if !IsHeapScalarTag(n.Tag) {
		return ScalarValue{}, false
	}
	return decodeScalarContent(n)
}

// HeapScalarForTag decodes a scalar leaf only when it carries the
// requested known scalar tag.
func HeapScalarForTag(tree HeapTree, nodeIdx int, tag int32) (ScalarValue, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return ScalarValue{}, false
	}
	n := tree.Nodes[nodeIdx]
	if n.Tag != tag || !IsHeapScalarTag(tag) {
		return ScalarValue{}, false
	}
	return decodeScalarContent(n)
}

// HeapColor decodes a known color heap leaf at tree.Nodes[nodeIdx].
// Only 4-byte payloads are accepted.
func HeapColor(tree HeapTree, nodeIdx int) (ColorValue, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return ColorValue{}, false
	}
	n := tree.Nodes[nodeIdx]
	if !IsHeapColorTag(n.Tag) {
		return ColorValue{}, false
	}
	return decodeColorContent(n.Tag, n.Content)
}

// HeapColorForTag decodes a color leaf only when it carries the
// requested known color tag.
func HeapColorForTag(tree HeapTree, nodeIdx int, tag int32) (ColorValue, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return ColorValue{}, false
	}
	n := tree.Nodes[nodeIdx]
	if n.Tag != tag || !IsHeapColorTag(tag) {
		return ColorValue{}, false
	}
	return decodeColorContent(tag, n.Content)
}

// FindScalarChild walks the children of tree.Nodes[parentIdx] and
// returns the first decodable child carrying the requested scalar tag.
func FindScalarChild(tree HeapTree, parentIdx int, tag int32) (ScalarValue, bool) {
	if parentIdx < 0 || parentIdx >= len(tree.Nodes) || !IsHeapScalarTag(tag) {
		return ScalarValue{}, false
	}
	parent := tree.Nodes[parentIdx]
	for _, ci := range parent.Children {
		if ci < 0 || ci >= len(tree.Nodes) || tree.Nodes[ci].Tag != tag {
			continue
		}
		if v, ok := HeapScalarForTag(tree, ci, tag); ok {
			return v, true
		}
	}
	return ScalarValue{}, false
}

// FindColorChild walks the children of tree.Nodes[parentIdx] and returns
// the first decodable child carrying the requested color tag.
func FindColorChild(tree HeapTree, parentIdx int, tag int32) (ColorValue, bool) {
	if parentIdx < 0 || parentIdx >= len(tree.Nodes) || !IsHeapColorTag(tag) {
		return ColorValue{}, false
	}
	parent := tree.Nodes[parentIdx]
	for _, ci := range parent.Children {
		if ci < 0 || ci >= len(tree.Nodes) || tree.Nodes[ci].Tag != tag {
			continue
		}
		if c, ok := HeapColorForTag(tree, ci, tag); ok {
			return c, true
		}
	}
	return ColorValue{}, false
}

// HexRGB returns the low 24-bit color as a CSS-style #RRGGBB string.
func (c ColorValue) HexRGB() string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

func decodeScalarContent(n HeapNode) (ScalarValue, bool) {
	if n.SizeSpec == byte(heap.SizeSpecBoolFalse) {
		return ScalarValue{Tag: n.Tag, Width: 0}, true
	}
	if n.SizeSpec == byte(heap.SizeSpecBoolTrue) {
		return ScalarValue{Tag: n.Tag, Width: 0, Signed: 1, Unsigned: 1}, true
	}
	if len(n.Content) > 8 {
		return ScalarValue{}, false
	}
	var u uint64
	for _, b := range n.Content {
		u = (u << 8) | uint64(b)
	}
	s := int64(u)
	if len(n.Content) > 0 {
		bits := uint(len(n.Content) * 8)
		if u&(1<<(bits-1)) != 0 {
			s = int64(u | (^uint64(0) << bits))
		}
	}
	return ScalarValue{
		Tag:      n.Tag,
		Width:    len(n.Content),
		Signed:   s,
		Unsigned: u,
		Raw:      append([]byte(nil), n.Content...),
	}, true
}

func decodeColorContent(tag int32, content []byte) (ColorValue, bool) {
	if len(content) != 4 {
		return ColorValue{}, false
	}
	raw := binary.BigEndian.Uint32(content)
	return ColorValue{
		Tag:    tag,
		Raw:    raw,
		Prefix: uint8(raw >> 24),
		R:      uint8(raw >> 16),
		G:      uint8(raw >> 8),
		B:      uint8(raw),
	}, true
}
