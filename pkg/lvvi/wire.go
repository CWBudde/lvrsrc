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
	// WireModeManualChain from the payload after byte0 / byte1.
	// For auto-chain it holds the raw payload bytes (each widened to
	// uint64) — the geometry is NOT LEB128, see HeapWire and
	// Numeric42_150px_down.vi. For manual-chain it is still the
	// LEB128-decoded varint stream (unverified, pending a controlled
	// fixture). Empty for chains with no trailing geometry (e.g.
	// `0208`).
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
	case WireModeAutoChain:
		// Auto-chain geometry is a sequence of raw single bytes, NOT
		// an LEB128 varint stream. Numeric42_150px_down.vi proves it:
		// a 150 px y-step is the single byte 0x96, where LEB128 would
		// require two bytes (96 01). Reading raw keeps values >= 128
		// intact instead of swallowing them as varint continuations.
		w.ChainGeometry = rawGeometryBytes(payload)
	case WireModeManualChain:
		// Manual-chain payloads are still parsed as LEB128 pending a
		// controlled ground-truth fixture (the corpus has only the
		// single Numeric42Bend.vi sample, whose ff-tokens are
		// ambiguous between raw 0xff and an LEB128 255 sentinel).
		w.ChainGeometry = decodeLEB128(payload)
	case WireModeTree:
		w.TreeRecords = splitTreeRecords(payload)
	}
	return w, true
}

// rawGeometryBytes widens each payload byte to a uint64 verbatim. For
// values < 128 this is identical to decodeLEB128; for values >= 128 it
// preserves the byte instead of treating the high bit as an LEB128
// continuation (see HeapWire / Numeric42_150px_down.vi).
func rawGeometryBytes(payload []byte) []uint64 {
	if len(payload) == 0 {
		return nil
	}
	out := make([]uint64, len(payload))
	for i, b := range payload {
		out[i] = uint64(b)
	}
	return out
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
	// source-glyph's output anchor, in pixels. It is NOT a glyph
	// constant — it is edit-history-dependent:
	//   - A freshly auto-routed (delete + reconnect) wire places the
	//     elbow at the canonical short distance for the current
	//     layout (~16 px for the I32-constant→numeric pair).
	//   - Moving a terminal afterwards freezes the existing elbow and
	//     only stretches the post-elbow segment, so the value stays at
	//     whatever it was when the wire was last routed (e.g. 65 in
	//     several corpus fixtures that were drawn then nudged).
	// Two VIs with identical terminal positions can therefore carry
	// different SourceAnchorX values. YStep, by contrast, reflects the
	// real vertical offset regardless of edit method. Zero when
	// Straight is true.
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
		// Raw-byte payload [signV, reserved, anchorX, yStep]:
		// payload[0]: y-direction flag (0 = down, 1 = up)
		// payload[1]: always 0 in our corpus (reserved / unknown)
		// payload[2]: elbow horizontal offset, a raw byte 0-255
		//             (edit-history-dependent: ~16 fresh, 65 stretched,
		//             but real-world wires span the whole range)
		// payload[3]: y-step magnitude, a raw byte 0-255
		// Because each field is a single raw byte (see HeapWire and
		// Numeric42_150px_down.vi), anchorX and yStep are inherently
		// bounded to 0-255 — no separate magnitude whitelist is needed
		// or wanted; the old {16,65} anchor bound was over-fitted to
		// the Numeric42 probes and rejected most real single-elbow
		// wires. Payloads longer than 4 bytes are multi-segment and
		// fall through to ok=false.
		var sign int
		switch w.ChainGeometry[0] {
		case 0:
			sign = 1
		case 1:
			sign = -1
		default:
			return ChainAutoPath{}, false
		}
		return ChainAutoPath{
			YStep:         sign * int(w.ChainGeometry[3]),
			SourceAnchorX: w.ChainGeometry[2],
		}, true
	}
	return ChainAutoPath{}, false
}

// LeftwardChainPath is the typed projection of a byte0=6 leftward
// auto-routed wire — the doubling-back route LabVIEW emits when the
// sink terminal sits to the left of the source, forcing the wire to
// exit right, loop around, and come back. ChainAutoPath rejects these
// (>4 varints); this accessor decodes the family that four controlled
// fixtures calibrated:
//
//	left_auto_8px_up            06 08 00 01 01 00 10 10 9c 18
//	left_auto_16px_up           06 08 00 01 01 00 10 10 9c 20
//	left_auto_8px_down          06 08 01 01 00 00 10 10 9c 18
//	left_auto_8px_down+8px_right 06 08 01 01 00 00 10 10 94 18
//
// Payload byte map (payload = Raw[2:], indices p0..p7):
//
//	p0,p2 — vertical direction: up=(p0=0,p2=1), down=(p0=1,p2=0)
//	p1    — always 0x01 in this family
//	p3    — always 0x00
//	p4,p5 — glyph anchor seed = 0x10 0x10 (I32-constant specific)
//	p6    — horizontal seed: 1 unit per pixel, decreasing as the sink
//	        moves right (toward the source). Absolute zero is not yet
//	        calibrated, so it is exposed raw.
//	p7    — vertical magnitude = 16 + pixels (1 unit per pixel),
//	        direction-independent (8px→0x18, 16px→0x20).
//
// Both axes are confirmed linear at 1 unit/pixel by the controlled
// pairs; the constant "+16" base on p7 and the absolute base of p6 are
// fixed routing margins that terminal-bounds correlation can resolve
// later. The anchor seed is glyph-specific, so the decoder only
// recognises the exact prefix it has ground truth for.
type LeftwardChainPath struct {
	// Up reports the vertical direction: true when the sink is above
	// the source, false when below.
	Up bool
	// VerticalPixels is the vertical offset magnitude in pixels
	// (p7 − 16). Always non-negative.
	VerticalPixels int
	// HorizontalSeed is the raw p6 byte. It moves 1 unit per pixel and
	// decreases as the sink approaches the source horizontally; its
	// absolute pixel zero is not yet calibrated.
	HorizontalSeed byte
}

// LeftwardChainPath returns the typed projection for a byte0=6
// leftward auto-route when the payload matches the calibrated family
// shape (auto mode, Waypoints==6, prefix `?? 01 ?? 00 10 10`, and a
// recognised direction pair in p0/p2). Returns ok=false for every
// other mode/shape — including single-elbow auto chains, tree wires,
// and the near-aligned Numeric42_indicator_left.vi sample (which has
// p1==0x00 and a different anchor seed).
func (w Wire) LeftwardChainPath() (LeftwardChainPath, bool) {
	if w.Mode != WireModeAutoChain || w.Waypoints != 6 {
		return LeftwardChainPath{}, false
	}
	if len(w.Raw) < 2 {
		return LeftwardChainPath{}, false
	}
	p := w.Raw[2:]
	if len(p) != 8 {
		return LeftwardChainPath{}, false
	}
	// Stable prefix shared by every ground-truth fixture in the family.
	if p[1] != 0x01 || p[3] != 0x00 || p[4] != 0x10 || p[5] != 0x10 {
		return LeftwardChainPath{}, false
	}
	var up bool
	switch {
	case p[0] == 0x00 && p[2] == 0x01:
		up = true
	case p[0] == 0x01 && p[2] == 0x00:
		up = false
	default:
		return LeftwardChainPath{}, false
	}
	// p7 = 16 + vertical pixels; a value below the base is evidence we
	// have not actually matched the family, so reject it.
	if p[7] < 16 {
		return LeftwardChainPath{}, false
	}
	return LeftwardChainPath{
		Up:             up,
		VerticalPixels: int(p[7]) - 16,
		HorizontalSeed: p[6],
	}, true
}

// TreeEndpoints returns the trailing per-branch coordinate records of a
// fan-out wire-network. Supports 2-branch (byte0=6) and 3-branch
// (byte0=7) shapes where the last byte0−4 records hold two bytes per
// branch, returned as a Mac-style Point (V, H).
//
// CAVEAT — the (V, H) interpretation is only fully ground-truthed for
// the straight-through main-line endpoint. Two fan-out topologies share
// byte0=7 and cannot be told apart from the header alone:
//
//   - Pure vertical Y-stack (all indicators near the same X): every
//     trailing record reads cleanly as a small (V, H) endpoint.
//     reference-find-by-id.vi is an independent corpus example.
//   - T-fork (one indicator far to the right, others tapping up/down):
//     the straight-through leg IS a genuine (V, H) endpoint — e.g.
//     Numeric42ThreeIndicatorsTfork.vi decodes Numeric 3 as {V:66,
//     H:196}, correctly far-right and mid-height — but the two tap
//     records are NOT plain endpoints: moving the bottom tap 8 px DOWN
//     (Numeric42ThreeIndicatorsTfork_bottom8pxdown.vi) changes the byte
//     this function labels H, not V, so that second tap byte tracks a
//     branch length/offset rather than a horizontal coordinate.
//
// The "last N = byte0−4 records" framing is ground-truthed (one record
// moves by exactly the edited pixel delta); the per-axis meaning of a
// tap record's two bytes is NOT, pending a controlled horizontal-move
// fixture. Callers should treat tap V/H as raw record bytes, not
// trusted screen coordinates.
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
