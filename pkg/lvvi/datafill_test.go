package lvvi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestDataFillKindString(t *testing.T) {
	cases := []struct {
		k    DataFillKind
		want string
	}{
		{DataFillKindUnknown, "unknown"},
		{DataFillKindRaw, "raw"},
		{DataFillKindInt, "int"},
		{DataFillKindUInt, "uint"},
		{DataFillKindFloat32, "float32"},
		{DataFillKindFloat64, "float64"},
		{DataFillKind(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.k.String(); got != c.want {
			t.Errorf("DataFillKind(%d).String() = %q, want %q", c.k, got, c.want)
		}
	}
}

func TestSignExtendBE(t *testing.T) {
	cases := []struct {
		name string
		buf  []byte
		want int64
	}{
		{"empty", nil, 0},
		{"1 byte 0", []byte{0x00}, 0},
		{"1 byte -1", []byte{0xFF}, -1},
		{"1 byte 5", []byte{0x05}, 5},
		{"1 byte -5", []byte{0xFB}, -5},
		{"2 bytes BE -2", []byte{0xFF, 0xFE}, -2},
		{"4 bytes BE positive", []byte{0x00, 0x00, 0xCA, 0xFE}, 0xCAFE},
		{"4 bytes BE negative", []byte{0xFF, 0xFF, 0xFF, 0xFE}, -2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := signExtendBE(c.buf)
			if err != nil {
				t.Fatalf("signExtendBE: %v", err)
			}
			if got != c.want {
				t.Errorf("signExtendBE(%v) = %d, want %d", c.buf, got, c.want)
			}
		})
	}
}

func TestSignExtendBERejectsOversize(t *testing.T) {
	if _, err := signExtendBE(make([]byte, 9)); err == nil {
		t.Error("signExtendBE(9 bytes) returned nil error, want oversize")
	}
}

// TestHeapDataFillRejectsNonDataFillTag verifies the function only
// claims DataFill tags and returns ok=false otherwise.
func TestHeapDataFillRejectsNonDataFillTag(t *testing.T) {
	m, _ := DecodeKnownResources(&lvrsrc.File{})
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 100, Parent: -1}, // not a DataFill tag
		},
		Roots: []int{0},
	}
	if _, ok := m.HeapDataFill(tree, 0); ok {
		t.Error("HeapDataFill on non-DataFill tag returned ok=true")
	}
}

// TestHeapDataFillRejectsOutOfRange verifies bounds checking.
func TestHeapDataFillRejectsOutOfRange(t *testing.T) {
	m, _ := DecodeKnownResources(&lvrsrc.File{})
	tree := HeapTree{Nodes: []HeapNode{}}
	if _, ok := m.HeapDataFill(tree, 0); ok {
		t.Error("HeapDataFill(idx=0) on empty tree returned ok=true")
	}
	if _, ok := m.HeapDataFill(tree, -1); ok {
		t.Error("HeapDataFill(idx=-1) returned ok=true")
	}
}

// TestHeapDataFillNoParent populates Raw but flags Unknown when the
// node has no parent (so no sibling typeDesc to consult).
func TestHeapDataFillNoParent(t *testing.T) {
	m, _ := DecodeKnownResources(&lvrsrc.File{})
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 513, Parent: -1, Content: []byte{0xCA, 0xFE}},
		},
		Roots: []int{0},
	}
	v, ok := m.HeapDataFill(tree, 0)
	if !ok {
		t.Fatal("HeapDataFill ok=false")
	}
	if v.Kind != DataFillKindUnknown {
		t.Errorf("Kind = %v, want Unknown", v.Kind)
	}
	if string(v.Raw) != string([]byte{0xCA, 0xFE}) {
		t.Errorf("Raw = %v, want [0xCA 0xFE]", v.Raw)
	}
	if v.HeapTypeID != 0 {
		t.Errorf("HeapTypeID = %d, want 0", v.HeapTypeID)
	}
}

// TestHeapDataFillCorpusSweep is the main acceptance test: every
// FPHb DataFill node in the corpus must resolve through HeapDataFill,
// the original content bytes must round-trip via Raw, and at least
// some entries must surface a typed numeric Kind.
func TestHeapDataFillCorpusSweep(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	histogram := map[DataFillKind]int{}
	resolvedNumeric := 0
	totalDataFill := 0
	totalFiles := 0
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
		tree, ok := m.FrontPanel()
		if !ok {
			continue
		}
		totalFiles++
		for i, n := range tree.Nodes {
			if !isDataFillTag(n.Tag) {
				continue
			}
			totalDataFill++
			val, ok := m.HeapDataFill(tree, i)
			if !ok {
				t.Errorf("%s: HeapDataFill(node=%d tag=%d) ok=false", e.Name(), i, n.Tag)
				continue
			}
			histogram[val.Kind]++
			if len(val.Raw) != len(n.Content) {
				t.Errorf("%s: HeapDataFill Raw length %d != content length %d",
					e.Name(), len(val.Raw), len(n.Content))
			}
			if val.Kind == DataFillKindInt || val.Kind == DataFillKindUInt ||
				val.Kind == DataFillKindFloat32 || val.Kind == DataFillKindFloat64 {
				resolvedNumeric++
				if val.ResolvedTypeIdx == 0 {
					t.Errorf("%s: numeric Kind=%v but ResolvedTypeIdx=0", e.Name(), val.Kind)
				}
				if val.FullType == "" {
					t.Errorf("%s: numeric Kind=%v but FullType empty", e.Name(), val.Kind)
				}
			}
		}
	}
	if totalDataFill == 0 {
		t.Skip("no DataFill tags found in corpus")
	}
	t.Logf("HeapDataFill swept %d files, %d DataFill nodes, %d resolved to typed numeric kinds",
		totalFiles, totalDataFill, resolvedNumeric)
	t.Logf("Kind histogram: %+v", histogram)
	// Acceptance: corpus has known UInt16 / Int32 hits — at least
	// one numeric resolution must succeed.
	if resolvedNumeric == 0 {
		t.Error("no DataFill node resolved to a typed numeric Kind — expected at least UInt16/Int32 hits from action.ctl/load-vi.vi")
	}
}
