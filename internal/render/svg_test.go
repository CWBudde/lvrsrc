package render

import (
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

func TestSVGIncludesViewBoxTitleAndPlaceholderClasses(t *testing.T) {
	scene := Scene{
		View:    ViewFrontPanel,
		ViewBox: Rect{Width: 320, Height: 160},
		Roots:   []int{0},
		Nodes: []Node{
			{
				Kind:        NodeKindGroup,
				Label:       "SL__object",
				Bounds:      Rect{X: 24, Y: 24, Width: 180, Height: 80},
				Parent:      -1,
				Children:    []int{1, 2},
				Path:        "SL__object",
				HeapIndex:   0,
				Z:           0,
				Placeholder: false,
			},
			{
				Kind:        NodeKindBox,
				Label:       "SL__object",
				Bounds:      Rect{X: 24, Y: 24, Width: 180, Height: 80},
				Parent:      0,
				Path:        "SL__object",
				HeapIndex:   0,
				Z:           1,
				Placeholder: false,
			},
			{
				Kind:        NodeKindLabel,
				Label:       "Tag(99999) <leaf>",
				Bounds:      Rect{X: 40, Y: 40, Width: 120, Height: 18},
				Parent:      0,
				Path:        "SL__object/Tag(99999)",
				HeapIndex:   2,
				Z:           2,
				Placeholder: true,
			},
		},
		Warnings: []string{"placeholder nodes rendered heuristically"},
	}

	got, err := SVG(scene, SVGOptions{Title: "Front Panel Scene"})
	if err != nil {
		t.Fatalf("SVG() err = %v", err)
	}
	if !strings.Contains(got, `<svg`) {
		t.Fatalf("SVG output missing <svg: %q", got)
	}
	if !strings.Contains(got, `viewBox="0 0 320 160"`) {
		t.Fatalf("SVG output missing viewBox: %q", got)
	}
	if !strings.Contains(got, `<title>Front Panel Scene</title>`) {
		t.Fatalf("SVG output missing title: %q", got)
	}
	if !strings.Contains(got, `class="lvrsrc-node lvrsrc-node-box"`) {
		t.Fatalf("SVG output missing box class: %q", got)
	}
	if !strings.Contains(got, `lvrsrc-node-placeholder`) {
		t.Fatalf("SVG output missing placeholder class: %q", got)
	}
	if !strings.Contains(got, `data-path="SL__object/Tag(99999)"`) {
		t.Fatalf("SVG output missing data-path: %q", got)
	}
	if !strings.Contains(got, `Tag(99999) &lt;leaf&gt;`) {
		t.Fatalf("SVG output did not escape text content: %q", got)
	}
	if !strings.Contains(got, `placeholder nodes rendered heuristically`) {
		t.Fatalf("SVG output missing warning text: %q", got)
	}
}

// Nodes that carry a WidgetKind should pick up an extra
// `lvrsrc-widget-{kind}` CSS class so the renderer can style each
// widget kind generically without matching the underlying class tag.
func TestSVGEmitsWidgetKindClass(t *testing.T) {
	scene := Scene{
		View:    ViewFrontPanel,
		ViewBox: Rect{Width: 200, Height: 120},
		Roots:   []int{0},
		Nodes: []Node{
			{
				Kind:       NodeKindGroup,
				Label:      "SL__stdBool",
				Bounds:     Rect{X: 24, Y: 24, Width: 60, Height: 30},
				Parent:     -1,
				Children:   []int{1},
				HeapIndex:  0,
				Path:       "SL__stdBool",
				WidgetKind: lvvi.WidgetKindBoolean,
			},
			{
				Kind:       NodeKindBox,
				Label:      "SL__stdBool",
				Bounds:     Rect{X: 24, Y: 24, Width: 60, Height: 30},
				Parent:     0,
				HeapIndex:  0,
				Path:       "SL__stdBool",
				Z:          1,
				WidgetKind: lvvi.WidgetKindBoolean,
			},
		},
	}
	got, err := SVG(scene, SVGOptions{})
	if err != nil {
		t.Fatalf("SVG() err = %v", err)
	}
	if !strings.Contains(got, "lvrsrc-widget-boolean") {
		t.Fatalf("SVG output missing widget class for boolean: %q", got)
	}
	// Default (.other / empty) styling must still be in the stylesheet so
	// unclassified nodes pick up a fallback skin.
	if !strings.Contains(got, ".lvrsrc-widget-other") {
		t.Fatalf("SVG output missing default widget styling: %q", got)
	}
}

func TestSVGOmitsWidgetKindClassWhenAbsent(t *testing.T) {
	// Nodes with WidgetKind == "" (e.g. helper / leaf labels) must not
	// emit a stray "lvrsrc-widget-" class fragment, otherwise the CSS
	// selector matches everything.
	scene := Scene{
		View:    ViewFrontPanel,
		ViewBox: Rect{Width: 100, Height: 50},
		Roots:   []int{0},
		Nodes: []Node{
			{
				Kind:   NodeKindBox,
				Bounds: Rect{X: 0, Y: 0, Width: 50, Height: 30},
				Parent: -1,
			},
		},
	}
	got, err := SVG(scene, SVGOptions{})
	if err != nil {
		t.Fatalf("SVG() err = %v", err)
	}
	if strings.Contains(got, `lvrsrc-widget-"`) || strings.Contains(got, `lvrsrc-widget- `) {
		t.Fatalf("SVG output emitted empty widget class: %q", got)
	}
}

// A NodeKindTerminal must emit both an outline rect at the bounds and
// a small filled circle at the anchor — that's the visual contract
// wires (12.5) will eventually attach to.
func TestSVGEmitsTerminalAnchorAndOutline(t *testing.T) {
	scene := Scene{
		View:    ViewBlockDiagram,
		ViewBox: Rect{Width: 200, Height: 120},
		Roots:   []int{0},
		Nodes: []Node{
			{
				Kind:       NodeKindTerminal,
				Label:      "SL__simTun",
				Bounds:     Rect{X: 40, Y: 60, Width: 8, Height: 8},
				Parent:     -1,
				HeapIndex:  0,
				Path:       "SL__simTun",
				WidgetKind: lvvi.WidgetKindTerminal,
				Anchor:     Point{X: 44, Y: 64},
			},
		},
	}
	got, err := SVG(scene, SVGOptions{})
	if err != nil {
		t.Fatalf("SVG() err = %v", err)
	}
	if !strings.Contains(got, "lvrsrc-node-terminal") {
		t.Fatalf("SVG output missing terminal class: %q", got)
	}
	if !strings.Contains(got, `<rect`) {
		t.Fatalf("SVG terminal missing outline rect: %q", got)
	}
	if !strings.Contains(got, `<circle`) {
		t.Fatalf("SVG terminal missing anchor circle: %q", got)
	}
	if !strings.Contains(got, `cx="44"`) || !strings.Contains(got, `cy="64"`) {
		t.Fatalf("SVG terminal anchor circle not at (44,64): %q", got)
	}
}

func TestSVGRejectsEmptyViewBox(t *testing.T) {
	_, err := SVG(Scene{}, SVGOptions{})
	if err == nil {
		t.Fatal("SVG() err = nil, want non-nil for empty scene")
	}
	if !strings.Contains(err.Error(), "view box") {
		t.Fatalf("SVG() err = %q, want mention of view box", err.Error())
	}
}
