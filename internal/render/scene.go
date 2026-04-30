package render

import (
	"fmt"
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
	NodeKindGroup NodeKind = "group"
	NodeKindBox   NodeKind = "box"
	NodeKindLabel NodeKind = "label"
	NodeKindPort  NodeKind = "port"
	// NodeKindTerminal is a flat anchor marker for BD tunnels and
	// front-panel terminals. It carries Bounds (the terminal's outer
	// rectangle) and Anchor (the connect-point wires will eventually
	// attach to). It does not have group / box / title-label children.
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
	// WidgetKind is the lvvi-resolved widget category for this scene
	// node — boolean / numeric / string / array / structure / primitive
	// / etc. Empty for label and helper nodes, "other" for unclassified
	// open-scope nodes.
	WidgetKind lvvi.WidgetKind
	// Anchor is the connect-point wires will attach to, in scene
	// coordinates. Only meaningful for NodeKindTerminal — for other
	// node kinds it is the zero Point. Phase 12.3 sources Anchor from
	// OF__termHotPoint when present, falling back to the terminal's
	// bounds centre.
	Anchor Point
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

type wireProjectionStats struct {
	Total    int
	Rendered int
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
	// sceneTerminalSize is the default extent for a terminal whose
	// heap node carries neither OF__termBounds nor OF__bounds — small
	// enough to read as an anchor marker rather than a full widget.
	sceneTerminalSize = 8.0
)

type layoutKind string

const (
	layoutKindGroup    layoutKind = "group"
	layoutKindLabel    layoutKind = "label"
	layoutKindTerminal layoutKind = "terminal"
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
	// widgetKind is the lvvi-resolved widget category for this item.
	// Stamped on the emitted scene Nodes so the SVG renderer can pick
	// a generic per-kind skin without re-resolving from the tag.
	widgetKind lvvi.WidgetKind
	// anchor is the per-item connect-point in the item's local
	// coordinate frame (relative to bounds.Left / bounds.Top). Only
	// populated for layoutKindTerminal items.
	anchor lvvi.Point
	// hasAnchor distinguishes "no hot-point recorded" from a hot-point
	// of {V:0, H:0}. When false, placeLayoutItem falls back to the
	// bounds centre.
	hasAnchor bool
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

	wireMix := lvvi.CountWireMix(tree)
	wireStats := wireProjectionStats{Total: wireMix.Total()}
	if view == ViewBlockDiagram {
		scene.Wires, wireStats = projectSceneWires(tree, scene, wireStats.Total)
	}

	scene.ViewBox = computeViewBox(scene, heuristicY)
	scene.Warnings = sceneWarnings(scene, items, wireMix, wireStats)

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
	for _, w := range scene.Wires {
		for _, p := range w.Points {
			right := p.X + sceneMarginX
			bottom := p.Y + sceneMarginY
			if right > maxX {
				maxX = right
			}
			if bottom > maxY {
				maxY = bottom
			}
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
		widgetKind := lvvi.WidgetKindForNode(n)
		// Terminal classes (tunnels, FP terminals) are rendered as a
		// flat anchor marker rather than a group / box / title-label
		// triple. Bounds come from OF__termBounds (preferred) or
		// OF__bounds (fallback); the anchor offset comes from
		// OF__termHotPoint when present.
		if widgetKind == lvvi.WidgetKindTerminal {
			item := &layoutItem{
				kind:        layoutKindTerminal,
				heapIndex:   idx,
				tag:         n.Tag,
				label:       label,
				placeholder: placeholder,
				path:        path,
				contentSize: len(n.Content),
				widgetKind:  widgetKind,
			}
			if b, ok := lvvi.FindTermBoundsChild(tree, idx); ok {
				b := b
				item.bounds = &b
			} else if b, ok := lvvi.FindBoundsChild(tree, idx); ok {
				b := b
				item.bounds = &b
			}
			if p, ok := lvvi.FindTermHotPointChild(tree, idx); ok {
				item.anchor = p
				item.hasAnchor = true
			}
			return item
		}
		item := &layoutItem{
			kind:        layoutKindGroup,
			heapIndex:   idx,
			tag:         n.Tag,
			label:       label,
			placeholder: placeholder,
			path:        path,
			contentSize: len(n.Content),
			widgetKind:  widgetKind,
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
	case layoutKindTerminal:
		// Terminals have no children; size comes from termBounds /
		// bounds. Falls back to a small fixed marker if no rect
		// payload was attached.
		if item.bounds != nil {
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
		item.width = sceneTerminalSize
		item.height = sceneTerminalSize
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
	case layoutKindTerminal:
		// Anchor offset is recorded in the item's local frame; convert
		// to scene coordinates here. When no hot-point was decoded,
		// fall back to the centre of the terminal's bounds so wires
		// always have a connect-point to attach to.
		var anchor Point
		if item.hasAnchor {
			anchor = Point{X: x + float64(item.anchor.H), Y: y + float64(item.anchor.V)}
		} else {
			anchor = Point{X: x + item.width/2, Y: y + item.height/2}
		}
		idx := appendNode(scene, Node{
			Kind:        NodeKindTerminal,
			Label:       item.label,
			Bounds:      Rect{X: x, Y: y, Width: item.width, Height: item.height},
			Parent:      parent,
			Z:           depth*10 + 1,
			Placeholder: item.placeholder,
			HeapIndex:   item.heapIndex,
			Tag:         item.tag,
			Path:        item.path,
			ContentSize: item.contentSize,
			WidgetKind:  item.widgetKind,
			Anchor:      anchor,
		})
		if parent >= 0 {
			scene.Nodes[parent].Children = append(scene.Nodes[parent].Children, idx)
		}
		return idx
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
			WidgetKind:  item.widgetKind,
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
			WidgetKind:  item.widgetKind,
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
			WidgetKind:  item.widgetKind,
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

func sceneWarnings(scene Scene, roots []*layoutItem, wireMix lvvi.WireMix, wireStats wireProjectionStats) []string {
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
		if total := wireMix.Total(); total > 0 {
			if wireStats.Rendered == 0 {
				warnings = append(warnings, "Block-diagram wires not yet rendered (terminals positioned, path drawing pending Phase 12.5).")
				warnings = append(warnings, fmt.Sprintf(
					"Block diagram has %d wire networks (%d auto-routed, %d manually-routed, %d branched, %d other); auto-routed L-shapes and 2- and 3-branch pure Y-trees are typed-decoded, multi-elbow / comb and 4+ branch chunks remain raw (Phase 12.4.4).",
					total, wireMix.AutoChain, wireMix.ManualChain, wireMix.Tree, wireMix.Other))
			} else {
				warnings = append(warnings, fmt.Sprintf(
					"Block diagram has %d wire networks (%d auto-routed, %d manually-routed, %d branched, %d other); rendered %d recognized network(s); auto-routed L-shapes and 2- and 3-branch pure Y-trees are typed-decoded, multi-elbow / comb and 4+ branch chunks remain raw (Phase 14.2).",
					total, wireMix.AutoChain, wireMix.ManualChain, wireMix.Tree, wireMix.Other, wireStats.Rendered))
			}
		} else {
			warnings = append(warnings, "Block-diagram wires not yet rendered (terminals positioned, path drawing pending Phase 12.5).")
		}
	}
	return warnings
}

func projectSceneWires(tree lvvi.HeapTree, scene Scene, total int) ([]Wire, wireProjectionStats) {
	stats := wireProjectionStats{Total: total}
	terminalByHeap := make(map[int]int)
	for i, n := range scene.Nodes {
		if n.Kind == NodeKindTerminal {
			terminalByHeap[n.HeapIndex] = i
		}
	}
	if len(terminalByHeap) == 0 {
		return nil, stats
	}

	var wires []Wire
	for i, n := range tree.Nodes {
		if n.Scope != "leaf" || n.Tag != int32(heap.FieldTagCompressedWireTable) || len(n.Content) == 0 {
			continue
		}
		w, ok := lvvi.HeapWire(tree, i)
		if !ok {
			continue
		}
		terms := terminalSceneNodesForWire(tree, i, terminalByHeap)
		switch w.Mode {
		case lvvi.WireModeAutoChain:
			if len(terms) < 2 {
				continue
			}
			path, ok := w.ChainAutoPath()
			if !ok {
				continue
			}
			from, to := leftToRightTerminals(scene, terms[0], terms[1])
			points := chainAutoScenePoints(scene.Nodes[from].Anchor, scene.Nodes[to].Anchor, path)
			if len(points) < 2 {
				continue
			}
			wires = append(wires, Wire{
				From:   from,
				To:     to,
				Z:      1000 + i,
				Points: points,
				Label:  lvvi.WireModeAutoChain.String(),
			})
			stats.Rendered++
		case lvvi.WireModeTree:
			endpoints, ok := w.TreeEndpoints()
			if !ok {
				continue
			}
			points := treeEndpointScenePoints(scene, endpoints, terms)
			if len(points) < 2 {
				continue
			}
			junction := treeJunction(points)
			for _, p := range points {
				branch := manhattanBranch(junction, p)
				if len(branch) < 2 {
					continue
				}
				wires = append(wires, Wire{
					From:   -1,
					To:     -1,
					Z:      1000 + i,
					Points: branch,
					Label:  lvvi.WireModeTree.String(),
				})
			}
			stats.Rendered++
		}
	}
	return wires, stats
}

func terminalSceneNodesForWire(tree lvvi.HeapTree, wireIdx int, terminalByHeap map[int]int) []int {
	if wireIdx < 0 || wireIdx >= len(tree.Nodes) {
		return nil
	}
	for p := tree.Nodes[wireIdx].Parent; p >= 0 && p < len(tree.Nodes); p = tree.Nodes[p].Parent {
		var out []int
		collectTerminalSceneNodes(tree, p, terminalByHeap, &out)
		if len(out) >= 2 {
			return out
		}
	}
	out := make([]int, 0, len(terminalByHeap))
	for heapIdx := range tree.Nodes {
		if sceneIdx, ok := terminalByHeap[heapIdx]; ok {
			out = append(out, sceneIdx)
		}
	}
	return out
}

func collectTerminalSceneNodes(tree lvvi.HeapTree, heapIdx int, terminalByHeap map[int]int, out *[]int) {
	if sceneIdx, ok := terminalByHeap[heapIdx]; ok {
		*out = append(*out, sceneIdx)
	}
	if heapIdx < 0 || heapIdx >= len(tree.Nodes) {
		return
	}
	for _, ci := range tree.Nodes[heapIdx].Children {
		collectTerminalSceneNodes(tree, ci, terminalByHeap, out)
	}
}

func leftToRightTerminals(scene Scene, a, b int) (int, int) {
	if a < 0 || a >= len(scene.Nodes) || b < 0 || b >= len(scene.Nodes) {
		return a, b
	}
	if scene.Nodes[a].Anchor.X <= scene.Nodes[b].Anchor.X {
		return a, b
	}
	return b, a
}

func chainAutoScenePoints(start, end Point, path lvvi.ChainAutoPath) []Point {
	points := []Point{start}
	if path.Straight {
		if start.X != end.X && start.Y != end.Y {
			points = appendPoint(points, Point{X: end.X, Y: start.Y})
		}
		return appendPoint(points, end)
	}

	elbowX := start.X + float64(path.SourceAnchorX)
	if end.X < start.X {
		elbowX = start.X - float64(path.SourceAnchorX)
	}
	elbowY := start.Y + float64(path.YStep)
	points = appendPoint(points, Point{X: elbowX, Y: start.Y})
	points = appendPoint(points, Point{X: elbowX, Y: elbowY})
	points = appendPoint(points, Point{X: end.X, Y: elbowY})
	return appendPoint(points, end)
}

func treeEndpointScenePoints(scene Scene, endpoints []lvvi.Point, terms []int) []Point {
	points := make([]Point, 0, len(endpoints))
	for _, ep := range endpoints {
		p := Point{X: float64(ep.H) + sceneMarginX, Y: float64(ep.V) + sceneMarginY}
		if match, ok := nearestTerminalAnchor(scene, p, terms); ok {
			p = match
		}
		points = append(points, p)
	}
	return points
}

func nearestTerminalAnchor(scene Scene, p Point, terms []int) (Point, bool) {
	const maxDistance = 4.0
	bestDistance := maxDistance + 1
	var best Point
	for _, idx := range terms {
		if idx < 0 || idx >= len(scene.Nodes) {
			continue
		}
		a := scene.Nodes[idx].Anchor
		d := math.Abs(a.X-p.X) + math.Abs(a.Y-p.Y)
		if d < bestDistance {
			bestDistance = d
			best = a
		}
	}
	if bestDistance <= maxDistance {
		return best, true
	}
	return Point{}, false
}

func treeJunction(points []Point) Point {
	if len(points) == 0 {
		return Point{}
	}
	minX := points[0].X
	sumY := 0.0
	for _, p := range points {
		if p.X < minX {
			minX = p.X
		}
		sumY += p.Y
	}
	return Point{X: minX, Y: sumY / float64(len(points))}
}

func manhattanBranch(from, to Point) []Point {
	points := []Point{from}
	points = appendPoint(points, Point{X: to.X, Y: from.Y})
	return appendPoint(points, to)
}

func appendPoint(points []Point, p Point) []Point {
	if len(points) > 0 {
		last := points[len(points)-1]
		if last.X == p.X && last.Y == p.Y {
			return points
		}
	}
	return append(points, p)
}
