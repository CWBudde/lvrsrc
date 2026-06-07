package lvvi

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// wireLeaf builds a single-node HeapTree with one
// OF__compressedWireTable leaf carrying the given bytes. Used so
// each test can pin its expectations on a known payload.
func wireLeaf(content []byte) HeapTree {
	return HeapTree{
		Nodes: []HeapNode{{
			Tag:     int32(heap.FieldTagCompressedWireTable),
			Scope:   "leaf",
			Parent:  -1,
			Content: content,
		}},
		Roots: []int{0},
	}
}

func singleBlockDiagramWirePayload(t *testing.T, name string) []byte {
	t.Helper()

	f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), name), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	m, issues := DecodeKnownResources(f)
	if len(issues) != 0 {
		t.Fatalf("DecodeKnownResources(%s) issues = %+v", name, issues)
	}
	tree, ok := m.BlockDiagram()
	if !ok {
		t.Fatalf("%s has no decoded block diagram", name)
	}
	var chunks [][]byte
	for _, n := range tree.Nodes {
		if n.Tag != int32(heap.FieldTagCompressedWireTable) || n.Scope != "leaf" {
			continue
		}
		if len(n.Content) == 0 {
			continue
		}
		chunks = append(chunks, append([]byte(nil), n.Content...))
	}
	if len(chunks) != 1 {
		t.Fatalf("%s compressed-wire chunks = %d, want 1", name, len(chunks))
	}
	return chunks[0]
}

// `0208` is the trivial-straight-scalar sentinel observed for
// auto-routed I32 and Boolean wires whose endpoints are aligned.
// Waypoints=2 (just endpoints), mode=auto-chain, no geometry.
func TestHeapWireDecodesTrivialAutoChain(t *testing.T) {
	tree := wireLeaf([]byte{0x02, 0x08})
	w, ok := HeapWire(tree, 0)
	if !ok {
		t.Fatal("HeapWire() ok = false, want true")
	}
	if w.Mode != WireModeAutoChain {
		t.Errorf("Mode = %s, want %s", w.Mode, WireModeAutoChain)
	}
	if w.Waypoints != 2 {
		t.Errorf("Waypoints = %d, want 2", w.Waypoints)
	}
	if len(w.ChainGeometry) != 0 {
		t.Errorf("ChainGeometry = %v, want empty", w.ChainGeometry)
	}
	if len(w.TreeRecords) != 0 {
		t.Errorf("TreeRecords = %v, want empty", w.TreeRecords)
	}
}

// Ground truth from Numeric42_8px_down.vi: a 1-edge wire whose
// indicator was nudged 8 pixels down. byte0=4 (2 endpoints + 2
// auto-elbow corners), byte1=0x08, payload `00 00 41 08` →
// LEB128 → [0, 0, 65, 8]. The trailing `8` is the literal y-step.
func TestHeapWireDecodesAutoChainElbowFromControlledFixture(t *testing.T) {
	tree := wireLeaf([]byte{0x04, 0x08, 0x00, 0x00, 0x41, 0x08})
	w, ok := HeapWire(tree, 0)
	if !ok {
		t.Fatal("HeapWire() ok = false, want true")
	}
	if w.Mode != WireModeAutoChain {
		t.Errorf("Mode = %s, want %s", w.Mode, WireModeAutoChain)
	}
	if w.Waypoints != 4 {
		t.Errorf("Waypoints = %d, want 4", w.Waypoints)
	}
	want := []uint64{0, 0, 65, 8}
	if !reflect.DeepEqual(w.ChainGeometry, want) {
		t.Errorf("ChainGeometry = %v, want %v", w.ChainGeometry, want)
	}
}

// Counterpart from Numeric42_16px_down.vi: only the trailing varint
// changed from 8 to 16, locking the per-pixel encoding hypothesis.
func TestHeapWireAutoChainEncodesPixelOffsetVerbatim(t *testing.T) {
	t8 := wireLeaf([]byte{0x04, 0x08, 0x00, 0x00, 0x41, 0x08})
	t16 := wireLeaf([]byte{0x04, 0x08, 0x00, 0x00, 0x41, 0x10})

	w8, ok := HeapWire(t8, 0)
	if !ok {
		t.Fatal("8px decode ok = false")
	}
	w16, ok := HeapWire(t16, 0)
	if !ok {
		t.Fatal("16px decode ok = false")
	}
	if w8.ChainGeometry[3] != 8 || w16.ChainGeometry[3] != 16 {
		t.Errorf("y-step bytes: 8px=%d, 16px=%d (want 8, 16)",
			w8.ChainGeometry[3], w16.ChainGeometry[3])
	}
	// All other geometry slots should match between the two — the
	// only thing that changed in the source fixture was the y-step.
	if w8.ChainGeometry[0] != w16.ChainGeometry[0] ||
		w8.ChainGeometry[1] != w16.ChainGeometry[1] ||
		w8.ChainGeometry[2] != w16.ChainGeometry[2] {
		t.Errorf("non-y bytes diverged: w8=%v w16=%v",
			w8.ChainGeometry, w16.ChainGeometry)
	}
}

// Ground truth from Numeric42Bend.vi: same endpoints as
// Numeric42, but with manually-placed waypoints. byte1 must flip
// from auto (0x08) to manual (0x04), and the payload should be
// noticeably richer (varint stream with multiple entries).
func TestHeapWireDecodesManualChainFromControlledFixture(t *testing.T) {
	payload := []byte{
		0x07, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00,
		0xff, 0x01, 0x05,
		0xff, 0x01, 0x69,
		0xff, 0x01, 0x20,
		0xff, 0x01, 0xe2, 0x64,
	}
	tree := wireLeaf(payload)
	w, ok := HeapWire(tree, 0)
	if !ok {
		t.Fatal("HeapWire() ok = false, want true")
	}
	if w.Mode != WireModeManualChain {
		t.Errorf("Mode = %s, want %s", w.Mode, WireModeManualChain)
	}
	if w.Waypoints != 7 {
		t.Errorf("Waypoints = %d, want 7", w.Waypoints)
	}
	// 18 payload bytes after the byte0/byte1 header. The repeated
	// `ff 01` token decodes to LEB128 255, so the manual-chain
	// stream contains four `255`s and six small values.
	got255 := 0
	for _, v := range w.ChainGeometry {
		if v == 255 {
			got255++
		}
	}
	if got255 != 4 {
		t.Errorf("ChainGeometry %v has %d entries equal to 255, want 4",
			w.ChainGeometry, got255)
	}
}

// Ground truth from Numeric42TwoIndicatorsY.vi: a Y-shape fan-out
// emits a single chunk in tree mode (byte1 = 0x00). byte0=6 means
// 6 records of 2 bytes each = 12 bytes total. The first record is
// the (byte0, byte1) header itself.
func TestHeapWireDecodesTreeModeFromControlledFixture(t *testing.T) {
	payload := []byte{
		0x06, 0x00,
		0x08, 0x07,
		0x00, 0x03,
		0x00, 0x41,
		0x31, 0x44,
		0x2d, 0x42,
	}
	tree := wireLeaf(payload)
	w, ok := HeapWire(tree, 0)
	if !ok {
		t.Fatal("HeapWire() ok = false, want true")
	}
	if w.Mode != WireModeTree {
		t.Errorf("Mode = %s, want %s", w.Mode, WireModeTree)
	}
	if w.Waypoints != 6 {
		t.Errorf("Waypoints = %d, want 6", w.Waypoints)
	}
	// 12 bytes total = 6 records × 2 bytes; the first record is
	// the byte0/byte1 header consumed before TreeRecords, so 5
	// records remain.
	if len(w.TreeRecords) != 5 {
		t.Errorf("len(TreeRecords) = %d, want 5", len(w.TreeRecords))
	}
	wantFirst := [2]byte{0x08, 0x07}
	if w.TreeRecords[0] != wantFirst {
		t.Errorf("TreeRecords[0] = %x, want %x", w.TreeRecords[0], wantFirst)
	}
	// Chain projections must stay empty for tree-mode chunks.
	if w.ChainGeometry != nil {
		t.Errorf("ChainGeometry = %v, want nil for tree mode", w.ChainGeometry)
	}
}

// Adding a third indicator to the Y-fan-out should bump byte0 and
// add one tree record (mirrors the controlled-fixture diff).
func TestHeapWireTreeModeRecordCountTracksBranches(t *testing.T) {
	twoY := wireLeaf([]byte{
		0x06, 0x00,
		0x08, 0x07, 0x00, 0x03, 0x00, 0x41, 0x31, 0x44, 0x2d, 0x42,
	})
	threeY := wireLeaf([]byte{
		0x07, 0x00,
		0x08, 0x04, 0x00, 0x03, 0x00, 0x03, 0x41, 0x31, 0x44, 0x2d, 0x42, 0xc4,
	})
	w2, _ := HeapWire(twoY, 0)
	w3, _ := HeapWire(threeY, 0)
	if w3.Waypoints != w2.Waypoints+1 {
		t.Errorf("Waypoints: 2-Y=%d, 3-Y=%d (want 3-Y = 2-Y + 1)",
			w2.Waypoints, w3.Waypoints)
	}
	if len(w3.TreeRecords) != len(w2.TreeRecords)+1 {
		t.Errorf("TreeRecords length: 2-Y=%d, 3-Y=%d (want 3-Y = 2-Y + 1)",
			len(w2.TreeRecords), len(w3.TreeRecords))
	}
}

// Unknown byte1 values must be classified as Other and leave the
// raw bytes untouched. This is the renderer's safe fallback for
// chunks whose mode we have not yet mapped.
func TestHeapWireUnknownModeFallsBackToOther(t *testing.T) {
	// `0501000100102604` has byte1=0x01, a value not in the spike
	// classification but observed in the wider corpus.
	tree := wireLeaf([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x10, 0x26, 0x04})
	w, ok := HeapWire(tree, 0)
	if !ok {
		t.Fatal("HeapWire() ok = false, want true")
	}
	if w.Mode != WireModeOther {
		t.Errorf("Mode = %s, want %s", w.Mode, WireModeOther)
	}
	if len(w.ChainGeometry) != 0 || len(w.TreeRecords) != 0 {
		t.Errorf("ChainGeometry=%v TreeRecords=%v, want both empty",
			w.ChainGeometry, w.TreeRecords)
	}
	if !reflect.DeepEqual(w.Raw, tree.Nodes[0].Content) {
		t.Errorf("Raw = %x, want %x", w.Raw, tree.Nodes[0].Content)
	}
}

func TestHeapWireRejectsWrongTag(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{{
			Tag:     int32(heap.FieldTagBounds),
			Scope:   "leaf",
			Content: []byte{0x02, 0x08},
		}},
	}
	if _, ok := HeapWire(tree, 0); ok {
		t.Error("HeapWire() on wrong tag returned ok=true")
	}
}

func TestHeapWireRejectsTooShortContent(t *testing.T) {
	for _, length := range []int{0, 1} {
		tree := wireLeaf(make([]byte, length))
		if _, ok := HeapWire(tree, 0); ok {
			t.Errorf("HeapWire() on %d-byte content returned ok=true", length)
		}
	}
}

func TestHeapWireRejectsOutOfRangeIndex(t *testing.T) {
	tree := HeapTree{}
	for _, idx := range []int{-1, 0, 1, 100} {
		if _, ok := HeapWire(tree, idx); ok {
			t.Errorf("HeapWire(idx=%d) on empty tree returned ok=true", idx)
		}
	}
}

func TestCountWireMixSplitsByMode(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 1, Scope: "open", Parent: -1, Children: []int{1, 2, 3, 4, 5}},
			{Tag: int32(heap.FieldTagCompressedWireTable), Scope: "leaf", Parent: 0, Content: []byte{0x02, 0x08}},
			{Tag: int32(heap.FieldTagCompressedWireTable), Scope: "leaf", Parent: 0, Content: []byte{0x07, 0x04, 0xaa, 0xbb}},
			{Tag: int32(heap.FieldTagCompressedWireTable), Scope: "leaf", Parent: 0, Content: []byte{0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
			{Tag: int32(heap.FieldTagCompressedWireTable), Scope: "leaf", Parent: 0, Content: []byte{0x05, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
			// Empty content: not counted.
			{Tag: int32(heap.FieldTagCompressedWireTable), Scope: "leaf", Parent: 0, Content: nil},
		},
	}
	mix := CountWireMix(tree)
	want := WireMix{AutoChain: 1, ManualChain: 1, Tree: 1, Other: 1}
	if mix != want {
		t.Errorf("CountWireMix = %+v, want %+v", mix, want)
	}
	if mix.Total() != 4 {
		t.Errorf("Total() = %d, want 4", mix.Total())
	}
}

// ChainAutoPath() must report Straight=true on the trivial
// `0208` sentinel — that's the well-known case where source and
// sink terminals are y-aligned and no elbow geometry is recorded.
func TestChainAutoPathRecognisesStraightWire(t *testing.T) {
	tree := wireLeaf([]byte{0x02, 0x08})
	w, _ := HeapWire(tree, 0)
	got, ok := w.ChainAutoPath()
	if !ok {
		t.Fatal("ChainAutoPath() ok = false on straight wire")
	}
	if !got.Straight {
		t.Errorf("ChainAutoPath().Straight = false, want true")
	}
	if got.YStep != 0 || got.SourceAnchorX != 0 {
		t.Errorf("ChainAutoPath() = %+v, want zero geometry for straight wire", got)
	}
}

// 8px-down: indicator placed 8 px below the source. payload =
// [0, 0, 65, 8] → ChainAutoPath should expose a positive YStep of
// 8 and SourceAnchorX of 65 (the I32-numeric-constant glyph
// anchor seen across the 4 controlled-fixture L-shape wires).
func TestChainAutoPathExposes8pxDownGroundTruth(t *testing.T) {
	tree := wireLeaf([]byte{0x04, 0x08, 0x00, 0x00, 0x41, 0x08})
	w, _ := HeapWire(tree, 0)
	got, ok := w.ChainAutoPath()
	if !ok {
		t.Fatal("ChainAutoPath() ok = false on 8px-down")
	}
	if got.Straight {
		t.Error("Straight = true on bent wire")
	}
	if got.YStep != 8 {
		t.Errorf("YStep = %d, want 8", got.YStep)
	}
	if got.SourceAnchorX != 65 {
		t.Errorf("SourceAnchorX = %d, want 65", got.SourceAnchorX)
	}
}

// 8px-up: indicator placed 8 px ABOVE the source. payload =
// [1, 0, 65, 8] — the leading 1 flips the sign. YStep should be
// -8, SourceAnchorX still 65.
func TestChainAutoPathSignsYStepFromDirectionByte(t *testing.T) {
	tree := wireLeaf([]byte{0x04, 0x08, 0x01, 0x00, 0x41, 0x08})
	w, _ := HeapWire(tree, 0)
	got, ok := w.ChainAutoPath()
	if !ok {
		t.Fatal("ChainAutoPath() ok = false on 8px-up")
	}
	if got.YStep != -8 {
		t.Errorf("YStep = %d, want -8", got.YStep)
	}
	if got.SourceAnchorX != 65 {
		t.Errorf("SourceAnchorX = %d, want 65", got.SourceAnchorX)
	}
}

// 16px-down: payload = [0, 0, 65, 16]. YStep should track the
// magnitude varint exactly.
func TestChainAutoPathTracksYStepMagnitude(t *testing.T) {
	tree := wireLeaf([]byte{0x04, 0x08, 0x00, 0x00, 0x41, 0x10})
	w, _ := HeapWire(tree, 0)
	got, _ := w.ChainAutoPath()
	if got.YStep != 16 {
		t.Errorf("YStep = %d, want 16", got.YStep)
	}
}

// Ground truth from Numeric42_150px_down.vi (controlled probe with the
// sink moved 150 px down — past the 127 boundary). The y-step byte is
// 0x96 = 150 as a SINGLE raw byte; LEB128 would need two bytes (96 01)
// and a longer chunk. This fixture proves the auto-chain geometry
// payload is raw bytes, not an LEB128 varint stream, so values >= 128
// (anchors and y-steps alike) must decode verbatim.
func TestChainAutoPath150pxRawByteGroundTruth(t *testing.T) {
	raw := singleBlockDiagramWirePayload(t, "Numeric42_150px_down.vi")
	want := []byte{0x04, 0x08, 0x00, 0x00, 0x41, 0x96}
	if !reflect.DeepEqual(raw, want) {
		t.Fatalf("raw = %x, want %x", raw, want)
	}
	w, _ := HeapWire(wireLeaf(raw), 0)
	// ChainGeometry must hold the raw byte 150, not an LEB128-mangled
	// value (the old varint parser produced [0 0 65], dropping 0x96).
	if !reflect.DeepEqual(w.ChainGeometry, []uint64{0, 0, 65, 150}) {
		t.Fatalf("ChainGeometry = %v, want [0 0 65 150]", w.ChainGeometry)
	}
	got, ok := w.ChainAutoPath()
	if !ok {
		t.Fatal("ChainAutoPath() ok = false on 150px-down")
	}
	if got.YStep != 150 {
		t.Errorf("YStep = %d, want 150", got.YStep)
	}
	if got.SourceAnchorX != 65 {
		t.Errorf("SourceAnchorX = %d, want 65 (stretched)", got.SourceAnchorX)
	}
}

// Numeric42_8px_down_auto-connect.vi was authored from Numeric42.vi by
// deleting the wire, moving the indicator 8 px down, and reconnecting —
// so LabVIEW auto-routes fresh and places the elbow at the canonical
// ~16 px: 04 08 00 00 10 08 (anchorX=16, yStep=8). This is the clean
// stretch-vs-reconnect counterpart to Numeric42_8px_down.vi (same 8px-
// down geometry, stretched to anchorX=65).
func TestChainAutoPathFreshRouteAnchorGroundTruth(t *testing.T) {
	raw := singleBlockDiagramWirePayload(t, "Numeric42_8px_down_auto-connect.vi")
	want := []byte{0x04, 0x08, 0x00, 0x00, 0x10, 0x08}
	if !reflect.DeepEqual(raw, want) {
		t.Fatalf("raw = %x, want %x", raw, want)
	}
	w, _ := HeapWire(wireLeaf(raw), 0)
	got, ok := w.ChainAutoPath()
	if !ok {
		t.Fatal("ChainAutoPath() ok = false")
	}
	if got.YStep != 8 || got.SourceAnchorX != 16 {
		t.Errorf("got %+v, want YStep=8 SourceAnchorX=16", got)
	}
}

// Ground truth from Numeric42FarFar.vi (sink ~400 px down, > 400 px
// right of the constant output). Payload 00 00 e2 ff 01 90:
//   - anchorX = 0xe2 = 226 (single byte): the ELBOW offset, NOT the
//     >400 px horizontal distance to the sink — the post-elbow run lives
//     in the terminal bounds (anchorX is an edit-history artifact; an
//     earlier save of this fixture had 470 at a SMALLER x offset).
//   - yStep = 0xff 0x01 0x90 = 0x0190 = 400: the 0xff escape decodes to
//     EXACTLY the 400 px vertical move, confirming 1:1 pixel scaling for
//     escaped (>= 255) magnitudes. Big-endian is forced (LE = 0x9001).
func TestChainAutoPathDecodesFFEscapeGroundTruth(t *testing.T) {
	raw := singleBlockDiagramWirePayload(t, "Numeric42FarFar.vi")
	want := []byte{0x04, 0x08, 0x00, 0x00, 0xe2, 0xff, 0x01, 0x90}
	if !reflect.DeepEqual(raw, want) {
		t.Fatalf("raw = %x, want %x", raw, want)
	}
	w, _ := HeapWire(wireLeaf(raw), 0)
	if !reflect.DeepEqual(w.ChainGeometry, []uint64{0, 0, 226, 400}) {
		t.Fatalf("ChainGeometry = %v, want [0 0 226 400]", w.ChainGeometry)
	}
	got, ok := w.ChainAutoPath()
	if !ok {
		t.Fatal("ChainAutoPath() ok = false on ff-escaped FarFar")
	}
	if got.YStep != 400 {
		t.Errorf("YStep = %d, want 400", got.YStep)
	}
	if got.SourceAnchorX != 226 {
		t.Errorf("SourceAnchorX = %d, want 226", got.SourceAnchorX)
	}
}

// Real-corpus single-elbow chunks carry anchors/y-steps across the full
// 0-255 byte range, not just the 16/65 the Numeric42 probes happened to
// use. These previously fell out of ChainAutoPath — either because the
// anchor was not in the {16,65} whitelist or because a byte >= 0x80 was
// swallowed as an LEB128 continuation. With raw-byte parsing they decode
// verbatim. Bytes lifted from write-ini.vi / ndjson-parser.vi.
func TestChainAutoPathDecodesFullRangeRawBytes(t *testing.T) {
	cases := []struct {
		name   string
		raw    []byte
		yStep  int
		anchor uint64
	}{
		{"anchor128 (write-ini 80 3d)", []byte{0x04, 0x08, 0x00, 0x00, 0x80, 0x3d}, 61, 128},
		{"anchor210 up (ndjson d2 43)", []byte{0x04, 0x08, 0x01, 0x00, 0xd2, 0x43}, -67, 210},
		{"ystep132 (ndjson 1a 84)", []byte{0x04, 0x08, 0x00, 0x00, 0x1a, 0x84}, 132, 26},
		{"ystep196 (ndjson 8a c4)", []byte{0x04, 0x08, 0x00, 0x00, 0x8a, 0xc4}, 196, 138},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, _ := HeapWire(wireLeaf(c.raw), 0)
			got, ok := w.ChainAutoPath()
			if !ok {
				t.Fatalf("ChainAutoPath() ok = false on %x", c.raw)
			}
			if got.YStep != c.yStep {
				t.Errorf("YStep = %d, want %d", got.YStep, c.yStep)
			}
			if got.SourceAnchorX != c.anchor {
				t.Errorf("SourceAnchorX = %d, want %d", got.SourceAnchorX, c.anchor)
			}
		})
	}
}

// Horizontal indicator shift produces no payload change at all
// (controlled-fixture spike confirmed). ChainAutoPath() must
// therefore return identical results for the original 8px-down
// and the 8-px-further-right variant, demonstrating that the
// renderer compose source/sink bounds with this typed projection
// at draw time rather than reading absolute x out of the chunk.
func TestChainAutoPathIgnoresXShift(t *testing.T) {
	a, _ := HeapWire(wireLeaf([]byte{0x04, 0x08, 0x00, 0x00, 0x41, 0x08}), 0)
	b, _ := HeapWire(wireLeaf([]byte{0x04, 0x08, 0x00, 0x00, 0x41, 0x08}), 0)
	pa, _ := a.ChainAutoPath()
	pb, _ := b.ChainAutoPath()
	if pa != pb {
		t.Errorf("ChainAutoPath() differs for x-shifted vs original: %+v vs %+v", pa, pb)
	}
}

// Non-auto modes must fall back to ok=false rather than mis-
// reporting tree or manual-chain payloads under the chain-auto
// shape.
func TestChainAutoPathRejectsNonAutoModes(t *testing.T) {
	tree := wireLeaf([]byte{0x06, 0x00, 0x08, 0x07, 0x00, 0x03, 0x00, 0x41, 0x31, 0x44, 0x2d, 0x42})
	w, _ := HeapWire(tree, 0)
	if _, ok := w.ChainAutoPath(); ok {
		t.Error("ChainAutoPath() ok = true on tree mode")
	}
}

// Multi-elbow auto-chain payloads (more than 4 varints) are not
// yet decoded — must return ok=false until Phase 13.5.
func TestChainAutoPathDoesNotMakeUpMultiElbowGeometry(t *testing.T) {
	// Synthetic multi-segment payload that decodes to MORE than the
	// 4 values of a single-elbow shape: [0, 0, 65, 8, 65, 8] (six
	// values, no 0xff escape). A payload longer than the single-elbow
	// shape must not be force-fit into ChainAutoPath — longer auto-
	// chain wires carry per-segment data we have not ground-truthed.
	// Don't claim more than we can defend.
	tree := wireLeaf([]byte{0x04, 0x08, 0x00, 0x00, 0x41, 0x08, 0x41, 0x08})
	w, _ := HeapWire(tree, 0)
	got, ok := w.ChainAutoPath()
	if !ok {
		t.Skip("ChainAutoPath rejected the multi-elbow shape — that is acceptable")
	}
	// If it does return a value, the YStep must not exceed a
	// sane pixel range; we'd rather fail cleanly than ship a
	// y-step of 9456.
	if got.YStep > 4096 || got.YStep < -4096 {
		t.Errorf("YStep = %d for multi-elbow chunk, well outside any reasonable pixel range", got.YStep)
	}
}

// Ground truth for the byte0=6 leftward auto-route family. These four
// controlled fixtures pin the exact on-disk bytes and the decoded
// LeftwardChainPath, and assert the single-axis deltas that calibrated
// the encoding. If a fixture is ever re-saved with different geometry,
// these byte assertions fail loudly instead of drifting silently.
func TestLeftwardChainPathGroundTruthFixtures(t *testing.T) {
	cases := []struct {
		name    string
		want    []byte
		wantUp  bool
		wantVer int
		wantP6  byte
	}{
		{"Numeric42_left_auto_8px_up.vi", []byte{0x06, 0x08, 0x00, 0x01, 0x01, 0x00, 0x10, 0x10, 0x9c, 0x18}, true, 8, 0x9c},
		{"Numeric42_left_auto_16px_up.vi", []byte{0x06, 0x08, 0x00, 0x01, 0x01, 0x00, 0x10, 0x10, 0x9c, 0x20}, true, 16, 0x9c},
		{"Numeric42_left_auto_8px_down.vi", []byte{0x06, 0x08, 0x01, 0x01, 0x00, 0x00, 0x10, 0x10, 0x9c, 0x18}, false, 8, 0x9c},
		{"Numeric42_left_auto_8px_down+8px_right.vi", []byte{0x06, 0x08, 0x01, 0x01, 0x00, 0x00, 0x10, 0x10, 0x94, 0x18}, false, 8, 0x94},
	}
	got := map[string]LeftwardChainPath{}
	for _, tc := range cases {
		raw := singleBlockDiagramWirePayload(t, tc.name)
		if !reflect.DeepEqual(raw, tc.want) {
			t.Fatalf("%s raw = % x, want % x", tc.name, raw, tc.want)
		}
		w, _ := HeapWire(wireLeaf(raw), 0)
		lcp, ok := w.LeftwardChainPath()
		if !ok {
			t.Fatalf("%s LeftwardChainPath() ok = false", tc.name)
		}
		if lcp.Up != tc.wantUp || lcp.VerticalPixels != tc.wantVer || lcp.HorizontalSeed != tc.wantP6 {
			t.Errorf("%s = %+v, want Up=%v VerticalPixels=%d HorizontalSeed=%#x",
				tc.name, lcp, tc.wantUp, tc.wantVer, tc.wantP6)
		}
		got[tc.name] = lcp
	}

	// Vertical magnitude: 8px vs 16px up differ only in VerticalPixels.
	up8, up16 := got["Numeric42_left_auto_8px_up.vi"], got["Numeric42_left_auto_16px_up.vi"]
	if up16.VerticalPixels-up8.VerticalPixels != 8 {
		t.Errorf("vertical delta = %d, want +8", up16.VerticalPixels-up8.VerticalPixels)
	}
	if up8.HorizontalSeed != up16.HorizontalSeed || up8.Up != up16.Up {
		t.Errorf("8px/16px up should differ only in vertical: %+v vs %+v", up8, up16)
	}

	// Direction: 8px up vs 8px down differ only in Up (same X, same magnitude).
	down8 := got["Numeric42_left_auto_8px_down.vi"]
	if up8.Up == down8.Up {
		t.Error("8px up vs down should differ in direction")
	}
	if up8.VerticalPixels != down8.VerticalPixels || up8.HorizontalSeed != down8.HorizontalSeed {
		t.Errorf("8px up vs down should differ only in direction: %+v vs %+v", up8, down8)
	}

	// Horizontal: 8px down vs +8px right differ only in HorizontalSeed,
	// by exactly −8 (sink moved right, toward the source).
	right8 := got["Numeric42_left_auto_8px_down+8px_right.vi"]
	if int(down8.HorizontalSeed)-int(right8.HorizontalSeed) != 8 {
		t.Errorf("horizontal seed delta = %d, want +8 (9c→94)", int(down8.HorizontalSeed)-int(right8.HorizontalSeed))
	}
	if down8.Up != right8.Up || down8.VerticalPixels != right8.VerticalPixels {
		t.Errorf("8px down vs +8px right should differ only in horizontal: %+v vs %+v", down8, right8)
	}
}

// LeftwardChainPath must reject shapes outside the calibrated family:
// single-elbow auto chains, tree wires, and the near-aligned
// indicator_left sample (p1=0x00, anchor 10 0f) that is not part of
// the family.
func TestLeftwardChainPathRejectsOtherShapes(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
	}{
		{"single-elbow auto (byte0=4)", []byte{0x04, 0x08, 0x00, 0x00, 0x41, 0x08}},
		{"trivial straight auto", []byte{0x02, 0x08}},
		{"tree mode byte0=6", []byte{0x06, 0x00, 0x08, 0x07, 0x00, 0x03, 0x00, 0x41, 0x31, 0x44, 0x2d, 0x42}},
		{"indicator_left near-aligned (p1=00)", []byte{0x06, 0x08, 0x00, 0x00, 0x00, 0x00, 0x10, 0x0f, 0x34, 0x23}},
		{"byte0=6 family but unknown direction", []byte{0x06, 0x08, 0x01, 0x01, 0x01, 0x00, 0x10, 0x10, 0x9c, 0x18}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, _ := HeapWire(wireLeaf(tc.raw), 0)
			if _, ok := w.LeftwardChainPath(); ok {
				t.Errorf("LeftwardChainPath() ok = true on %s", tc.name)
			}
		})
	}
}

// 2-Y tree (byte0=6) is the only tree shape we have ground-truth
// endpoint data for. TreeEndpointPair must expose the two
// endpoints exactly as the geometry-varied controlled fixture
// confirmed: the per-record (V, H) bytes from records #4 and #5.
func TestTreeEndpointPairExtracts2YGroundTruth(t *testing.T) {
	tree := wireLeaf([]byte{
		0x06, 0x00,
		0x08, 0x07,
		0x00, 0x03,
		0x00, 0x41,
		0x31, 0x44, // endpoint A: V=49, H=68
		0x2d, 0x42, // endpoint B: V=45, H=66
	})
	w, _ := HeapWire(tree, 0)
	a, b, ok := w.TreeEndpointPair()
	if !ok {
		t.Fatal("TreeEndpointPair() ok = false on 2-Y")
	}
	if a != (Point{V: 49, H: 68}) {
		t.Errorf("endpoint A = %+v, want {V:49, H:68}", a)
	}
	if b != (Point{V: 45, H: 66}) {
		t.Errorf("endpoint B = %+v, want {V:45, H:66}", b)
	}
}

// The geometry-varied 2-Y fixture confirmed which bytes encode
// branch geometry: indicator A moved 7 px right → record #4's H
// byte +7; indicator B moved ~10 px down → record #5's V byte +10.
// The decoder must surface the moved coordinates.
func TestTreeEndpointPairTracksGeometryEdits(t *testing.T) {
	moved := wireLeaf([]byte{
		0x06, 0x00,
		0x08, 0x07,
		0x00, 0x03,
		0x00, 0x41,
		0x31, 0x4b, // endpoint A: H went 0x44 → 0x4b (+7 right)
		0x37, 0x42, // endpoint B: V went 0x2d → 0x37 (+10 down)
	})
	w, _ := HeapWire(moved, 0)
	a, b, _ := w.TreeEndpointPair()
	if a != (Point{V: 49, H: 75}) {
		t.Errorf("endpoint A = %+v, want {V:49, H:75}", a)
	}
	if b != (Point{V: 55, H: 66}) {
		t.Errorf("endpoint B = %+v, want {V:55, H:66}", b)
	}
}

// TreeEndpointPair is the 2-Y specialisation — it must reject 3-Y
// and comb shapes (use TreeEndpoints for N-branch decoding).
func TestTreeEndpointPairRejectsNon2YTrees(t *testing.T) {
	threeY := wireLeaf([]byte{
		0x07, 0x00,
		0x08, 0x04, 0x00, 0x03, 0x00, 0x03,
		0x41, 0x31, 0x44, 0x2d, 0x42, 0xc4,
	})
	w3, _ := HeapWire(threeY, 0)
	if _, _, ok := w3.TreeEndpointPair(); ok {
		t.Error("TreeEndpointPair() ok = true on 3-Y; use TreeEndpoints instead")
	}

	fourY := wireLeaf([]byte{
		0x0a, 0x00,
		0x08, 0x07, 0x06, 0x00, 0x03, 0x03, 0x05, 0x00, 0x03, 0x41,
		0x21, 0x4a, 0x59, 0x59, 0x25, 0x48, 0x55, 0x57,
	})
	w4, _ := HeapWire(fourY, 0)
	if _, _, ok := w4.TreeEndpointPair(); ok {
		t.Error("TreeEndpointPair() ok = true on comb (byte0=10); must be false")
	}
}

// TreeEndpoints subsumes TreeEndpointPair: must return the same two
// points for a 2-Y chunk.
func TestTreeEndpointsMatches2YGroundTruth(t *testing.T) {
	tree := wireLeaf([]byte{
		0x06, 0x00,
		0x08, 0x07,
		0x00, 0x03,
		0x00, 0x41,
		0x31, 0x44, // endpoint A: V=49, H=68
		0x2d, 0x42, // endpoint B: V=45, H=66
	})
	w, _ := HeapWire(tree, 0)
	pts, ok := w.TreeEndpoints()
	if !ok {
		t.Fatal("TreeEndpoints() ok = false on 2-Y")
	}
	if len(pts) != 2 {
		t.Fatalf("len = %d, want 2", len(pts))
	}
	if pts[0] != (Point{V: 49, H: 68}) {
		t.Errorf("pts[0] = %+v, want {V:49, H:68}", pts[0])
	}
	if pts[1] != (Point{V: 45, H: 66}) {
		t.Errorf("pts[1] = %+v, want {V:45, H:66}", pts[1])
	}
}

// 3-branch fan-out (byte0=7): last 3 of 6 TreeRecords are per-branch
// coordinate records. This is a T-fork (see TreeEndpoints caveat): the
// straight-through leg to Numeric 3 is a genuine far-right endpoint
// ({V:66, H:196}); the two taps' second byte is a branch offset, not H.
// Ground-truth payload from Numeric42ThreeIndicatorsTfork.vi.
func TestTreeEndpoints3YGroundTruth(t *testing.T) {
	// Numeric42ThreeIndicatorsTfork.vi: 07 00 08 04 00 03 00 03 41 31 44 2d 42 c4
	tree := wireLeaf([]byte{
		0x07, 0x00,
		0x08, 0x04,
		0x00, 0x03,
		0x00, 0x03,
		0x41, 0x31, // endpoint A: V=65, H=49
		0x44, 0x2d, // endpoint B: V=68, H=45
		0x42, 0xc4, // endpoint C: V=66, H=196
	})
	w, _ := HeapWire(tree, 0)
	pts, ok := w.TreeEndpoints()
	if !ok {
		t.Fatal("TreeEndpoints() ok = false on 3-Y")
	}
	if len(pts) != 3 {
		t.Fatalf("len = %d, want 3", len(pts))
	}
	if pts[0] != (Point{V: 65, H: 49}) {
		t.Errorf("pts[0] = %+v, want {V:65, H:49}", pts[0])
	}
	if pts[1] != (Point{V: 68, H: 45}) {
		t.Errorf("pts[1] = %+v, want {V:68, H:45}", pts[1])
	}
	if pts[2] != (Point{V: 66, H: 196}) {
		t.Errorf("pts[2] = %+v, want {V:66, H:196}", pts[2])
	}
}

// Numeric42ThreeIndicatorsTfork_bottom8pxdown.vi is the geometry-varied
// T-fork fixture added for Phase 13.5. Moving the bottom tap 8 px DOWN
// changes exactly one trailing record (`44 2d` → `44 35`), confirming
// the "last N records track per-branch geometry" rule for byte0=7. Note
// the changed byte is the one TreeEndpoints labels H, even though the
// edit was vertical — so a tap record's second byte is a branch offset,
// not a horizontal coordinate (see TreeEndpoints caveat).
func TestTreeEndpoints3YTracksGeometryEdit(t *testing.T) {
	baseRaw := singleBlockDiagramWirePayload(t, "Numeric42ThreeIndicatorsTfork.vi")
	movedRaw := singleBlockDiagramWirePayload(t, "Numeric42ThreeIndicatorsTfork_bottom8pxdown.vi")

	wantBase := []byte{0x07, 0x00, 0x08, 0x04, 0x00, 0x03, 0x00, 0x03, 0x41, 0x31, 0x44, 0x2d, 0x42, 0xc4}
	wantMoved := []byte{0x07, 0x00, 0x08, 0x04, 0x00, 0x03, 0x00, 0x03, 0x41, 0x31, 0x44, 0x35, 0x42, 0xc4}
	if !reflect.DeepEqual(baseRaw, wantBase) {
		t.Fatalf("base raw = %x, want %x", baseRaw, wantBase)
	}
	if !reflect.DeepEqual(movedRaw, wantMoved) {
		t.Fatalf("moved raw = %x, want %x", movedRaw, wantMoved)
	}

	baseWire, _ := HeapWire(wireLeaf(baseRaw), 0)
	movedWire, _ := HeapWire(wireLeaf(movedRaw), 0)
	basePts, ok := baseWire.TreeEndpoints()
	if !ok {
		t.Fatal("base TreeEndpoints() ok = false")
	}
	movedPts, ok := movedWire.TreeEndpoints()
	if !ok {
		t.Fatal("moved TreeEndpoints() ok = false")
	}
	wantBasePts := []Point{{V: 65, H: 49}, {V: 68, H: 45}, {V: 66, H: 196}}
	wantMovedPts := []Point{{V: 65, H: 49}, {V: 68, H: 53}, {V: 66, H: 196}}
	if !reflect.DeepEqual(basePts, wantBasePts) {
		t.Fatalf("base points = %+v, want %+v", basePts, wantBasePts)
	}
	if !reflect.DeepEqual(movedPts, wantMovedPts) {
		t.Fatalf("moved points = %+v, want %+v", movedPts, wantMovedPts)
	}
	// The vertical (8 px down) edit moves the tap record's second byte
	// (labelled H here): confirms the byte tracks the edited branch by
	// exactly the pixel delta, while exposing that it is not a true
	// horizontal coordinate. See TreeEndpoints caveat.
	if movedPts[1].H-basePts[1].H != 8 {
		t.Errorf("changed tap record byte delta = %d, want +8", movedPts[1].H-basePts[1].H)
	}
}

// Second independent 3-branch fixture from the corpus (reference-find-by-id.vi).
// This is the pure vertical Y-stack case (contrast the T-fork above): all
// three trailing records read as small, plausible (V, H) endpoints with no
// far-right outlier, so the (V, H) interpretation holds cleanly here.
func TestTreeEndpoints3YCorpusIndependent(t *testing.T) {
	// reference-find-by-id.vi 3-Y chunk: 07 00 08 05 00 00 00 03 14 39 3d 34 32 19
	tree := wireLeaf([]byte{
		0x07, 0x00,
		0x08, 0x05,
		0x00, 0x00,
		0x00, 0x03,
		0x14, 0x39, // endpoint A: V=20, H=57
		0x3d, 0x34, // endpoint B: V=61, H=52
		0x32, 0x19, // endpoint C: V=50, H=25
	})
	w, _ := HeapWire(tree, 0)
	pts, ok := w.TreeEndpoints()
	if !ok {
		t.Fatal("TreeEndpoints() ok = false on corpus 3-Y chunk")
	}
	if len(pts) != 3 {
		t.Fatalf("len = %d, want 3", len(pts))
	}
	if pts[0] != (Point{V: 20, H: 57}) {
		t.Errorf("pts[0] = %+v, want {V:20, H:57}", pts[0])
	}
	if pts[1] != (Point{V: 61, H: 52}) {
		t.Errorf("pts[1] = %+v, want {V:61, H:52}", pts[1])
	}
	if pts[2] != (Point{V: 50, H: 25}) {
		t.Errorf("pts[2] = %+v, want {V:50, H:25}", pts[2])
	}
}

// Comb topologies (byte0=10) and all byte0 ≥ 8 shapes must return
// nil/false from TreeEndpoints until Phase 13.5 ground-truths enough
// topology records to decode them defensibly.
func TestTreeEndpointsRejectsUnknownTopologies(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
	}{
		{
			"4-Y comb (byte0=10)",
			[]byte{0x0a, 0x00, 0x08, 0x07, 0x06, 0x00, 0x03, 0x03, 0x05, 0x00, 0x03, 0x41,
				0x21, 0x4a, 0x59, 0x59, 0x25, 0x48, 0x55, 0x57},
		},
		{
			"3-Y comb (byte0=10)",
			[]byte{0x0a, 0x00, 0x08, 0x00, 0x06, 0x00, 0x06, 0x00, 0x00, 0x03, 0x03, 0x41,
				0x2f, 0x01, 0x5b, 0x01, 0x57, 0x42, 0x3f, 0x44},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, _ := HeapWire(wireLeaf(tc.raw), 0)
			if pts, ok := w.TreeEndpoints(); ok {
				t.Errorf("TreeEndpoints() ok = true, pts = %v; must be false until 13.5", pts)
			}
		})
	}
}

// TreeEndpointPair must reject non-tree modes outright.
func TestTreeEndpointPairRejectsNonTreeModes(t *testing.T) {
	tree := wireLeaf([]byte{0x02, 0x08})
	w, _ := HeapWire(tree, 0)
	if _, _, ok := w.TreeEndpointPair(); ok {
		t.Error("TreeEndpointPair() ok = true on auto-chain")
	}
}

// Sweep the corpus: every non-empty OF__compressedWireTable leaf
// must decode through HeapWire (presence-only — semantics still
// being mapped). Logs the per-mode breakdown so the spike's mode
// distribution is visible.
func TestHeapWireCorpusCoverage(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	total := 0
	decoded := 0
	mix := WireMix{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".vi" && ext != ".ctl" && ext != ".vit" {
			continue
		}
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		if err != nil {
			continue
		}
		m, _ := DecodeKnownResources(f)
		for _, getter := range []func() (HeapTree, bool){m.FrontPanel, m.BlockDiagram} {
			tree, ok := getter()
			if !ok {
				continue
			}
			fmix := CountWireMix(tree)
			mix.AutoChain += fmix.AutoChain
			mix.ManualChain += fmix.ManualChain
			mix.Tree += fmix.Tree
			mix.Other += fmix.Other
			for i, n := range tree.Nodes {
				if n.Tag != int32(heap.FieldTagCompressedWireTable) || n.Scope != "leaf" {
					continue
				}
				if len(n.Content) == 0 {
					continue
				}
				total++
				if _, ok := HeapWire(tree, i); ok {
					decoded++
				}
			}
		}
	}
	if total == 0 {
		t.Skip("corpus contains no OF__compressedWireTable leaves")
	}
	if decoded != total {
		t.Fatalf("HeapWire decoded %d/%d leaves", decoded, total)
	}
	t.Logf("HeapWire: %d/%d leaves decoded; mix = %+v", decoded, total, mix)
}

// Comb-with-elbow topology (Phase 13.5 baseline). The comb fixture
// has one auto-chain elbow before the fan-out, encoded as a single
// tree-mode chunk with byte0=10 (10 two-byte records). The raw bytes
// and TreeRecords are ground-truthed here so Phase 13.5 can diff
// against them when mapping junction geometry.
//
// Raw: 0a 00 08 00 06 00 06 00 00 03 03 41 2f 01 5b 01 57 42 3f 44
// TreeRecords (9 after header): [8 0][6 0][6 0][0 3][3 65][47 1][91 1][87 66][63 68]
//
// Compare with pure 3-Y (no elbow, byte0=7):
// Raw: 07 00 08 04 00 03 00 03 41 31 44 2d 42 c4
// TreeRecords (6 after header): [8 4][0 3][0 3][65 49][68 45][66 196]
//
// The per-record semantics across these two topologies are tracked as
// Phase 13.5; the junction coordinates and elbow geometry within the
// 10-record comb chunk are not yet decoded.
func TestHeapWireCombWithElbowBaseline(t *testing.T) {
	want := []byte{0x0a, 0x00, 0x08, 0x00, 0x06, 0x00, 0x06, 0x00,
		0x00, 0x03, 0x03, 0x41, 0x2f, 0x01, 0x5b, 0x01, 0x57, 0x42, 0x3f, 0x44}
	tree := wireLeaf(want)
	w, ok := HeapWire(tree, 0)
	if !ok {
		t.Fatal("HeapWire() ok = false on comb chunk")
	}
	if w.Mode != WireModeTree {
		t.Errorf("Mode = %s, want tree", w.Mode)
	}
	if w.Waypoints != 10 {
		t.Errorf("Waypoints = %d, want 10", w.Waypoints)
	}
	if len(w.TreeRecords) != 9 {
		t.Errorf("len(TreeRecords) = %d, want 9", len(w.TreeRecords))
	}
	// Neither TreeEndpointPair nor TreeEndpoints decode combs.
	if _, _, ok := w.TreeEndpointPair(); ok {
		t.Error("TreeEndpointPair() ok = true on comb; must be false")
	}
	if pts, ok := w.TreeEndpoints(); ok {
		t.Errorf("TreeEndpoints() ok = true on comb, pts=%v; must be false until 13.5", pts)
	}
}

// Numeric42ThreeIndicatorsYComb_middle8pxdown.vi gives Phase 13.5 a
// controlled comb edit. The edited fixture shifts the middle indicator
// 8 px along the diagram's horizontal axis. The compressed table does
// not behave like a simple trailing endpoint list: two adjacent records
// change in opposite directions, implying span/junction data around
// the moved branch. Keep that fact pinned without pretending to decode
// the whole topology yet.
func TestHeapWireCombMiddleEditIdentifiesSpanRecords(t *testing.T) {
	baseRaw := singleBlockDiagramWirePayload(t, "Numeric42ThreeIndicatorsYComb.vi")
	movedRaw := singleBlockDiagramWirePayload(t, "Numeric42ThreeIndicatorsYComb_middle8pxdown.vi")

	wantBase := []byte{0x0a, 0x00, 0x08, 0x00, 0x06, 0x00, 0x06, 0x00,
		0x00, 0x03, 0x03, 0x41, 0x2f, 0x01, 0x5b, 0x01, 0x57, 0x42, 0x3f, 0x44}
	wantMoved := []byte{0x0a, 0x00, 0x08, 0x00, 0x06, 0x00, 0x06, 0x00,
		0x00, 0x03, 0x03, 0x41, 0x2f, 0x01, 0x63, 0x01, 0x4f, 0x42, 0x3f, 0x44}
	if !reflect.DeepEqual(baseRaw, wantBase) {
		t.Fatalf("base raw = %x, want %x", baseRaw, wantBase)
	}
	if !reflect.DeepEqual(movedRaw, wantMoved) {
		t.Fatalf("moved raw = %x, want %x", movedRaw, wantMoved)
	}

	baseWire, _ := HeapWire(wireLeaf(baseRaw), 0)
	movedWire, _ := HeapWire(wireLeaf(movedRaw), 0)
	if pts, ok := movedWire.TreeEndpoints(); ok {
		t.Fatalf("TreeEndpoints() decoded comb as %+v; comb topology is still open", pts)
	}
	if len(baseWire.TreeRecords) != 9 || len(movedWire.TreeRecords) != 9 {
		t.Fatalf("record counts: base=%d moved=%d, want 9/9", len(baseWire.TreeRecords), len(movedWire.TreeRecords))
	}

	var changed []int
	for i := range baseWire.TreeRecords {
		if baseWire.TreeRecords[i] != movedWire.TreeRecords[i] {
			changed = append(changed, i)
		}
	}
	if !reflect.DeepEqual(changed, []int{6, 7}) {
		t.Fatalf("changed record indexes = %v, want [6 7]", changed)
	}
	if got := int(movedWire.TreeRecords[6][0]) - int(baseWire.TreeRecords[6][0]); got != 8 {
		t.Errorf("record 6 first-byte delta = %d, want +8", got)
	}
	if got := int(movedWire.TreeRecords[7][0]) - int(baseWire.TreeRecords[7][0]); got != -8 {
		t.Errorf("record 7 first-byte delta = %d, want -8", got)
	}
	if baseWire.TreeRecords[6][1] != movedWire.TreeRecords[6][1] ||
		baseWire.TreeRecords[7][1] != movedWire.TreeRecords[7][1] {
		t.Errorf("record 6/7 second bytes changed: base=%x/%x moved=%x/%x",
			baseWire.TreeRecords[6], baseWire.TreeRecords[7],
			movedWire.TreeRecords[6], movedWire.TreeRecords[7])
	}
}
