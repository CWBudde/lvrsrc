package lvvi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestHeapTermBoundsHappyPath(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{
				Tag:    int32(heap.FieldTagTermBounds),
				Scope:  "leaf",
				Parent: -1,
				Content: []byte{
					0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x00, 0x07,
				},
			},
		},
		Roots: []int{0},
	}
	got, ok := HeapTermBounds(tree, 0)
	if !ok {
		t.Fatal("HeapTermBounds() ok = false, want true")
	}
	want := Bounds{Left: 0, Top: 0, Right: 7, Bottom: 7}
	if got != want {
		t.Errorf("HeapTermBounds() = %+v, want %+v", got, want)
	}
}

func TestHeapTermBoundsRejectsWrongTag(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{
				Tag:     int32(heap.FieldTagBounds), // not termBounds
				Scope:   "leaf",
				Parent:  -1,
				Content: []byte{0, 0, 0, 0, 0, 7, 0, 7},
			},
		},
		Roots: []int{0},
	}
	if _, ok := HeapTermBounds(tree, 0); ok {
		t.Error("HeapTermBounds() on non-termBounds tag returned ok=true")
	}
}

func TestHeapTermBoundsRejectsBadContentLength(t *testing.T) {
	for _, length := range []int{0, 4, 7, 9, 16} {
		tree := HeapTree{
			Nodes: []HeapNode{{
				Tag:     int32(heap.FieldTagTermBounds),
				Scope:   "leaf",
				Content: make([]byte, length),
			}},
		}
		if _, ok := HeapTermBounds(tree, 0); ok {
			t.Errorf("HeapTermBounds() on %d-byte content returned ok=true", length)
		}
	}
}

func TestFindTermBoundsChildLocatesLeaf(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 1, Scope: "open", Parent: -1, Children: []int{1, 2}},
			{Tag: int32(heap.FieldTagBgColor), Scope: "leaf", Parent: 0},
			{
				Tag:     int32(heap.FieldTagTermBounds),
				Scope:   "leaf",
				Parent:  0,
				Content: []byte{0, 5, 0, 7, 0, 0x6E, 0, 0x4D},
			},
		},
		Roots: []int{0},
	}
	got, ok := FindTermBoundsChild(tree, 0)
	if !ok {
		t.Fatal("FindTermBoundsChild() ok = false, want true")
	}
	want := Bounds{Left: 5, Top: 7, Right: 110, Bottom: 77}
	if got != want {
		t.Errorf("FindTermBoundsChild() = %+v, want %+v", got, want)
	}
}

func TestHeapTermHotPointHappyPath(t *testing.T) {
	// pylabview convention is Mac Point — vertical (V) before
	// horizontal (H). 0x0000 0xFFFC = V=0, H=-4.
	tree := HeapTree{
		Nodes: []HeapNode{{
			Tag:     int32(heap.FieldTagTermHotPoint),
			Scope:   "leaf",
			Parent:  -1,
			Content: []byte{0x00, 0x00, 0xFF, 0xFC},
		}},
	}
	got, ok := HeapTermHotPoint(tree, 0)
	if !ok {
		t.Fatal("HeapTermHotPoint() ok = false, want true")
	}
	want := Point{V: 0, H: -4}
	if got != want {
		t.Errorf("HeapTermHotPoint() = %+v, want %+v", got, want)
	}
}

func TestHeapTermHotPointRejectsBadContentLength(t *testing.T) {
	for _, length := range []int{0, 1, 2, 3, 5, 8} {
		tree := HeapTree{
			Nodes: []HeapNode{{
				Tag:     int32(heap.FieldTagTermHotPoint),
				Scope:   "leaf",
				Content: make([]byte, length),
			}},
		}
		if _, ok := HeapTermHotPoint(tree, 0); ok {
			t.Errorf("HeapTermHotPoint() on %d-byte content returned ok=true", length)
		}
	}
}

func TestFindTermHotPointChildLocatesLeaf(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 1, Scope: "open", Parent: -1, Children: []int{1}},
			{
				Tag:     int32(heap.FieldTagTermHotPoint),
				Scope:   "leaf",
				Parent:  0,
				Content: []byte{0x00, 0x05, 0x00, 0x09},
			},
		},
		Roots: []int{0},
	}
	got, ok := FindTermHotPointChild(tree, 0)
	if !ok {
		t.Fatal("FindTermHotPointChild() ok = false, want true")
	}
	want := Point{V: 5, H: 9}
	if got != want {
		t.Errorf("FindTermHotPointChild() = %+v, want %+v", got, want)
	}
}

// Sweep the corpus: every OF__termBounds leaf (8 B) should decode to
// a Bounds with non-pathological coordinates, and every
// OF__termHotPoint leaf (4 B) should decode similarly. This catches
// drift in the field-tag classification or content sizing.
func TestHeapTerminalCorpusCoverage(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalBounds, decodedBounds := 0, 0
	totalHot, decodedHot := 0, 0
	exercised := 0
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
				if n.Scope != "leaf" {
					continue
				}
				switch n.Tag {
				case int32(heap.FieldTagTermBounds):
					totalBounds++
					if _, ok := HeapTermBounds(tree, i); ok {
						decodedBounds++
					}
				case int32(heap.FieldTagTermHotPoint):
					totalHot++
					if _, ok := HeapTermHotPoint(tree, i); ok {
						decodedHot++
					}
				}
			}
		}
	}
	if exercised == 0 {
		t.Skip("no corpus VI yielded a heap tree")
	}
	if totalBounds == 0 && totalHot == 0 {
		t.Skip("corpus contains no OF__termBounds or OF__termHotPoint leaves")
	}
	if totalBounds > 0 && decodedBounds != totalBounds {
		t.Fatalf("HeapTermBounds decoded %d/%d leaves", decodedBounds, totalBounds)
	}
	if totalHot > 0 && decodedHot != totalHot {
		t.Fatalf("HeapTermHotPoint decoded %d/%d leaves", decodedHot, totalHot)
	}
	t.Logf("HeapTermBounds: %d/%d, HeapTermHotPoint: %d/%d across %d trees",
		decodedBounds, totalBounds, decodedHot, totalHot, exercised)
}
