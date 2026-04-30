package lvvi

import "github.com/CWBudde/lvrsrc/internal/codecs/heap"

// HeapCompressedWireTable returns the raw payload of an
// OF__compressedWireTable leaf at tree.Nodes[nodeIdx]. The byte layout
// is undocumented — pylabview's `LVheap.py` carries the enum number
// only, with no decoder — so callers receive the bytes verbatim and
// decide how to interpret them. Phase 13 uses this as a presence
// signal ("there are wires here") and a stable round-trip fixture for
// current and future decoder work.
//
// Returns ok=false when:
//   - nodeIdx is out of range,
//   - the node's tag is not FieldTagCompressedWireTable (456), or
//   - the node's content is empty.
func HeapCompressedWireTable(tree HeapTree, nodeIdx int) ([]byte, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return nil, false
	}
	n := tree.Nodes[nodeIdx]
	if n.Tag != int32(heap.FieldTagCompressedWireTable) {
		return nil, false
	}
	if len(n.Content) == 0 {
		return nil, false
	}
	return n.Content, true
}

// CountCompressedWireTables walks tree.Nodes and reports how many
// OF__compressedWireTable leaves carry non-empty payloads. Used by
// the scene projection to surface a "wires present but topology not
// yet decoded" annotation when the BD heap has at least one chunk.
func CountCompressedWireTables(tree HeapTree) int {
	count := 0
	for _, n := range tree.Nodes {
		if n.Scope != "leaf" {
			continue
		}
		if n.Tag != int32(heap.FieldTagCompressedWireTable) {
			continue
		}
		if len(n.Content) == 0 {
			continue
		}
		count++
	}
	return count
}
