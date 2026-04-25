package heap

import (
	"encoding/binary"
	"fmt"
)

// SizeSpec values pylabview recognises in the heap-entry command word.
// See LVheap.py / LVblock.py:5131.
//
//	0      → no content (boolean false)
//	1..4   → that many bytes of opaque content
//	5      → reserved/error
//	6      → variable-length content (u124 length prefix)
//	7      → no content (boolean true)
const (
	SizeSpecBoolFalse byte = 0
	SizeSpecVarLength byte = 6
	SizeSpecBoolTrue  byte = 7
	maxRawTagID            = 1023
)

// Attribute is one entry from a heap-node attribute list, decoded as
// (S124 id, S24 value). Pylabview occasionally rewrites the integer
// value through `attributeValueIntToIntOrEn` to map known IDs onto
// enum members; that semantic projection is intentionally deferred so
// every observed attribute round-trips byte-for-byte.
type Attribute struct {
	ID    int32
	Value int32
}

// Node is one decoded entry from a heap tag stream. The walker emits
// nodes in their on-disk order, attaching each to its current parent
// based on the running scope.
type Node struct {
	// Tag is the post-offset tag identifier. For the typical
	// 10-bit-encoded tag, Tag = RawTagID - 31. For the escape form
	// (RawTagID == 1023), Tag is the explicit signed int32 that
	// follows.
	Tag int32
	// RawTagID is the 10-bit value extracted from the command word,
	// preserved verbatim for round-trip.
	RawTagID uint16
	// HasExplicitTag is true when the entry used the 0x3FF escape and
	// stores the tag as a trailing int32. Encoders need this to
	// reproduce the on-disk byte sequence exactly.
	HasExplicitTag bool
	// Scope distinguishes opening, leaf, and closing tags.
	Scope NodeScope
	// SizeSpec is the raw 3-bit size selector from the command word
	// (0..7). It determines how Content was framed:
	//   0 → no content, bool false
	//   1..4 → exactly N bytes
	//   6 → u124-prefixed variable length
	//   7 → no content, bool true
	SizeSpec byte
	// Attribs is the decoded attribute list (empty when the
	// hasAttrList bit was clear).
	Attribs []Attribute
	// Content is the opaque content bytes (nil for SizeSpec 0/7).
	// Per-tag typed decoding (StdInt, TypeId, Rect, …) is the next
	// layer; this walker stops here so the heap tree round-trips
	// byte-for-byte before any per-tag interpretation lands.
	Content []byte
	// ByteSize is the total wire byte count of this entry, including
	// its command word, optional explicit tag, attribute list, size
	// prefix, and content.
	ByteSize int
	// Children is the list of nodes nested between this node's
	// TagOpen and the matching TagClose (sibling, not child, of the
	// close per pylabview semantics).
	Children []*Node
	// parent is the parent Node when this entry was attached to one;
	// nil for top-level entries.
	parent *Node
}

// Parent returns the node this entry was attached to, or nil for
// top-level entries.
func (n *Node) Parent() *Node { return n.parent }

// HasContent reports whether the entry carries non-bool content bytes.
func (n *Node) HasContent() bool {
	return n.SizeSpec != SizeSpecBoolFalse && n.SizeSpec != SizeSpecBoolTrue
}

// IsBool reports whether the SizeSpec encodes a bare boolean.
func (n *Node) IsBool() bool {
	return n.SizeSpec == SizeSpecBoolFalse || n.SizeSpec == SizeSpecBoolTrue
}

// BoolValue returns the bool encoded by SizeSpec. It is meaningful only
// when IsBool returns true.
func (n *Node) BoolValue() bool { return n.SizeSpec == SizeSpecBoolTrue }

// WalkResult bundles the walker's outputs. Flat is the linear list of
// entries in their on-disk order (mirrors pylabview's
// `section.objects`); Roots is the tree projection where each
// top-level entry is a root and TagOpen children appear under their
// parent.
type WalkResult struct {
	Flat  []*Node
	Roots []*Node
}

// Walk consumes content (the tag-stream bytes from Envelope.Content)
// and returns a WalkResult. It returns an error if the stream is
// truncated or if any entry fails to parse cleanly. The walker stops
// when it has consumed exactly len(content) bytes; trailing bytes are
// surfaced as an error so a corrupt heap never silently produces a
// short tree.
func Walk(content []byte) (WalkResult, error) {
	res := WalkResult{}
	pos := 0
	var parent *Node
	for pos < len(content) {
		node, n, err := decodeEntry(content[pos:])
		if err != nil {
			return WalkResult{}, fmt.Errorf("heap entry at offset %d: %w", pos, err)
		}

		// pylabview semantics: a TagClose walks the parent up
		// *before* the close-tag node is created and attached.
		if node.Scope == NodeScope(NodeScopeTagClose) && parent != nil {
			parent = parent.parent
		}

		node.parent = parent
		res.Flat = append(res.Flat, node)
		if parent != nil {
			parent.Children = append(parent.Children, node)
		} else {
			res.Roots = append(res.Roots, node)
		}

		// TagOpen makes the node the new parent for subsequent entries.
		if node.Scope == NodeScope(NodeScopeTagOpen) {
			parent = node
		}

		pos += n
	}
	if pos != len(content) {
		return WalkResult{}, fmt.Errorf("heap walk consumed %d bytes, want %d", pos, len(content))
	}
	return res, nil
}

// decodeEntry parses a single heap entry at the start of buf. Returns
// the populated Node, the number of bytes consumed, and any error.
func decodeEntry(buf []byte) (*Node, int, error) {
	if len(buf) < 2 {
		return nil, 0, fmt.Errorf("entry header truncated: have %d bytes", len(buf))
	}
	cmdHi := buf[0]
	cmdLo := buf[1]
	sizeSpec := (cmdHi >> 5) & 7
	hasAttrList := (cmdHi >> 4) & 1
	scopeInfo := (cmdHi >> 2) & 3
	rawTagID := uint16(cmdLo) | (uint16(cmdHi&3) << 8)
	pos := 2

	node := &Node{
		RawTagID: rawTagID,
		Scope:    NodeScope(scopeInfo),
		SizeSpec: sizeSpec,
	}

	if rawTagID == maxRawTagID {
		if pos+4 > len(buf) {
			return nil, 0, fmt.Errorf("explicit tag truncated: need 4 bytes, have %d", len(buf)-pos)
		}
		node.Tag = int32(binary.BigEndian.Uint32(buf[pos : pos+4]))
		node.HasExplicitTag = true
		pos += 4
	} else {
		node.Tag = int32(rawTagID) - 31
	}

	if hasAttrList != 0 {
		count, n, err := readU124(buf[pos:])
		if err != nil {
			return nil, 0, fmt.Errorf("attribute count: %w", err)
		}
		pos += n
		node.Attribs = make([]Attribute, 0, count)
		for i := uint32(0); i < count; i++ {
			id, na, err := readS124(buf[pos:])
			if err != nil {
				return nil, 0, fmt.Errorf("attribute %d id: %w", i, err)
			}
			pos += na
			val, nv, err := readS24(buf[pos:])
			if err != nil {
				return nil, 0, fmt.Errorf("attribute %d value: %w", i, err)
			}
			pos += nv
			node.Attribs = append(node.Attribs, Attribute{ID: id, Value: val})
		}
	}

	switch {
	case sizeSpec == SizeSpecBoolFalse, sizeSpec == SizeSpecBoolTrue:
		// no content bytes
	case sizeSpec >= 1 && sizeSpec <= 4:
		if pos+int(sizeSpec) > len(buf) {
			return nil, 0, fmt.Errorf("fixed-size content truncated: need %d bytes, have %d", sizeSpec, len(buf)-pos)
		}
		node.Content = append([]byte(nil), buf[pos:pos+int(sizeSpec)]...)
		pos += int(sizeSpec)
	case sizeSpec == SizeSpecVarLength:
		size, n, err := readU124(buf[pos:])
		if err != nil {
			return nil, 0, fmt.Errorf("var-length size prefix: %w", err)
		}
		pos += n
		if pos+int(size) > len(buf) {
			return nil, 0, fmt.Errorf("var-length content truncated: declared %d bytes, have %d", size, len(buf)-pos)
		}
		node.Content = append([]byte(nil), buf[pos:pos+int(size)]...)
		pos += int(size)
	case sizeSpec == 5:
		return nil, 0, fmt.Errorf("reserved sizeSpec 5 not supported")
	}

	node.ByteSize = pos
	return node, pos, nil
}
