package lvvi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestBoundsWidthHeight(t *testing.T) {
	b := Bounds{Left: 10, Top: 20, Right: 130, Bottom: 80}
	if got, want := b.Width(), int16(120); got != want {
		t.Errorf("Width() = %d, want %d", got, want)
	}
	if got, want := b.Height(), int16(60); got != want {
		t.Errorf("Height() = %d, want %d", got, want)
	}
}

func TestHeapBoundsHappyPath(t *testing.T) {
	// OF__bounds leaf with left=10, top=20, right=300, bottom=-1
	tree := HeapTree{
		Nodes: []HeapNode{
			{
				Tag:    int32(heap.FieldTagBounds),
				Scope:  "leaf",
				Parent: -1,
				Content: []byte{
					0x00, 0x0A, 0x00, 0x14, 0x01, 0x2C, 0xFF, 0xFF,
				},
			},
		},
		Roots: []int{0},
	}
	got, ok := HeapBounds(tree, 0)
	if !ok {
		t.Fatal("HeapBounds() ok = false, want true")
	}
	want := Bounds{Left: 10, Top: 20, Right: 300, Bottom: -1}
	if got != want {
		t.Errorf("HeapBounds() = %+v, want %+v", got, want)
	}
}

func TestHeapBoundsRejectsWrongTag(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{
				Tag:     int32(heap.FieldTagBgColor),
				Scope:   "leaf",
				Parent:  -1,
				Content: []byte{0x00, 0x0A, 0x00, 0x14, 0x01, 0x2C, 0xFF, 0xFF},
			},
		},
		Roots: []int{0},
	}
	if _, ok := HeapBounds(tree, 0); ok {
		t.Error("HeapBounds() on non-bounds tag returned ok=true, want false")
	}
}

func TestHeapBoundsRejectsBadContentLength(t *testing.T) {
	for _, length := range []int{0, 4, 7, 9, 16} {
		tree := HeapTree{
			Nodes: []HeapNode{
				{
					Tag:     int32(heap.FieldTagBounds),
					Scope:   "leaf",
					Parent:  -1,
					Content: make([]byte, length),
				},
			},
			Roots: []int{0},
		}
		if _, ok := HeapBounds(tree, 0); ok {
			t.Errorf("HeapBounds() on %d-byte content returned ok=true, want false", length)
		}
	}
}

func TestHeapBoundsRejectsOutOfRangeIndex(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{}, Roots: []int{}}
	for _, idx := range []int{-1, 0, 1, 100} {
		if _, ok := HeapBounds(tree, idx); ok {
			t.Errorf("HeapBounds(idx=%d) on empty tree returned ok=true, want false", idx)
		}
	}
}

func TestFindBoundsChildLocatesOFBoundsLeafAmongSiblings(t *testing.T) {
	// Synthetic control node (parent at index 0) with three children:
	//   1: OF__bgColor leaf  (not bounds)
	//   2: OF__bounds  leaf  ← target
	//   3: OF__objFlags leaf (not bounds)
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 1, Scope: "open", Parent: -1, Children: []int{1, 2, 3}},
			{Tag: int32(heap.FieldTagBgColor), Scope: "leaf", Parent: 0},
			{
				Tag:    int32(heap.FieldTagBounds),
				Scope:  "leaf",
				Parent: 0,
				Content: []byte{
					0x00, 0x05, 0x00, 0x07, 0x00, 0x6E, 0x00, 0x4D,
				},
			},
			{Tag: int32(heap.FieldTagObjFlags), Scope: "leaf", Parent: 0},
		},
		Roots: []int{0},
	}
	got, ok := FindBoundsChild(tree, 0)
	if !ok {
		t.Fatal("FindBoundsChild() ok = false, want true")
	}
	want := Bounds{Left: 5, Top: 7, Right: 110, Bottom: 77}
	if got != want {
		t.Errorf("FindBoundsChild() = %+v, want %+v", got, want)
	}
}

func TestFindBoundsChildReturnsFalseWhenNoBoundsSibling(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 1, Scope: "open", Parent: -1, Children: []int{1}},
			{Tag: int32(heap.FieldTagBgColor), Scope: "leaf", Parent: 0},
		},
		Roots: []int{0},
	}
	if _, ok := FindBoundsChild(tree, 0); ok {
		t.Error("FindBoundsChild() with no OF__bounds sibling returned ok=true, want false")
	}
}

// TestHeapBoundsCorpusCoverage exercises HeapBounds against every
// OF__bounds leaf in every FPHb/BDHb tree of the corpus. It records
// per-fixture and aggregate decoding rates, and asserts non-empty
// coverage so a regression in tag classification or content sizing is
// caught here. It is informational rather than strict — coordinates are
// only sanity-checked for non-pathological extremes (≤ ±10000).
func TestHeapBoundsCorpusCoverage(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalBoundsLeaves := 0
	totalDecoded := 0
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
				if n.Tag != int32(heap.FieldTagBounds) || n.Scope != "leaf" {
					continue
				}
				totalBoundsLeaves++
				b, ok := HeapBounds(tree, i)
				if !ok {
					continue
				}
				totalDecoded++
				for _, v := range []int16{b.Left, b.Top, b.Right, b.Bottom} {
					if v < -10000 || v > 10000 {
						t.Errorf("%s node %d: extreme bounds coord %d in %+v",
							e.Name(), i, v, b)
					}
				}
			}
		}
	}
	if exercised == 0 {
		t.Skip("no corpus VI yielded an FPHb or BDHb tree")
	}
	if totalBoundsLeaves == 0 {
		t.Fatalf("found 0 OF__bounds leaves across %d trees — tag classification regressed", exercised)
	}
	if totalDecoded != totalBoundsLeaves {
		t.Fatalf("HeapBounds decoded %d/%d OF__bounds leaves — bounds payload format inconsistent",
			totalDecoded, totalBoundsLeaves)
	}
	t.Logf("HeapBounds: %d/%d OF__bounds leaves decoded across %d trees",
		totalDecoded, totalBoundsLeaves, exercised)
}
