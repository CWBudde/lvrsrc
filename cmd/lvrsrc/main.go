package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/example/lvrsrc/pkg/lvrsrc"
)

type cliApp struct {
	v      *viper.Viper
	stdout io.Writer
	stderr io.Writer
}

func main() {
	cmd := newRootCmd(os.Stdout, os.Stderr)
	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd(stdout, stderr io.Writer) *cobra.Command {
	app := &cliApp{
		v:      viper.New(),
		stdout: stdout,
		stderr: stderr,
	}

	app.v.SetEnvPrefix("LVRSRC")
	app.v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	app.v.AutomaticEnv()
	app.v.SetDefault("format", "text")
	app.v.SetDefault("strict", false)
	app.v.SetDefault("log-level", "info")
	app.v.SetDefault("out", "")
	app.v.SetDefault("config", "")

	rootCmd := &cobra.Command{
		Use:           "lvrsrc",
		Short:         "Inspect LabVIEW RSRC/VI files",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return app.initConfig(cmd)
		},
	}

	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	flags := rootCmd.PersistentFlags()
	flags.String("config", "", "config file path")
	flags.String("format", "text", "output format")
	flags.Bool("strict", false, "enable strict parsing")
	flags.String("log-level", "info", "log verbosity")
	flags.String("out", "", "write command output to file")

	_ = app.v.BindPFlag("config", flags.Lookup("config"))
	_ = app.v.BindPFlag("format", flags.Lookup("format"))
	_ = app.v.BindPFlag("strict", flags.Lookup("strict"))
	_ = app.v.BindPFlag("log-level", flags.Lookup("log-level"))
	_ = app.v.BindPFlag("out", flags.Lookup("out"))

	rootCmd.AddCommand(
		app.newInspectCmd(),
		app.newDumpCmd(),
		app.newListResourcesCmd(),
	)

	return rootCmd
}

func (a *cliApp) initConfig(cmd *cobra.Command) error {
	configPath := a.v.GetString("config")
	if configPath == "" {
		return nil
	}

	a.v.SetConfigFile(configPath)
	if err := a.v.ReadInConfig(); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	// Rebind command flags after config load so explicit CLI flags still win.
	flags := cmd.Root().PersistentFlags()
	_ = a.v.BindPFlag("config", flags.Lookup("config"))
	_ = a.v.BindPFlag("format", flags.Lookup("format"))
	_ = a.v.BindPFlag("strict", flags.Lookup("strict"))
	_ = a.v.BindPFlag("log-level", flags.Lookup("log-level"))
	_ = a.v.BindPFlag("out", flags.Lookup("out"))
	return nil
}

func (a *cliApp) newInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <file>",
		Short: "Inspect RSRC container structure",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := lvrsrc.Open(args[0], lvrsrc.OpenOptions{Strict: a.v.GetBool("strict")})
			if err != nil {
				return err
			}

			w, closeFn, err := a.outputWriter(cmd)
			if err != nil {
				return err
			}
			defer closeFn()

			return a.writeInspect(w, file)
		},
	}
}

func (a *cliApp) newDumpCmd() *cobra.Command {
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "dump <file>",
		Short: "Dump parsed RSRC container data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := lvrsrc.Open(args[0], lvrsrc.OpenOptions{Strict: a.v.GetBool("strict")})
			if err != nil {
				return err
			}

			w, closeFn, err := a.outputWriter(cmd)
			if err != nil {
				return err
			}
			defer closeFn()

			if jsonFlag || strings.EqualFold(a.v.GetString("format"), "json") {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(file)
			}

			return a.writeDumpText(w, file)
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "emit JSON output")
	return cmd
}

func (a *cliApp) newListResourcesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-resources <file>",
		Short: "List resources in compact tabular form",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := lvrsrc.Open(args[0], lvrsrc.OpenOptions{Strict: a.v.GetBool("strict")})
			if err != nil {
				return err
			}

			w, closeFn, err := a.outputWriter(cmd)
			if err != nil {
				return err
			}
			defer closeFn()

			return a.writeResources(w, file)
		},
	}
}

func (a *cliApp) outputWriter(cmd *cobra.Command) (io.Writer, func(), error) {
	outPath := a.v.GetString("out")
	if outPath == "" {
		return cmd.OutOrStdout(), func() {}, nil
	}

	f, err := os.Create(outPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open output file: %w", err)
	}
	return f, func() { _ = f.Close() }, nil
}

func (a *cliApp) writeInspect(w io.Writer, file *lvrsrc.File) error {
	if _, err := fmt.Fprintf(w, "Kind: %s\n", kindLabel(file.Kind)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Format Version: %d\n", file.Header.FormatVersion); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Type: %s\n", file.Header.Type); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Creator: %s\n", file.Header.Creator); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Data: offset=%d size=%d\n", file.Header.DataOffset, file.Header.DataSize); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Info: offset=%d size=%d\n", file.Header.InfoOffset, file.Header.InfoSize); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Blocks:"); err != nil {
		return err
	}
	for _, block := range file.Blocks {
		if _, err := fmt.Fprintf(w, "- %s sections=%d\n", block.Type, len(block.Sections)); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, "Warnings: none")
	return err
}

func (a *cliApp) writeDumpText(w io.Writer, file *lvrsrc.File) error {
	if err := a.writeInspect(w, file); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "Names: %d\nResources: %d\n", len(file.Names), len(file.Resources()))
	return err
}

func (a *cliApp) writeResources(w io.Writer, file *lvrsrc.File) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "TYPE\tID\tNAME\tSIZE"); err != nil {
		return err
	}
	for _, resource := range file.Resources() {
		if _, err := fmt.Fprintf(tw, "%s\t%d\t%s\t%d\n", resource.Type, resource.ID, resource.Name, resource.Size); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func kindLabel(kind lvrsrc.FileKind) string {
	switch kind {
	case lvrsrc.FileKindVI:
		return "VI"
	case lvrsrc.FileKindControl:
		return "Control"
	case lvrsrc.FileKindTemplate:
		return "Template"
	case lvrsrc.FileKindLibrary:
		return "Library"
	default:
		return "Unknown"
	}
}
