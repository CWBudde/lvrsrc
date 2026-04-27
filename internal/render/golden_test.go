package render

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/internal/golden"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
)

func TestSceneAndSVGGolden(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		fixture string
		view    View
	}{
		{name: "simple-vi-front-panel", fixture: "format-string.vi", view: ViewFrontPanel},
		{name: "control-front-panel", fixture: "action.ctl", view: ViewFrontPanel},
		{name: "structure-heavy-block-diagram", fixture: "load-vi.vi", view: ViewBlockDiagram},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			scene := mustSceneFromFixture(t, tc.fixture, tc.view)
			svg, err := SVG(scene, SVGOptions{Title: "golden render"})
			if err != nil {
				t.Fatalf("SVG(%s,%s) err = %v", tc.fixture, tc.view, err)
			}

			snap := renderGoldenSnapshot{
				Fixture:      tc.fixture,
				View:         string(tc.view),
				ViewBox:      scene.ViewBox,
				PreferCanvas: PreferCanvas(scene),
				Warnings:     append([]string(nil), scene.Warnings...),
				Roots:        append([]int(nil), scene.Roots...),
				Nodes:        snapshotNodes(scene.Nodes),
				SVG:          svg,
			}

			got, err := json.MarshalIndent(snap, "", "  ")
			if err != nil {
				t.Fatalf("MarshalIndent(%s,%s) err = %v", tc.fixture, tc.view, err)
			}
			got = append(got, '\n')

			golden.Assert(t, filepath.Join("testdata", "golden", tc.name+".golden.json"), got)
		})
	}
}

type renderGoldenSnapshot struct {
	Fixture      string               `json:"fixture"`
	View         string               `json:"view"`
	ViewBox      Rect                 `json:"view_box"`
	PreferCanvas bool                 `json:"prefer_canvas"`
	Warnings     []string             `json:"warnings,omitempty"`
	Roots        []int                `json:"roots"`
	Nodes        []renderNodeSnapshot `json:"nodes"`
	SVG          string               `json:"svg"`
}

type renderNodeSnapshot struct {
	Kind        NodeKind        `json:"kind"`
	Label       string          `json:"label,omitempty"`
	Bounds      Rect            `json:"bounds"`
	Parent      int             `json:"parent"`
	Children    []int           `json:"children,omitempty"`
	Z           int             `json:"z"`
	Placeholder bool            `json:"placeholder,omitempty"`
	HeapIndex   int             `json:"heap_index"`
	Path        string          `json:"path,omitempty"`
	WidgetKind  lvvi.WidgetKind `json:"widget_kind,omitempty"`
	Anchor      *Point          `json:"anchor,omitempty"`
}

func snapshotNodes(nodes []Node) []renderNodeSnapshot {
	out := make([]renderNodeSnapshot, len(nodes))
	for i, n := range nodes {
		var anchor *Point
		if n.Kind == NodeKindTerminal {
			a := n.Anchor
			anchor = &a
		}
		out[i] = renderNodeSnapshot{
			Kind:        n.Kind,
			Label:       n.Label,
			Bounds:      n.Bounds,
			Parent:      n.Parent,
			Children:    append([]int(nil), n.Children...),
			Z:           n.Z,
			Placeholder: n.Placeholder,
			HeapIndex:   n.HeapIndex,
			Path:        n.Path,
			WidgetKind:  n.WidgetKind,
			Anchor:      anchor,
		}
	}
	return out
}

func mustSceneFromFixture(t *testing.T, fixture string, view View) Scene {
	t.Helper()

	f, err := lvrsrc.Open(corpus.Path(fixture), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open(%s) err = %v", fixture, err)
	}
	m, _ := lvvi.DecodeKnownResources(f)
	switch view {
	case ViewFrontPanel:
		scene, ok := FrontPanelScene(m)
		if !ok {
			t.Fatalf("FrontPanelScene(%s) ok = false", fixture)
		}
		return scene
	case ViewBlockDiagram:
		scene, ok := BlockDiagramScene(m)
		if !ok {
			t.Fatalf("BlockDiagramScene(%s) ok = false", fixture)
		}
		return scene
	default:
		t.Fatalf("unsupported golden view %q", view)
		return Scene{}
	}
}
