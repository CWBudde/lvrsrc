package lvvi

import (
	"github.com/CWBudde/lvrsrc/internal/codecs"
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
