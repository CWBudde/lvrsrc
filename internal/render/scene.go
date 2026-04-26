package render

import (
	"math"
	"strings"

	"github.com/CWBudde/lvrsrc/internal/codecs/heap"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

// View identifies which heap surface the scene represents.
type View string

const (
	ViewFrontPanel   View = "front-panel"
	ViewBlockDiagram View = "block-diagram"
)

// NodeKind identifies the rendering primitive a scene node represents.
type NodeKind string

const (
	NodeKindGroup    NodeKind = "group"
	NodeKindBox      NodeKind = "box"
	NodeKindLabel    NodeKind = "label"
	NodeKindPort     NodeKind = "port"
	NodeKindTerminal NodeKind = "terminal"
)

// Rect is an axis-aligned logical rectangle in scene coordinates.
type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// Point is one logical coordinate in scene space.
type Point struct {
	X float64
	Y float64
}

// Node is one scene-graph entry shared by the demo and future CLI
// renderers. Parent/Children form the containment graph; Bounds and Z
// carry the geometry needed for SVG or canvas output.
type Node struct {
	Kind        NodeKind
	Label       string
	Bounds      Rect
	Parent      int
	Children    []int
	Z           int
	Placeholder bool
	HeapIndex   int
	Tag         int32
	Path        string
	LeafCount   int
	ContentSize int
}

// Wire is the future hook for explicit block-diagram routing. The
// initial Phase 11.1 projection does not emit wires yet, but the scene
// model reserves the type so demo and CLI output can share it later.
type Wire struct {
	From   int
	To     int
	Z      int
	Points []Point
	Label  string
}

// Scene is the renderer-neutral representation emitted from decoded
// front-panel / block-diagram heaps.
type Scene struct {
	View     View
	ViewBox  Rect
	Nodes    []Node
	Wires    []Wire
	Roots    []int
	Warnings []string
}

const (
	sceneMarginX        = 24.0
	sceneMarginY        = 24.0
	sceneRootGapY       = 20.0
	sceneGroupPadX      = 16.0
	sceneGroupPadY      = 12.0
	sceneGroupIndentX   = 18.0
	sceneHeaderHeight   = 22.0
	sceneChildGapY      = 10.0
	sceneTitleGapY      = 8.0
	sceneMinGroupW      = 180.0
	sceneMinLabelW      = 96.0
	sceneLabelH         = 18.0
	sceneCharW          = 7.0
	canvasNodeThreshold = 180
	canvasAreaThreshold = 1200 * 1200
)

type layoutKind string

const (
	layoutKindGroup layoutKind = "group"
	layoutKindLabel layoutKind = "label"
)

type layoutItem struct {
	kind        layoutKind
	heapIndex   int
	tag         int32
	label       string
	placeholder bool
	path        string
	leafCount   int
	contentSize int
	children    []*layoutItem
	width       float64
	height      float64
	// bounds is the decoded OF__bounds rectangle for this group, when
	// present. nil means the heuristic layout pass owns position and
	// size for this item.
	bounds *lvvi.Bounds
}

// ProjectHeapTree converts a decoded heap tree into a renderer-neutral
// scene graph with explicit logical bounds and containment.
//
// Phase 11.1: items whose heap node carries a decoded OF__bounds child
// are positioned and sized from those coordinates. Items without
// decoded bounds keep the prior heuristic (vertical stack indented by
// depth). The viewBox is sized to encompass both kinds.
func ProjectHeapTree(tree lvvi.HeapTree, view View) Scene {
	scene := Scene{View: view}
	if len(tree.Roots) == 0 || len(tree.Nodes) == 0 {
		return scene
	}

	items := make([]*layoutItem, 0, len(tree.Roots))
	for _, ri := range tree.Roots {
		item := buildLayoutItem(tree, ri, "")
		if item == nil {
			continue
		}
		measureLayout(item)
		items = append(items, item)
	}
	if len(items) == 0 {
		return scene
	}

	// Roots without decoded bounds stack top-to-bottom from the
	// margin; roots with bounds are placed at their decoded coords.
	heuristicY := sceneMarginY
	for _, item := range items {
		var rootIdx int
		if item.bounds != nil {
			x := float64(item.bounds.Left) + sceneMarginX
			y := float64(item.bounds.Top) + sceneMarginY
			rootIdx = placeLayoutItem(&scene, item, x, y, -1, 0)
		} else {
			rootIdx = placeLayoutItem(&scene, item, sceneMarginX, heuristicY, -1, 0)
			heuristicY += item.height + sceneRootGapY
		}
		scene.Roots = append(scene.Roots, rootIdx)
	}

	scene.ViewBox = computeViewBox(scene, heuristicY)
	scene.Warnings = sceneWarnings(scene, items)

	return scene
}

// computeViewBox sizes the scene viewbox so it encompasses every
// emitted node's bounds plus a fixed margin. heuristicEnd is the
// y-extent the heuristic stack would have reached if every root used
// it; we keep this floor so a heuristic-only scene retains its prior
// layout shape.
func computeViewBox(scene Scene, heuristicEnd float64) Rect {
	maxX := sceneMarginX * 2
	maxY := math.Max(heuristicEnd+sceneMarginY, sceneMarginY*2)
	for _, n := range scene.Nodes {
		right := n.Bounds.X + n.Bounds.Width + sceneMarginX
		bottom := n.Bounds.Y + n.Bounds.Height + sceneMarginY
		if right > maxX {
			maxX = right
		}
		if bottom > maxY {
			maxY = bottom
		}
	}
	return Rect{X: 0, Y: 0, Width: maxX, Height: maxY}
}

// FrontPanelScene projects the decoded FPHb tree from the lvvi model
// into a renderer-neutral scene graph.
func FrontPanelScene(m *lvvi.Model) (Scene, bool) {
	if m == nil {
		return Scene{}, false
	}
	tree, ok := m.FrontPanel()
	if !ok {
		return Scene{}, false
	}
	return ProjectHeapTree(tree, ViewFrontPanel), true
}

// BlockDiagramScene projects the decoded BDHb tree from the lvvi model
// into a renderer-neutral scene graph.
func BlockDiagramScene(m *lvvi.Model) (Scene, bool) {
	if m == nil {
		return Scene{}, false
	}
	tree, ok := m.BlockDiagram()
	if !ok {
		return Scene{}, false
	}
	return ProjectHeapTree(tree, ViewBlockDiagram), true
}

// PreferCanvas reports whether a scene is large enough that a canvas
// render path is likely to be more responsive than a DOM-heavy SVG view.
func PreferCanvas(scene Scene) bool {
	area := scene.ViewBox.Width * scene.ViewBox.Height
	return len(scene.Nodes) >= canvasNodeThreshold || area >= canvasAreaThreshold
}

func buildLayoutItem(tree lvvi.HeapTree, idx int, parentPath string) *layoutItem {
	if idx < 0 || idx >= len(tree.Nodes) {
		return nil
	}
	n := tree.Nodes[idx]
	label := lvvi.HeapTagName(n)
	path := label
	if parentPath != "" {
		path = parentPath + "/" + label
	}
	placeholder := strings.Contains(label, "(")

	switch n.Scope {
	case "open":
		item := &layoutItem{
			kind:        layoutKindGroup,
			heapIndex:   idx,
			tag:         n.Tag,
			label:       label,
			placeholder: placeholder,
			path:        path,
			contentSize: len(n.Content),
		}
		if b, ok := lvvi.FindBoundsChild(tree, idx); ok {
			b := b
			item.bounds = &b
		}
		for _, ci := range n.Children {
			if ci < 0 || ci >= len(tree.Nodes) {
				continue
			}
			child := tree.Nodes[ci]
			// OF__bounds leaves carry the parent's geometry only;
			// they do not represent visible content, so once we've
			// promoted the bounds onto the parent we drop the leaf
			// from the layout to avoid the redundant text label.
			if child.Scope == "leaf" && child.Tag == int32(heap.FieldTagBounds) {
				continue
			}
			switch child.Scope {
			case "open":
				if childItem := buildLayoutItem(tree, ci, path); childItem != nil {
					item.children = append(item.children, childItem)
				}
			case "leaf":
				item.leafCount++
				if childItem := buildLayoutItem(tree, ci, path); childItem != nil {
					item.children = append(item.children, childItem)
				}
			}
		}
		return item
	case "leaf":
		return &layoutItem{
			kind:        layoutKindLabel,
			heapIndex:   idx,
			tag:         n.Tag,
			label:       label,
			placeholder: placeholder,
			path:        path,
			contentSize: len(n.Content),
		}
	default:
		return nil
	}
}

func measureLayout(item *layoutItem) {
	if item == nil {
		return
	}
	labelW := textWidth(item.label)
	switch item.kind {
	case layoutKindLabel:
		item.width = math.Max(sceneMinLabelW, labelW+12)
		item.height = sceneLabelH
	case layoutKindGroup:
		// Recurse into children regardless so they pick up their
		// own decoded bounds before we measure the parent.
		for _, child := range item.children {
			measureLayout(child)
		}
		if item.bounds != nil {
			// Decoded LabVIEW geometry takes precedence — render the
			// group at the exact pixel size LabVIEW recorded, even if
			// children would heuristically need more room. Visible
			// overflow in this case is preferable to silently widening
			// the box (which would misrepresent the source layout).
			w := float64(item.bounds.Width())
			h := float64(item.bounds.Height())
			if w < 1 {
				w = 1
			}
			if h < 1 {
				h = 1
			}
			item.width = w
			item.height = h
			return
		}
		contentW := 0.0
		contentH := 0.0
		for i, child := range item.children {
			contentW = math.Max(contentW, child.width)
			contentH += child.height
			if i > 0 {
				contentH += sceneChildGapY
			}
		}
		item.width = math.Max(sceneMinGroupW, labelW+sceneGroupPadX*2)
		if len(item.children) > 0 {
			item.width = math.Max(item.width, sceneGroupPadX*2+sceneGroupIndentX+contentW)
		}
		item.height = sceneGroupPadY + sceneHeaderHeight + sceneGroupPadY
		if len(item.children) > 0 {
			item.height += sceneTitleGapY + contentH
		}
	}
}

func placeLayoutItem(scene *Scene, item *layoutItem, x, y float64, parent, depth int) int {
	switch item.kind {
	case layoutKindLabel:
		idx := appendNode(scene, Node{
			Kind:        NodeKindLabel,
			Label:       item.label,
			Bounds:      Rect{X: x, Y: y, Width: item.width, Height: item.height},
			Parent:      parent,
			Z:           depth*10 + 2,
			Placeholder: item.placeholder,
			HeapIndex:   item.heapIndex,
			Tag:         item.tag,
			Path:        item.path,
			ContentSize: item.contentSize,
		})
		if parent >= 0 {
			scene.Nodes[parent].Children = append(scene.Nodes[parent].Children, idx)
		}
		return idx

	case layoutKindGroup:
		groupIdx := appendNode(scene, Node{
			Kind:        NodeKindGroup,
			Label:       item.label,
			Bounds:      Rect{X: x, Y: y, Width: item.width, Height: item.height},
			Parent:      parent,
			Z:           depth * 10,
			Placeholder: item.placeholder,
			HeapIndex:   item.heapIndex,
			Tag:         item.tag,
			Path:        item.path,
			LeafCount:   item.leafCount,
			ContentSize: item.contentSize,
		})
		if parent >= 0 {
			scene.Nodes[parent].Children = append(scene.Nodes[parent].Children, groupIdx)
		}

		boxIdx := appendNode(scene, Node{
			Kind:        NodeKindBox,
			Label:       item.label,
			Bounds:      Rect{X: x, Y: y, Width: item.width, Height: item.height},
			Parent:      groupIdx,
			Z:           depth*10 + 1,
			Placeholder: item.placeholder,
			HeapIndex:   item.heapIndex,
			Tag:         item.tag,
			Path:        item.path,
			LeafCount:   item.leafCount,
			ContentSize: item.contentSize,
		})
		scene.Nodes[groupIdx].Children = append(scene.Nodes[groupIdx].Children, boxIdx)

		// Title label is inset by sceneGroupPadX on both sides; for
		// groups whose decoded bounds are smaller than the inset it
		// would otherwise come out negative, so clamp to ≥ 0.
		titleW := math.Max(0, item.width-sceneGroupPadX*2)
		titleIdx := appendNode(scene, Node{
			Kind:        NodeKindLabel,
			Label:       item.label,
			Bounds:      Rect{X: x + sceneGroupPadX, Y: y + sceneGroupPadY, Width: titleW, Height: sceneHeaderHeight},
			Parent:      groupIdx,
			Z:           depth*10 + 2,
			Placeholder: item.placeholder,
			HeapIndex:   item.heapIndex,
			Tag:         item.tag,
			Path:        item.path,
			LeafCount:   item.leafCount,
			ContentSize: item.contentSize,
		})
		scene.Nodes[groupIdx].Children = append(scene.Nodes[groupIdx].Children, titleIdx)

		cy := y + sceneGroupPadY + sceneHeaderHeight + sceneTitleGapY
		for _, child := range item.children {
			if child.kind == layoutKindGroup && child.bounds != nil {
				// Nested controls with their own decoded bounds use
				// scene-absolute coordinates rather than stacking
				// below the parent's title. Origin shift mirrors
				// ProjectHeapTree so the math is consistent at every
				// level.
				cx := float64(child.bounds.Left) + sceneMarginX
				cyAbs := float64(child.bounds.Top) + sceneMarginY
				placeLayoutItem(scene, child, cx, cyAbs, groupIdx, depth+1)
				continue
			}
			placeLayoutItem(scene, child, x+sceneGroupPadX+sceneGroupIndentX, cy, groupIdx, depth+1)
			cy += child.height + sceneChildGapY
		}
		return groupIdx

	default:
		return -1
	}
}

func appendNode(scene *Scene, node Node) int {
	idx := len(scene.Nodes)
	scene.Nodes = append(scene.Nodes, node)
	return idx
}

func textWidth(s string) float64 {
	runes := 0
	for range s {
		runes++
	}
	return float64(runes) * sceneCharW
}

func sceneWarnings(scene Scene, roots []*layoutItem) []string {
	var warnings []string
	allRootsHaveBounds := len(roots) > 0
	for _, item := range roots {
		if item.bounds == nil {
			allRootsHaveBounds = false
			break
		}
	}
	if !allRootsHaveBounds {
		warnings = append(warnings, "Layout is heuristic: some positions and sizes are derived from heap structure rather than decoded LabVIEW geometry.")
	}

	placeholders := 0
	for _, node := range scene.Nodes {
		if node.Placeholder {
			placeholders++
		}
	}
	if placeholders > 0 {
		warnings = append(warnings, "Placeholder nodes are present for unresolved object classes or fields.")
	}
	if scene.View == ViewBlockDiagram {
		warnings = append(warnings, "Block-diagram wire routing and terminal positions are not rendered yet.")
	}
	return warnings
}
