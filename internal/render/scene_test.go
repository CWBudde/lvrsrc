package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

func TestProjectHeapTreeBuildsSceneGraphForOpenAndLeafNodes(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{Tag: -3, Scope: "open", Parent: -1, Children: []int{1, 2}},
			{Tag: 99999, Scope: "open", Parent: 0},
			{Tag: 88888, Scope: "leaf", Parent: 0},
		},
		Roots: []int{0},
	}

	scene := ProjectHeapTree(tree, ViewFrontPanel)

	if scene.View != ViewFrontPanel {
		t.Fatalf("scene.View = %q, want %q", scene.View, ViewFrontPanel)
	}
	if len(scene.Roots) != 1 {
		t.Fatalf("len(scene.Roots) = %d, want 1", len(scene.Roots))
	}
	if scene.ViewBox.Width <= 0 || scene.ViewBox.Height <= 0 {
		t.Fatalf("scene.ViewBox = %+v, want positive width/height", scene.ViewBox)
	}

	root := scene.Nodes[scene.Roots[0]]
	if root.Kind != NodeKindGroup {
		t.Fatalf("root.Kind = %q, want %q", root.Kind, NodeKindGroup)
	}
	if root.HeapIndex != 0 {
		t.Fatalf("root.HeapIndex = %d, want 0", root.HeapIndex)
	}
	if root.Label != "SL__object" {
		t.Fatalf("root.Label = %q, want SL__object", root.Label)
	}
	if root.LeafCount != 1 {
		t.Fatalf("root.LeafCount = %d, want 1", root.LeafCount)
	}

	rootBox, ok := findNode(scene.Nodes, func(n Node) bool {
		return n.HeapIndex == 0 && n.Kind == NodeKindBox
	})
	if !ok {
		t.Fatal("did not emit a box node for the open-scope heap object")
	}
	rootTitle, ok := findNode(scene.Nodes, func(n Node) bool {
		return n.HeapIndex == 0 && n.Kind == NodeKindLabel
	})
	if !ok {
		t.Fatal("did not emit a title label for the open-scope heap object")
	}
	if rootTitle.Label != "SL__object" {
		t.Fatalf("root title label = %q, want SL__object", rootTitle.Label)
	}
	if !containsRect(root.Bounds, rootBox.Bounds) {
		t.Fatalf("root group bounds %+v do not contain box bounds %+v", root.Bounds, rootBox.Bounds)
	}
	if !containsRect(root.Bounds, rootTitle.Bounds) {
		t.Fatalf("root group bounds %+v do not contain title bounds %+v", root.Bounds, rootTitle.Bounds)
	}

	childGroup, ok := findNode(scene.Nodes, func(n Node) bool {
		return n.HeapIndex == 1 && n.Kind == NodeKindGroup
	})
	if !ok {
		t.Fatal("did not emit a child group node for the nested open-scope heap object")
	}
	if !childGroup.Placeholder {
		t.Fatal("nested unknown tag was not marked as a placeholder")
	}
	if childGroup.Parent != scene.Roots[0] {
		t.Fatalf("childGroup.Parent = %d, want %d", childGroup.Parent, scene.Roots[0])
	}
	if got := childGroup.Label; got != "Tag(99999)" {
		t.Fatalf("childGroup.Label = %q, want Tag(99999)", got)
	}
	if got := childGroup.Path; !strings.Contains(got, "SL__object/Tag(99999)") {
		t.Fatalf("childGroup.Path = %q, want it to include parent/object path", got)
	}
	if !containsRect(root.Bounds, childGroup.Bounds) {
		t.Fatalf("root bounds %+v do not contain child group bounds %+v", root.Bounds, childGroup.Bounds)
	}

	leafLabel, ok := findNode(scene.Nodes, func(n Node) bool {
		return n.HeapIndex == 2 && n.Kind == NodeKindLabel
	})
	if !ok {
		t.Fatal("did not emit a label node for the leaf heap entry")
	}
	if leafLabel.Parent != scene.Roots[0] {
		t.Fatalf("leafLabel.Parent = %d, want %d", leafLabel.Parent, scene.Roots[0])
	}
	if leafLabel.Bounds.Width <= 0 || leafLabel.Bounds.Height <= 0 {
		t.Fatalf("leafLabel bounds = %+v, want positive width/height", leafLabel.Bounds)
	}
}

func TestProjectHeapTreeStacksMultipleRootsTopToBottom(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{Tag: -3, Scope: "open", Parent: -1},
			{Tag: -4, Scope: "open", Parent: -1},
		},
		Roots: []int{0, 1},
	}

	scene := ProjectHeapTree(tree, ViewBlockDiagram)

	if scene.View != ViewBlockDiagram {
		t.Fatalf("scene.View = %q, want %q", scene.View, ViewBlockDiagram)
	}
	if len(scene.Roots) != 2 {
		t.Fatalf("len(scene.Roots) = %d, want 2", len(scene.Roots))
	}
	first := scene.Nodes[scene.Roots[0]]
	second := scene.Nodes[scene.Roots[1]]
	if !(first.Bounds.Y < second.Bounds.Y) {
		t.Fatalf("root bounds Y = %v and %v, want top-to-bottom ordering", first.Bounds.Y, second.Bounds.Y)
	}
	if overlapsVertically(first.Bounds, second.Bounds) {
		t.Fatalf("root bounds overlap vertically: first=%+v second=%+v", first.Bounds, second.Bounds)
	}
}

func TestProjectHeapTreeAddsFidelityWarnings(t *testing.T) {
	tree := lvvi.HeapTree{
		Nodes: []lvvi.HeapNode{
			{Tag: -3, Scope: "open", Parent: -1, Children: []int{1}},
			{Tag: 99999, Scope: "open", Parent: 0},
		},
		Roots: []int{0},
	}

	fp := ProjectHeapTree(tree, ViewFrontPanel)
	if len(fp.Warnings) == 0 {
		t.Fatal("front-panel scene warnings = empty, want heuristic/placeholder warning")
	}
	if !containsString(fp.Warnings, "heuristic") {
		t.Fatalf("front-panel warnings = %v, want heuristic warning", fp.Warnings)
	}
	if !containsString(fp.Warnings, "placeholder") {
		t.Fatalf("front-panel warnings = %v, want placeholder warning", fp.Warnings)
	}

	bd := ProjectHeapTree(tree, ViewBlockDiagram)
	if !containsString(bd.Warnings, "wires not yet rendered") {
		t.Fatalf("block-diagram warnings = %v, want wires-not-rendered warning", bd.Warnings)
	}
}

func TestPreferCanvasForLargeScenes(t *testing.T) {
	small := Scene{
		ViewBox: Rect{Width: 320, Height: 240},
		Nodes: []Node{
			{Kind: NodeKindBox, Bounds: Rect{Width: 120, Height: 60}},
			{Kind: NodeKindLabel, Bounds: Rect{Width: 80, Height: 18}},
		},
	}
	if PreferCanvas(small) {
		t.Fatal("PreferCanvas(small) = true, want false")
	}

	large := Scene{
		ViewBox: Rect{Width: 2400, Height: 1800},
	}
	for i := 0; i < 220; i++ {
		large.Nodes = append(large.Nodes, Node{
			Kind:   NodeKindBox,
			Bounds: Rect{X: float64(i * 12), Y: float64(i * 8), Width: 180, Height: 90},
		})
	}
	if !PreferCanvas(large) {
		t.Fatal("PreferCanvas(large) = false, want true")
	}
}

func TestFrontPanelSceneOnCorpusProducesPositiveViewBox(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
		if err != nil {
			continue
		}
		m, _ := lvvi.DecodeKnownResources(f)
		scene, ok := FrontPanelScene(m)
		if !ok {
			continue
		}
		if scene.View != ViewFrontPanel {
			t.Fatalf("%s: scene.View = %q, want %q", e.Name(), scene.View, ViewFrontPanel)
		}
		if len(scene.Roots) == 0 || len(scene.Nodes) == 0 {
			t.Fatalf("%s: empty scene from decodable front panel", e.Name())
		}
		if scene.ViewBox.Width <= 0 || scene.ViewBox.Height <= 0 {
			t.Fatalf("%s: non-positive view box %+v", e.Name(), scene.ViewBox)
		}
		return
	}
	t.Skip("no corpus VI with a decodable FPHb scene")
}

func containsString(ss []string, needle string) bool {
	for _, s := range ss {
		if strings.Contains(strings.ToLower(s), strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func findNode(nodes []Node, pred func(Node) bool) (Node, bool) {
	for _, n := range nodes {
		if pred(n) {
			return n, true
		}
	}
	return Node{}, false
}

func containsRect(outer, inner Rect) bool {
	return inner.X >= outer.X &&
		inner.Y >= outer.Y &&
		inner.X+inner.Width <= outer.X+outer.Width &&
		inner.Y+inner.Height <= outer.Y+outer.Height
}

func overlapsVertically(a, b Rect) bool {
	return a.Y+a.Height > b.Y
}
