package lvvi

import "github.com/CWBudde/lvrsrc/internal/codecs/heap"

// ContainerValue is a structural projection of an open-scope heap field
// that groups child nodes but carries no direct payload bytes.
type ContainerValue struct {
	Tag        int32
	Children   []int
	ByteSize   int
	ChildCount int
}

// IsHeapContainerTag reports whether tag is a known structural heap
// field that primarily acts as a child-list/container in observed
// FPHb/BDHb trees.
func IsHeapContainerTag(tag int32) bool {
	switch tag {
	case int32(heap.FieldTagBrowseOptions),
		int32(heap.FieldTagDataNodeList),
		int32(heap.FieldTagDcoList),
		int32(heap.FieldTagDdoList),
		int32(heap.FieldTagDdoListList),
		int32(heap.FieldTagDiagramList),
		int32(heap.FieldTagFboxlineList),
		int32(heap.FieldTagFilterNodeList),
		int32(heap.FieldTagGrowTermsList),
		int32(heap.FieldTagHGrowNodeList),
		int32(heap.FieldTagImage),
		int32(heap.FieldTagKeyMappingList),
		int32(heap.FieldTagLsrDCOList),
		int32(heap.FieldTagNodeList),
		int32(heap.FieldTagOrderList),
		int32(heap.FieldTagPageList),
		int32(heap.FieldTagPartsList),
		int32(heap.FieldTagPermDCOList),
		int32(heap.FieldTagPrivDataList),
		int32(heap.FieldTagPropList),
		int32(heap.FieldTagRefList),
		int32(heap.FieldTagSeqLocDCOList),
		int32(heap.FieldTagSequenceList),
		int32(heap.FieldTagSignalList),
		int32(heap.FieldTagSrDCOList),
		int32(heap.FieldTagTermList):
		return true
	default:
		return false
	}
}

// HeapContainer decodes a known open-scope heap container at
// tree.Nodes[nodeIdx]. Returns ok=false for out-of-range indices,
// non-container tags, or non-open nodes.
func HeapContainer(tree HeapTree, nodeIdx int) (ContainerValue, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return ContainerValue{}, false
	}
	n := tree.Nodes[nodeIdx]
	if !IsHeapContainerTag(n.Tag) || n.Scope != "open" {
		return ContainerValue{}, false
	}
	return ContainerValue{
		Tag:        n.Tag,
		Children:   append([]int(nil), n.Children...),
		ByteSize:   n.ByteSize,
		ChildCount: len(n.Children),
	}, true
}

// HeapContainerForTag decodes a container only when it carries the
// requested known container tag.
func HeapContainerForTag(tree HeapTree, nodeIdx int, tag int32) (ContainerValue, bool) {
	if nodeIdx < 0 || nodeIdx >= len(tree.Nodes) {
		return ContainerValue{}, false
	}
	n := tree.Nodes[nodeIdx]
	if n.Tag != tag || !IsHeapContainerTag(tag) || n.Scope != "open" {
		return ContainerValue{}, false
	}
	return HeapContainer(tree, nodeIdx)
}

// FindContainerChild walks the children of tree.Nodes[parentIdx] and
// returns the first open-scope child carrying the requested container tag.
func FindContainerChild(tree HeapTree, parentIdx int, tag int32) (ContainerValue, bool) {
	if parentIdx < 0 || parentIdx >= len(tree.Nodes) || !IsHeapContainerTag(tag) {
		return ContainerValue{}, false
	}
	parent := tree.Nodes[parentIdx]
	for _, ci := range parent.Children {
		if ci < 0 || ci >= len(tree.Nodes) || tree.Nodes[ci].Tag != tag {
			continue
		}
		if v, ok := HeapContainerForTag(tree, ci, tag); ok {
			return v, true
		}
	}
	return ContainerValue{}, false
}
