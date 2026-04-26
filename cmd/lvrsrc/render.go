package main

import (
	"fmt"
	"strings"

	irender "github.com/CWBudde/lvrsrc/internal/render"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
	"github.com/CWBudde/lvrsrc/pkg/lvvi"
	"github.com/spf13/cobra"
)

func (a *cliApp) newRenderCmd() *cobra.Command {
	var viewFlag string
	var formatFlag string

	cmd := &cobra.Command{
		Use:   "render <file>",
		Short: "Render an approximate front-panel or block-diagram SVG",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !strings.EqualFold(formatFlag, "svg") {
				return fmt.Errorf("unsupported render format %q (supported: svg)", formatFlag)
			}

			file, err := lvrsrc.Open(args[0], lvrsrc.OpenOptions{Strict: a.v.GetBool("strict")})
			if err != nil {
				return err
			}
			model, _ := lvvi.DecodeKnownResources(file)

			var (
				scene irender.Scene
				ok    bool
				title string
			)
			switch strings.ToLower(viewFlag) {
			case "front-panel":
				scene, ok = irender.FrontPanelScene(model)
				title = "LabVIEW front-panel render"
			case "block-diagram":
				scene, ok = irender.BlockDiagramScene(model)
				title = "LabVIEW block-diagram render"
			default:
				return fmt.Errorf("unsupported render view %q (supported: front-panel, block-diagram)", viewFlag)
			}
			if !ok {
				return fmt.Errorf("file has no decodable %s scene", viewFlag)
			}

			svg, err := irender.SVG(scene, irender.SVGOptions{Title: title})
			if err != nil {
				return err
			}

			w, closeFn, err := a.outputWriter(cmd)
			if err != nil {
				return err
			}
			defer closeFn()

			_, err = fmt.Fprint(w, svg)
			return err
		},
	}
	cmd.Flags().StringVar(&viewFlag, "view", "front-panel", "scene to render: front-panel or block-diagram")
	cmd.Flags().StringVar(&formatFlag, "format", "svg", "render output format")
	return cmd
}
