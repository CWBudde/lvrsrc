package lvvi

import (
	"encoding/binary"
	"math"

	"github.com/CWBudde/lvrsrc/internal/codecs/vctp"
)

// FlatKind classifies a node in a decoded flattened-data tree (FlatValue).
// LabVIEW stores a control's composite default value (and any cluster/array
// constant) as "flattened data": each leaf is serialised big-endian in
// declaration order with no padding, strings carry a 4-byte length prefix,
// and a cluster is just its members back-to-back. FlatKind tells callers how
// to read a given FlatValue node.
type FlatKind int

const (
	// FlatKindUnknown is a node whose VCTP type the unflattener does not
	// model; Raw holds the bytes it consumed (which may be zero).
	FlatKindUnknown FlatKind = iota
	// FlatKindInt is a signed big-endian integer in Int (I8…I64).
	FlatKindInt
	// FlatKindUint is an unsigned big-endian integer in Uint (U8…U64).
	FlatKindUint
	// FlatKindFloat is an IEEE-754 real in Float (SGL binary32 / DBL binary64).
	FlatKindFloat
	// FlatKindBool is a 1- or 2-byte boolean in Bool (any non-zero byte → true).
	FlatKindBool
	// FlatKindString is a 4-byte-length-prefixed byte string in String.
	FlatKindString
	// FlatKindCluster is an aggregate; its members are in Children, in
	// declaration order.
	FlatKindCluster
	// FlatKindVariant is an LVVariant blob the unflattener keeps opaque in
	// Raw. A variant is only decoded when it is the trailing member of its
	// cluster (it then consumes the rest of the buffer); the variant's own
	// internal structure is not parsed.
	FlatKindVariant
)

// String returns a stable lowercase label for the kind.
func (k FlatKind) String() string {
	switch k {
	case FlatKindInt:
		return "int"
	case FlatKindUint:
		return "uint"
	case FlatKindFloat:
		return "float"
	case FlatKindBool:
		return "bool"
	case FlatKindString:
		return "string"
	case FlatKindCluster:
		return "cluster"
	case FlatKindVariant:
		return "variant"
	default:
		return "unknown"
	}
}

// FlatValue is one node of a decoded flattened-data tree. Scalar leaves carry
// their typed value in the matching field (Int/Uint/Float/Bool/String); a
// cluster carries its decoded members in Children. FullType is the resolved
// VCTP type name and Label is the member's VCTP label (the cluster field name)
// when one was stored, both for self-documenting output. Raw holds the verbatim
// bytes for opaque kinds (variant/unknown).
type FlatValue struct {
	Kind     FlatKind
	FullType string
	Label    string
	Int      int64
	Uint     uint64
	Float    float64
	Bool     bool
	String   string
	Children []FlatValue
	Raw      []byte
}

// UnflattenValue decodes a flattened-data blob against the VCTP type at the
// given flat index, returning the decoded tree. Returns ok=false when the
// VCTP pool is unavailable, the index is out of range, the type is one the
// unflattener does not model, or the blob does not match the type's flat
// layout (a member runs past the end, a string length overflows, …).
//
// The caller supplies the governing type explicitly, so this is the principled
// entry point — no type guessing. FrontPanelDefaults layers a structural
// resolver on top for panel control defaults, whose panel-heap type pointer is
// not yet decoded.
//
// ok=true does not require the blob to be fully consumed; callers that need an
// exact fit (e.g. the structural resolver) compare the returned FlatValue's
// span themselves via UnflattenValueN.
func (m *Model) UnflattenValue(flatTypeIndex int, blob []byte) (FlatValue, bool) {
	fv, _, ok := m.UnflattenValueN(flatTypeIndex, blob)
	return fv, ok
}

// UnflattenValueN is UnflattenValue that also reports how many bytes the
// decode consumed, so callers can assert an exact fit (consumed == len(blob)).
func (m *Model) UnflattenValueN(flatTypeIndex int, blob []byte) (FlatValue, int, bool) {
	if m == nil || m.file == nil {
		return FlatValue{}, 0, false
	}
	descs, _, ok := decodeVCTP(m.file)
	if !ok {
		return FlatValue{}, 0, false
	}
	return unflattenValue(descs, flatTypeIndex, blob, 0)
}

// maxUnflattenDepth bounds cluster recursion so a malformed (cyclic) member
// list cannot loop forever. The deepest corpus cluster nests two levels
// (error-response: outer cluster → error sub-cluster → scalars); 32 is a
// generous ceiling.
const maxUnflattenDepth = 32

// unflattenValue is the recursive worker behind UnflattenValueN. It decodes
// the VCTP type at flatIdx from the front of blob and returns the decoded
// node, the number of bytes it consumed, and ok. Variable-width leaves
// (String, Cluster) read their span from the data; fixed-width scalars from
// the type. A leaf that would overrun blob fails with ok=false rather than
// panicking, so a wrong type never reads out of bounds.
func unflattenValue(descs []vctp.TypeDescriptor, flatIdx int, blob []byte, depth int) (FlatValue, int, bool) {
	if flatIdx < 0 || flatIdx >= len(descs) || depth > maxUnflattenDepth {
		return FlatValue{}, 0, false
	}
	d := descs[flatIdx]
	fv := FlatValue{FullType: d.FullType.String(), Label: d.Label}

	switch d.FullType {
	case vctp.FullTypeNumInt8, vctp.FullTypeNumInt16, vctp.FullTypeNumInt32, vctp.FullTypeNumInt64:
		w := numWidth(d.FullType)
		if len(blob) < w {
			return FlatValue{}, 0, false
		}
		u := beUint(blob[:w])
		fv.Kind = FlatKindInt
		fv.Uint = u
		fv.Int = signExtend(u, w)
		return fv, w, true

	case vctp.FullTypeNumUInt8, vctp.FullTypeNumUInt16, vctp.FullTypeNumUInt32, vctp.FullTypeNumUInt64:
		w := numWidth(d.FullType)
		if len(blob) < w {
			return FlatValue{}, 0, false
		}
		fv.Kind = FlatKindUint
		fv.Uint = beUint(blob[:w])
		fv.Int = int64(fv.Uint)
		return fv, w, true

	case vctp.FullTypeNumFloat32:
		if len(blob) < 4 {
			return FlatValue{}, 0, false
		}
		fv.Kind = FlatKindFloat
		fv.Float = float64(math.Float32frombits(binary.BigEndian.Uint32(blob[:4])))
		return fv, 4, true

	case vctp.FullTypeNumFloat64:
		if len(blob) < 8 {
			return FlatValue{}, 0, false
		}
		fv.Kind = FlatKindFloat
		fv.Float = math.Float64frombits(binary.BigEndian.Uint64(blob[:8]))
		return fv, 8, true

	case vctp.FullTypeBoolean, vctp.FullTypeBooleanU16:
		// A flattened boolean is one byte; the U16 form pads to two. Both
		// corpus widths read true on any non-zero byte (see constWidthMatches).
		w := 1
		if d.FullType == vctp.FullTypeBooleanU16 {
			w = 2
		}
		if len(blob) < w {
			return FlatValue{}, 0, false
		}
		fv.Kind = FlatKindBool
		fv.Bool = beUint(blob[:w]) != 0
		fv.Uint = beUint(blob[:w])
		return fv, w, true

	case vctp.FullTypeString, vctp.FullTypeString2, vctp.FullTypePasString, vctp.FullTypeCString:
		if len(blob) < 4 {
			return FlatValue{}, 0, false
		}
		n := int(binary.BigEndian.Uint32(blob[:4]))
		if n < 0 || 4+n > len(blob) {
			return FlatValue{}, 0, false
		}
		fv.Kind = FlatKindString
		fv.String = string(blob[4 : 4+n])
		fv.Raw = append([]byte(nil), blob[4:4+n]...)
		return fv, 4 + n, true

	case vctp.FullTypeLVVariant:
		// Opaque. Only ever decoded as a trailing cluster member in the
		// corpus, where consuming the remainder is exact; a non-trailing
		// variant would make the following members mis-parse and the
		// enclosing cluster fail the exact-fit check, which is the honest
		// signal that this layer cannot place it.
		fv.Kind = FlatKindVariant
		fv.Raw = append([]byte(nil), blob...)
		return fv, len(blob), true

	case vctp.FullTypeCluster:
		members, ok := clusterMembers(d.Inner)
		if !ok {
			return FlatValue{}, 0, false
		}
		fv.Kind = FlatKindCluster
		used := 0
		for _, mem := range members {
			if used > len(blob) {
				return FlatValue{}, 0, false
			}
			child, c, ok := unflattenValue(descs, mem, blob[used:], depth+1)
			if !ok {
				return FlatValue{}, 0, false
			}
			fv.Children = append(fv.Children, child)
			used += c
		}
		return fv, used, true

	default:
		return FlatValue{}, 0, false
	}
}

// clusterMembers reads a Cluster type's inner config — a U2 member count
// followed by that many U2 flat-VCTP indices — into the member index list.
// Returns ok=false when the inner bytes are too short for the declared count.
func clusterMembers(inner []byte) ([]int, bool) {
	if len(inner) < 2 {
		return nil, false
	}
	cnt := int(binary.BigEndian.Uint16(inner[:2]))
	if 2+cnt*2 > len(inner) {
		return nil, false
	}
	out := make([]int, cnt)
	for i := 0; i < cnt; i++ {
		out[i] = int(binary.BigEndian.Uint16(inner[2+i*2 : 4+i*2]))
	}
	return out, true
}

// numWidth returns the byte width of a fixed-width VCTP numeric integer type.
func numWidth(ft vctp.FullType) int {
	switch ft {
	case vctp.FullTypeNumInt8, vctp.FullTypeNumUInt8:
		return 1
	case vctp.FullTypeNumInt16, vctp.FullTypeNumUInt16:
		return 2
	case vctp.FullTypeNumInt32, vctp.FullTypeNumUInt32:
		return 4
	case vctp.FullTypeNumInt64, vctp.FullTypeNumUInt64:
		return 8
	default:
		return 0
	}
}

// beUint reads up to 8 bytes as a big-endian unsigned integer.
func beUint(b []byte) uint64 {
	var u uint64
	for _, c := range b {
		u = (u << 8) | uint64(c)
	}
	return u
}

// resolveCompositeDefault decodes a panel control's composite (cluster)
// default blob without a usable panel-heap type pointer. The panel DDO that
// owns the OF__DefaultData leaf references only LabVIEW's internal transaction
// fields (flg/oRt/eof/udf/txd), not the user's data cluster, so the governing
// cluster type is recovered structurally: among every VCTP Cluster type, the
// ones whose flat layout consumes the blob *exactly* are candidates, and the
// best-labelled candidate wins.
//
// Exact byte-consumption across nested length-prefixed strings and back-to-back
// scalars is a very tight constraint — a blob carrying embedded ASCII (e.g.
// "ok", "error") does not consume cleanly against an unrelated cluster — so the
// candidate set is small. Where it holds more than one entry they are
// structurally equivalent and decode to the same value (e.g. the user cluster
// and its internal `txd` twin); the tie-break on labelled-member count just
// picks the copy that carries the real field names. Returns the decoded tree,
// the winning flat index, and ok.
//
// All five corpus composite defaults resolve this way: response.ctl
// {id:"", status:"ok", result:variant}, request.ctl / ndjson-parser.vi
// {id:"", version:1, command:"", params:variant}, error-response.ctl
// {error:{code:0, message:"", details:""}, status:"error", id:""}, and
// show-panel-argument--cluster.ctl {Show Panel?: true}.
func resolveCompositeDefault(descs []vctp.TypeDescriptor, blob []byte) (FlatValue, int, bool) {
	if len(blob) == 0 {
		return FlatValue{}, -1, false
	}
	bestIdx := -1
	var best FlatValue
	bestLabels, bestMembers := -1, -1
	for i := range descs {
		if descs[i].FullType != vctp.FullTypeCluster {
			continue
		}
		fv, used, ok := unflattenValue(descs, i, blob, 0)
		if !ok || used != len(blob) {
			continue
		}
		labels := labelledChildren(fv)
		members := len(fv.Children)
		// Prefer the candidate with the most named fields, then the most
		// members, then the lowest flat index (deterministic).
		if labels > bestLabels || (labels == bestLabels && members > bestMembers) {
			best, bestIdx, bestLabels, bestMembers = fv, i, labels, members
		}
	}
	if bestIdx < 0 {
		return FlatValue{}, -1, false
	}
	return best, bestIdx, true
}

// labelledChildren counts how many direct members of a cluster FlatValue carry
// a non-empty VCTP label (field name).
func labelledChildren(fv FlatValue) int {
	n := 0
	for _, c := range fv.Children {
		if c.Label != "" {
			n++
		}
	}
	return n
}
