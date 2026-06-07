package lvvi

import "github.com/CWBudde/lvrsrc/internal/codecs/heap"

// FrontPanelType joins one front-panel data object to the VCTP type it
// declares. Each front-panel control, indicator, and tunnel carries an
// OF__typeDesc (tag 283) leaf in the FPHb (front-panel heap) whose content
// is a 0-based ordinal into the VCTP top-types list — the same TopTypes
// indirection the block-diagram constant join uses (see TypedConst).
// TopTypes[TopIndex] is the flat VCTP index of the object's type.
//
// This is the front-panel mirror of BlockDiagramConstants' type join: the
// block diagram resolves a constant's type from the nearest-preceding
// OF__typeDesc within its enclosing object, while the panel heap declares
// each data object's type with its own OF__typeDesc leaf directly. The
// resolution verifies cleanly against the controlled fixtures — Numeric42's
// DBL indicator resolves to NumFloat64, the _I8 control to NumInt8, _SGL to
// NumFloat32, and _CDB to NumComplex128.
type FrontPanelType struct {
	// NodeIndex is the FPHb heap node index of the OF__typeDesc leaf.
	NodeIndex int
	// Parent is the heap node index of the enclosing data object, or -1.
	Parent int
	// TopIndex is the 0-based top-types ordinal read from the leaf content,
	// or -1 when the content is not a readable ordinal.
	TopIndex int
	// TypeIndex is the resolved flat VCTP index (TopTypes[TopIndex]), or -1
	// when the ordinal is out of range.
	TypeIndex int
	// Type is the resolved VCTP type descriptor; valid only when HasType.
	Type TypeDescriptor
	// HasType reports whether a VCTP type resolved for the object.
	HasType bool
}

// FrontPanelTypes returns every front-panel data object in the wrapped
// file, each joined to its VCTP type. Returns ok=false when there is no
// FPHb (front-panel heap) section.
//
// One entry is produced per OF__typeDesc (tag 283) leaf in panel order.
// The leaf content is a big-endian top-types ordinal; TopTypes[ordinal] is
// the flat VCTP index and descs[flat] the resolved type. Objects whose
// ordinal is out of range (or whose content is not a readable ordinal)
// still appear, with HasType=false, so the panel-order mapping is never
// silently truncated.
func (m *Model) FrontPanelTypes() ([]FrontPanelType, bool) {
	if m == nil || m.file == nil {
		return nil, false
	}
	tree, ok := m.FrontPanel()
	if !ok {
		return nil, false
	}
	descs, tops, _ := decodeVCTP(m.file)

	var out []FrontPanelType
	for i, n := range tree.Nodes {
		if n.Tag != int32(heap.FieldTagTypeDesc) || n.Scope != "leaf" {
			continue
		}
		fp := FrontPanelType{
			NodeIndex: i,
			Parent:    n.Parent,
			TopIndex:  -1,
			TypeIndex: -1,
		}
		if top, ok := bigEndianUint(n.Content); ok && top < uint64(len(tops)) {
			fp.TopIndex = int(top)
			flat := int(tops[top])
			fp.TypeIndex = flat
			if flat >= 0 && flat < len(descs) {
				fp.Type = projectTypeDesc(descs[flat])
				fp.HasType = true
			}
		}
		out = append(out, fp)
	}
	return out, true
}

// FrontPanelDefault is a raw OF__DefaultData (tag 571) leaf from the FPHb
// heap — the flattened default value of a front-panel data object.
//
// In the observed corpus every OF__DefaultData is a *composite* cluster
// default (e.g. a request/response cluster carrying strings and a version
// number), not a fixed-width numeric scalar. The object-level type does not
// resolve cleanly through the TopTypes ordinal here — it points at a
// Void/placeholder wrapper rather than the cluster type — so decoding the
// value would require a recursive VCTP cluster flatten/unflatten walk that
// this type-only layer deliberately does not attempt. The raw bytes are
// surfaced verbatim for callers that want them, without an invented type.
type FrontPanelDefault struct {
	// NodeIndex is the FPHb heap node index of the OF__DefaultData leaf.
	NodeIndex int
	// Parent is the heap node index of the enclosing data object, or -1.
	Parent int
	// Raw is the verbatim OF__DefaultData payload (a flattened, typically
	// composite, default value).
	Raw []byte
}

// FrontPanelDefaults returns every raw OF__DefaultData (tag 571) leaf in
// the wrapped file's front-panel heap, in panel order. Returns ok=false
// when there is no FPHb section. The bytes are surfaced verbatim; see
// FrontPanelDefault for why no typed value is decoded.
func (m *Model) FrontPanelDefaults() ([]FrontPanelDefault, bool) {
	if m == nil || m.file == nil {
		return nil, false
	}
	tree, ok := m.FrontPanel()
	if !ok {
		return nil, false
	}
	var out []FrontPanelDefault
	for i, n := range tree.Nodes {
		if n.Tag != int32(heap.FieldTagDefaultData) || n.Scope != "leaf" {
			continue
		}
		out = append(out, FrontPanelDefault{
			NodeIndex: i,
			Parent:    n.Parent,
			Raw:       append([]byte(nil), n.Content...),
		})
	}
	return out, true
}
