package heap

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Rect is a heap-node rectangle: pylabview's HeapNodeRect (LVheap.py:1725).
// Bytes: 4 × big-endian int16 in the order Left, Top, Right, Bottom.
type Rect struct {
	Left, Top, Right, Bottom int16
}

// Point is a heap-node 2D point: pylabview's HeapNodePoint (LVheap.py:1765).
// Bytes: 2 × big-endian int16 in the order X, Y. (Note: pylabview's XML
// serialiser writes "(y, x)" but the *binary* layout puts X first; the
// codec follows the binary layout.)
type Point struct {
	X, Y int16
}

// AsStdInt interprets the node's Content as a big-endian integer of
// length len(Content). pylabview's HeapNodeStdInt (LVheap.py:1664)
// supports both signed and unsigned variants.
//
// Returns an error when:
//   - the node is bool-shaped (SizeSpec 0 or 7, no content body), or
//   - len(Content) > 8 (cannot fit a Go int64).
//
// Single-byte through 8-byte payloads decode in the obvious way.
func (n *Node) AsStdInt(signed bool) (int64, error) {
	if n.IsBool() {
		return 0, fmt.Errorf("AsStdInt: node is bool-shaped (SizeSpec=%d)", n.SizeSpec)
	}
	if len(n.Content) > 8 {
		return 0, fmt.Errorf("AsStdInt: %d-byte content cannot fit int64", len(n.Content))
	}
	if len(n.Content) == 0 {
		return 0, nil
	}
	var u uint64
	for _, b := range n.Content {
		u = (u << 8) | uint64(b)
	}
	if !signed {
		return int64(u), nil
	}
	// Sign-extend from len(Content) bytes.
	bits := uint(len(n.Content) * 8)
	if u&(1<<(bits-1)) != 0 {
		// Set all higher bits to 1.
		u |= ^uint64(0) << bits
	}
	return int64(u), nil
}

// AsTypeID is a thin alias for AsStdInt(true): pylabview's HeapNodeTypeId
// (LVheap.py:1707) is HeapNodeStdInt with btlen=-1, signed=True.
func (n *Node) AsTypeID() (int64, error) {
	return n.AsStdInt(true)
}

// AsRect decodes the node's Content as a Rect. The content must be
// exactly 8 bytes.
func (n *Node) AsRect() (Rect, error) {
	if len(n.Content) != 8 {
		return Rect{}, fmt.Errorf("AsRect: content is %d bytes, want 8", len(n.Content))
	}
	c := n.Content
	return Rect{
		Left:   int16(binary.BigEndian.Uint16(c[0:2])),
		Top:    int16(binary.BigEndian.Uint16(c[2:4])),
		Right:  int16(binary.BigEndian.Uint16(c[4:6])),
		Bottom: int16(binary.BigEndian.Uint16(c[6:8])),
	}, nil
}

// AsPoint decodes the node's Content as a Point. The content must be
// exactly 4 bytes.
func (n *Node) AsPoint() (Point, error) {
	if len(n.Content) != 4 {
		return Point{}, fmt.Errorf("AsPoint: content is %d bytes, want 4", len(n.Content))
	}
	c := n.Content
	return Point{
		X: int16(binary.BigEndian.Uint16(c[0:2])),
		Y: int16(binary.BigEndian.Uint16(c[2:4])),
	}, nil
}

// AsFloat32 decodes the node's Content as a big-endian IEEE-754 float32.
// pylabview's `HeapNodeTDDataFill.parseRSRCContentDirect` (LVheap.py:1986)
// reads exactly 4 bytes for `NumFloat32`/`UnitFloat32`. Float content is
// not subject to pylabview's `shrinkRepeatedBits` truncation — only
// integers are.
func (n *Node) AsFloat32() (float32, error) {
	if len(n.Content) != 4 {
		return 0, fmt.Errorf("AsFloat32: content is %d bytes, want 4", len(n.Content))
	}
	bits := binary.BigEndian.Uint32(n.Content)
	return math.Float32frombits(bits), nil
}

// AsFloat64 decodes the node's Content as a big-endian IEEE-754 float64.
// pylabview's `HeapNodeTDDataFill.parseRSRCContentDirect` reads exactly
// 8 bytes for `NumFloat64`/`UnitFloat64`.
func (n *Node) AsFloat64() (float64, error) {
	if len(n.Content) != 8 {
		return 0, fmt.Errorf("AsFloat64: content is %d bytes, want 8", len(n.Content))
	}
	bits := binary.BigEndian.Uint64(n.Content)
	return math.Float64frombits(bits), nil
}

// AsString returns the node's Content as a Go string plus a NULL flag.
// pylabview's HeapNodeString (LVheap.py:1797) treats a node whose
// content is the boolean False (SizeSpec 0, no body) as the special
// `[NULL]` value; SizeSpec 6 with a zero-length payload is a
// legitimate empty string.
//
// The returned bytes are not transcoded — callers that need a specific
// charset should decode `Node.Content` themselves.
func (n *Node) AsString() (string, bool, error) {
	if n.SizeSpec == SizeSpecBoolFalse {
		return "", true, nil
	}
	if n.SizeSpec == SizeSpecBoolTrue {
		// Pylabview also treats bool-true content as a malformed string;
		// surface the case rather than guessing.
		return "", true, nil
	}
	return string(n.Content), false, nil
}
