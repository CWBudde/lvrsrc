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
// block-diagram heap and decodes it as a scalar.
func findConstValue(t *testing.T, fixture string) (ScalarValue, []byte) {
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
		sv, ok := HeapScalarForTag(tree, i, int32(heap.FieldTagConstValue))
		if !ok {
			t.Fatalf("%s: HeapScalarForTag(OF__ConstValue) ok=false", fixture)
		}
		return sv, append([]byte(nil), n.Content...)
	}
	t.Fatalf("%s: no OF__ConstValue leaf found", fixture)
	return ScalarValue{}, nil
}

// TestHeapConstValueGroundTruth pins the block-diagram numeric-constant
// literal encoding. The OF__ConstValue leaf (tag 589) of an I32 constant
// is a fixed-width 4-byte big-endian int32 written verbatim — NOT the
// byte-stream-with-0xff-escape used by the compressed-wire geometry.
//
// The fixtures straddle both single-byte boundaries: 255/256 cross the
// 1-byte limit and 65535/65536 cross the 2-byte limit. A varint or
// escape scheme would change byte width or insert a 0xff prefix at those
// boundaries; the raw bytes show neither — only positional big-endian.
func TestHeapConstValueGroundTruth(t *testing.T) {
	cases := []struct {
		fixture string
		want    int64
		raw     []byte
	}{
		{"Numeric42.vi", 42, []byte{0x00, 0x00, 0x00, 0x2a}},
		{"Numeric255.vi", 255, []byte{0x00, 0x00, 0x00, 0xff}},
		{"Numeric256.vi", 256, []byte{0x00, 0x00, 0x01, 0x00}},
		{"Numeric65535.vi", 65535, []byte{0x00, 0x00, 0xff, 0xff}},
		{"Numeric65536.vi", 65536, []byte{0x00, 0x01, 0x00, 0x00}},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			sv, raw := findConstValue(t, tc.fixture)
			if sv.Width != 4 {
				t.Errorf("Width = %d, want 4 (I32 constant)", sv.Width)
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

// TestHeapConstValueDBLGroundTruth pins the floating-point constant
// encoding. A DBL constant's OF__ConstValue widens to 8 bytes holding a
// big-endian IEEE-754 double written verbatim — confirming the literal
// width tracks the constant's data type (4 bytes for I32, 8 for DBL)
// rather than being a magnitude-dependent varint.
func TestHeapConstValueDBLGroundTruth(t *testing.T) {
	const fixture = "Numeric9876Dot5432.vi"
	want := 9876.5432
	wantRaw := []byte{0x40, 0xc3, 0x4a, 0x45, 0x87, 0x93, 0xdd, 0x98}

	sv, raw := findConstValue(t, fixture)
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
	// Unsigned carries the raw big-endian IEEE-754 bits; reinterpret.
	got := math.Float64frombits(sv.Unsigned)
	if got != want {
		t.Errorf("decoded DBL = %v, want %v", got, want)
	}
	if math.Float64frombits(binary.BigEndian.Uint64(raw)) != want {
		t.Errorf("raw big-endian DBL = %v, want %v", math.Float64frombits(binary.BigEndian.Uint64(raw)), want)
	}
}
