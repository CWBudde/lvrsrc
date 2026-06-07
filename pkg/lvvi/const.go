package lvvi

import (
	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
)

// ConstValue is a decoded block-diagram numeric-constant literal taken
// from an OF__ConstValue heap leaf (tag 589).
//
// A numeric constant is stored as a fixed-width big-endian value whose
// width tracks the constant's data type, written verbatim (no varint,
// no 0xff escape). Observed representation widths:
//
//	I8 / U8       1 byte   raw byte (signedness lives in the type, not
//	                       the bytes: I8 and U8 of 42 both store 0x2a)
//	I16 / U16     2 bytes  big-endian int
//	I32 / U32     4 bytes  big-endian int
//	I64 / U64     8 bytes  big-endian int
//	SGL          4 bytes   big-endian IEEE-754 binary32
//	DBL          8 bytes   big-endian IEEE-754 binary64
//	EXT          16 bytes  big-endian IEEE-754 binary128 (quad)
//	FXD          8 bytes   big-endian fixed-point container (value x 2^k)
//
// Raw preserves the payload bytes; Signed/Unsigned are big-endian
// integer interpretations of those bytes (for SGL/DBL, Unsigned holds
// the raw IEEE-754 bits — reinterpret with math.Float32/64frombits when
// the constant's representation is known to be floating point).
//
// HeapConstValue decodes the integer/float widths up to 8 bytes. It
// reports ok=false for payloads wider than 8 bytes: that band covers
// both 16-byte EXT constants and non-numeric constants (string, array,
// cluster, which carry arbitrary widths), and the leaf alone cannot
// distinguish them without the constant's VCTP type — so those are left
// to a future type-aware layer rather than guessed here.
type ConstValue struct {
	Tag      int32
	Width    int
	Signed   int64
	Unsigned uint64
	Raw      []byte
}

// HeapConstValue decodes the OF__ConstValue leaf at tree.Nodes[nodeIdx].
// It returns ok=false for out-of-range indices, non-ConstValue tags, and
// payloads wider than 8 bytes (non-numeric constants).
func HeapConstValue(tree HeapTree, nodeIdx int) (ConstValue, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return ConstValue{}, false
	}
	n := tree.Nodes[nodeIdx]
	if n.Tag != int32(heap.FieldTagConstValue) || n.Scope != "leaf" {
		return ConstValue{}, false
	}
	sv, ok := decodeScalarContent(n)
	if !ok {
		return ConstValue{}, false
	}
	return ConstValue{
		Tag:      n.Tag,
		Width:    sv.Width,
		Signed:   sv.Signed,
		Unsigned: sv.Unsigned,
		Raw:      sv.Raw,
	}, true
}

// FindConstValueChild returns the decoded OF__ConstValue leaf nested
// under parentIdx, if present.
func FindConstValueChild(tree HeapTree, parentIdx int) (ConstValue, bool) {
	if parentIdx < 0 || parentIdx >= len(tree.Nodes) {
		return ConstValue{}, false
	}
	for _, childIdx := range tree.Nodes[parentIdx].Children {
		if cv, ok := HeapConstValue(tree, childIdx); ok {
			return cv, true
		}
	}
	return ConstValue{}, false
}
