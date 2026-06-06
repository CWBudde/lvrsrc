package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
	"github.com/spf13/cobra"
)

// wireReport is the JSON shape emitted by `lvrsrc wires --json`.
type wireReport struct {
	File  string     `json:"file"`
	Wires []wireInfo `json:"wires"`
}

type wireInfo struct {
	Index         int       `json:"index"`
	Raw           string    `json:"raw"`
	Mode          string    `json:"mode"`
	Waypoints     int       `json:"waypoints"`
	ChainGeometry []uint64  `json:"chainGeometry,omitempty"`
	ChainAuto     *autoInfo `json:"chainAuto,omitempty"`
	LeftwardChain *leftInfo `json:"leftwardChain,omitempty"`
	TreeEndpoints []ptInfo  `json:"treeEndpoints,omitempty"`
	TreeRecords   []string  `json:"treeRecords,omitempty"`
}

type autoInfo struct {
	Straight      bool   `json:"straight"`
	YStep         int    `json:"yStep"`
	SourceAnchorX uint64 `json:"sourceAnchorX"`
}

type leftInfo struct {
	Up             bool `json:"up"`
	VerticalPixels int  `json:"verticalPixels"`
	HorizontalSeed int  `json:"horizontalSeed"`
}

type ptInfo struct {
	V int `json:"v"`
	H int `json:"h"`
}

func (a *cliApp) newWiresCmd() *cobra.Command {
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "wires <file>",
		Short: "Dump decoded block-diagram wire chunks (OF__compressedWireTable)",
		Long: "wires decodes every block-diagram compressed-wire chunk and reports its\n" +
			"raw bytes, mode (auto-chain / manual-chain / tree), waypoint count, and any\n" +
			"recognised typed geometry (single-elbow chain-auto path, leftward multi-elbow\n" +
			"path, or tree endpoints). It is a diagnostic aid for reverse-engineering and\n" +
			"verifying controlled-wire fixtures.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := lvrsrc.Open(args[0], lvrsrc.OpenOptions{Strict: a.v.GetBool("strict")})
			if err != nil {
				return err
			}
			model, _ := lvvi.DecodeKnownResources(file)

			report := wireReport{File: args[0], Wires: collectWires(model)}

			w, closeFn, err := a.outputWriter(cmd)
			if err != nil {
				return err
			}
			defer closeFn()

			if jsonFlag || strings.EqualFold(a.v.GetString("format"), "json") {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}
			return writeWiresText(w, report)
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "emit JSON output")
	return cmd
}

// collectWires walks the decoded block diagram and projects every
// compressed-wire leaf into a wireInfo with whatever typed geometry is
// recognised.
func collectWires(model *lvvi.Model) []wireInfo {
	tree, ok := model.BlockDiagram()
	if !ok {
		return nil
	}
	var out []wireInfo
	for i, n := range tree.Nodes {
		if n.Scope != "leaf" {
			continue
		}
		w, ok := lvvi.HeapWire(tree, i)
		if !ok || len(w.Raw) == 0 {
			continue
		}
		info := wireInfo{
			Index:         len(out) + 1,
			Raw:           hexBytes(w.Raw),
			Mode:          w.Mode.String(),
			Waypoints:     int(w.Waypoints),
			ChainGeometry: w.ChainGeometry,
		}
		if ca, ok := w.ChainAutoPath(); ok {
			info.ChainAuto = &autoInfo{Straight: ca.Straight, YStep: ca.YStep, SourceAnchorX: ca.SourceAnchorX}
		}
		if lcp, ok := w.LeftwardChainPath(); ok {
			info.LeftwardChain = &leftInfo{Up: lcp.Up, VerticalPixels: lcp.VerticalPixels, HorizontalSeed: int(lcp.HorizontalSeed)}
		}
		if pts, ok := w.TreeEndpoints(); ok {
			for _, p := range pts {
				info.TreeEndpoints = append(info.TreeEndpoints, ptInfo{V: int(p.V), H: int(p.H)})
			}
		}
		for _, r := range w.TreeRecords {
			info.TreeRecords = append(info.TreeRecords, fmt.Sprintf("%02x %02x", r[0], r[1]))
		}
		out = append(out, info)
	}
	return out
}

func writeWiresText(w io.Writer, report wireReport) error {
	if _, err := fmt.Fprintf(w, "File: %s\n", report.File); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Block-diagram wire chunks: %d\n", len(report.Wires)); err != nil {
		return err
	}
	for _, info := range report.Wires {
		if _, err := fmt.Fprintf(w, "\n#%d  raw=%s  mode=%s  waypoints=%d\n", info.Index, info.Raw, info.Mode, info.Waypoints); err != nil {
			return err
		}
		if len(info.ChainGeometry) > 0 {
			if _, err := fmt.Fprintf(w, "    chain-geometry: %v\n", info.ChainGeometry); err != nil {
				return err
			}
		}
		if info.ChainAuto != nil {
			if _, err := fmt.Fprintf(w, "    chain-auto: straight=%t yStep=%d sourceAnchorX=%d\n",
				info.ChainAuto.Straight, info.ChainAuto.YStep, info.ChainAuto.SourceAnchorX); err != nil {
				return err
			}
		}
		if info.LeftwardChain != nil {
			if _, err := fmt.Fprintf(w, "    leftward-chain: up=%t verticalPixels=%d horizontalSeed=0x%02x\n",
				info.LeftwardChain.Up, info.LeftwardChain.VerticalPixels, info.LeftwardChain.HorizontalSeed); err != nil {
				return err
			}
		}
		if len(info.TreeEndpoints) > 0 {
			if _, err := fmt.Fprintf(w, "    tree-endpoints: %v\n", info.TreeEndpoints); err != nil {
				return err
			}
		}
	}
	return nil
}

func hexBytes(b []byte) string {
	parts := make([]string, len(b))
	for i, c := range b {
		parts[i] = fmt.Sprintf("%02x", c)
	}
	return strings.Join(parts, " ")
}
