package lvvi

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestHeapCompressedWireTableHappyPath(t *testing.T) {
	payload := []byte{0x0a, 0x00, 0x08, 0x05, 0x00, 0x03, 0x01, 0x00, 0x05, 0x00,
		0x03, 0x0d, 0x20, 0x1b, 0x4e, 0x04, 0x46, 0x96, 0x0d, 0xa1}
	tree := HeapTree{
		Nodes: []HeapNode{{
			Tag:     int32(heap.FieldTagCompressedWireTable),
			Scope:   "leaf",
			Parent:  -1,
			Content: payload,
		}},
		Roots: []int{0},
	}
	got, ok := HeapCompressedWireTable(tree, 0)
	if !ok {
		t.Fatal("HeapCompressedWireTable() ok = false, want true")
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("HeapCompressedWireTable() = %x, want %x", got, payload)
	}
}

func TestHeapCompressedWireTableRejectsWrongTag(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{{
			Tag:     int32(heap.FieldTagWireTable), // not the compressed form
			Scope:   "leaf",
			Content: []byte{0x02, 0x08},
		}},
	}
	if _, ok := HeapCompressedWireTable(tree, 0); ok {
		t.Error("HeapCompressedWireTable() on non-compressed tag returned ok=true")
	}
}

func TestHeapCompressedWireTableRejectsEmptyContent(t *testing.T) {
	// A 0-byte payload is meaningless — there must be at least one
	// compressed entry to qualify as a wire-table chunk.
	tree := HeapTree{
		Nodes: []HeapNode{{
			Tag:     int32(heap.FieldTagCompressedWireTable),
			Scope:   "leaf",
			Content: nil,
		}},
	}
	if _, ok := HeapCompressedWireTable(tree, 0); ok {
		t.Error("HeapCompressedWireTable() on empty content returned ok=true")
	}
}

func TestHeapCompressedWireTableRejectsOutOfRangeIndex(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{}, Roots: []int{}}
	for _, idx := range []int{-1, 0, 1, 100} {
		if _, ok := HeapCompressedWireTable(tree, idx); ok {
			t.Errorf("HeapCompressedWireTable(idx=%d) on empty tree returned ok=true", idx)
		}
	}
}

func TestCountCompressedWireTablesCountsLeaves(t *testing.T) {
	tree := HeapTree{
		Nodes: []HeapNode{
			{Tag: 1, Scope: "open", Parent: -1, Children: []int{1, 2, 3, 4}},
			{Tag: int32(heap.FieldTagCompressedWireTable), Scope: "leaf", Parent: 0, Content: []byte{0x02, 0x08}},
			{Tag: int32(heap.FieldTagBgColor), Scope: "leaf", Parent: 0},
			{Tag: int32(heap.FieldTagCompressedWireTable), Scope: "leaf", Parent: 0, Content: []byte{0x03, 0x08, 0x00, 0x1b}},
			// Empty content: not a real wire-table chunk → not counted.
			{Tag: int32(heap.FieldTagCompressedWireTable), Scope: "leaf", Parent: 0, Content: nil},
		},
	}
	if got, want := CountCompressedWireTables(tree), 2; got != want {
		t.Errorf("CountCompressedWireTables() = %d, want %d", got, want)
	}
}

func TestCountCompressedWireTablesEmptyTree(t *testing.T) {
	if got := CountCompressedWireTables(HeapTree{}); got != 0 {
		t.Errorf("CountCompressedWireTables(empty) = %d, want 0", got)
	}
}

// Sweep every BD heap in the corpus: every OF__compressedWireTable
// leaf should yield non-empty bytes through the accessor. This is a
// presence check only — it does not validate the compression scheme.
func TestHeapCompressedWireTableCorpusCoverage(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	totalLeaves := 0
	totalDecoded := 0
	totalChunks := 0
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
			totalChunks += CountCompressedWireTables(tree)
			for i, n := range tree.Nodes {
				if n.Tag != int32(heap.FieldTagCompressedWireTable) || n.Scope != "leaf" {
					continue
				}
				if len(n.Content) == 0 {
					continue
				}
				totalLeaves++
				if got, ok := HeapCompressedWireTable(tree, i); ok && len(got) == len(n.Content) {
					totalDecoded++
				}
			}
		}
	}
	if exercised == 0 {
		t.Skip("no corpus VI yielded a heap tree")
	}
	if totalLeaves == 0 {
		t.Skip("corpus contains no OF__compressedWireTable leaves")
	}
	if totalDecoded != totalLeaves {
		t.Fatalf("HeapCompressedWireTable returned %d/%d leaves verbatim",
			totalDecoded, totalLeaves)
	}
	if totalChunks != totalLeaves {
		t.Errorf("CountCompressedWireTables = %d, want %d (must agree with leaf count)",
			totalChunks, totalLeaves)
	}
	t.Logf("HeapCompressedWireTable: %d/%d leaves decoded across %d trees",
		totalDecoded, totalLeaves, exercised)
}
