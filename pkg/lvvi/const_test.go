package lvvi

import (
	"encoding/binary"
	"math"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// findConstValue locates the single OF__ConstValue leaf in a fixture's
// block-diagram heap and decodes it.
func findConstValue(t *testing.T, fixture string) (ConstValue, []byte) {
	t.Helper()
	f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), fixture), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open(%s): %v", fixture, err)
	}
	m, _ := DecodeKnownResources(f)
	tree, ok := m.BlockDiagram()
	if !ok {
		t.Fatalf("%s: no block-diagram heap", fixture)
	}
	for i, n := range tree.Nodes {
		if n.Tag != int32(heap.FieldTagConstValue) || n.Scope != "leaf" {
			continue
		}
		cv, ok := HeapConstValue(tree, i)
		if !ok {
			t.Fatalf("%s: HeapConstValue ok=false", fixture)
		}
		return cv, append([]byte(nil), n.Content...)
	}
	t.Fatalf("%s: no OF__ConstValue leaf found", fixture)
	return ConstValue{}, nil
}

// rawConstLeaf returns the raw OF__ConstValue leaf bytes regardless of
// width (used for EXT, which exceeds HeapConstValue's 8-byte band). The
// bool reports whether HeapConstValue also accepts the leaf.
func rawConstLeaf(t *testing.T, fixture string) ([]byte, bool) {
	t.Helper()
	f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), fixture), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open(%s): %v", fixture, err)
	}
	m, _ := DecodeKnownResources(f)
	tree, ok := m.BlockDiagram()
	if !ok {
		t.Fatalf("%s: no block-diagram heap", fixture)
	}
	for i, n := range tree.Nodes {
		if n.Tag != int32(heap.FieldTagConstValue) || n.Scope != "leaf" {
			continue
		}
		_, accepted := HeapConstValue(tree, i)
		return append([]byte(nil), n.Content...), accepted
	}
	t.Fatalf("%s: no OF__ConstValue leaf found", fixture)
	return nil, false
}

// TestHeapConstValueGroundTruth pins the block-diagram integer-constant
// literal encoding. The OF__ConstValue leaf (tag 589) is a fixed-width
// big-endian integer written verbatim — NOT the byte-stream-with-0xff-
// escape used by the compressed-wire geometry — and its width tracks the
// constant's data type, not its magnitude.
//
// The I32 fixtures straddle both single-byte boundaries (255/256 cross
// the 1-byte limit, 65535/65536 the 2-byte limit) with no width change
// and no 0xff prefix — only positional big-endian. The I8/U8 pair shows
// signedness is carried by the type, not the bytes (both store 0x2a),
// and I64 confirms the 8-byte integer width.
func TestHeapConstValueGroundTruth(t *testing.T) {
	cases := []struct {
		fixture string
		width   int
		want    int64
		raw     []byte
	}{
		{"Numeric42.vi", 4, 42, []byte{0x00, 0x00, 0x00, 0x2a}},
		{"Numeric255.vi", 4, 255, []byte{0x00, 0x00, 0x00, 0xff}},
		{"Numeric256.vi", 4, 256, []byte{0x00, 0x00, 0x01, 0x00}},
		{"Numeric65535.vi", 4, 65535, []byte{0x00, 0x00, 0xff, 0xff}},
		{"Numeric65536.vi", 4, 65536, []byte{0x00, 0x01, 0x00, 0x00}},
		{"Numeric42_I8.vi", 1, 42, []byte{0x2a}},
		{"Numeric42_U8.vi", 1, 42, []byte{0x2a}},
		{"Numeric65536_I64.vi", 8, 65536, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00}},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			sv, raw := findConstValue(t, tc.fixture)
			if sv.Width != tc.width {
				t.Errorf("Width = %d, want %d", sv.Width, tc.width)
			}
			if sv.Signed != tc.want {
				t.Errorf("Signed = %d, want %d", sv.Signed, tc.want)
			}
			if sv.Unsigned != uint64(tc.want) {
				t.Errorf("Unsigned = %d, want %d", sv.Unsigned, tc.want)
			}
			if len(raw) != len(tc.raw) {
				t.Fatalf("raw length = %d, want %d (% x)", len(raw), len(tc.raw), raw)
			}
			for i := range tc.raw {
				if raw[i] != tc.raw[i] {
					t.Fatalf("raw = % x, want % x (no varint/escape across the byte boundary)", raw, tc.raw)
				}
			}
		})
	}
}

// TestHeapConstValueFloatGroundTruth pins the IEEE-754 floating-point
// constant encodings. A SGL constant's OF__ConstValue is a 4-byte
// big-endian binary32; a DBL constant's is an 8-byte big-endian binary64
// — both written verbatim. SGL stores the float32 rounding of the typed
// value (9876.5432 -> 9876.54297), confirming the literal width tracks
// the constant's data type.
func TestHeapConstValueFloatGroundTruth(t *testing.T) {
	t.Run("SGL", func(t *testing.T) {
		sv, raw := findConstValue(t, "Numeric9876Dot5432_SGL.vi")
		wantRaw := []byte{0x46, 0x1a, 0x52, 0x2c}
		if sv.Width != 4 {
			t.Errorf("Width = %d, want 4 (SGL constant)", sv.Width)
		}
		if len(raw) != 4 {
			t.Fatalf("raw length = %d, want 4 (% x)", len(raw), raw)
		}
		for i := range wantRaw {
			if raw[i] != wantRaw[i] {
				t.Fatalf("raw = % x, want % x", raw, wantRaw)
			}
		}
		got := math.Float32frombits(uint32(sv.Unsigned))
		if want := float32(9876.5432); got != want {
			t.Errorf("decoded SGL = %v, want %v", got, want)
		}
	})

	t.Run("DBL", func(t *testing.T) {
		sv, raw := findConstValue(t, "Numeric9876Dot5432.vi")
		want := 9876.5432
		wantRaw := []byte{0x40, 0xc3, 0x4a, 0x45, 0x87, 0x93, 0xdd, 0x98}
		if sv.Width != 8 {
			t.Errorf("Width = %d, want 8 (DBL constant)", sv.Width)
		}
		if len(raw) != 8 {
			t.Fatalf("raw length = %d, want 8 (% x)", len(raw), raw)
		}
		for i := range wantRaw {
			if raw[i] != wantRaw[i] {
				t.Fatalf("raw = % x, want % x", raw, wantRaw)
			}
		}
		if got := math.Float64frombits(sv.Unsigned); got != want {
			t.Errorf("decoded DBL = %v, want %v", got, want)
		}
		if got := math.Float64frombits(binary.BigEndian.Uint64(raw)); got != want {
			t.Errorf("raw big-endian DBL = %v, want %v", got, want)
		}
	})
}

// TestHeapConstValueExtendedAndFixedPoint pins the two wide
// representations. EXT is a 16-byte big-endian IEEE-754 binary128 (quad)
// — wider than HeapConstValue's 8-byte numeric band, so the accessor
// reports ok=false and the literal is asserted from the raw leaf. FXD is
// an 8-byte big-endian fixed-point container; here 9876.5432 is stored
// as the magnitude 0x26948b = 2528395 = round(9876.5432 * 2^8).
func TestHeapConstValueExtendedAndFixedPoint(t *testing.T) {
	t.Run("EXT", func(t *testing.T) {
		raw, ok := rawConstLeaf(t, "Numeric9876Dot5432_EXT.vi")
		if len(raw) != 16 {
			t.Fatalf("EXT OF__ConstValue length = %d, want 16 (% x)", len(raw), raw)
		}
		wantRaw := []byte{0x40, 0x0c, 0x34, 0xa4, 0x58, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		for i := range wantRaw {
			if raw[i] != wantRaw[i] {
				t.Fatalf("EXT raw = % x, want % x", raw, wantRaw)
			}
		}
		// binary128: sign(1) exponent(15, bias 16383) fraction(112).
		hi := binary.BigEndian.Uint64(raw[:8])
		exp := int((hi>>48)&0x7fff) - 16383
		if exp != 13 { // 9876.5432 ∈ [2^13, 2^14)
			t.Errorf("EXT binary128 unbiased exponent = %d, want 13", exp)
		}
		// HeapConstValue declines >8-byte payloads (EXT vs string is
		// indistinguishable at the leaf without the VCTP type).
		if ok {
			t.Error("HeapConstValue should decline the 16-byte EXT leaf")
		}
	})

	t.Run("FXD", func(t *testing.T) {
		sv, raw := findConstValue(t, "Numeric9876Dot5432_FXD.vi")
		wantRaw := []byte{0x00, 0x00, 0x26, 0x94, 0x8b, 0x00, 0x00, 0x00}
		if sv.Width != 8 {
			t.Errorf("FXD Width = %d, want 8", sv.Width)
		}
		for i := range wantRaw {
			if raw[i] != wantRaw[i] {
				t.Fatalf("FXD raw = % x, want % x", raw, wantRaw)
			}
		}
		// Magnitude 0x26948b = round(9876.5432 * 2^8).
		const mag = 0x26948b
		if got := (sv.Unsigned >> 24) & 0xffffff; got != mag {
			t.Errorf("FXD magnitude = %#x, want %#x", got, mag)
		}
		if want := 9876.5432; math.Abs(float64(mag)/256.0-want) > 0.01 {
			t.Errorf("FXD magnitude/2^8 = %v, want ~%v", float64(mag)/256.0, want)
		}
	})
}
