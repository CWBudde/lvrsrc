package lvvi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestHeapPointHappyPath(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{{
			Tag:     int32(heap.FieldTagOrigin),
			Scope:   "leaf",
			Parent:  -1,
			Content: []byte{0xFF, 0xFE, 0x00, 0x03},
		}},
		Roots: []int{0},
	}
	got, ok := HeapPoint(tree, 0)
	if !ok {
		t.Fatal("HeapPoint() ok = false, want true")
	}
	want := PointValue{Tag: int32(heap.FieldTagOrigin), X: -2, Y: 3}
	if got != want {
		t.Fatalf("HeapPoint() = %+v, want %+v", got, want)
	}

	got, ok = HeapPointForTag(tree, 0, int32(heap.FieldTagOrigin))
	if !ok {
		t.Fatal("HeapPointForTag() ok = false, want true")
	}
	if got != want {
		t.Fatalf("HeapPointForTag() = %+v, want %+v", got, want)
	}
}

func TestHeapPointRejectsUnknownTagAndBadLength(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: int32(heap.FieldTagObjFlags), Scope: "leaf", Content: []byte{0, 1, 2, 3}},
			{Tag: int32(heap.FieldTagOrigin), Scope: "leaf", Content: []byte{0, 1, 2}},
		},
	}
	if _, ok := HeapPoint(tree, 0); ok {
		t.Error("HeapPoint() on unknown tag returned ok=true")
	}
	if _, ok := HeapPointForTag(tree, 0, int32(heap.FieldTagOrigin)); ok {
		t.Error("HeapPointForTag() on wrong tag returned ok=true")
	}
	if _, ok := HeapPoint(tree, 1); ok {
		t.Error("HeapPoint() on bad payload length returned ok=true")
	}
}

func TestFindPointChild(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 1, Scope: "open", Parent: -1, Children: []int{1, 2}},
			{Tag: int32(heap.FieldTagObjFlags), Scope: "leaf", Parent: 0, Content: []byte{0}},
			{Tag: int32(heap.FieldTagMinPaneSize), Scope: "leaf", Parent: 0, Content: []byte{0, 10, 0, 20}},
		},
	}
	got, ok := FindPointChild(tree, 0, int32(heap.FieldTagMinPaneSize))
	if !ok {
		t.Fatal("FindPointChild() ok = false, want true")
	}
	want := PointValue{Tag: int32(heap.FieldTagMinPaneSize), X: 10, Y: 20}
	if got != want {
		t.Fatalf("FindPointChild() = %+v, want %+v", got, want)
	}
}

func TestHeapPointCorpusCoverage(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalPoint := 0
	decodedPoint := 0
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
				if n.Scope != "leaf" || !IsHeapPointTag(n.Tag) {
					continue
				}
				totalPoint++
				countsByTag[n.Tag]++
				if _, ok := HeapPoint(tree, i); ok {
					decodedPoint++
				} else {
					undecodedByTag[n.Tag]++
				}
			}
		}
	}
	if exercised == 0 {
		t.Skip("no corpus VI yielded an FPHb or BDHb tree")
	}
	if totalPoint == 0 {
		t.Fatal("found 0 known point leaves")
	}
	if decodedPoint != totalPoint {
		t.Fatalf("HeapPoint decoded %d/%d known point leaves; undecoded=%v", decodedPoint, totalPoint, undecodedByTag)
	}
	t.Logf("HeapPoint: %d/%d leaves decoded across %d trees; tags=%v",
		decodedPoint, totalPoint, exercised, countsByTag)
}
