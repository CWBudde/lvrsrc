package lvvi

import (
	"encoding/binary"
	"math"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/codecs/vctp"
)

// ConstKind classifies how a typed numeric constant's OF__ConstValue
// bytes should be interpreted once its VCTP type is known.
type ConstKind int

const (
	// ConstKindUnknown is used when no VCTP type resolved or the type is
	// not a numeric scalar the layer decodes (string, array, cluster, …).
	ConstKindUnknown ConstKind = iota
	// ConstKindSignedInt is a two's-complement big-endian integer (I8…I64).
	ConstKindSignedInt
	// ConstKindUnsignedInt is an unsigned big-endian integer (U8…U64).
	ConstKindUnsignedInt
	// ConstKindFloat is an IEEE-754 real (SGL binary32, DBL binary64, EXT
	// binary128 decoded to the nearest float64).
	ConstKindFloat
	// ConstKindComplex is a complex value stored as two IEEE-754 floats,
	// real-then-imaginary (CSG/CDB/CEXT).
	ConstKindComplex
	// ConstKindFixedPoint is an FXD container holding a scaled magnitude;
	// the radix lives in the FXD type config, not the VCTP header, so only
	// the raw magnitude is surfaced for now.
	ConstKindFixedPoint
	// ConstKindBoolean is a boolean literal stored as a 1- or 2-byte word.
	ConstKindBoolean
)

// String returns a stable lowercase name for the kind.
func (k ConstKind) String() string {
	switch k {
	case ConstKindSignedInt:
		return "signed-int"
	case ConstKindUnsignedInt:
		return "unsigned-int"
	case ConstKindFloat:
		return "float"
	case ConstKindComplex:
		return "complex"
	case ConstKindFixedPoint:
		return "fixed-point"
	case ConstKindBoolean:
		return "boolean"
	default:
		return "unknown"
	}
}

// TypedConst is a block-diagram numeric constant joined to the VCTP type
// that governs how its OF__ConstValue bytes decode.
//
// The raw OF__ConstValue leaf is a fixed-width big-endian value whose
// width tracks the constant's data type (see ConstValue). Width alone is
// not a unique type key past 8 bytes — 8 = I64/U64/DBL/CSG/FXD, 16 = EXT
// or CDB, 32 = CEXT or a string constant — so the exact interpretation
// requires the constant's VCTP type. This struct resolves that type from
// the constant object's OF__typeDesc (tag 283) reference and uses it to
// produce the correct typed value, including the 16/32-byte EXT/CDB/CEXT
// forms that HeapConstValue declines.
type TypedConst struct {
	// NodeIndex is the BD heap node index of the OF__ConstValue leaf.
	NodeIndex int
	// Raw is the verbatim OF__ConstValue payload.
	Raw []byte
	// TypeIndex is the 0-based flat VCTP index resolved from the governing
	// OF__typeDesc leaf, or -1 when no type resolved.
	TypeIndex int
	// FullType is the VCTP FullType name (e.g. "NumInt32"); empty when
	// unresolved.
	FullType string
	// HasType reports whether a VCTP type was resolved for the constant.
	HasType bool
	// WidthMatch reports whether the resolved type's constant-literal
	// width equals len(Raw) — a consistency check on the heap↔VCTP join.
	WidthMatch bool
	// Kind classifies the typed interpretation of Raw.
	Kind ConstKind
	// Int / Uint hold the integer value for ConstKindSignedInt /
	// ConstKindUnsignedInt (Int is sign-extended; Uint is the raw word).
	Int  int64
	Uint uint64
	// Float holds the real value for ConstKindFloat. EXT (binary128) is
	// decoded to the nearest float64.
	Float float64
	// Real / Imag hold the components for ConstKindComplex, each decoded
	// from the underlying float width (binary32/64/128).
	Real float64
	Imag float64
	// FixedRaw holds the raw fixed-point magnitude for ConstKindFixedPoint
	// (radix governed by the FXD config, not yet parsed).
	FixedRaw uint64
}

// BlockDiagramConstants returns every block-diagram numeric constant in
// the wrapped file, each joined to its VCTP type. Returns ok=false when
// there is no BDHb (block-diagram heap) section.
//
// For each OF__ConstValue leaf the constant's governing type is resolved
// from the nearest preceding OF__typeDesc (tag 283) leaf within the same
// enclosing heap object. That leaf's content is a 0-based index into the
// VCTP top-types list; TopTypes[content] is the flat VCTP index of the
// type. The resolved VCTP FullType then drives the typed decode — which
// is what disambiguates the wide width collisions (EXT vs CDB at 16
// bytes, CEXT vs string at 32) and lets the layer decode EXT/CDB/CEXT to
// exact typed values, unlike HeapConstValue's leaf-only ≤8-byte view.
func (m *Model) BlockDiagramConstants() ([]TypedConst, bool) {
	if m == nil || m.file == nil {
		return nil, false
	}
	tree, ok := m.BlockDiagram()
	if !ok {
		return nil, false
	}
	descs, tops, _ := decodeVCTP(m.file)

	var out []TypedConst
	for i, n := range tree.Nodes {
		if n.Tag != int32(heap.FieldTagConstValue) || n.Scope != "leaf" {
			continue
		}
		tc := TypedConst{
			NodeIndex: i,
			Raw:       append([]byte(nil), n.Content...),
			TypeIndex: -1,
			Kind:      ConstKindUnknown,
		}
		if flat, ok := resolveConstTypeIndex(tree, i, tops); ok && flat >= 0 && flat < len(descs) {
			tc.TypeIndex = flat
			fillTypedConst(&tc, descs[flat].FullType)
		}
		out = append(out, tc)
	}
	return out, true
}

// resolveConstTypeIndex finds the flat VCTP index governing the
// OF__ConstValue leaf at constIdx. The constant's own type is the nearest
// OF__typeDesc (tag 283) leaf preceding the value leaf within the same
// enclosing heap object — for a diagram constant the object carries two
// typeDesc references (the wired indicator's type and the constant's own
// type); the constant's is the inner, closer one. The leaf's content is a
// big-endian unsigned 0-based index into the top-types list (tops), and
// tops[content] is the flat VCTP index.
//
// The TopTypes indirection is essential, not cosmetic. Evidence:
//
//   - Numeric42.vi: an I32 constant wired to a DBL indicator carries
//     typeDesc 0x03 → tops[3]=3 → VCTP[3]=NumFloat64 (indicator) and 0x04
//     → tops[4]=4 → VCTP[4]=NumInt32 (constant); the nearest-preceding
//     rule selects 0x04, matching the 4-byte I32 literal.
//   - Add17Plus25.vi (two I32 constants, only 5 VCTP types): typeDesc
//     0x07 → tops[7]=4 → VCTP[4]=NumInt32 and 0x09 → tops[9]=4 →
//     VCTP[4]. Indexing VCTP flat with 0x07/0x09 directly would be out of
//     range — the content is a top-types ordinal, not a flat index.
func resolveConstTypeIndex(tree HeapTree, constIdx int, tops []uint32) (int, bool) {
	if constIdx < 0 || constIdx >= len(tree.Nodes) {
		return 0, false
	}
	enclosing := tree.Nodes[constIdx].Parent
	best := -1
	for i := 0; i < constIdx; i++ {
		n := tree.Nodes[i]
		if n.Tag != int32(heap.FieldTagTypeDesc) || n.Scope != "leaf" {
			continue
		}
		if enclosing >= 0 && !isDescendantOrSelf(tree, i, enclosing) {
			continue
		}
		if i > best {
			best = i
		}
	}
	if best < 0 {
		return 0, false
	}
	topIdx, ok := bigEndianUint(tree.Nodes[best].Content)
	if !ok || topIdx >= uint64(len(tops)) {
		return 0, false
	}
	return int(tops[topIdx]), true
}

// isDescendantOrSelf reports whether node is ancestor or a descendant of
// ancestor in the heap tree.
func isDescendantOrSelf(tree HeapTree, node, ancestor int) bool {
	for node >= 0 {
		if node == ancestor {
			return true
		}
		node = tree.Nodes[node].Parent
	}
	return false
}

// bigEndianUint reads up to 8 content bytes as a big-endian unsigned int.
func bigEndianUint(b []byte) (uint64, bool) {
	if len(b) == 0 || len(b) > 8 {
		return 0, false
	}
	var u uint64
	for _, c := range b {
		u = (u << 8) | uint64(c)
	}
	return u, true
}

// fillTypedConst sets the typed fields of tc from the resolved VCTP
// FullType ft and tc.Raw.
func fillTypedConst(tc *TypedConst, ft vctp.FullType) {
	tc.FullType = ft.String()
	tc.HasType = true
	if w, ok := constLiteralWidth(ft); ok {
		tc.WidthMatch = w == len(tc.Raw)
	}

	raw := tc.Raw
	switch ft {
	case vctp.FullTypeNumInt8, vctp.FullTypeNumInt16,
		vctp.FullTypeNumInt32, vctp.FullTypeNumInt64:
		tc.Kind = ConstKindSignedInt
		if u, ok := bigEndianUint(raw); ok {
			tc.Uint = u
			tc.Int = signExtend(u, len(raw))
		}
	case vctp.FullTypeNumUInt8, vctp.FullTypeNumUInt16,
		vctp.FullTypeNumUInt32, vctp.FullTypeNumUInt64:
		tc.Kind = ConstKindUnsignedInt
		if u, ok := bigEndianUint(raw); ok {
			tc.Uint = u
			tc.Int = int64(u)
		}
	case vctp.FullTypeNumFloat32:
		tc.Kind = ConstKindFloat
		if len(raw) >= 4 {
			tc.Float = float64(math.Float32frombits(binary.BigEndian.Uint32(raw[:4])))
		}
	case vctp.FullTypeNumFloat64:
		tc.Kind = ConstKindFloat
		if len(raw) >= 8 {
			tc.Float = math.Float64frombits(binary.BigEndian.Uint64(raw[:8]))
		}
	case vctp.FullTypeNumFloatExt:
		tc.Kind = ConstKindFloat
		if len(raw) >= 16 {
			tc.Float = decodeBinary128(raw[:16])
		}
	case vctp.FullTypeNumComplex64:
		tc.Kind = ConstKindComplex
		if len(raw) >= 8 {
			tc.Real = float64(math.Float32frombits(binary.BigEndian.Uint32(raw[0:4])))
			tc.Imag = float64(math.Float32frombits(binary.BigEndian.Uint32(raw[4:8])))
		}
	case vctp.FullTypeNumComplex128:
		tc.Kind = ConstKindComplex
		if len(raw) >= 16 {
			tc.Real = math.Float64frombits(binary.BigEndian.Uint64(raw[0:8]))
			tc.Imag = math.Float64frombits(binary.BigEndian.Uint64(raw[8:16]))
		}
	case vctp.FullTypeNumComplexExt:
		tc.Kind = ConstKindComplex
		if len(raw) >= 32 {
			tc.Real = decodeBinary128(raw[0:16])
			tc.Imag = decodeBinary128(raw[16:32])
		}
	case vctp.FullTypeFixedPoint:
		tc.Kind = ConstKindFixedPoint
		if u, ok := bigEndianUint(raw); ok {
			tc.FixedRaw = u
		}
	case vctp.FullTypeBoolean, vctp.FullTypeBooleanU16:
		tc.Kind = ConstKindBoolean
		if u, ok := bigEndianUint(raw); ok {
			tc.Uint = u
		}
	default:
		tc.Kind = ConstKindUnknown
	}
}

// signExtend sign-extends the low width*8 bits of u to a signed int64.
func signExtend(u uint64, width int) int64 {
	if width <= 0 || width >= 8 {
		return int64(u)
	}
	bits := uint(width * 8)
	if u&(1<<(bits-1)) != 0 {
		return int64(u | (^uint64(0) << bits))
	}
	return int64(u)
}

// constLiteralWidth returns the OF__ConstValue byte width for a numeric
// VCTP type, and ok=false for types whose constant literal has no fixed
// numeric width (string, array, cluster, …). The widths track the
// observed encoding: EXT is the 16-byte binary128 constant form (not the
// 10/16-byte in-memory x87 extended), and the complex widths are twice
// their component float.
func constLiteralWidth(ft vctp.FullType) (int, bool) {
	switch ft {
	case vctp.FullTypeNumInt8, vctp.FullTypeNumUInt8, vctp.FullTypeBoolean:
		return 1, true
	case vctp.FullTypeNumInt16, vctp.FullTypeNumUInt16, vctp.FullTypeBooleanU16:
		return 2, true
	case vctp.FullTypeNumInt32, vctp.FullTypeNumUInt32, vctp.FullTypeNumFloat32:
		return 4, true
	case vctp.FullTypeNumInt64, vctp.FullTypeNumUInt64, vctp.FullTypeNumFloat64,
		vctp.FullTypeNumComplex64, vctp.FullTypeFixedPoint:
		return 8, true
	case vctp.FullTypeNumFloatExt, vctp.FullTypeNumComplex128:
		return 16, true
	case vctp.FullTypeNumComplexExt:
		return 32, true
	default:
		return 0, false
	}
}

// decodeBinary128 decodes a 16-byte big-endian IEEE-754 binary128 (quad)
// to the nearest float64. The 112-bit fraction is truncated to float64's
// 52-bit fraction, so the result is approximate for values that use more
// than 52 fraction bits; LabVIEW EXT constants observed so far store far
// fewer significant bits, so the round-trip is exact in practice. Inf/NaN
// and subnormals are handled; the common path is normal numbers.
func decodeBinary128(b []byte) float64 {
	if len(b) < 16 {
		return 0
	}
	hi := binary.BigEndian.Uint64(b[0:8])
	lo := binary.BigEndian.Uint64(b[8:16])
	sign := hi >> 63
	exp := int((hi >> 48) & 0x7fff)
	fracHi := hi & 0xFFFFFFFFFFFF // low 48 bits of the high word
	// Top 52 bits of the 112-bit fraction: 48 from fracHi + top 4 of lo.
	frac52 := (fracHi << 4) | (lo >> 60)

	var val float64
	switch exp {
	case 0x7fff:
		if frac52 == 0 && fracHi == 0 && lo == 0 {
			val = math.Inf(1)
		} else {
			val = math.NaN()
		}
	case 0:
		// Subnormal (or zero): no implicit leading 1.
		if frac52 == 0 {
			val = 0
		} else {
			val = math.Ldexp(float64(frac52)/math.Exp2(52), 1-16383)
		}
	default:
		significand := 1 + float64(frac52)/math.Exp2(52)
		val = math.Ldexp(significand, exp-16383)
	}
	if sign != 0 {
		val = -val
	}
	return val
}
