package lvvi

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// Synthetic two-terminal wire: source ID 51, sink ID 95. The wire's
// arrayElement parent owns one OF__termList container with two LEAF
// arrayElement children carrying the IDs. Two canonical OPEN
// arrayElements declare those IDs elsewhere in the tree.
func TestWireTerminalIDsTwoTerminalNetwork(t *testing.T) {
	tree := buildSyntheticWireTree(t, []HeapObjectID{51, 95})
	wireIdx := findCompressedWire(t, tree)
	ids, ok := WireTerminalIDs(tree, wireIdx)
	if !ok {
		t.Fatalf("WireTerminalIDs ok=false, want true")
	}
	if !reflect.DeepEqual(ids, []HeapObjectID{51, 95}) {
		t.Fatalf("ids = %v, want [51 95]", ids)
	}
}

func TestWireTerminalsResolvesToCanonicalDeclarations(t *testing.T) {
	tree := buildSyntheticWireTree(t, []HeapObjectID{51, 95, 109})
	wireIdx := findCompressedWire(t, tree)
	idx, _ := BuildHeapObjectIndex(tree)
	terms, ok := WireTerminals(tree, idx, wireIdx)
	if !ok {
		t.Fatalf("WireTerminals ok=false, want true")
	}
	if len(terms) != 3 {
		t.Fatalf("len(terms) = %d, want 3", len(terms))
	}
	for i, ti := range terms {
		if ti < 0 {
			t.Fatalf("terms[%d] = -1, expected canonical declaration to resolve", i)
		}
		if !isCanonicalDeclaration(tree.Nodes[ti]) {
			t.Fatalf("terms[%d] -> node %d is not canonical OPEN declaration", i, ti)
		}
	}
}

// When an ID has no canonical declaration anywhere, the resolver must
// return -1 in that slot rather than silently dropping the reference.
func TestWireTerminalsLeavesUnknownIDAsMinusOne(t *testing.T) {
	tree := buildSyntheticWireTree(t, []HeapObjectID{51, 999})
	wireIdx := findCompressedWire(t, tree)
	idx, _ := BuildHeapObjectIndex(tree)
	// Drop ID 999 from the index to simulate a dangling reference.
	delete(idx, 999)
	terms, ok := WireTerminals(tree, idx, wireIdx)
	if !ok {
		t.Fatalf("WireTerminals ok=false, want true")
	}
	if len(terms) != 2 {
		t.Fatalf("len(terms) = %d, want 2", len(terms))
	}
	if terms[1] != -1 {
		t.Fatalf("terms[1] = %d, want -1 for missing ID", terms[1])
	}
}

func TestWireTerminalIDsRejectsNonWireNode(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{
		{Tag: 1, Scope: "leaf", Content: []byte{0x01}},
	}}
	if _, ok := WireTerminalIDs(tree, 0); ok {
		t.Fatalf("WireTerminalIDs on non-wire node returned ok=true")
	}
}

// Corpus-level guarantee: the bug we hit during the Phase 14.2
// investigation was that every wire in a complex VI rendered with the
// same source/sink terminal pair. Once the wire-terminal index lands,
// every wire in ndjson-parser.vi and format-string.vi must resolve to
// distinct (source-ID, sink-ID) pairs (or unique multi-sink tuples for
// tree networks).
func TestWireTerminalsCorpusUniquePerWire(t *testing.T) {
	for _, name := range []string{"format-string.vi", "ndjson-parser.vi", "reference-find-by-id.vi"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", "corpus", name)
			file, err := lvrsrc.Open(path, lvrsrc.OpenOptions{})
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			model, _ := DecodeKnownResources(file)
			tree, ok := model.BlockDiagram()
			if !ok {
				t.Fatalf("no BD heap in %s", name)
			}
			idx, _ := BuildHeapObjectIndex(tree)
			seen := map[string]int{}
			wires := 0
			for i, n := range tree.Nodes {
				if n.Scope != "leaf" || n.Tag != int32(heap.FieldTagCompressedWireTable) || len(n.Content) == 0 {
					continue
				}
				ids, ok := WireTerminalIDs(tree, i)
				if !ok {
					continue
				}
				wires++
				key := tupleKey(ids)
				seen[key]++
				terms, _ := WireTerminals(tree, idx, i)
				for k, ti := range terms {
					if ti == -1 {
						t.Errorf("wire %d term %d (id=%d) failed to resolve", i, k, ids[k])
					}
				}
			}
			if wires < 2 {
				t.Skipf("%s has fewer than 2 wires (%d), skipping uniqueness check", name, wires)
			}
			for key, count := range seen {
				if count > 1 {
					t.Errorf("%s: %d wires share termList tuple %s — wire identity collapsed", name, count, key)
				}
			}
		})
	}
}

func tupleKey(ids []HeapObjectID) string {
	out := make([]byte, 0, len(ids)*4)
	for _, id := range ids {
		out = append(out, byte(id>>24), byte(id>>16), byte(id>>8), byte(id))
	}
	return string(out)
}

// buildSyntheticWireTree constructs a minimal HeapTree containing one
// compressedWireTable chunk whose termList references the supplied IDs,
// plus one canonical OPEN arrayElement declaration per ID elsewhere in
// the tree. Returned tree mirrors the structural pattern observed in
// real BD heaps.
func buildSyntheticWireTree(t *testing.T, ids []HeapObjectID) HeapTree {
	t.Helper()
	if len(ids) < 2 {
		t.Fatalf("buildSyntheticWireTree requires at least 2 IDs, got %d", len(ids))
	}
	var nodes []HeapNode
	add := func(n HeapNode) int {
		nodes = append(nodes, n)
		return len(nodes) - 1
	}
	// Canonical OPEN declarations.
	declIdx := make(map[HeapObjectID]int, len(ids))
	for _, id := range ids {
		declIdx[id] = add(HeapNode{
			Scope: "open",
			Tag:   -6,
			Attributes: []HeapAttribute{
				{ID: HeapAttrIndex, Value: 21},
				{ID: HeapAttrObjectID, Value: int32(id)},
			},
			Parent: -1,
		})
	}
	// Wire-bearing arrayElement parent.
	wireParentIdx := add(HeapNode{
		Scope:    "open",
		Tag:      -6,
		Attributes: []HeapAttribute{{ID: HeapAttrIndex, Value: 23}, {ID: HeapAttrObjectID, Value: 999}},
		Parent:   -1,
		Children: nil,
	})
	// termList container.
	termListIdx := add(HeapNode{
		Scope:      "open",
		Tag:        tagTermListContext,
		Attributes: []HeapAttribute{{ID: HeapAttrChildCount, Value: int32(len(ids))}},
		Parent:     wireParentIdx,
	})
	// termList children: LEAF arrayElement per ID.
	leafChildren := make([]int, 0, len(ids))
	for _, id := range ids {
		i := add(HeapNode{
			Scope:      "leaf",
			Tag:        -6,
			Attributes: []HeapAttribute{{ID: HeapAttrObjectID, Value: int32(id)}},
			Parent:     termListIdx,
		})
		leafChildren = append(leafChildren, i)
	}
	// Compressed-wire chunk.
	wireIdx := add(HeapNode{
		Scope:   "leaf",
		Tag:     int32(heap.FieldTagCompressedWireTable),
		Content: []byte{0x02, 0x08},
		Parent:  wireParentIdx,
	})
	// Wire up children of the open containers.
	nodes[wireParentIdx].Children = []int{termListIdx, wireIdx}
	nodes[termListIdx].Children = leafChildren
	return HeapTree{Nodes: nodes, Roots: collectRoots(nodes)}
}

func findCompressedWire(t *testing.T, tree HeapTree) int {
	t.Helper()
	for i, n := range tree.Nodes {
		if n.Tag == int32(heap.FieldTagCompressedWireTable) && n.Scope == "leaf" {
			return i
		}
	}
	t.Fatalf("buildSyntheticWireTree did not produce a compressedWireTable node")
	return -1
}

func collectRoots(nodes []HeapNode) []int {
	var roots []int
	for i, n := range nodes {
		if n.Parent == -1 {
			roots = append(roots, i)
		}
	}
	return roots
}
