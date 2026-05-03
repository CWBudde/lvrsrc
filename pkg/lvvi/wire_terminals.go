package lvvi

import "github.com/CWBudde/lvrsrc/internal/codecs/heap"

// tagTermListContext identifies the heap-tag value (268) that, in the
// BD wire-identity context, names an OF__termList container. The same
// numeric value names SL__udClassDDO in the ClassTag namespace; the
// resolver in HeapTagName cannot distinguish without a context stack,
// so wire-terminal code matches the integer directly and documents the
// collision rather than relying on the resolved name.
const tagTermListContext = int32(heap.FieldTagTermList)

// WireTerminalIDs returns the heap-object IDs of the terminals
// referenced by the OF__compressedWireTable chunk at tree.Nodes[wireIdx].
//
// The IDs are returned in the heap-stream order in which the wire's
// OF__termList container lists them. Empirically (verified across all
// rendered wires in the existing 40-fixture corpus) this is
// [source, sink, sink, …] — termList[0] is the network source, the
// remaining IDs are the sinks. Pylabview has no decoder for this
// surface, so the semantics may need a version gate once older corpus
// fixtures arrive.
//
// Returns ok=false when:
//   - wireIdx is out of range,
//   - the node is not an OF__compressedWireTable leaf with content,
//   - the wire's arrayElement parent has no OF__termList sibling,
//   - or the termList contains zero ID-bearing children.
func WireTerminalIDs(tree HeapTree, wireIdx int) ([]HeapObjectID, bool) {
	if _, ok := HeapCompressedWireTable(tree, wireIdx); !ok {
		return nil, false
	}
	parentIdx := tree.Nodes[wireIdx].Parent
	if parentIdx < 0 || parentIdx >= len(tree.Nodes) {
		return nil, false
	}
	parent := tree.Nodes[parentIdx]
	for _, c := range parent.Children {
		if c < 0 || c >= len(tree.Nodes) {
			continue
		}
		cn := tree.Nodes[c]
		if cn.Scope != "open" || cn.Tag != tagTermListContext {
			continue
		}
		ids := make([]HeapObjectID, 0, len(cn.Children))
		for _, leafIdx := range cn.Children {
			if leafIdx < 0 || leafIdx >= len(tree.Nodes) {
				continue
			}
			id, hasID := HeapNodeID(tree, leafIdx)
			if !hasID {
				continue
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			return nil, false
		}
		return ids, true
	}
	return nil, false
}

// WireTerminals resolves a wire's termList IDs to canonical heap-tree
// node indices using the supplied object index. Each result entry is
// either:
//   - a non-negative index into tree.Nodes pointing at the canonical
//     OPEN-scope arrayElement that wraps the terminal, or
//   - -1 when the ID is absent from the index (referenced terminal not
//     declared in this BD heap — should not happen in well-formed
//     corpus but is preserved as an explicit gap rather than silently
//     dropped so callers can warn).
//
// The slice positions correspond 1:1 with the IDs returned by
// WireTerminalIDs (i.e. index 0 is the source, indices 1..N are sinks).
func WireTerminals(tree HeapTree, idx HeapObjectIndex, wireIdx int) ([]int, bool) {
	ids, ok := WireTerminalIDs(tree, wireIdx)
	if !ok {
		return nil, false
	}
	out := make([]int, len(ids))
	for i, id := range ids {
		if node, ok := idx[id]; ok {
			out[i] = node
		} else {
			out[i] = -1
		}
	}
	return out, true
}
