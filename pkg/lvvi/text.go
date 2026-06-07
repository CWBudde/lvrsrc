package lvvi

import (
	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
)

// HeapTextKind distinguishes the two ways LabVIEW stores text in a heap.
type HeapTextKind string

const (
	// HeapTextString is a single raw string stored directly as the node's
	// content (no length prefix), e.g. a control's NAME_LABEL text via
	// OF__text, a block-diagram node name via OF__nodeName, or a numeric
	// format via OF__format.
	HeapTextString HeapTextKind = "string"
	// HeapTextLabelList is a list of one-byte-length-prefixed strings
	// stored under an SL__multiLabel object (OF__buf) — the item text of
	// ring/enum controls and the per-state text of booleans.
	HeapTextLabelList HeapTextKind = "label-list"
)

// heapStringTagNames is the set of context-resolved tag names whose heap
// content is a raw text string, mirroring pylabview's NODE_STRING_TAGS_LIST.
//
// Detection keys on the *resolved* name (heap.ResolveTagName) rather than
// the bare tag id because several of these collide with non-text fields in
// other classes — most importantly OF__text is tag 3, which means
// OF__activePlot outside SL__textHair. Resolving in the parent-class
// context disambiguates them exactly the way pylabview does.
var heapStringTagNames = map[string]struct{}{
	"OF__text":         {}, // control / pane / terminal name label (under SL__textHair)
	"OF__format":       {}, // numeric/string display format spec
	"OF__methName":     {}, // block-diagram method name
	"OF__nodeName":     {}, // block-diagram node name
	"OF__tagDLLName":   {}, // Call Library node DLL name
	"OF__PropItemName": {}, // property-node item name
}

// HeapStringAt returns the decoded raw-string content of tree.Nodes[idx]
// when that node is a text-bearing string tag in its parent-class context.
// ok is false for any non-text node.
func HeapStringAt(tree HeapTree, idx int) (string, bool) {
	if idx < 0 || idx >= len(tree.Nodes) {
		return "", false
	}
	n := tree.Nodes[idx]
	name, _, ok := heap.ResolveTagName(n.Tag, ParentTopClass(tree, n.Parent))
	if !ok {
		return "", false
	}
	if _, isStr := heapStringTagNames[name]; !isStr {
		return "", false
	}
	return string(n.Content), true
}

// HeapLabelListAt decodes the P-string list stored in an OF__buf node whose
// enclosing object is an SL__multiLabel — the item text of ring/enum
// controls and the per-state text of booleans. ok is false for any other
// node; on a truncated list it returns the strings decoded so far with
// ok=false so callers can still surface partial text.
//
// Only SL__multiLabel is decoded, matching pylabview: SL__bigMultiLabel
// uses a different (wider length) layout and does not appear in the corpus.
func HeapLabelListAt(tree HeapTree, idx int) ([]string, bool) {
	if idx < 0 || idx >= len(tree.Nodes) {
		return nil, false
	}
	n := tree.Nodes[idx]
	if n.Tag != int32(heap.FieldTagBuf) {
		return nil, false
	}
	if ParentTopClass(tree, n.Parent) != heap.ClassTagMultiLabel {
		return nil, false
	}
	return decodePStrList(n.Content)
}

// decodePStrList splits a buffer of consecutive one-byte-length-prefixed
// strings. ok is false when a length runs past the end of the buffer; the
// strings decoded before the truncation are still returned.
func decodePStrList(b []byte) ([]string, bool) {
	out := []string{}
	for len(b) > 0 {
		n := int(b[0])
		b = b[1:]
		if n > len(b) {
			return out, false
		}
		out = append(out, string(b[:n]))
		b = b[n:]
	}
	return out, true
}

// HeapText is one decoded text element in a heap tree together with the
// context needed to attribute it.
type HeapText struct {
	// NodeIndex is the index of the text-bearing node in tree.Nodes.
	NodeIndex int
	// Tag is the node's raw tag; TagName is its resolved name.
	Tag     int32
	TagName string
	// Kind is the storage encoding (string vs label-list).
	Kind HeapTextKind
	// Role is the PartID of the nearest enclosing part, identifying the
	// label's role (NAME_LABEL, RING_TEXT, …). PartIDNone when no
	// enclosing part carries an OF__partID.
	Role PartID
	// OwnerClass is the nearest enclosing object class that is not a text
	// wrapper (skips SL__textHair / SL__fontRun / SL__multiLabel), e.g.
	// SL__label for a name label or SL__stdRing for ring item text.
	// classTagNone (-1) when none is found; OwnerName is "" in that case.
	OwnerClass heap.ClassTag
	OwnerName  string
	// Lines holds the text: one entry for a string, N entries for a
	// label-list.
	Lines []string
}

// HeapTexts enumerates every decoded text element in the tree in node
// order, attributing each to its part role and owning object class. It is
// the showcase-facing accessor: callers can filter by Kind or Role (e.g.
// Role.IsLabel) to pull control names, captions, or ring item text.
func HeapTexts(tree HeapTree) []HeapText {
	var out []HeapText
	for i := range tree.Nodes {
		if s, ok := HeapStringAt(tree, i); ok {
			out = append(out, newHeapText(tree, i, HeapTextString, []string{s}))
			continue
		}
		if lines, ok := HeapLabelListAt(tree, i); ok {
			out = append(out, newHeapText(tree, i, HeapTextLabelList, lines))
		}
	}
	return out
}

func newHeapText(tree HeapTree, idx int, kind HeapTextKind, lines []string) HeapText {
	n := tree.Nodes[idx]
	owner, ownerName := nearestTextOwner(tree, n.Parent)
	return HeapText{
		NodeIndex:  idx,
		Tag:        n.Tag,
		TagName:    HeapTagNameAt(tree, idx),
		Kind:       kind,
		Role:       nearestPartRole(tree, n.Parent),
		OwnerClass: owner,
		OwnerName:  ownerName,
		Lines:      lines,
	}
}

// textOwnerSkip are the cosmetic/wrapper classes that hold text but are not
// the meaningful owner of it; nearestTextOwner walks past them.
var textOwnerSkip = map[heap.ClassTag]bool{
	heap.ClassTagTextHair:      true,
	heap.ClassTagFontRun:       true,
	heap.ClassTagMultiLabel:    true,
	heap.ClassTagBigMultiLabel: true,
}

// classTagNone is the sentinel HeapText.OwnerClass value used when no
// enclosing owner class is found. ClassTag 0 is a real class (SL__fontRun),
// so -1 is used instead.
const classTagNone heap.ClassTag = -1

// nearestTextOwner walks up from idx and returns the first class attribute
// that is not a text wrapper. Returns (classTagNone, "") when none.
func nearestTextOwner(tree HeapTree, idx int) (heap.ClassTag, string) {
	for i := 0; i < 128 && idx >= 0 && idx < len(tree.Nodes); i++ {
		if cls, ok := HeapNodeClass(tree.Nodes[idx]); ok && !textOwnerSkip[cls] {
			return cls, cls.String()
		}
		idx = tree.Nodes[idx].Parent
	}
	return classTagNone, ""
}

// nearestPartRole walks up from idx and returns the PartID of the first
// ancestor (inclusive) that carries an OF__partID child, identifying the
// role of the part that contains the text. PartIDNone when none is found.
func nearestPartRole(tree HeapTree, idx int) PartID {
	for i := 0; i < 128 && idx >= 0 && idx < len(tree.Nodes); i++ {
		if v, ok := FindScalarChild(tree, idx, int32(heap.FieldTagPartID)); ok {
			return PartID(v.Unsigned)
		}
		idx = tree.Nodes[idx].Parent
	}
	return PartIDNone
}

// FrontPanelTexts decodes and enumerates all text in the front-panel heap.
// ok is false when the file has no decodable FPHb section.
func (m *Model) FrontPanelTexts() ([]HeapText, bool) {
	tree, ok := m.FrontPanel()
	if !ok {
		return nil, false
	}
	return HeapTexts(tree), true
}

// BlockDiagramTexts decodes and enumerates all text in the block-diagram
// heap. ok is false when the file has no decodable BDHb section.
func (m *Model) BlockDiagramTexts() ([]HeapText, bool) {
	tree, ok := m.BlockDiagram()
	if !ok {
		return nil, false
	}
	return HeapTexts(tree), true
}
