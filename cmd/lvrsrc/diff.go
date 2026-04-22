package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/CWBudde/lvrsrc/pkg/lvdiff"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
	"github.com/spf13/cobra"
)

func (a *cliApp) newDiffCmd() *cobra.Command {
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "diff <a> <b>",
		Short: "Diff two RSRC files (header, blocks, sections)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := lvrsrc.OpenOptions{Strict: a.v.GetBool("strict")}
			fa, err := lvrsrc.Open(args[0], opts)
			if err != nil {
				return fmt.Errorf("open %s: %w", args[0], err)
			}
			fb, err := lvrsrc.Open(args[1], opts)
			if err != nil {
				return fmt.Errorf("open %s: %w", args[1], err)
			}

			diff := lvdiff.Files(fa, fb)

			exitCode := 0
			if !diff.IsEmpty() {
				exitCode = 1
			}

			w, closeFn, err := a.outputWriter(cmd)
			if err != nil {
				return err
			}
			defer closeFn()

			if jsonFlag || strings.EqualFold(a.v.GetString("format"), "json") {
				if err := writeDiffJSON(w, diff, exitCode); err != nil {
					return err
				}
			} else {
				if err := writeDiffText(w, diff, args[0], args[1]); err != nil {
					return err
				}
			}

			if exitCode == 0 {
				return nil
			}
			return &exitCodeError{
				code: exitCode,
				err:  fmt.Errorf("files differ (%d item(s))", len(diff.Items)),
			}
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "emit JSON output")
	return cmd
}

type diffSummary struct {
	Added    int `json:"added"`
	Removed  int `json:"removed"`
	Modified int `json:"modified"`
}

func summarize(diff *lvdiff.Diff) diffSummary {
	var s diffSummary
	for _, it := range diff.Items {
		switch it.Category {
		case lvdiff.CategoryAdded:
			s.Added++
		case lvdiff.CategoryRemoved:
			s.Removed++
		case lvdiff.CategoryModified:
			s.Modified++
		}
	}
	return s
}

type diffItemJSON struct {
	Kind     string `json:"kind"`
	Category string `json:"category"`
	Path     string `json:"path"`
	Old      any    `json:"old,omitempty"`
	New      any    `json:"new,omitempty"`
	Message  string `json:"message,omitempty"`
}

func writeDiffJSON(w io.Writer, diff *lvdiff.Diff, exitCode int) error {
	items := make([]diffItemJSON, 0, len(diff.Items))
	for _, it := range diff.Items {
		items = append(items, diffItemJSON{
			Kind:     string(it.Kind),
			Category: string(it.Category),
			Path:     it.Path,
			Old:      it.Old,
			New:      it.New,
			Message:  it.Message,
		})
	}

	payload := struct {
		Identical bool           `json:"identical"`
		ExitCode  int            `json:"exitCode"`
		Summary   diffSummary    `json:"summary"`
		Items     []diffItemJSON `json:"items"`
	}{
		Identical: diff.IsEmpty(),
		ExitCode:  exitCode,
		Summary:   summarize(diff),
		Items:     items,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func writeDiffText(w io.Writer, diff *lvdiff.Diff, aLabel, bLabel string) error {
	if _, err := fmt.Fprintf(w, "--- %s\n+++ %s\n", aLabel, bLabel); err != nil {
		return err
	}
	if diff.IsEmpty() {
		_, err := fmt.Fprintln(w, "files are identical")
		return err
	}

	s := summarize(diff)
	if _, err := fmt.Fprintf(w, "Summary: %d added, %d removed, %d modified\n", s.Added, s.Removed, s.Modified); err != nil {
		return err
	}

	items := append([]lvdiff.DiffItem(nil), diff.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return kindOrder(items[i].Kind) < kindOrder(items[j].Kind)
		}
		return items[i].Path < items[j].Path
	})

	var currentKind lvdiff.Kind
	for _, it := range items {
		if it.Kind != currentKind {
			currentKind = it.Kind
			if _, err := fmt.Fprintf(w, "\n@@ %s @@\n", kindHeading(it.Kind)); err != nil {
				return err
			}
		}

		prefix := "~"
		switch it.Category {
		case lvdiff.CategoryAdded:
			prefix = "+"
		case lvdiff.CategoryRemoved:
			prefix = "-"
		}

		if _, err := fmt.Fprintf(w, "%s %s", prefix, it.Path); err != nil {
			return err
		}
		switch it.Category {
		case lvdiff.CategoryModified:
			if _, err := fmt.Fprintf(w, ": %v -> %v\n", formatValue(it.Old), formatValue(it.New)); err != nil {
				return err
			}
		case lvdiff.CategoryAdded:
			if it.New != nil {
				if _, err := fmt.Fprintf(w, ": %v", formatValue(it.New)); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		case lvdiff.CategoryRemoved:
			if it.Old != nil {
				if _, err := fmt.Fprintf(w, ": %v", formatValue(it.Old)); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}
	return nil
}

func kindOrder(k lvdiff.Kind) int {
	switch k {
	case lvdiff.KindHeader:
		return 0
	case lvdiff.KindBlock:
		return 1
	case lvdiff.KindSection:
		return 2
	case lvdiff.KindDecoded:
		return 3
	default:
		return 4
	}
}

func kindHeading(k lvdiff.Kind) string {
	switch k {
	case lvdiff.KindHeader:
		return "header"
	case lvdiff.KindBlock:
		return "blocks"
	case lvdiff.KindSection:
		return "sections"
	case lvdiff.KindDecoded:
		return "decoded"
	default:
		return string(k)
	}
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
