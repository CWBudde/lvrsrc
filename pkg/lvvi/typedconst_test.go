package lvvi

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// typedConst opens a fixture and returns its single block-diagram typed
// constant. Every controlled-constant fixture carries exactly one.
func typedConst(t *testing.T, fixture string) TypedConst {
	t.Helper()
	f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), fixture), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open(%s): %v", fixture, err)
	}
	m, _ := DecodeKnownResources(f)
	consts, ok := m.BlockDiagramConstants()
	if !ok {
		t.Fatalf("%s: no block-diagram heap", fixture)
	}
	if len(consts) != 1 {
		t.Fatalf("%s: got %d constants, want exactly 1", fixture, len(consts))
	}
	return consts[0]
}

// TestBlockDiagramConstantTypeJoin pins the heap↔VCTP join: every
// controlled-constant fixture resolves its OF__ConstValue leaf to the
// correct VCTP type and decodes the matching typed value.
//
// Numeric42.vi is the disambiguator. The I32 constant is wired to a DBL
// "Numeric" indicator, so the constant object references two VCTP types —
// VCTP[3]=NumFloat64 (indicator) and VCTP[4]=NumInt32 (constant). The
// resolver must select the constant's own type (NumInt32, 4-byte), not
// the indicator's (NumFloat64, 8-byte); width alone cannot tell them
// apart, the VCTP join can.
func TestBlockDiagramConstantTypeJoin(t *testing.T) {
	cases := []struct {
		fixture  string
		fullType string
		kind     ConstKind
	}{
		{"Numeric42.vi", "NumInt32", ConstKindSignedInt},
		{"Numeric42_I8.vi", "NumInt8", ConstKindSignedInt},
		{"Numeric42_U8.vi", "NumUInt8", ConstKindUnsignedInt},
		{"Numeric65536_I64.vi", "NumInt64", ConstKindSignedInt},
		{"Numeric9876Dot5432_SGL.vi", "NumFloat32", ConstKindFloat},
		{"Numeric9876Dot5432.vi", "NumFloat64", ConstKindFloat},
		{"Numeric9876Dot5432_EXT.vi", "NumFloatExt", ConstKindFloat},
		{"Numeric9876Dot5432_FXD.vi", "FixedPoint", ConstKindFixedPoint},
		{"Numeric42_CSG.vi", "NumComplex64", ConstKindComplex},
		{"Numeric42_i5_CDB.vi", "NumComplex128", ConstKindComplex},
		{"Numeric42_CEXT.vi", "NumComplexExt", ConstKindComplex},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			c := typedConst(t, tc.fixture)
			if !c.HasType {
				t.Fatalf("HasType = false, want a resolved VCTP type")
			}
			if c.FullType != tc.fullType {
				t.Errorf("FullType = %q, want %q", c.FullType, tc.fullType)
			}
			if c.Kind != tc.kind {
				t.Errorf("Kind = %v, want %v", c.Kind, tc.kind)
			}
			if !c.WidthMatch {
				t.Errorf("WidthMatch = false: resolved type %q width != %d raw bytes",
					c.FullType, len(c.Raw))
			}
		})
	}
}

// TestBlockDiagramConstantBoolean pins the dual-width Boolean literal. Across
// the whole corpus a TRUE Boolean constant is stored as a 1-byte 0x01 and a
// FALSE as a 2-byte 0x0000; both resolve to the same Boolean VCTP type (the
// descriptor flags do not distinguish the widths), so both must come back as
// Kind=boolean with WidthMatch=true and the value decoded from any non-zero
// byte. BoolToLED carries the 1-byte TRUE; WhileLoop_Numeric42 and
// reference-find-by-id carry the 2-byte FALSE — the WidthMatch=false outlier
// this test closes.
func TestBlockDiagramConstantBoolean(t *testing.T) {
	cases := []struct {
		fixture string
		node    int
		rawLen  int
		value   uint64 // 0 = false, non-zero = true
	}{
		{"BoolToLED.vi", 48, 1, 1},
		{"WhileLoop_Numeric42.vi", 169, 2, 0},
		{"reference-find-by-id.vi", 1180, 2, 0},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), tc.fixture), lvrsrc.OpenOptions{})
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			m, _ := DecodeKnownResources(f)
			consts, ok := m.BlockDiagramConstants()
			if !ok {
				t.Fatal("no block-diagram heap")
			}
			var c *TypedConst
			for i := range consts {
				if consts[i].NodeIndex == tc.node {
					c = &consts[i]
					break
				}
			}
			if c == nil {
				t.Fatalf("no constant at node %d", tc.node)
			}
			if c.FullType != "Boolean" {
				t.Errorf("FullType = %q, want Boolean", c.FullType)
			}
			if c.Kind != ConstKindBoolean {
				t.Errorf("Kind = %v, want boolean", c.Kind)
			}
			if len(c.Raw) != tc.rawLen {
				t.Errorf("rawLen = %d, want %d", len(c.Raw), tc.rawLen)
			}
			if !c.WidthMatch {
				t.Errorf("WidthMatch = false; a %d-byte Boolean literal must be accepted", len(c.Raw))
			}
			if c.Uint != tc.value {
				t.Errorf("Uint = %#x, want %#x", c.Uint, tc.value)
			}
		})
	}
}

// TestBlockDiagramConstantTopTypesIndirection pins the TopTypes
// indirection that the single-constant controlled fixtures cannot prove.
// The OF__typeDesc content is a 0-based index into the VCTP top-types
// list, not a flat VCTP index; tops[content] is the flat index.
//
// Add17Plus25.vi has only 5 VCTP descriptors but its two I32 constants
// carry OF__typeDesc 0x07 and 0x09 — indexing the flat list directly
// would be out of range. Both must route through tops (tops[7]=tops[9]=4
// → VCTP[4]=NumInt32). module-timeout--constant.vi confirms the same with
// a non-identity mapping: tops[4]=1 → VCTP[1]=NumInt32 "Module Timeout".
func TestBlockDiagramConstantTopTypesIndirection(t *testing.T) {
	t.Run("Add17Plus25", func(t *testing.T) {
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), "Add17Plus25.vi"), lvrsrc.OpenOptions{})
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		m, _ := DecodeKnownResources(f)
		consts, ok := m.BlockDiagramConstants()
		if !ok {
			t.Fatal("no block-diagram heap")
		}
		got := map[int64]bool{}
		for _, c := range consts {
			if c.Kind != ConstKindSignedInt {
				t.Errorf("constant at node %d: Kind = %v, want signed-int", c.NodeIndex, c.Kind)
			}
			if c.FullType != "NumInt32" {
				t.Errorf("constant at node %d: FullType = %q, want NumInt32 (via tops, not flat[7]/flat[9])", c.NodeIndex, c.FullType)
			}
			if c.TypeIndex != 4 {
				t.Errorf("constant at node %d: TypeIndex = %d, want 4 (flat), not the 0x07/0x09 top-types ordinal", c.NodeIndex, c.TypeIndex)
			}
			got[c.Int] = true
		}
		for _, want := range []int64{17, 25} {
			if !got[want] {
				t.Errorf("missing constant value %d (got %v)", want, got)
			}
		}
	})

	t.Run("module-timeout", func(t *testing.T) {
		c := typedConst(t, "module-timeout--constant.vi")
		if c.FullType != "NumInt32" {
			t.Errorf("FullType = %q, want NumInt32 (tops[4]=1 -> VCTP[1])", c.FullType)
		}
		if c.TypeIndex != 1 {
			t.Errorf("TypeIndex = %d, want 1 (non-identity tops mapping)", c.TypeIndex)
		}
		if c.Int != 5000 {
			t.Errorf("Int = %d, want 5000", c.Int)
		}
	})
}

// TestBlockDiagramConstantIntegers pins the resolved integer values, with
// the I8/U8 pair showing signedness comes from the VCTP type (both store
// 0x2a, but U8 decodes as ConstKindUnsignedInt).
func TestBlockDiagramConstantIntegers(t *testing.T) {
	cases := []struct {
		fixture string
		want    int64
	}{
		{"Numeric42.vi", 42},
		{"Numeric42_I8.vi", 42},
		{"Numeric42_U8.vi", 42},
		{"Numeric65536_I64.vi", 65536},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			c := typedConst(t, tc.fixture)
			if c.Int != tc.want {
				t.Errorf("Int = %d, want %d", c.Int, tc.want)
			}
			if c.Uint != uint64(tc.want) {
				t.Errorf("Uint = %d, want %d", c.Uint, tc.want)
			}
		})
	}
}

// TestBlockDiagramConstantFloats pins the typed float decode across all
// three IEEE-754 widths. SGL stores the float32 rounding; DBL is exact;
// EXT is decoded from binary128 to the nearest float64 (the stored bytes
// carry ~float32 precision, so a tight tolerance still confirms a real
// decode rather than a misread).
func TestBlockDiagramConstantFloats(t *testing.T) {
	const want = 9876.5432

	t.Run("SGL", func(t *testing.T) {
		c := typedConst(t, "Numeric9876Dot5432_SGL.vi")
		if got, w := c.Float, float64(float32(want)); got != w {
			t.Errorf("Float = %v, want %v (float32 rounding)", got, w)
		}
	})
	t.Run("DBL", func(t *testing.T) {
		c := typedConst(t, "Numeric9876Dot5432.vi")
		if c.Float != want {
			t.Errorf("Float = %v, want %v (exact)", c.Float, want)
		}
	})
	t.Run("EXT", func(t *testing.T) {
		c := typedConst(t, "Numeric9876Dot5432_EXT.vi")
		if math.Abs(c.Float-want) > 0.05 {
			t.Errorf("Float = %v, want ~%v (binary128 decode)", c.Float, want)
		}
	})
}

// TestBlockDiagramConstantComplex pins the complex decode for all three
// component widths — this is the payoff of the type join: at the leaf, a
// 16-byte CDB is indistinguishable from a 16-byte EXT and a 32-byte CEXT
// from a string, but the VCTP type resolves them to NumComplex128 /
// NumComplexExt and decodes real-then-imaginary components.
func TestBlockDiagramConstantComplex(t *testing.T) {
	cases := []struct {
		fixture string
		re, im  float64
		tol     float64
	}{
		{"Numeric42_CSG.vi", 42, 0, 0},
		{"Numeric42_i5_CDB.vi", 42, 5, 0},
		{"Numeric42_CEXT.vi", 42, 0, 0.05},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			c := typedConst(t, tc.fixture)
			if c.Kind != ConstKindComplex {
				t.Fatalf("Kind = %v, want complex", c.Kind)
			}
			if math.Abs(c.Real-tc.re) > tc.tol {
				t.Errorf("Real = %v, want %v", c.Real, tc.re)
			}
			if math.Abs(c.Imag-tc.im) > tc.tol {
				t.Errorf("Imag = %v, want %v", c.Imag, tc.im)
			}
		})
	}
}

// TestBlockDiagramConstantFixedPoint pins the fixed-point magnitude. The
// FXD constant stores the scaled magnitude in an 8-byte container; here
// 9876.5432 is 0x26948b = round(9876.5432 * 2^8) lodged at bits 24..47.
func TestBlockDiagramConstantFixedPoint(t *testing.T) {
	c := typedConst(t, "Numeric9876Dot5432_FXD.vi")
	if c.Kind != ConstKindFixedPoint {
		t.Fatalf("Kind = %v, want fixed-point", c.Kind)
	}
	const mag = 0x26948b
	if got := (c.FixedRaw >> 24) & 0xffffff; got != mag {
		t.Errorf("FXD magnitude = %#x, want %#x", got, mag)
	}
	if want := 9876.5432; math.Abs(float64(mag)/256.0-want) > 0.01 {
		t.Errorf("FXD magnitude/2^8 = %v, want ~%v", float64(mag)/256.0, want)
	}
}
