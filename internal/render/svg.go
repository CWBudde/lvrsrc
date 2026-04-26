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

const svgStyle = `<style>
.lvrsrc-node-box { fill: #f4f0e8; stroke: #5f4b32; stroke-width: 1.5; rx: 8; ry: 8; }
.lvrsrc-node-label { fill: #16324f; font: 13px "Helvetica Neue", Helvetica, Arial, sans-serif; }
.lvrsrc-node-placeholder { fill: #7d4f50; stroke: #7d4f50; stroke-dasharray: 5 3; }
.lvrsrc-warning { fill: #7d4f50; font: 12px "Helvetica Neue", Helvetica, Arial, sans-serif; }
</style>
`

func writeSVGNode(buf *bytes.Buffer, node Node) error {
	classes := []string{"lvrsrc-node", "lvrsrc-node-" + string(node.Kind)}
	if node.Placeholder {
		classes = append(classes, "lvrsrc-node-placeholder")
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
	default:
		return nil
	}
}

func esc(s string) string {
	return html.EscapeString(s)
}
