package lvvi

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// frontPanelTypes opens a fixture and returns its front-panel type joins.
func frontPanelTypes(t *testing.T, fixture string) []FrontPanelType {
	t.Helper()
	f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), fixture), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open(%s): %v", fixture, err)
	}
	m, _ := DecodeKnownResources(f)
	types, ok := m.FrontPanelTypes()
	if !ok {
		t.Fatalf("%s: no front-panel heap", fixture)
	}
	return types
}

// hasResolvedType reports whether any entry resolved to the named VCTP
// FullType.
func hasResolvedType(types []FrontPanelType, fullType string) bool {
	for _, ft := range types {
		if ft.HasType && ft.Type.FullType == fullType {
			return true
		}
	}
	return false
}

// TestFrontPanelTypeJoin pins the FPHb↔VCTP type join: each controlled
// fixture's single front-panel control resolves through the TopTypes
// indirection to the control's declared VCTP type. The constant on the
// block diagram and the control on the panel are independent — Numeric42's
// panel indicator is a DBL (NumFloat64) even though the wired diagram
// constant is an I32 — so resolving the panel type confirms the join works
// on the front-panel heap, not just the block diagram.
func TestFrontPanelTypeJoin(t *testing.T) {
	cases := []struct {
		fixture     string
		controlType string
	}{
		{"Numeric42.vi", "NumFloat64"},
		{"Numeric42_I8.vi", "NumInt8"},
		{"Numeric42_U8.vi", "NumUInt8"},
		{"Numeric65536_I64.vi", "NumInt64"},
		{"Numeric9876Dot5432_SGL.vi", "NumFloat32"},
		{"Numeric9876Dot5432.vi", "NumFloat64"},
		{"Numeric42_CSG.vi", "NumComplex64"},
		{"Numeric42_i5_CDB.vi", "NumComplex128"},
		{"NumericDblInput.vi", "NumFloat64"},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			types := frontPanelTypes(t, tc.fixture)
			if len(types) == 0 {
				t.Fatal("no OF__typeDesc leaves resolved on the front panel")
			}
			for _, ft := range types {
				if !ft.HasType {
					t.Errorf("node %d: type unresolved (TopIndex=%d, TypeIndex=%d)",
						ft.NodeIndex, ft.TopIndex, ft.TypeIndex)
				}
			}
			if !hasResolvedType(types, tc.controlType) {
				got := make([]string, 0, len(types))
				for _, ft := range types {
					got = append(got, ft.Type.FullType)
				}
				t.Errorf("control type %q not among resolved panel types %v",
					tc.controlType, got)
			}
		})
	}
}

// TestFrontPanelTypeTopTypesIndirection pins that the panel join routes
// through TopTypes, not a flat VCTP index. Numeric42's control typeDesc
// content is 0x02, and TopTypes[2]=3 → VCTP[3]=NumFloat64; indexing the
// flat list with 0x02 directly would hit VCTP[2], a different type.
func TestFrontPanelTypeTopTypesIndirection(t *testing.T) {
	f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), "Numeric42.vi"), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	m, _ := DecodeKnownResources(f)
	types, ok := m.FrontPanelTypes()
	if !ok {
		t.Fatal("no front-panel heap")
	}
	tops, ok := m.TopTypes()
	if !ok {
		t.Fatal("no top-types")
	}

	var dbl *FrontPanelType
	for i := range types {
		if types[i].HasType && types[i].Type.FullType == "NumFloat64" {
			dbl = &types[i]
			break
		}
	}
	if dbl == nil {
		t.Fatal("no NumFloat64 control resolved")
	}
	if dbl.TopIndex < 0 || dbl.TopIndex >= len(tops) {
		t.Fatalf("TopIndex %d out of range for %d tops", dbl.TopIndex, len(tops))
	}
	if int(tops[dbl.TopIndex]) != dbl.TypeIndex {
		t.Errorf("TypeIndex %d != TopTypes[%d]=%d (join must route through tops)",
			dbl.TypeIndex, dbl.TopIndex, tops[dbl.TopIndex])
	}
	if dbl.TopIndex == dbl.TypeIndex {
		t.Errorf("TopIndex == TypeIndex (%d); Numeric42's control proves the indirection is non-identity (2 → tops[2]=3)", dbl.TopIndex)
	}
}

// TestFrontPanelDefaultScalar pins the scalar control-default decode. A DBL
// control whose current value was committed as the default (Make Current
// Value Default) stores an 8-byte OF__DefaultData that resolves — through
// the same nearest-preceding-typeDesc → TopTypes join as a block-diagram
// constant — to NumFloat64 and decodes to the exact value 9876.5432.
func TestFrontPanelDefaultScalar(t *testing.T) {
	f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), "NumericDblInput.vi"), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	m, _ := DecodeKnownResources(f)
	defs, ok := m.FrontPanelDefaults()
	if !ok {
		t.Fatal("no front-panel heap")
	}
	if len(defs) != 1 {
		t.Fatalf("got %d default leaves, want exactly 1", len(defs))
	}
	d := defs[0]
	if !d.HasType {
		t.Fatal("default did not resolve a VCTP type")
	}
	if d.FullType != "NumFloat64" {
		t.Errorf("FullType = %q, want NumFloat64", d.FullType)
	}
	if d.Kind != ConstKindFloat {
		t.Errorf("Kind = %v, want float", d.Kind)
	}
	if !d.WidthMatch {
		t.Errorf("WidthMatch = false: %d raw bytes for NumFloat64", len(d.Raw))
	}
	if d.Float != 9876.5432 {
		t.Errorf("Float = %v, want 9876.5432 (exact)", d.Float)
	}
}

// TestFrontPanelDefaultComposite pins that composite (cluster) defaults never
// masquerade as fixed-width scalars: their raw blob carries the embedded field
// literals and WidthMatch stays false. The structured decode of those blobs
// lives in TestFrontPanelDefaultCompositeDecode (the value is read from
// Composite, not from the scalar fields).
func TestFrontPanelDefaultComposite(t *testing.T) {
	cases := []struct {
		fixture string
		wantSub []byte
	}{
		{"response.ctl", []byte("ok")},
		{"error-response.ctl", []byte("error")},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), tc.fixture), lvrsrc.OpenOptions{})
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			m, _ := DecodeKnownResources(f)
			defs, ok := m.FrontPanelDefaults()
			if !ok {
				t.Fatal("no front-panel heap")
			}
			if len(defs) == 0 {
				t.Fatal("expected at least one OF__DefaultData leaf")
			}
			found := false
			for _, d := range defs {
				if bytes.Contains(d.Raw, tc.wantSub) {
					found = true
					if d.WidthMatch {
						t.Errorf("composite default WidthMatch=true; a cluster blob must not pass as a fixed-width scalar")
					}
				}
			}
			if !found {
				t.Errorf("no DefaultData blob contained %q; got %d blobs", tc.wantSub, len(defs))
			}
		})
	}
}

// TestFrontPanelTypesNoPanel confirms the ok=false contract when a fixture
// has no decodable front-panel heap is not silently a success. Every
// corpus VI has an FPHb, so this guards the nil-receiver / no-file paths.
func TestFrontPanelTypesNoPanel(t *testing.T) {
	var m *Model
	if _, ok := m.FrontPanelTypes(); ok {
		t.Error("nil model: FrontPanelTypes ok=true, want false")
	}
	if _, ok := m.FrontPanelDefaults(); ok {
		t.Error("nil model: FrontPanelDefaults ok=true, want false")
	}
}
