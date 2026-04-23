package main

import (
	"fmt"

	"github.com/CWBudde/lvrsrc/pkg/lvmeta"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
	"github.com/spf13/cobra"
)

// newSetMetaCmd registers the `lvrsrc set-meta` command.
//
// It applies Tier 2 metadata edits via pkg/lvmeta and writes the result
// to --out, then re-parses the output and runs structural validation as
// a belt-and-suspenders post-write safety gate. The --unsafe flag is
// accepted but gated off; Tier 3 raw patching is not implemented yet
// and attempting to opt in returns an error.
func (a *cliApp) newSetMetaCmd() *cobra.Command {
	var (
		description string
		name        string
		unsafe      bool
	)

	cmd := &cobra.Command{
		Use:   "set-meta <file>",
		Short: "Edit VI metadata (description, name) under the Tier 2 safety model",
		Long: "set-meta applies targeted metadata edits to a LabVIEW RSRC file.\n" +
			"At least one of --description or --name must be provided, and --out\n" +
			"must point at the path to write the edited file to.\n\n" +
			"Edits go through pkg/lvmeta and are guarded by the Tier 2 safety gate\n" +
			"(codec-level validation, version-range enforcement, structural round-\n" +
			"trip check). After the file is written, it is re-parsed and validated\n" +
			"one more time; any structural errors abort the command.\n\n" +
			"--unsafe is reserved for Tier 3 raw patching, which is not implemented\n" +
			"yet; passing it currently returns an error.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			descSet := cmd.Flags().Changed("description")
			nameSet := cmd.Flags().Changed("name")

			outPath := a.v.GetString("out")
			if outPath == "" {
				return fmt.Errorf("set-meta requires --out")
			}
			if !descSet && !nameSet {
				return fmt.Errorf("set-meta requires at least one of --description or --name")
			}
			if unsafe {
				return fmt.Errorf("--unsafe (Tier 3 raw patching) is not implemented yet")
			}

			strict := a.v.GetBool("strict")
			file, err := lvrsrc.Open(args[0], lvrsrc.OpenOptions{Strict: strict})
			if err != nil {
				return err
			}

			mut := lvmeta.Mutator{Strict: strict}
			if descSet {
				if err := mut.SetDescription(file, description); err != nil {
					return fmt.Errorf("set-meta description: %w", err)
				}
			}
			if nameSet {
				if err := mut.SetName(file, name); err != nil {
					return fmt.Errorf("set-meta name: %w", err)
				}
			}

			if err := file.WriteToFile(outPath); err != nil {
				return fmt.Errorf("write output: %w", err)
			}

			if err := postWriteValidate(outPath, strict); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "set the VI description (maps to the STRG resource)")
	cmd.Flags().StringVar(&name, "name", "", "set the VI name (LVSR section name)")
	cmd.Flags().BoolVar(&unsafe, "unsafe", false, "enable Tier 3 raw patching (reserved, not yet implemented)")
	return cmd
}

// postWriteValidate re-opens the file at path and fails when the
// structural validator reports any severity-error issue. Warnings are
// tolerated at the CLI boundary; callers who need strict warning
// policies can pass --strict, which routes through the Mutator's
// strict-mode codec-level gate on the pre-write path.
func postWriteValidate(path string, strict bool) error {
	f, err := lvrsrc.Open(path, lvrsrc.OpenOptions{Strict: strict})
	if err != nil {
		return fmt.Errorf("post-write validate: re-parse %q: %w", path, err)
	}
	for _, iss := range f.Validate() {
		if iss.Severity == lvrsrc.SeverityError {
			return fmt.Errorf("post-write validate: %s: %s", iss.Code, iss.Message)
		}
	}
	return nil
}
