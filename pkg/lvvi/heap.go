package lvvi

import (
	"fmt"
	"strings"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/bdhb"
	"github.com/CWBudde/lvrsrc/internal/codecs/fphb"
	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
)

// HeapAttribute is one entry from a heap-node attribute list, projected
// out of internal/codecs/heap.Attribute so callers do not have to import
// the internal package.
type HeapAttribute struct {
	ID    int32
	Value int32
}

// HeapNode is the lvvi-level projection of one decoded heap entry. It
// mirrors internal/codecs/heap.Node but resolves the parent pointer
// cycle into a Parent index and the Children pointer slice into an
// index slice, so callers can serialise or compare HeapTree without
// worrying about cycles.
//
// Parent is -1 for top-level nodes; otherwise it is a valid index into
// the enclosing HeapTree.Nodes slice.
type HeapNode struct {
	// Tag is the post-offset tag identifier (RawTagID minus the 31
	// offset, or the explicit signed int32 from the 0x3FF escape).
	Tag int32
	// RawTagID is the 10-bit value extracted from the command word.
	RawTagID uint16
	// HasExplicitTag is true when the entry used the 0x3FF escape.
	HasExplicitTag bool
	// Scope is "open", "leaf", or "close".
	Scope string
	// SizeSpec is the raw 3-bit size selector from the command word.
	SizeSpec byte
	// Attributes is the decoded attribute list (empty when absent).
	Attributes []HeapAttribute
	// Content is the opaque content bytes (nil for SizeSpec 0/7).
	Content []byte
	// ByteSize is the wire byte count of the entry.
	ByteSize int
	// Parent is the index of the parent HeapNode in the enclosing
	// HeapTree.Nodes, or -1 for top-level entries.
	Parent int
	// Children are indices into the enclosing HeapTree.Nodes for the
	// nodes nested between this entry's TagOpen and matching TagClose
	// (per pylabview semantics, the close-tag itself is a sibling, not
	// a child).
	Children []int
}

// HeapTree is the public projection of a decoded FPHb (Front-Panel
// Heap) tag stream. Nodes is the flat list of entries in their on-disk
// order; Roots holds the indices of the top-level entries.
type HeapTree struct {
	Nodes []HeapNode
	Roots []int
}

// FrontPanel returns the decoded FPHb (Front-Panel Heap) tree for the
// wrapped file. Returns ok=false when no FPHb section is present or the
// codec failed to decode it (the codec records its own validation
// issues; this accessor surfaces only the success/empty distinction).
//
// The returned HeapTree is a projection of the internal heap.WalkResult
// with parent/child cycles resolved into integer indices.
func (m *Model) FrontPanel() (HeapTree, bool) {
	if m == nil || m.file == nil {
		return HeapTree{}, false
	}
	refs := sectionsOf(m.file, string(fphb.FourCC))
	if len(refs) == 0 {
		return HeapTree{}, false
	}
	ctx := codecs.Context{FileVersion: m.file.Header.FormatVersion, Kind: m.file.Kind}
	raw, err := (fphb.Codec{}).Decode(ctx, refs[0].Payload)
	if err != nil {
		return HeapTree{}, false
	}
	v, ok := raw.(fphb.Value)
	if !ok {
		return HeapTree{}, false
	}
	return projectHeapTree(v.Tree), true
}

func projectHeapTree(w heap.WalkResult) HeapTree {
	idx := make(map[*heap.Node]int, len(w.Flat))
	for i, n := range w.Flat {
		idx[n] = i
	}

	nodes := make([]HeapNode, len(w.Flat))
	for i, n := range w.Flat {
		parent := -1
		if p := n.Parent(); p != nil {
			if pi, ok := idx[p]; ok {
				parent = pi
			}
		}
		var children []int
		if len(n.Children) > 0 {
			children = make([]int, 0, len(n.Children))
			for _, c := range n.Children {
				if ci, ok := idx[c]; ok {
					children = append(children, ci)
				}
			}
		}
		var attribs []HeapAttribute
		if len(n.Attribs) > 0 {
			attribs = make([]HeapAttribute, len(n.Attribs))
			for j, a := range n.Attribs {
				attribs[j] = HeapAttribute{ID: a.ID, Value: a.Value}
			}
		}
		var content []byte
		if len(n.Content) > 0 {
			content = append([]byte(nil), n.Content...)
		}
		nodes[i] = HeapNode{
			Tag:            n.Tag,
			RawTagID:       n.RawTagID,
			HasExplicitTag: n.HasExplicitTag,
			Scope:          scopeString(n.Scope),
			SizeSpec:       n.SizeSpec,
			Attributes:     attribs,
			Content:        content,
			ByteSize:       n.ByteSize,
			Parent:         parent,
			Children:       children,
		}
	}

	roots := make([]int, 0, len(w.Roots))
	for _, r := range w.Roots {
		if ri, ok := idx[r]; ok {
			roots = append(roots, ri)
		}
	}
	return HeapTree{Nodes: nodes, Roots: roots}
}

// BlockDiagram returns the decoded BDHb (Block-Diagram Heap) tree for
// the wrapped file. Returns ok=false when no BDHb section is present or
// the codec failed to decode it. The returned HeapTree shares the same
// projection conventions as FrontPanel() — Parent/Children indices are
// pre-resolved, and Scope is one of "open", "leaf", "close".
func (m *Model) BlockDiagram() (HeapTree, bool) {
	if m == nil || m.file == nil {
		return HeapTree{}, false
	}
	refs := sectionsOf(m.file, string(bdhb.FourCC))
	if len(refs) == 0 {
		return HeapTree{}, false
	}
	ctx := codecs.Context{FileVersion: m.file.Header.FormatVersion, Kind: m.file.Kind}
	raw, err := (bdhb.Codec{}).Decode(ctx, refs[0].Payload)
	if err != nil {
		return HeapTree{}, false
	}
	v, ok := raw.(bdhb.Value)
	if !ok {
		return HeapTree{}, false
	}
	return projectHeapTree(v.Tree), true
}

// HeapNodeClass returns the object class of a heap node, taken from its
// SL__class attribute. Object (open) nodes carry this attribute; field
// leaves do not. ok is false when the node carries no class attribute.
func HeapNodeClass(n HeapNode) (heap.ClassTag, bool) {
	for _, a := range n.Attributes {
		if a.ID == int32(heap.SystemAttribTagClass) {
			return heap.ClassTag(a.Value), true
		}
	}
	return 0, false
}

// ParentTopClass returns the class of the nearest enclosing object at or
// above tree.Nodes[idx], found by walking the parent chain for the first
// SL__class attribute. It mirrors pylabview's parentTopClassEn and is the
// context used to resolve a child node's field tags. When no ancestor
// carries a class, it returns heap.ClassDefault (SL__oHExt), matching
// pylabview's fallback. Callers naming a node N pass N.Parent here.
func ParentTopClass(tree HeapTree, idx int) heap.ClassTag {
	for i := 0; i < 128 && idx >= 0 && idx < len(tree.Nodes); i++ {
		if cls, ok := HeapNodeClass(tree.Nodes[idx]); ok {
			return cls
		}
		idx = tree.Nodes[idx].Parent
	}
	return heap.ClassDefault
}

// HeapTagNameAt resolves the best display name for tree.Nodes[idx],
// honouring LabVIEW's context-dependent tag namespace.
//
// LabVIEW heaps reuse the same integer tag namespace across unrelated
// enum families, and a positive tag's meaning depends on the enclosing
// object's class (pylabview's tagIdToEnum). This resolver replicates that:
//
//  1. Negative (system) tags name the node's structural role directly —
//     SL__rootObject, SL__arrayElement — which is more descriptive than
//     the object's class, so these win even when a class is present.
//  2. Positive-tag object nodes (those carrying an SL__class attribute)
//     are named by their class, e.g. SL__Image or SL__pane — the node's
//     true identity (the older resolver wrongly used the tag here, naming
//     a pane "SL__aInsDCO").
//  3. Remaining nodes (field leaves) are named by their tag resolved in
//     the parent object's class context: positive tags resolve in the
//     parent class's per-class field list, then the generic OBJ_FIELD_TAGS
//     family (e.g. tag 0 inside an SL__Image is OF__ImageResID, and tag
//     172 anywhere is OF__objFlags, never the colliding SL__grouper).
//
// Unresolved tags fall back to "Tag(N)" so the UI never sees an empty
// string; callers can detect a fallback by the parenthesis.
func HeapTagNameAt(tree HeapTree, idx int) string {
	if idx < 0 || idx >= len(tree.Nodes) {
		return "Tag(?)"
	}
	n := tree.Nodes[idx]
	if n.Tag >= 0 {
		if cls, ok := HeapNodeClass(n); ok {
			if name := heap.ClassTag(cls).String(); !strings.Contains(name, "(") {
				return name
			}
		}
	}
	if name, _, ok := heap.ResolveTagName(n.Tag, ParentTopClass(tree, n.Parent)); ok {
		return name
	}
	return fmt.Sprintf("Tag(%d)", n.Tag)
}

// HeapTagName resolves a single HeapNode's tag to its best-known name
// without parent context. Prefer HeapTagNameAt, which knows the enclosing
// object's class and so resolves context-dependent field tags correctly;
// this context-free form is kept for callers that only hold a node. It
// applies the same naming policy as HeapTagNameAt (system role, then
// class, then field) but resolves field tags in the default (SL__oHExt)
// class context since it has no parent chain to walk.
func HeapTagName(n HeapNode) string {
	if n.Tag >= 0 {
		if cls, ok := HeapNodeClass(n); ok {
			if name := heap.ClassTag(cls).String(); !strings.Contains(name, "(") {
				return name
			}
		}
	}
	if name, _, ok := heap.ResolveTagName(n.Tag, heap.ClassDefault); ok {
		return name
	}
	return fmt.Sprintf("Tag(%d)", n.Tag)
}

func scopeString(s heap.NodeScope) string {
	switch s {
	case heap.NodeScopeTagOpen:
		return "open"
	case heap.NodeScopeTagClose:
		return "close"
	case heap.NodeScopeTagLeaf:
		return "leaf"
	default:
		return "leaf"
	}
}
