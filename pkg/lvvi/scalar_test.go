package lvvi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestHeapScalarHappyPath(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{{
			Tag:      int32(heap.FieldTagObjFlags),
			Scope:    "leaf",
			Parent:   -1,
			SizeSpec: 4,
			Content:  []byte{0xFF, 0xFF, 0xFF, 0xFE},
		}},
		Roots: []int{0},
	}
	got, ok := HeapScalar(tree, 0)
	if !ok {
		t.Fatal("HeapScalar() ok = false, want true")
	}
	if got.Tag != int32(heap.FieldTagObjFlags) || got.Width != 4 || got.Signed != -2 || got.Unsigned != 0xFFFFFFFE {
		t.Fatalf("HeapScalar() = %+v, want tag/width/signed/unsigned decoded", got)
	}
	if len(got.Raw) != 4 || got.Raw[3] != 0xFE {
		t.Fatalf("HeapScalar().Raw = % x, want preserved payload", got.Raw)
	}
}

func TestHeapScalarRejectsWrongTagAndOversize(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: int32(heap.FieldTagBounds), Scope: "leaf", SizeSpec: 4, Content: []byte{0, 1}},
			{Tag: int32(heap.FieldTagObjFlags), Scope: "leaf", SizeSpec: 6, Content: make([]byte, 9)},
		},
	}
	if _, ok := HeapScalar(tree, 0); ok {
		t.Error("HeapScalar() on non-scalar tag returned ok=true")
	}
	if _, ok := HeapScalar(tree, 1); ok {
		t.Error("HeapScalar() on oversize scalar returned ok=true")
	}
}

func TestHeapScalarDecodesBoolShapedScalars(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: int32(heap.FieldTagActiveDiag), Scope: "leaf", SizeSpec: byte(heap.SizeSpecBoolFalse)},
			{Tag: int32(heap.FieldTagActiveDiag), Scope: "leaf", SizeSpec: byte(heap.SizeSpecBoolTrue)},
		},
	}
	got, ok := HeapScalar(tree, 0)
	if !ok {
		t.Fatal("HeapScalar(false) ok = false, want true")
	}
	if got.Width != 0 || got.Unsigned != 0 || got.Signed != 0 {
		t.Fatalf("HeapScalar(false) = %+v, want zero bool scalar", got)
	}
	got, ok = HeapScalar(tree, 1)
	if !ok {
		t.Fatal("HeapScalar(true) ok = false, want true")
	}
	if got.Width != 0 || got.Unsigned != 1 || got.Signed != 1 {
		t.Fatalf("HeapScalar(true) = %+v, want one bool scalar", got)
	}
}

func TestHeapColorHappyPath(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{{
			Tag:      int32(heap.FieldTagBgColor),
			Scope:    "leaf",
			Parent:   -1,
			SizeSpec: 4,
			Content:  []byte{0x00, 0x11, 0x22, 0x33},
		}},
		Roots: []int{0},
	}
	got, ok := HeapColor(tree, 0)
	if !ok {
		t.Fatal("HeapColor() ok = false, want true")
	}
	if got.Raw != 0x00112233 || got.Prefix != 0 || got.R != 0x11 || got.G != 0x22 || got.B != 0x33 {
		t.Fatalf("HeapColor() = %+v, want raw/prefix/rgb decoded", got)
	}
	if got.HexRGB() != "#112233" {
		t.Fatalf("HexRGB() = %q, want #112233", got.HexRGB())
	}
}

func TestHeapColorRejectsWrongTagAndLength(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: int32(heap.FieldTagObjFlags), Scope: "leaf", Content: []byte{0, 1, 2, 3}},
			{Tag: int32(heap.FieldTagBgColor), Scope: "leaf", Content: []byte{0, 1, 2}},
		},
	}
	if _, ok := HeapColor(tree, 0); ok {
		t.Error("HeapColor() on non-color tag returned ok=true")
	}
	if _, ok := HeapColor(tree, 1); ok {
		t.Error("HeapColor() on 3-byte color returned ok=true")
	}
}

func TestFindScalarAndColorChild(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 1, Scope: "open", Parent: -1, Children: []int{1, 2}},
			{Tag: int32(heap.FieldTagObjFlags), Scope: "leaf", Parent: 0, SizeSpec: 2, Content: []byte{0x12, 0x34}},
			{Tag: int32(heap.FieldTagFgColor), Scope: "leaf", Parent: 0, SizeSpec: 4, Content: []byte{0, 0xAA, 0xBB, 0xCC}},
		},
	}
	scalar, ok := FindScalarChild(tree, 0, int32(heap.FieldTagObjFlags))
	if !ok {
		t.Fatal("FindScalarChild() ok = false, want true")
	}
	if scalar.Unsigned != 0x1234 {
		t.Fatalf("FindScalarChild() unsigned = %#x, want 0x1234", scalar.Unsigned)
	}
	color, ok := FindColorChild(tree, 0, int32(heap.FieldTagFgColor))
	if !ok {
		t.Fatal("FindColorChild() ok = false, want true")
	}
	if color.HexRGB() != "#AABBCC" {
		t.Fatalf("FindColorChild().HexRGB() = %q, want #AABBCC", color.HexRGB())
	}
}

func TestHeapScalarCorpusCoverage(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalScalar := 0
	decodedScalar := 0
	totalColor := 0
	decodedColor := 0
	nonFourByteColor := 0
	exercised := 0
	countsByTag := map[int32]int{}
	undecodedByTag := map[int32]int{}
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
			exercised++
			for i, n := range tree.Nodes {
				if n.Scope != "leaf" || !IsHeapScalarTag(n.Tag) {
					continue
				}
				totalScalar++
				countsByTag[n.Tag]++
				if _, ok := HeapScalar(tree, i); ok {
					decodedScalar++
				} else {
					undecodedByTag[n.Tag]++
				}
				if IsHeapColorTag(n.Tag) {
					totalColor++
					if _, ok := HeapColor(tree, i); ok {
						decodedColor++
					} else {
						nonFourByteColor++
					}
				}
			}
		}
	}
	if exercised == 0 {
		t.Skip("no corpus VI yielded an FPHb or BDHb tree")
	}
	if totalScalar == 0 {
		t.Fatal("found 0 known scalar leaves")
	}
	if decodedScalar != totalScalar {
		t.Fatalf("HeapScalar decoded %d/%d known scalar leaves; undecoded=%v", decodedScalar, totalScalar, undecodedByTag)
	}
	if totalColor == 0 || decodedColor == 0 {
		t.Fatalf("HeapColor decoded %d/%d color leaves, want non-zero coverage", decodedColor, totalColor)
	}
	t.Logf("HeapScalar: %d/%d leaves decoded across %d trees; tags=%v", decodedScalar, totalScalar, exercised, countsByTag)
	t.Logf("HeapColor: %d/%d 4-byte color leaves decoded (%d non-4-byte color leaves skipped)",
		decodedColor, totalColor, nonFourByteColor)
}
