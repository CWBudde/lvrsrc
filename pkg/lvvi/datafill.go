package lvvi

import (
	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/dthp"
	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/codecs/vctp"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// DataFillKind classifies the typed interpretation of a heap node's
// content bytes once its TypeDesc has been resolved through DTHP+VCTP.
//
// pylabview's `HeapNodeTDDataFill` (LVheap.py:1911) drives this: each
// `OF__StdNumMin`/`OF__StdNumMax`/`OF__StdNumInc` node interprets its
// content as a numeric value of the surrounding object's TypeDesc. We
// surface only the primitive numeric kinds; anything pylabview can't
// decode directly (Boolean, Cluster, String, complex, quad-float, …)
// falls back to `DataFillKindRaw` with the resolved FullType label so
// callers stay round-trip-safe.
type DataFillKind int

const (
	// DataFillKindUnknown means the node's TypeDesc could not be
	// resolved at all (no DTHP, no VCTP, no parent typeDesc child). The
	// raw bytes are still preserved in DataFillValue.Raw.
	DataFillKindUnknown DataFillKind = iota
	// DataFillKindRaw means the TypeDesc resolved but its FullType is
	// not one of the supported numeric primitives. DataFillValue.Raw
	// still holds the original content bytes.
	DataFillKindRaw
	// DataFillKindInt means the value was decoded as a signed integer
	// stored in DataFillValue.Int. Width is the declared TypeDesc width
	// in bytes (1, 2, 4, or 8); the on-disk content may be shorter
	// thanks to pylabview's shrinkRepeatedBits truncation.
	DataFillKindInt
	// DataFillKindUInt means the value was decoded as an unsigned
	// integer stored in DataFillValue.Uint, masked to Width bytes.
	DataFillKindUInt
	// DataFillKindFloat32 means the content is a 4-byte big-endian
	// IEEE-754 single-precision float, stored in DataFillValue.Float.
	DataFillKindFloat32
	// DataFillKindFloat64 means the content is an 8-byte big-endian
	// IEEE-754 double-precision float, stored in DataFillValue.Float.
	DataFillKindFloat64
)

// String returns a short label suitable for logs and tests.
func (k DataFillKind) String() string {
	switch k {
	case DataFillKindUnknown:
		return "unknown"
	case DataFillKindRaw:
		return "raw"
	case DataFillKindInt:
		return "int"
	case DataFillKindUInt:
		return "uint"
	case DataFillKindFloat32:
		return "float32"
	case DataFillKindFloat64:
		return "float64"
	default:
		return "unknown"
	}
}

// DataFillValue is the public typed projection of a HeapNodeTDDataFill
// node (`OF__StdNumMin`/`OF__StdNumMax`/`OF__StdNumInc`). Raw is always
// populated with the original content bytes so callers retain
// round-trip-safe access regardless of Kind.
type DataFillValue struct {
	// Kind is the resolved typed interpretation. See DataFillKind.
	Kind DataFillKind
	// Int is meaningful only when Kind == DataFillKindInt.
	Int int64
	// Uint is meaningful only when Kind == DataFillKindUInt.
	Uint uint64
	// Float is meaningful when Kind == DataFillKindFloat32 or
	// DataFillKindFloat64. Float32 values are stored after a widening
	// conversion to float64.
	Float float64
	// Width is the declared TypeDesc width in bytes (1/2/4/8) for
	// numeric kinds, 0 otherwise.
	Width int
	// HeapTypeID is the heap-local TypeID read from the parent's
	// `OF__typeDesc` child, or 0 when no such child was found.
	HeapTypeID uint32
	// ResolvedTypeIdx is the 1-based VCTP flat ID the heap-local TypeID
	// resolved to (matches Model.TypeAt's numbering), or 0 when
	// resolution failed.
	ResolvedTypeIdx int
	// FullType is the resolved descriptor's type name (e.g.
	// "NumInt32", "Boolean"). Empty when ResolvedTypeIdx is 0.
	FullType string
	// Raw is the original content bytes from the heap node.
	Raw []byte
}

// HeapDataFill resolves a HeapNode tagged StdNumMin/Max/Inc (FieldTag
// 513/514/515) against the file's DTHP+VCTP, returning a typed value.
//
// Returns ok=false only when nodeIdx is out of range or the node's tag
// is not one of the DataFill tags. All other failure modes (no DTHP,
// no VCTP, missing parent typeDesc child, unsupported FullType) return
// ok=true with `Kind ∈ {Unknown, Raw}` and `Raw` populated, so callers
// always receive the original content bytes.
//
// Resolution mirrors pylabview's `HeapNodeTDDataFill.findAndStoreTD`
// (LVheap.py:1929) and `Block.getHeapTD` (LVblock.py:3280):
//
//  1. Take the node's parent (the surrounding TagOpen).
//  2. Find the parent's `OF__typeDesc` child (FieldTag 283).
//  3. Read its content as a signed BE integer — that is the heap-local
//     TypeID.
//  4. Resolve via VCTP.descriptors[DTHP.IndexShift + heapTypeID - 1].
//  5. Switch on FullType to choose the typed decoder.
func (m *Model) HeapDataFill(tree HeapTree, nodeIdx int) (DataFillValue, bool) {
	if m == nil || m.file == nil {
		return DataFillValue{}, false
	}
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return DataFillValue{}, false
	}
	node := tree.Nodes[nodeIdx]
	if !isDataFillTag(node.Tag) {
		return DataFillValue{}, false
	}

	val := DataFillValue{Kind: DataFillKindUnknown}
	if len(node.Content) > 0 {
		val.Raw = append([]byte(nil), node.Content...)
	}

	// Step 1+2: find the parent's OF__typeDesc child.
	heapTypeID, hasTypeID := findTypeDescSibling(tree, nodeIdx)
	if !hasTypeID {
		return val, true
	}
	val.HeapTypeID = heapTypeID

	// Step 3+4: resolve through DTHP shift + VCTP descriptors.
	descs, _, ok := decodeVCTP(m.file)
	if !ok {
		return val, true
	}
	shift, hasDTHP := decodeDTHPShift(m.file)
	if !hasDTHP {
		// No DTHP block at all — pylabview returns None in this case.
		// Treat as unresolved.
		return val, true
	}
	tdIndex := int(shift) + int(heapTypeID) - 1
	if tdIndex < 0 || tdIndex >= len(descs) {
		return val, true
	}
	desc := descs[tdIndex]
	val.ResolvedTypeIdx = tdIndex + 1
	val.FullType = desc.FullType.String()
	val.Kind = DataFillKindRaw

	// Step 5: typed-content decode based on resolved FullType.
	decodeDataFillContent(&val, node.Content, desc.FullType)
	return val, true
}

// isDataFillTag reports whether tag is one of the FieldTag values that
// pylabview routes through HeapNodeTDDataFill: OF__StdNumMin (513),
// OF__StdNumMax (514), OF__StdNumInc (515).
func isDataFillTag(tag int32) bool {
	return tag == int32(heap.FieldTagStdNumMin) ||
		tag == int32(heap.FieldTagStdNumMax) ||
		tag == int32(heap.FieldTagStdNumInc)
}

// findTypeDescSibling looks at the parent of tree.Nodes[nodeIdx] and
// returns the heap-local TypeID stored in its OF__typeDesc child (tag
// 283). Returns (0, false) when there is no parent or no typeDesc
// child or the child's content can't be read as a signed integer.
func findTypeDescSibling(tree HeapTree, nodeIdx int) (uint32, bool) {
	node := tree.Nodes[nodeIdx]
	if node.Parent < 0 || node.Parent >= len(tree.Nodes) {
		return 0, false
	}
	parent := tree.Nodes[node.Parent]
	for _, ci := range parent.Children {
		if ci < 0 || ci >= len(tree.Nodes) {
			continue
		}
		child := tree.Nodes[ci]
		if child.Tag != int32(heap.FieldTagTypeDesc) {
			continue
		}
		// Decode the typeDesc content as a signed BE integer and
		// reject negative or zero values (pylabview's getHeapTD
		// requires heapTypeId >= 1).
		v, err := signExtendBE(child.Content)
		if err != nil || v < 1 {
			return 0, false
		}
		return uint32(v), true
	}
	return 0, false
}

// signExtendBE reads buf as a big-endian signed integer of length
// len(buf) and sign-extends to int64. Mirrors heap.Node.AsStdInt's
// signed branch — duplicated here so we can call it on a HeapNode
// projection without round-tripping back through the internal type.
func signExtendBE(buf []byte) (int64, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	if len(buf) > 8 {
		return 0, errOversizeContent
	}
	var u uint64
	for _, b := range buf {
		u = (u << 8) | uint64(b)
	}
	bits := uint(len(buf) * 8)
	if u&(1<<(bits-1)) != 0 {
		u |= ^uint64(0) << bits
	}
	return int64(u), nil
}

// errOversizeContent is returned when integer-shaped content would
// overflow int64. Kept as a package var so callers can check identity,
// though the data-fill path silently treats it as unresolved.
var errOversizeContent = &dataFillError{msg: "content > 8 bytes"}

type dataFillError struct{ msg string }

func (e *dataFillError) Error() string { return e.msg }

// decodeDataFillContent populates the typed fields of val based on the
// resolved FullType. On any failure (length mismatch, unsupported
// type) the function leaves val.Kind at DataFillKindRaw — Raw is
// already populated, so callers still see the original bytes.
func decodeDataFillContent(val *DataFillValue, content []byte, ft vctp.FullType) {
	switch ft {
	case vctp.FullTypeNumInt8:
		setInt(val, content, 1)
	case vctp.FullTypeNumInt16:
		setInt(val, content, 2)
	case vctp.FullTypeNumInt32:
		setInt(val, content, 4)
	case vctp.FullTypeNumInt64:
		setInt(val, content, 8)
	case vctp.FullTypeNumUInt8:
		setUint(val, content, 1)
	case vctp.FullTypeNumUInt16:
		setUint(val, content, 2)
	case vctp.FullTypeNumUInt32:
		setUint(val, content, 4)
	case vctp.FullTypeNumUInt64:
		setUint(val, content, 8)
	case vctp.FullTypeNumFloat32:
		if len(content) == 4 {
			n := &heap.Node{Content: append([]byte(nil), content...), SizeSpec: 4}
			if f, err := n.AsFloat32(); err == nil {
				val.Kind = DataFillKindFloat32
				val.Width = 4
				val.Float = float64(f)
			}
		}
	case vctp.FullTypeNumFloat64:
		if len(content) == 8 {
			n := &heap.Node{Content: append([]byte(nil), content...), SizeSpec: 6}
			if f, err := n.AsFloat64(); err == nil {
				val.Kind = DataFillKindFloat64
				val.Width = 8
				val.Float = f
			}
		}
	}
	// Anything else stays Kind=Raw.
}

// setInt sign-extends content into val.Int and marks val as
// DataFillKindInt at the declared width.
func setInt(val *DataFillValue, content []byte, width int) {
	v, err := signExtendBE(content)
	if err != nil {
		return
	}
	val.Kind = DataFillKindInt
	val.Width = width
	val.Int = v
}

// setUint reads content as signed (per pylabview's
// sign-extend-then-treat-as-unsigned dance from
// HeapNodeTDDataFill.parseRSRCContentDirect, LVheap.py:1976) and masks
// to the declared width.
func setUint(val *DataFillValue, content []byte, width int) {
	v, err := signExtendBE(content)
	if err != nil {
		return
	}
	val.Kind = DataFillKindUInt
	val.Width = width
	if width >= 8 {
		val.Uint = uint64(v)
		return
	}
	mask := (uint64(1) << uint(width*8)) - 1
	val.Uint = uint64(v) & mask
}

// decodeDTHPShift returns the file's DTHP IndexShift, or (0, false)
// when no DTHP block is present or it could not be decoded. A DTHP
// section with TDCount=0 still counts as "present" (the shift is 0 by
// definition); callers that need to distinguish should look at the
// resolved descriptor index.
func decodeDTHPShift(f *lvrsrc.File) (uint32, bool) {
	refs := sectionsOf(f, string(dthp.FourCC))
	if len(refs) == 0 {
		return 0, false
	}
	ctx := codecs.Context{FileVersion: f.Header.FormatVersion, Kind: f.Kind}
	raw, err := (dthp.Codec{}).Decode(ctx, refs[0].Payload)
	if err != nil {
		return 0, false
	}
	v, ok := raw.(dthp.Value)
	if !ok {
		return 0, false
	}
	return v.IndexShift, true
}
