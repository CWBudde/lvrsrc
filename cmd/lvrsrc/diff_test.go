package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/pkg/lvdiff"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestDiffCommandIdenticalFilesExitZero(t *testing.T) {
	path := fixturePath(t, "config-data.ctl")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"diff", path, path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "identical") {
		t.Fatalf("expected 'identical' in output, got %q", stdout.String())
	}
}

func TestDiffCommandDifferingFilesExitOne(t *testing.T) {
	aPath := fixturePath(t, "config-data.ctl")
	bPath := writeMutatedFixture(t)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"diff", aPath, bPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	var exitErr *exitCodeError
	if !asExitCodeError(err, &exitErr) {
		t.Fatalf("Execute() error = %T, want *exitCodeError", err)
	}
	if got, want := exitErr.Code(), 1; got != want {
		t.Fatalf("exit code = %d, want %d", got, want)
	}

	out := stdout.String()
	if !strings.Contains(out, "~") && !strings.Contains(out, "+") && !strings.Contains(out, "-") {
		t.Fatalf("expected unified-diff-style markers in output, got %q", out)
	}
}

func TestDiffCommandJSON(t *testing.T) {
	aPath := fixturePath(t, "config-data.ctl")
	bPath := writeMutatedFixture(t)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"diff", aPath, bPath, "--json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	var exitErr *exitCodeError
	if !asExitCodeError(err, &exitErr) {
		t.Fatalf("Execute() error = %T, want *exitCodeError", err)
	}
	if got, want := exitErr.Code(), 1; got != want {
		t.Fatalf("exit code = %d, want %d", got, want)
	}

	var got struct {
		Identical bool `json:"identical"`
		ExitCode  int  `json:"exitCode"`
		Summary   struct {
			Added    int `json:"added"`
			Removed  int `json:"removed"`
			Modified int `json:"modified"`
		} `json:"summary"`
		Items []struct {
			Kind     string `json:"kind"`
			Category string `json:"category"`
			Path     string `json:"path"`
		} `json:"items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\noutput=%s", err, stdout.String())
	}
	if got.Identical {
		t.Fatalf("identical = true, want false")
	}
	if got.ExitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", got.ExitCode)
	}
	if len(got.Items) == 0 {
		t.Fatal("expected at least one diff item")
	}
	if got.Summary.Added+got.Summary.Removed+got.Summary.Modified != len(got.Items) {
		t.Fatalf("summary totals (%d) do not match items (%d)", got.Summary.Added+got.Summary.Removed+got.Summary.Modified, len(got.Items))
	}
}

func TestDiffCommandJSONIdenticalExitZero(t *testing.T) {
	path := fixturePath(t, "config-data.ctl")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"diff", path, path, "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got struct {
		Identical bool `json:"identical"`
		ExitCode  int  `json:"exitCode"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\noutput=%s", err, stdout.String())
	}
	if !got.Identical {
		t.Fatalf("identical = false, want true")
	}
	if got.ExitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", got.ExitCode)
	}
}

// writeMutatedFixture round-trips config-data.ctl through the writer after
// mutating one section's payload so the resulting file differs structurally
// from the original corpus fixture while still being a parseable RSRC file.
func writeMutatedFixture(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile(fixturePath(t, "config-data.ctl"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	f, err := lvrsrc.Parse(data, lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Bump the format version so the header diff fires — a guaranteed change
	// independent of writer preservation details.
	f.Header.FormatVersion++
	f.SecondaryHeader.FormatVersion = f.Header.FormatVersion

	path := filepath.Join(t.TempDir(), "mutated.ctl")
	if err := f.WriteToFile(path); err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}
	return path
}

// TestKindOrderCoversAllCases drives each switch arm of kindOrder so
// the diff helper's relative-ordering table doesn't sit unexercised.
func TestKindOrderCoversAllCases(t *testing.T) {
	cases := []struct {
		k    string
		want int
	}{
		{"header", 0},
		{"block", 1},
		{"section", 2},
		{"decoded", 3},
		{"unknown-kind", 4},
	}
	for _, tc := range cases {
		// Cast through lvdiff.Kind via the package-internal helper.
		if got := kindOrder(stringToKind(tc.k)); got != tc.want {
			t.Errorf("kindOrder(%q) = %d, want %d", tc.k, got, tc.want)
		}
		if got := kindHeading(stringToKind(tc.k)); got == "" {
			t.Errorf("kindHeading(%q) returned empty", tc.k)
		}
	}
}

// TestFormatValueCoversBranches walks the formatValue type switch.
func TestFormatValueCoversBranches(t *testing.T) {
	if got := formatValue(nil); got != "" {
		t.Errorf("formatValue(nil) = %q, want empty", got)
	}
	if got := formatValue("hi"); got != `"hi"` {
		t.Errorf(`formatValue("hi") = %q, want %q`, got, `"hi"`)
	}
	if got := formatValue(42); got != "42" {
		t.Errorf("formatValue(42) = %q, want 42", got)
	}
}

func stringToKind(s string) lvdiffKind { return lvdiffKind(s) }

type lvdiffKind = lvdiff.Kind
