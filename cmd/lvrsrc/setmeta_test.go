package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) = %v", path, err)
	}
	return data
}

// runSetMeta executes the set-meta command and returns (stdout, stderr, err).
// Callers pass only the subcommand args; the binary name is added here.
func runSetMeta(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs(append([]string{"set-meta"}, args...))
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func readSTRGText(t *testing.T, f *lvrsrc.File) string {
	t.Helper()
	for _, b := range f.Blocks {
		if b.Type != "STRG" {
			continue
		}
		if len(b.Sections) == 0 {
			t.Fatalf("STRG block has no sections")
		}
		p := b.Sections[0].Payload
		if len(p) < 4 {
			t.Fatalf("STRG payload too short: %d", len(p))
		}
		size := binary.BigEndian.Uint32(p[:4])
		if 4+int(size) > len(p) {
			t.Fatalf("STRG size overruns payload")
		}
		return string(p[4 : 4+size])
	}
	return ""
}

func lvsrName(t *testing.T, f *lvrsrc.File) string {
	t.Helper()
	for _, b := range f.Blocks {
		if b.Type == "LVSR" && len(b.Sections) > 0 {
			return b.Sections[0].Name
		}
	}
	return ""
}

func TestSetMetaCommandSetsDescription(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "edited.vi")

	_, stderr, err := runSetMeta(t,
		fixturePath(t, "format-string.vi"),
		"--description", "Edited via set-meta.",
		"--out", outPath,
	)
	if err != nil {
		t.Fatalf("set-meta err = %v (stderr=%q)", err, stderr)
	}

	f, err := lvrsrc.Parse(mustReadFile(t, outPath), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse(written) err = %v", err)
	}
	if got := readSTRGText(t, f); got != "Edited via set-meta." {
		t.Fatalf("STRG text = %q, want %q", got, "Edited via set-meta.")
	}
}

func TestSetMetaCommandSetsName(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "renamed.vi")

	_, stderr, err := runSetMeta(t,
		fixturePath(t, "format-string.vi"),
		"--name", "via-set-meta.vi",
		"--out", outPath,
	)
	if err != nil {
		t.Fatalf("set-meta err = %v (stderr=%q)", err, stderr)
	}

	f, err := lvrsrc.Parse(mustReadFile(t, outPath), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse(written) err = %v", err)
	}
	if got := lvsrName(t, f); got != "via-set-meta.vi" {
		t.Fatalf("LVSR Name = %q, want %q", got, "via-set-meta.vi")
	}
}

func TestSetMetaCommandSetsBothFlagsInOneCall(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "both.vi")

	_, _, err := runSetMeta(t,
		fixturePath(t, "format-string.vi"),
		"--description", "double edit",
		"--name", "both.vi",
		"--out", outPath,
	)
	if err != nil {
		t.Fatalf("set-meta err = %v", err)
	}

	f, err := lvrsrc.Parse(mustReadFile(t, outPath), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	if got := readSTRGText(t, f); got != "double edit" {
		t.Fatalf("STRG = %q, want double edit", got)
	}
	if got := lvsrName(t, f); got != "both.vi" {
		t.Fatalf("LVSR Name = %q, want both.vi", got)
	}
}

func TestSetMetaCommandCreatesNewSTRGWhenAbsent(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "added-strg.vi")

	_, _, err := runSetMeta(t,
		fixturePath(t, "is-float.vi"),
		"--description", "freshly inserted",
		"--out", outPath,
	)
	if err != nil {
		t.Fatalf("set-meta err = %v", err)
	}

	f, err := lvrsrc.Parse(mustReadFile(t, outPath), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	if got := readSTRGText(t, f); got != "freshly inserted" {
		t.Fatalf("STRG = %q, want freshly inserted", got)
	}
}

func TestSetMetaCommandEmptyDescriptionAllowed(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "empty-desc.vi")

	// Passing an explicit empty description must be respected (not
	// confused with "flag not provided"), producing a zero-length STRG.
	_, _, err := runSetMeta(t,
		fixturePath(t, "format-string.vi"),
		"--description", "",
		"--out", outPath,
	)
	if err != nil {
		t.Fatalf("set-meta empty description err = %v", err)
	}

	f, err := lvrsrc.Parse(mustReadFile(t, outPath), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	if got := readSTRGText(t, f); got != "" {
		t.Fatalf("STRG = %q, want empty", got)
	}
}

func TestSetMetaCommandRequiresOutFlag(t *testing.T) {
	_, _, err := runSetMeta(t,
		fixturePath(t, "format-string.vi"),
		"--description", "x",
	)
	if err == nil {
		t.Fatalf("expected error requiring --out, got nil")
	}
	if !strings.Contains(err.Error(), "--out") {
		t.Fatalf("err = %q, want mention of --out", err.Error())
	}
}

func TestSetMetaCommandRequiresAtLeastOneEditFlag(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "never-written.vi")

	_, _, err := runSetMeta(t,
		fixturePath(t, "format-string.vi"),
		"--out", outPath,
	)
	if err == nil {
		t.Fatalf("expected error requiring --description or --name, got nil")
	}
	if !strings.Contains(err.Error(), "description") {
		t.Fatalf("err = %q, want mention of description/name requirement", err.Error())
	}
	// Output file must NOT have been created when the flag check fails up
	// front.
	if _, statErr := os.Stat(outPath); statErr == nil {
		t.Fatalf("output file unexpectedly created: %s", outPath)
	}
}

func TestSetMetaCommandUnsafeFlagRejected(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "never.vi")

	_, _, err := runSetMeta(t,
		fixturePath(t, "format-string.vi"),
		"--description", "whatever",
		"--unsafe",
		"--out", outPath,
	)
	if err == nil {
		t.Fatalf("expected error for --unsafe, got nil")
	}
	if !strings.Contains(err.Error(), "unsafe") && !strings.Contains(err.Error(), "Tier 3") {
		t.Fatalf("err = %q, want mention of unsafe/Tier 3", err.Error())
	}
}

func TestSetMetaCommandPropagatesLvmetaErrorForNameOverflow(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "overflow.vi")

	overflow := strings.Repeat("a", 256) // > 255 → ErrNameTooLong
	_, _, err := runSetMeta(t,
		fixturePath(t, "format-string.vi"),
		"--name", overflow,
		"--out", outPath,
	)
	if err == nil {
		t.Fatalf("expected error for overflow name, got nil")
	}
	if !strings.Contains(err.Error(), "Pascal") && !strings.Contains(err.Error(), "length") {
		t.Fatalf("err = %q, want mention of length/Pascal limit", err.Error())
	}
}

func TestSetMetaCommandPostWriteValidationPassesOnCorpus(t *testing.T) {
	// A happy-path edit on a corpus file must emerge from the post-write
	// validation step without errors. This locks the gate into the test
	// matrix so regressions in the gate surface here rather than in
	// downstream tooling.
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "passes.vi")
	_, stderr, err := runSetMeta(t,
		fixturePath(t, "format-string.vi"),
		"--description", "post-write check",
		"--out", outPath,
	)
	if err != nil {
		t.Fatalf("set-meta err = %v (stderr=%q)", err, stderr)
	}

	f, err := lvrsrc.Parse(mustReadFile(t, outPath), lvrsrc.OpenOptions{Strict: true})
	if err != nil {
		t.Fatalf("strict Parse err = %v", err)
	}
	for _, iss := range f.Validate() {
		if iss.Severity == lvrsrc.SeverityError {
			t.Fatalf("post-write re-parse reported error: %s: %s", iss.Code, iss.Message)
		}
	}
}
