package render

import (
	"strings"
	"testing"
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

func TestSVGRejectsEmptyViewBox(t *testing.T) {
	_, err := SVG(Scene{}, SVGOptions{})
	if err == nil {
		t.Fatal("SVG() err = nil, want non-nil for empty scene")
	}
	if !strings.Contains(err.Error(), "view box") {
		t.Fatalf("SVG() err = %q, want mention of view box", err.Error())
	}
}
