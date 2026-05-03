package lvvi

import (
	"reflect"
	"testing"
)

func TestHeapNodeID(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{
		{Attributes: []HeapAttribute{{ID: HeapAttrObjectID, Value: 42}}},
		{Attributes: []HeapAttribute{{ID: HeapAttrIndex, Value: 7}}},
		{Attributes: nil},
	}}
	if got, ok := HeapNodeID(tree, 0); !ok || got != 42 {
		t.Fatalf("HeapNodeID(0) = (%d, %v), want (42, true)", got, ok)
	}
	if _, ok := HeapNodeID(tree, 1); ok {
		t.Fatalf("HeapNodeID(1): node has no -3 attr, want ok=false")
	}
	if _, ok := HeapNodeID(tree, 2); ok {
		t.Fatalf("HeapNodeID(2): node has no attrs, want ok=false")
	}
	if _, ok := HeapNodeID(tree, -1); ok {
		t.Fatalf("HeapNodeID(-1): out of range, want ok=false")
	}
	if _, ok := HeapNodeID(tree, 99); ok {
		t.Fatalf("HeapNodeID(99): out of range, want ok=false")
	}
}

// BuildHeapObjectIndex must prefer the OPEN canonical declaration
// (which carries both -2 and -3) over forward-declaration LEAF stubs
// that carry only -3.
func TestBuildHeapObjectIndexPrefersCanonicalOverLeafStub(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{
		// LEAF stub appears first in stream order.
		{Scope: "leaf", Attributes: []HeapAttribute{{ID: HeapAttrObjectID, Value: 51}}},
		// Canonical OPEN declaration appears later.
		{Scope: "open", Attributes: []HeapAttribute{{ID: HeapAttrIndex, Value: 21}, {ID: HeapAttrObjectID, Value: 51}}},
	}}
	idx, dupes := BuildHeapObjectIndex(tree)
	if got, ok := idx[51]; !ok || got != 1 {
		t.Fatalf("idx[51] = (%d, %v), want (1, true)", got, ok)
	}
	wantDupes := map[HeapObjectID][]int{51: {0, 1}}
	if !reflect.DeepEqual(dupes, wantDupes) {
		t.Fatalf("dupes = %v, want %v", dupes, wantDupes)
	}
}

// When OPEN canonical comes first, the LEAF stub must not displace it.
func TestBuildHeapObjectIndexCanonicalFirstStaysCanonical(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{
		{Scope: "open", Attributes: []HeapAttribute{{ID: HeapAttrIndex, Value: 21}, {ID: HeapAttrObjectID, Value: 51}}},
		{Scope: "leaf", Attributes: []HeapAttribute{{ID: HeapAttrObjectID, Value: 51}}},
	}}
	idx, _ := BuildHeapObjectIndex(tree)
	if got, ok := idx[51]; !ok || got != 0 {
		t.Fatalf("idx[51] = (%d, %v), want (0, true)", got, ok)
	}
}

// IDs whose only declaration is a LEAF (no OPEN canonical observed)
// must still be resolvable so callers can recover when the corpus
// surprises us.
func TestBuildHeapObjectIndexLeafOnlyStillResolves(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{
		{Scope: "leaf", Attributes: []HeapAttribute{{ID: HeapAttrObjectID, Value: 99}}},
	}}
	idx, dupes := BuildHeapObjectIndex(tree)
	if got, ok := idx[99]; !ok || got != 0 {
		t.Fatalf("idx[99] = (%d, %v), want (0, true)", got, ok)
	}
	if len(dupes) != 0 {
		t.Fatalf("dupes for single-entry ID should be empty, got %v", dupes)
	}
}

// Nodes without HeapAttrObjectID are ignored entirely.
func TestBuildHeapObjectIndexIgnoresNodesWithoutObjectID(t *testing.T) {
	tree := HeapTree{Nodes: []HeapNode{
		{Scope: "open", Attributes: []HeapAttribute{{ID: HeapAttrIndex, Value: 21}}},
		{Scope: "leaf", Attributes: nil},
	}}
	idx, dupes := BuildHeapObjectIndex(tree)
	if len(idx) != 0 {
		t.Fatalf("expected empty index, got %v", idx)
	}
	if len(dupes) != 0 {
		t.Fatalf("expected empty dupes, got %v", dupes)
	}
}
