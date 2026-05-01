package lvvi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestHeapContainerHappyPath(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: int32(heap.FieldTagImage), Scope: "open", Parent: -1, ByteSize: 2, Children: []int{1, 2}},
			{Tag: int32(heap.ClassTagFontRun), Scope: "leaf", Parent: 0},
			{Tag: int32(heap.ClassTagFontRun), Scope: "leaf", Parent: 0},
		},
		Roots: []int{0},
	}
	got, ok := HeapContainer(tree, 0)
	if !ok {
		t.Fatal("HeapContainer() ok = false, want true")
	}
	if got.Tag != int32(heap.FieldTagImage) || got.ChildCount != 2 || got.ByteSize != 2 {
		t.Fatalf("HeapContainer() = %+v, want image container with 2 children", got)
	}
	if len(got.Children) != 2 || got.Children[0] != 1 || got.Children[1] != 2 {
		t.Fatalf("HeapContainer().Children = %v, want [1 2]", got.Children)
	}

	got.Children[0] = 99
	if tree.Nodes[0].Children[0] != 1 {
		t.Fatal("HeapContainer() returned aliased Children slice")
	}
}

func TestHeapContainerRejectsWrongTagAndScope(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: int32(heap.FieldTagObjFlags), Scope: "open", Parent: -1},
			{Tag: int32(heap.FieldTagImage), Scope: "leaf", Parent: -1},
			{Tag: int32(heap.FieldTagImage), Scope: "close", Parent: -1},
		},
	}
	if _, ok := HeapContainer(tree, 0); ok {
		t.Error("HeapContainer() on non-container tag returned ok=true")
	}
	if _, ok := HeapContainer(tree, 1); ok {
		t.Error("HeapContainer() on leaf container tag returned ok=true")
	}
	if _, ok := HeapContainer(tree, 2); ok {
		t.Error("HeapContainer() on close container tag returned ok=true")
	}
}

func TestFindContainerChild(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 1, Scope: "open", Parent: -1, Children: []int{1, 2}},
			{Tag: int32(heap.FieldTagObjFlags), Scope: "leaf", Parent: 0},
			{Tag: int32(heap.FieldTagNodeList), Scope: "open", Parent: 0, Children: []int{3}},
			{Tag: int32(heap.SystemTagArrayElement), Scope: "leaf", Parent: 2},
		},
	}
	got, ok := FindContainerChild(tree, 0, int32(heap.FieldTagNodeList))
	if !ok {
		t.Fatal("FindContainerChild() ok = false, want true")
	}
	if got.ChildCount != 1 || got.Children[0] != 3 {
		t.Fatalf("FindContainerChild() = %+v, want one child index 3", got)
	}
}

func TestHeapContainerCorpusCoverage(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalOpen := 0
	decodedOpen := 0
	closeCount := 0
	leafCount := 0
	exercised := 0
	countsByTag := map[int32]int{}
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
				if !IsHeapContainerTag(n.Tag) {
					continue
				}
				countsByTag[n.Tag]++
				switch n.Scope {
				case "open":
					totalOpen++
					if _, ok := HeapContainer(tree, i); ok {
						decodedOpen++
					}
				case "leaf":
					leafCount++
				case "close":
					closeCount++
				}
			}
		}
	}
	if exercised == 0 {
		t.Skip("no corpus VI yielded an FPHb or BDHb tree")
	}
	if totalOpen == 0 {
		t.Fatal("found 0 known open container nodes")
	}
	if decodedOpen != totalOpen {
		t.Fatalf("HeapContainer decoded %d/%d known open container nodes", decodedOpen, totalOpen)
	}
	t.Logf("HeapContainer: %d/%d open containers decoded across %d trees; close=%d leaf=%d tags=%v",
		decodedOpen, totalOpen, exercised, closeCount, leafCount, countsByTag)
}
