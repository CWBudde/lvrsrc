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
	// NodeIndex is the heap node index of the value leaf — the
	// OF__ConstValue leaf for a block-diagram constant
	// (BlockDiagramConstants) or the OF__DefaultData leaf for a
	// front-panel control default (FrontPanelDefaults).
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
	// FixedRaw holds the raw fixed-point mantissa for ConstKindFixedPoint —
	// the verbatim big-endian word, before applying the binary-point scale.
	FixedRaw uint64
	// FixedWordLength / FixedIntWordLength are the FXP word length and
	// integer word length in bits, read from the resolved FixedPoint VCTP
	// type config. Valid only when FixedConfigOK.
	FixedWordLength    int
	FixedIntWordLength int
	// FixedRadix is the fractional bit count (FixedWordLength -
	// FixedIntWordLength); FixedValue is the scaled value
	// (signExtend(FixedRaw) / 2^FixedRadix). Both valid only when
	// FixedConfigOK, which reports whether the FXP config parsed.
	FixedRadix    int
	FixedValue    float64
	FixedConfigOK bool
	// Composite is the decoded flattened-data tree for a composite
	// (cluster) default whose blob unflattened exactly against a resolved
	// VCTP cluster type, or nil for scalar / unresolved constants.
	// CompositeTypeIndex is that cluster's flat VCTP index and CompositeOK
	// reports whether the composite decode succeeded. Populated only by
	// FrontPanelDefaults (panel control defaults); see
	// resolveCompositeDefault.
	Composite          *FlatValue
	CompositeTypeIndex int
	CompositeOK        bool
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
			fillTypedConst(&tc, descs[flat])
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

// fillTypedConst sets the typed fields of tc from the resolved VCTP type
// descriptor and tc.Raw. The whole descriptor (not just its FullType) is
// needed so the FixedPoint case can read the binary-point scale from the
// type's inner config bytes.
func fillTypedConst(tc *TypedConst, desc vctp.TypeDescriptor) {
	ft := desc.FullType
	tc.FullType = ft.String()
	tc.HasType = true
	tc.WidthMatch = constWidthMatches(ft, len(tc.Raw))

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
		if wl, iwl, ok := fixedPointConfig(desc.Inner); ok {
			tc.FixedWordLength = wl
			tc.FixedIntWordLength = iwl
			tc.FixedRadix = wl - iwl
			tc.FixedConfigOK = true
			// The mantissa is the low wl bits of the container (which is
			// wider than wl when wl < 64 — w32i8 holds a 32-bit value in an
			// 8-byte leaf), sign-extended from bit wl-1. Both corpus FXP
			// constants are positive, so negative-mantissa handling is not
			// yet corpus-verified.
			mantissa := signExtendBits(tc.FixedRaw, wl)
			tc.FixedValue = float64(mantissa) * math.Ldexp(1, -tc.FixedRadix)
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

// fixedPointConfig reads the FXP word length and integer word length (both
// in bits) from a FixedPoint VCTP descriptor's inner config bytes. The radix
// (fractional bit count) is wordLength - intWordLength — the standard
// LabVIEW FXP definition, where the integer word length counts the bits to
// the left of the binary point (sign included).
//
// Inner layout (big-endian), read off Numeric9876Dot5432_FXD.vi's two
// FixedPoint descriptors:
//
//	off 0:  0x03                     sub-record marker (constant across the corpus)
//	off 1:  representation byte       0x51 for the signed FXP observed
//	off 2:  U16 word length (bits)    32 / 64  — also the constant's byte width × 8
//	off 4:  U32 integer word length   16 / 32
//	off 24: U32 word length - 1       31 / 63  (cross-check)
//	off 28: U32 integer word length-1 15 / 31  (cross-check)
//
// The off-2 / off-4 positions are pinned by the cross-check fields at off-24
// / off-28 and by off-2 matching the constant's container width. Both corpus
// FXP types happen to have wordLength = 2 × intWordLength, so this fixture
// alone cannot tell radix = wordLength - intWordLength apart from radix =
// intWordLength (they coincide); the former is the documented FXP semantics
// and produces the correct value (9876.54296875) for the 64/32 constant.
func fixedPointConfig(inner []byte) (wordLen, intWordLen int, ok bool) {
	if len(inner) < 8 {
		return 0, 0, false
	}
	wordLen = int(binary.BigEndian.Uint16(inner[2:4]))
	intWordLen = int(binary.BigEndian.Uint32(inner[4:8]))
	if wordLen <= 0 {
		return 0, 0, false
	}
	return wordLen, intWordLen, true
}

// signExtendBits sign-extends the low `bits` bits of u to a signed int64,
// masking off anything above bit bits-1. bits >= 64 (or <= 0) returns u
// reinterpreted as int64 unchanged.
func signExtendBits(u uint64, bits int) int64 {
	if bits <= 0 || bits >= 64 {
		return int64(u)
	}
	mask := (uint64(1) << uint(bits)) - 1
	u &= mask
	if u&(uint64(1)<<uint(bits-1)) != 0 {
		u |= ^mask
	}
	return int64(u)
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

// constWidthMatches reports whether a raw OF__ConstValue / OF__DefaultData
// length n is consistent with the resolved VCTP numeric type ft — a
// consistency check on the heap↔VCTP join that callers expose as
// TypedConst.WidthMatch.
//
// Most numeric types have a single fixed literal width (constLiteralWidth).
// Boolean is the exception: across the corpus its constant literal is 1 byte
// for TRUE (0x01) and 2 bytes for FALSE (0x0000) — BoolToLED/format-string
// store the 1-byte TRUE, WhileLoop_Numeric42/reference-find-by-id the 2-byte
// FALSE. The VCTP descriptor (including its Flags) is identical for both
// widths, so the width cannot be predicted from the type alone; the value
// decode (any non-zero byte → true) is width-independent, so both 1- and
// 2-byte Boolean literals are accepted here. Why FALSE takes the extra byte
// is not yet explained — only the widths are corpus-confirmed.
//
// Returns false for types whose literal has no fixed numeric width (string,
// array, cluster, variant, …) so callers treat them as not-a-scalar.
func constWidthMatches(ft vctp.FullType, n int) bool {
	if ft == vctp.FullTypeBoolean || ft == vctp.FullTypeBooleanU16 {
		return n == 1 || n == 2
	}
	w, ok := constLiteralWidth(ft)
	return ok && w == n
}

// constLiteralWidth returns the fixed OF__ConstValue byte width for a numeric
// VCTP type, and ok=false for types whose constant literal has no single
// fixed numeric width (string, array, cluster, the dual-width Boolean, …).
// The widths track the observed encoding: EXT is the 16-byte binary128
// constant form (not the 10/16-byte in-memory x87 extended), and the complex
// widths are twice their component float. Boolean is handled by
// constWidthMatches, not here, because its literal width is not fixed.
func constLiteralWidth(ft vctp.FullType) (int, bool) {
	switch ft {
	case vctp.FullTypeNumInt8, vctp.FullTypeNumUInt8:
		return 1, true
	case vctp.FullTypeNumInt16, vctp.FullTypeNumUInt16:
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
