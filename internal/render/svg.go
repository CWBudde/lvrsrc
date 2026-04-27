package render

import (
	"bytes"
	"fmt"
	"html"
	"sort"
	"strings"
)

// SVGOptions controls how a Scene is emitted as SVG.
type SVGOptions struct {
	Title string
}

// SVG renders a scene as a standalone SVG document.
func SVG(scene Scene, opts SVGOptions) (string, error) {
	if scene.ViewBox.Width <= 0 || scene.ViewBox.Height <= 0 {
		return "", fmt.Errorf("scene view box must have positive width and height")
	}

	title := opts.Title
	if title == "" {
		title = "lvrsrc scene render"
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="%s">`+"\n",
		int(scene.ViewBox.Width), int(scene.ViewBox.Height), int(scene.ViewBox.Width), int(scene.ViewBox.Height), esc(title))
	fmt.Fprintf(&buf, "<title>%s</title>\n", esc(title))
	buf.WriteString(svgStyle)

	if len(scene.Warnings) > 0 {
		y := 18.0
		for _, warning := range scene.Warnings {
			fmt.Fprintf(&buf, `<text class="lvrsrc-warning" x="12" y="%.0f">%s</text>`+"\n", y, esc(warning))
			y += 16
		}
	}

	nodes := append([]Node(nil), scene.Nodes...)
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Z != nodes[j].Z {
			return nodes[i].Z < nodes[j].Z
		}
		if nodes[i].Parent != nodes[j].Parent {
			return nodes[i].Parent < nodes[j].Parent
		}
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind < nodes[j].Kind
		}
		return nodes[i].Path < nodes[j].Path
	})
	for _, node := range nodes {
		if err := writeSVGNode(&buf, node); err != nil {
			return "", err
		}
	}
	buf.WriteString("</svg>\n")
	return buf.String(), nil
}

// Per-WidgetKind skins. Phase 12.2a aims for "you can tell the
// booleans from the numerics from the strings"; pixel-faithful per-
// class styling is Phase 12.6 (Stage 2 fidelity). The .other style is
// always emitted so the existing-style baseline matches unclassified
// nodes; subsequent rules win via class specificity. Keep the rule
// order stable for golden-test reproducibility.
const svgStyle = `<style>
.lvrsrc-node-box { fill: #f4f0e8; stroke: #5f4b32; stroke-width: 1.5; rx: 8; ry: 8; }
.lvrsrc-node-label { fill: #16324f; font: 13px "Helvetica Neue", Helvetica, Arial, sans-serif; }
.lvrsrc-node-placeholder { fill: #7d4f50; stroke: #7d4f50; stroke-dasharray: 5 3; }
.lvrsrc-warning { fill: #7d4f50; font: 12px "Helvetica Neue", Helvetica, Arial, sans-serif; }
.lvrsrc-widget-other { fill: #f4f0e8; stroke: #5f4b32; }
.lvrsrc-widget-boolean { fill: #d8eed3; stroke: #3a7a3a; }
.lvrsrc-widget-numeric { fill: #d6e6f2; stroke: #1f6f8b; }
.lvrsrc-widget-string { fill: #ecdcef; stroke: #5f3a7a; }
.lvrsrc-widget-cluster { fill: #ecdfcd; stroke: #7a5a3a; }
.lvrsrc-widget-array { fill: #e1e8c8; stroke: #5a6f1f; }
.lvrsrc-widget-graph { fill: #d2e6e6; stroke: #1f6b6b; }
.lvrsrc-widget-decoration { fill: #f0f0f0; stroke: #9a9a9a; stroke-dasharray: 3 2; }
.lvrsrc-widget-structure { fill: #f1d9c2; stroke: #8b4f1f; stroke-width: 2; }
.lvrsrc-widget-primitive { fill: #d3dceb; stroke: #1f3a6b; }
.lvrsrc-widget-terminal { fill: #ffffff; stroke: #2a2a2a; stroke-width: 1; }
.lvrsrc-node-terminal-anchor { fill: #2a2a2a; stroke: none; }
</style>
`

func writeSVGNode(buf *bytes.Buffer, node Node) error {
	classes := []string{"lvrsrc-node", "lvrsrc-node-" + string(node.Kind)}
	if node.Placeholder {
		classes = append(classes, "lvrsrc-node-placeholder")
	}
	if node.WidgetKind != "" {
		classes = append(classes, "lvrsrc-widget-"+string(node.WidgetKind))
	}
	common := fmt.Sprintf(`class="%s" data-path="%s" data-heap-index="%d"`,
		strings.Join(classes, " "), esc(node.Path), node.HeapIndex)

	switch node.Kind {
	case NodeKindBox:
		_, err := fmt.Fprintf(buf,
			`<rect %s x="%.0f" y="%.0f" width="%.0f" height="%.0f"/>`+"\n",
			common, node.Bounds.X, node.Bounds.Y, node.Bounds.Width, node.Bounds.Height)
		return err
	case NodeKindLabel:
		_, err := fmt.Fprintf(buf,
			`<text %s x="%.0f" y="%.0f">%s</text>`+"\n",
			common, node.Bounds.X, node.Bounds.Y+node.Bounds.Height-4, esc(node.Label))
		return err
	case NodeKindTerminal:
		// Terminal anchors are drawn as a small outline rect at the
		// bounds plus a filled dot at the anchor — wires (12.5) will
		// attach at the dot.
		if _, err := fmt.Fprintf(buf,
			`<rect %s x="%.0f" y="%.0f" width="%.0f" height="%.0f"/>`+"\n",
			common, node.Bounds.X, node.Bounds.Y, node.Bounds.Width, node.Bounds.Height); err != nil {
			return err
		}
		anchorClasses := append([]string{}, "lvrsrc-node-terminal-anchor")
		anchorAttr := fmt.Sprintf(`class="%s" data-path="%s" data-heap-index="%d"`,
			strings.Join(anchorClasses, " "), esc(node.Path), node.HeapIndex)
		_, err := fmt.Fprintf(buf,
			`<circle %s cx="%.0f" cy="%.0f" r="2"/>`+"\n",
			anchorAttr, node.Anchor.X, node.Anchor.Y)
		return err
	default:
		return nil
	}
}

func esc(s string) string {
	return html.EscapeString(s)
}
