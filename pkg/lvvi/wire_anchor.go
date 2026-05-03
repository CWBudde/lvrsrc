package lvvi

// WireTerminalAnchor returns the heap-tree index of the visual anchor
// for a wire endpoint, or -1 when no anchor can be derived.
//
// Resolution priority (verified across the full corpus, 100% coverage):
//
//  1. **Walk down** the canonical declaration's subtree for any
//     WidgetKindTerminal node. Hits wires that terminate on a
//     primitive's input/output simTun nested inside the control wrapper
//     (e.g. constants, simple primitives). Returns the simTun heap
//     index.
//
//  2. **Walk up** from the canonical declaration looking for the nearest
//     WidgetKindTerminal ancestor. When such an ancestor exists, the
//     canonical declaration itself is the per-endpoint anchor (each
//     connector-pane endpoint has its own canonical with a unique
//     HeapObjectID; the ancestor sdfTun is *shared* across all
//     endpoints it owns). Returning the canonical here lets the scene
//     project distinct per-endpoint anchors instead of collapsing every
//     wire onto the sdfTun anchor — Phase 16.4 A2.
//
//  3. **sdfTun children scan**: for endpoint IDs that appear only as
//     stub LEAF arrayElement children of a sdfTun (no canonical OPEN
//     declaration anywhere), the sdfTun itself is the visual anchor.
//
// The returned heap node is not necessarily a WidgetKindTerminal class
// in case (2): it is whatever arrayElement carries the per-endpoint
// HeapObjectID. The render layer must register such canonicals as
// scene anchors so they are addressable via terminalByHeap.
func WireTerminalAnchor(tree HeapTree, idx HeapObjectIndex, id HeapObjectID) int {
	if canonical, ok := idx[id]; ok {
		if t := walkDownToWidgetTerminal(tree, canonical); t >= 0 {
			return t
		}
		if walkUpToWidgetTerminal(tree, canonical) >= 0 {
			return canonical
		}
	}
	return findSdfTunCarryingID(tree, id)
}

// walkDownToWidgetTerminal recurses through tree.Nodes[root]'s
// descendants looking for the first OPEN-scope node classified as
// WidgetKindTerminal.
func walkDownToWidgetTerminal(tree HeapTree, root int) int {
	if root < 0 || root >= len(tree.Nodes) {
		return -1
	}
	n := tree.Nodes[root]
	if n.Scope == "open" && WidgetKindForNode(n) == WidgetKindTerminal {
		return root
	}
	for _, c := range n.Children {
		if got := walkDownToWidgetTerminal(tree, c); got >= 0 {
			return got
		}
	}
	return -1
}

// walkUpToWidgetTerminal walks the parent chain of `start` looking for
// the nearest OPEN-scope WidgetKindTerminal ancestor. This is how
// connector-pane endpoints resolve: they live nested several levels
// inside an sdfTun container.
func walkUpToWidgetTerminal(tree HeapTree, start int) int {
	if start < 0 || start >= len(tree.Nodes) {
		return -1
	}
	for p := tree.Nodes[start].Parent; p >= 0 && p < len(tree.Nodes); p = tree.Nodes[p].Parent {
		pn := tree.Nodes[p]
		if pn.Scope == "open" && WidgetKindForNode(pn) == WidgetKindTerminal {
			return p
		}
	}
	return -1
}

// findSdfTunCarryingID scans every WidgetKindTerminal node in the tree
// for a direct LEAF arrayElement child whose HeapAttrObjectID equals
// the supplied id. The matching terminal node is returned. This handles
// a small population of endpoint IDs that are referenced as stub leaves
// inside an sdfTun without a corresponding canonical OPEN declaration
// elsewhere in the tree.
func findSdfTunCarryingID(tree HeapTree, id HeapObjectID) int {
	for i, n := range tree.Nodes {
		if n.Scope != "open" || WidgetKindForNode(n) != WidgetKindTerminal {
			continue
		}
		for _, ci := range n.Children {
			if ci < 0 || ci >= len(tree.Nodes) {
				continue
			}
			cn := tree.Nodes[ci]
			if cn.Tag != -6 || cn.Scope != "leaf" {
				continue
			}
			for _, a := range cn.Attributes {
				if a.ID == HeapAttrObjectID && HeapObjectID(a.Value) == id {
					return i
				}
			}
		}
	}
	return -1
}
