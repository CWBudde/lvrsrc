package lvvi

import "github.com/CWBudde/lvrsrc/internal/codecs/heap"

// WireMode classifies an OF__compressedWireTable chunk by its `byte1`
// flag — the family of encoding used for the rest of the payload.
//
// The controlled-fixture spike (phases 13.2–13.5) observed three distinct
// values across all initial corpus chunks and deliberately-varied
// fixtures. Anything else falls to WireModeOther so the decoder
// degrades gracefully on unfamiliar shapes.
type WireMode uint8

// Members of WireMode.
const (
	// WireModeAutoChain (0x08) is the most common form: a single
	// edge between two terminals where LabVIEW's auto-router infers
	// the path from the terminal positions. The trailing payload is
	// LEB128 varints encoding only the deltas the router cannot
	// reconstruct from terminal bounds (perpendicular step at each
	// elbow, source-side seed values). Empty payload (`0208`) means
	// terminals are aligned and the wire is straight.
	WireModeAutoChain WireMode = 0x08

	// WireModeManualChain (0x04) flags a single edge whose user
	// placed explicit waypoints. The payload still LEB128-decodes
	// but contains many more entries than the auto-routed form
	// because each manual bend records its own coordinates rather
	// than being inferred.
	WireModeManualChain WireMode = 0x04

	// WireModeTree (0x00) is a multi-endpoint wire network — a
	// fan-out or join. Payload size always equals `byte0 × 2`; the
	// trailing data is a stream of fixed-width 2-byte records, the
	// first of which is the (byte0, byte1) header itself, and the
	// remaining `byte0 - 1` records describe branches plus a shared
	// geometry tail. Per-record semantics are tracked as Phase
	// Phase 13.3.
	WireModeTree WireMode = 0x00

	// WireModeOther covers any byte1 value we have not yet
	// classified. Payload is preserved as Raw bytes only.
	WireModeOther WireMode = 0xFF
)

// String returns the symbolic mode name for diagnostics.
func (m WireMode) String() string {
	switch m {
	case WireModeAutoChain:
		return "auto-chain"
	case WireModeManualChain:
		return "manual-chain"
	case WireModeTree:
		return "tree"
	case WireModeOther:
		return "other"
	}
	return "other"
}

// Wire is the typed projection of an OF__compressedWireTable leaf.
//
// Wire is shape-aware but content-agnostic — the per-varint and
// per-record semantics inside ChainGeometry / TreeRecords are still
// being mapped (Phase 13.5). Callers that need round-trip-safe
// access to the original bytes should use Raw; ChainGeometry and
// TreeRecords are projections of those bytes for diagnostic and
// renderer use.
type Wire struct {
	// Mode is the classifier read from byte1 of the payload.
	Mode WireMode
	// Waypoints is the byte0 of the payload — for chain modes, the
	// total number of path waypoints (endpoints + internal corners);
	// for tree mode, the total record count including the header.
	Waypoints uint8
	// ChainGeometry is populated for WireModeAutoChain and
	// WireModeManualChain. It holds the LEB128-decoded varints from
	// the payload after byte0 / byte1. Empty for chains with no
	// trailing geometry (e.g. `0208`).
	ChainGeometry []uint64
	// TreeRecords is populated for WireModeTree. It splits the
	// payload after byte0 / byte1 into 2-byte records.
	TreeRecords [][2]byte
	// Raw is the original payload bytes, preserved verbatim so
	// callers can re-examine or round-trip them.
	Raw []byte
}

// HeapWire decodes an OF__compressedWireTable leaf at
// tree.Nodes[nodeIdx] into a Wire. Returns ok=false on out-of-range
// index, wrong tag, or empty content.
//
// A non-empty payload always yields a populated Wire: at minimum
// Mode, Waypoints, and Raw are set. ChainGeometry / TreeRecords are
// populated only for the matching modes.
func HeapWire(tree HeapTree, nodeIdx int) (Wire, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return Wire{}, false
	}
	n := tree.Nodes[nodeIdx]
	if n.Tag != int32(heap.FieldTagCompressedWireTable) {
		return Wire{}, false
	}
	if len(n.Content) < 2 {
		return Wire{}, false
	}
	w := Wire{
		Waypoints: n.Content[0],
		Mode:      classifyWireMode(n.Content[1]),
		Raw:       n.Content,
	}
	payload := n.Content[2:]
	switch w.Mode {
	case WireModeAutoChain, WireModeManualChain:
		w.ChainGeometry = decodeLEB128(payload)
	case WireModeTree:
		w.TreeRecords = splitTreeRecords(payload)
	}
	return w, true
}

func classifyWireMode(b byte) WireMode {
	switch WireMode(b) {
	case WireModeAutoChain, WireModeManualChain, WireModeTree:
		return WireMode(b)
	}
	return WireModeOther
}

// decodeLEB128 reads payload as a stream of unsigned LEB128 varints,
// returning the decoded values. A trailing byte with the
// continuation bit still set (truncated varint) is silently dropped
// — the controlled-fixture spike never observed truncation in
// chain-mode payloads.
func decodeLEB128(payload []byte) []uint64 {
	if len(payload) == 0 {
		return nil
	}
	var out []uint64
	var v uint64
	var shift uint
	for _, b := range payload {
		v |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			out = append(out, v)
			v = 0
			shift = 0
			continue
		}
		shift += 7
	}
	return out
}

// splitTreeRecords slices payload into 2-byte records. A trailing
// odd byte is dropped — tree-mode chunks the spike observed always
// satisfied `len(content) == byte0 * 2`, so an odd trailing byte
// would itself be a malformed chunk worth surfacing later.
func splitTreeRecords(payload []byte) [][2]byte {
	if len(payload) == 0 {
		return nil
	}
	out := make([][2]byte, 0, len(payload)/2)
	for i := 0; i+1 < len(payload); i += 2 {
		out = append(out, [2]byte{payload[i], payload[i+1]})
	}
	return out
}

// CountWireNetworks reports how many wire-network chunks tree
// carries. Equivalent to CountCompressedWireTables, kept under the
// new name so the renderer can use accurate "network" terminology
// post-Phase 13.1 spike.
func CountWireNetworks(tree HeapTree) int {
	return CountCompressedWireTables(tree)
}

// ChainAutoPath is the typed Phase 13 projection of a
// WireModeAutoChain payload. It is populated by Wire.ChainAutoPath
// when the payload matches a recognised shape (straight wire or
// single-elbow L-shape); longer multi-elbow auto-routed payloads
// currently return ok=false and callers fall back to the raw
// ChainGeometry varint stream.
//
// Renderer composition (Phase 14): a wire is drawn from the
// source-glyph's output anchor → +SourceAnchorX horizontally →
// ±YStep vertically (sign from YStep) → continue horizontally to
// the sink-glyph's input anchor. The sink's x-position comes from
// its OF__bounds — the chunk does not encode it because the
// auto-router stretches the post-elbow segment to fill the gap.
type ChainAutoPath struct {
	// Straight reports whether the wire takes a direct line with
	// no elbow — equivalent to the trivial `0208` sentinel
	// (terminals y-aligned, no encoded geometry).
	Straight bool
	// YStep is the signed perpendicular step at the wire's single
	// elbow, in pixels. Positive when the sink is below the source,
	// negative when above. Zero when Straight is true.
	YStep int
	// SourceAnchorX is the elbow's horizontal distance from the
	// source-glyph's output anchor, in pixels. This is glyph-
	// specific (= 65 for the I32 Numeric Constant in our corpus)
	// and stays stable as the sink moves horizontally because the
	// auto-router stretches the post-elbow segment, not the pre-
	// elbow one. Zero when Straight is true.
	SourceAnchorX uint64
}

// ChainAutoPath returns the typed L-shape projection for a chain-
// auto wire when the payload matches a recognised shape. Returns
// ok=false for non-auto modes, multi-elbow payloads, or any payload
// shape outside the spike's empirical lookup.
func (w Wire) ChainAutoPath() (ChainAutoPath, bool) {
	if w.Mode != WireModeAutoChain {
		return ChainAutoPath{}, false
	}
	switch len(w.ChainGeometry) {
	case 0:
		return ChainAutoPath{Straight: true}, true
	case 4:
		// payload[0]: y-direction flag (0 = down, 1 = up)
		// payload[1]: always 0 in our corpus (reserved / unknown)
		// payload[2]: source-glyph elbow anchor (= 65 for I32 const)
		// payload[3]: y-step magnitude (unsigned pixels)
		var sign int
		switch w.ChainGeometry[0] {
		case 0:
			sign = 1
		case 1:
			sign = -1
		default:
			return ChainAutoPath{}, false
		}
		yMag := w.ChainGeometry[3]
		// A real BD elbow's y-step measures in tens or low
		// hundreds of pixels. Reject implausibly large values as
		// evidence we have not actually hit the L-shape pattern —
		// multi-elbow auto-chain wires can encode routing indices
		// rather than pixel steps in the same payload positions.
		// 4096 is generous: BD canvases rarely exceed that overall,
		// let alone a single elbow step.
		const maxReasonableYStep = 4096
		if yMag > maxReasonableYStep {
			return ChainAutoPath{}, false
		}
		// Source-glyph anchors are also bounded — the spike saw
		// `65` for I32 numeric constants. Reject implausibly large
		// anchors for the same reason.
		if w.ChainGeometry[2] > 1024 {
			return ChainAutoPath{}, false
		}
		return ChainAutoPath{
			YStep:         sign * int(yMag),
			SourceAnchorX: w.ChainGeometry[2],
		}, true
	}
	return ChainAutoPath{}, false
}

// TreeEndpoints returns the (V, H) endpoint coordinates for a pure
// Y-tree wire-network. Supports 2-branch (byte0=6) and 3-branch
// (byte0=7) shapes where the last byte0−4 records hold one (V, H)
// endpoint per branch, each encoded as two bytes in Mac-style Point
// (V, H) order.
//
// The rule "last N = byte0−4 endpoint records" was ground-truthed for
// 2-branch by the geometry-varied controlled fixture. For 3-branch it
// is now confirmed by Numeric42ThreeIndicatorsY_bottom8pxdown.vi,
// which moves exactly one endpoint record by 8 px; the independent
// corpus reference-find-by-id.vi chunk also matches the same shape.
//
// Returns nil, false for:
//   - non-tree modes
//   - comb topologies (byte0 ≥ 8), which have additional junction
//     records whose positions are topology-dependent and not yet
//     ground-truthed (Phase 13.5)
//   - truncated or malformed record slices
func (w Wire) TreeEndpoints() ([]Point, bool) {
	if w.Mode != WireModeTree {
		return nil, false
	}
	var n int
	switch w.Waypoints {
	case 6:
		n = 2
	case 7:
		n = 3
	default:
		return nil, false
	}
	// Pure Y-trees have exactly 3 header records before the N
	// endpoint records: the (byte0, byte1) header is consumed into
	// Waypoints/Mode, so TreeRecords[0..2] are the 3 structural
	// records and TreeRecords[len-n..] are the N endpoints.
	if len(w.TreeRecords) != n+3 {
		return nil, false
	}
	base := len(w.TreeRecords) - n
	out := make([]Point, n)
	for i := range n {
		r := w.TreeRecords[base+i]
		out[i] = Point{V: int16(r[0]), H: int16(r[1])}
	}
	return out, true
}

// TreeEndpointPair returns the two endpoint coordinates of a
// 2-branch Y-tree wire-network. It is a convenience wrapper over
// TreeEndpoints for the byte0=6 case; callers that handle N-branch
// trees should use TreeEndpoints directly.
//
// Returns ok=false for non-tree modes, 3-branch and comb topologies,
// or truncated record slices.
func (w Wire) TreeEndpointPair() (a, b Point, ok bool) {
	if w.Waypoints != 6 {
		return Point{}, Point{}, false
	}
	pts, ok := w.TreeEndpoints()
	if !ok {
		return Point{}, Point{}, false
	}
	return pts[0], pts[1], true
}

// WireMix counts wire networks broken down by mode. Useful for
// surfacing per-mode warnings in the scene graph.
type WireMix struct {
	AutoChain   int
	ManualChain int
	Tree        int
	Other       int
}

// Total returns the sum across all modes.
func (m WireMix) Total() int {
	return m.AutoChain + m.ManualChain + m.Tree + m.Other
}

// CountWireMix walks tree.Nodes and classifies every
// OF__compressedWireTable leaf by its mode. Empty leaves are
// skipped (consistent with CountCompressedWireTables).
func CountWireMix(tree HeapTree) WireMix {
	var mix WireMix
	for i, n := range tree.Nodes {
		if n.Scope != "leaf" {
			continue
		}
		if n.Tag != int32(heap.FieldTagCompressedWireTable) {
			continue
		}
		if len(n.Content) == 0 {
			continue
		}
		w, ok := HeapWire(tree, i)
		if !ok {
			continue
		}
		switch w.Mode {
		case WireModeAutoChain:
			mix.AutoChain++
		case WireModeManualChain:
			mix.ManualChain++
		case WireModeTree:
			mix.Tree++
		default:
			mix.Other++
		}
	}
	return mix
}
