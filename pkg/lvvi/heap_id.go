package lvvi

// Heap-node attribute IDs observed in BDHb / FPHb headers. Pylabview
// keeps these as bare negative integers; the names below come from
// empirical correlation with corpus VIs (notably the BD wire-identity
// investigation that grounded the OF__termList → -3 ID mapping).
//
// The numeric values are treated as the source of truth and exported as
// constants so other lvvi accessors can refer to them by name without
// re-encoding the magic number.
const (
	// HeapAttrObjectID is the ID assigned to a heap node by the encoder
	// (pylabview emits this as a per-section monotonic counter). Other
	// nodes reference it by storing the same value, e.g. the source and
	// sink heap-object IDs inside a wire's termList.
	HeapAttrObjectID int32 = -3
	// HeapAttrIndex is the secondary attribute the encoder co-emits
	// alongside HeapAttrObjectID on canonical OPEN-scope nodes. Its
	// exact meaning is not yet decoded; for indexing purposes we treat
	// its presence as the marker that distinguishes a definition from
	// the forward-declaration LEAF stubs that carry only HeapAttrObjectID.
	HeapAttrIndex int32 = -2
	// HeapAttrChildCount is the attribute carried by container nodes
	// such as OF__termList that records how many children follow.
	HeapAttrChildCount int32 = -5
)

// HeapObjectID is the integer identifier carried by HeapAttrObjectID on
// a heap node. IDs are unique within a single decoded heap section
// (FPHb / BDHb) and are used for cross-references such as wire-to-
// terminal mapping where storing the heap-tree index would not survive
// a re-decode.
type HeapObjectID int32

// HeapNodeID returns the HeapObjectID declared by attribute -3 on
// tree.Nodes[idx], or (0, false) when the index is out of range or the
// node has no -3 attribute.
func HeapNodeID(tree HeapTree, idx int) (HeapObjectID, bool) {
	if idx < 0 || idx >= len(tree.Nodes) {
		return 0, false
	}
	for _, a := range tree.Nodes[idx].Attributes {
		if a.ID == HeapAttrObjectID {
			return HeapObjectID(a.Value), true
		}
	}
	return 0, false
}

// HeapObjectIndex resolves a HeapObjectID to the heap-tree index of its
// canonical declaration. The canonical declaration is the OPEN-scope
// node that carries both HeapAttrIndex and HeapAttrObjectID. Forward-
// declaration LEAF stubs (which carry only HeapAttrObjectID) are not
// recorded as the canonical entry but are still tracked in the dupes
// map returned by BuildHeapObjectIndex for diagnostics.
type HeapObjectIndex map[HeapObjectID]int

// BuildHeapObjectIndex scans tree.Nodes once and returns:
//   - the canonical HeapObjectIndex (one entry per ID, preferring
//     OPEN-scope nodes that also carry HeapAttrIndex),
//   - a duplicates map listing every node index that also declares the
//     same ID — useful for diagnosing unexpected collisions and for
//     surfacing the LEAF stubs the encoder emitted alongside each
//     canonical OPEN node.
//
// When the corpus has only LEAF declarations for an ID (no OPEN
// canonical), the first LEAF wins and is recorded as canonical so
// callers can still resolve the reference.
func BuildHeapObjectIndex(tree HeapTree) (HeapObjectIndex, map[HeapObjectID][]int) {
	idx := make(HeapObjectIndex)
	dupes := make(map[HeapObjectID][]int)
	for i, n := range tree.Nodes {
		id, ok := nodeIDAndIndex(n)
		if !ok {
			continue
		}
		dupes[id] = append(dupes[id], i)
		existing, seen := idx[id]
		if !seen {
			idx[id] = i
			continue
		}
		// Prefer an OPEN canonical (HeapAttrIndex present) over a LEAF
		// stub. If the existing entry is already canonical, keep it.
		if isCanonicalDeclaration(tree.Nodes[existing]) {
			continue
		}
		if isCanonicalDeclaration(n) {
			idx[id] = i
		}
	}
	// Collapse single-entry dupes (no actual duplication) so the
	// returned map only flags real multi-declaration cases.
	for id, list := range dupes {
		if len(list) <= 1 {
			delete(dupes, id)
		}
	}
	return idx, dupes
}

// isCanonicalDeclaration reports whether n looks like the encoder's
// canonical OPEN-scope declaration of its HeapObjectID — i.e. it is in
// open scope and also carries HeapAttrIndex.
func isCanonicalDeclaration(n HeapNode) bool {
	if n.Scope != "open" {
		return false
	}
	hasIndex := false
	for _, a := range n.Attributes {
		if a.ID == HeapAttrIndex {
			hasIndex = true
			break
		}
	}
	return hasIndex
}

// nodeIDAndIndex returns the HeapObjectID declared by node n, if any.
func nodeIDAndIndex(n HeapNode) (HeapObjectID, bool) {
	for _, a := range n.Attributes {
		if a.ID == HeapAttrObjectID {
			return HeapObjectID(a.Value), true
		}
	}
	return 0, false
}
