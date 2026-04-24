package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/internal/rsrcwire"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestInspectCommand(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"inspect", fixturePath(t, "config-data.ctl")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Kind: Control") {
		t.Fatalf("inspect output missing kind: %q", out)
	}
	if !strings.Contains(out, "Format Version: 3") {
		t.Fatalf("inspect output missing format version: %q", out)
	}
	if !strings.Contains(out, "Type: LVCC") {
		t.Fatalf("inspect output missing type: %q", out)
	}
	if !strings.Contains(out, "Blocks:") {
		t.Fatalf("inspect output missing block list: %q", out)
	}
	if !strings.Contains(out, "Warnings: none") {
		t.Fatalf("inspect output missing warnings line: %q", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestInspectCommandLibraryFixture(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"inspect", llbFixturePath(t, "empty-libfile.llb")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Kind: Library") {
		t.Fatalf("inspect output missing library kind: %q", out)
	}
	if !strings.Contains(out, "Type: LVAR") {
		t.Fatalf("inspect output missing library type: %q", out)
	}
	if !strings.Contains(out, "- ADir sections=1") {
		t.Fatalf("inspect output missing ADir block: %q", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestDumpJSONCommand(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"dump", fixturePath(t, "config-data.ctl"), "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got struct {
		Kind   string `json:"kind"`
		Header struct {
			Type string `json:"type"`
		} `json:"header"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\noutput=%s", err, stdout.String())
	}
	if got.Kind != "ctl" {
		t.Fatalf("kind = %q, want %q", got.Kind, "ctl")
	}
	if got.Header.Type != "LVCC" {
		t.Fatalf("header.type = %q, want %q", got.Header.Type, "LVCC")
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestListResourcesCommand(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"list-resources", fixturePath(t, "config-data.ctl")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "TYPE") || !strings.Contains(out, "ID") || !strings.Contains(out, "SIZE") {
		t.Fatalf("list-resources output missing table header: %q", out)
	}
	if !strings.Contains(out, "LIBN") {
		t.Fatalf("list-resources output missing LIBN entry: %q", out)
	}
	if !strings.Contains(out, "Config Data.ctl") {
		t.Fatalf("list-resources output missing named resource: %q", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestValidateCommandValidFile(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"validate", fixturePath(t, "config-data.ctl")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "VALID") {
		t.Fatalf("validate output missing VALID status: %q", out)
	}
	if !strings.Contains(out, "Warnings: 0") || !strings.Contains(out, "Errors: 0") {
		t.Fatalf("validate output missing summary: %q", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestValidateCommandJSONWarningExitCode(t *testing.T) {
	path := writeWarningFixture(t)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"validate", path, "--json"})

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
		Valid    bool `json:"valid"`
		ExitCode int  `json:"exitCode"`
		Summary  struct {
			Warnings int `json:"warnings"`
			Errors   int `json:"errors"`
		} `json:"summary"`
		Issues []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\noutput=%s", err, stdout.String())
	}
	if got.Valid {
		t.Fatalf("valid = true, want false")
	}
	if got.ExitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", got.ExitCode)
	}
	if got.Summary.Warnings != 1 || got.Summary.Errors != 0 {
		t.Fatalf("summary = %+v, want warnings=1 errors=0", got.Summary)
	}
	if len(got.Issues) == 0 || got.Issues[0].Code != "section.size.zero" {
		t.Fatalf("issues = %+v, want section.size.zero", got.Issues)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestValidateCommandErrorExitCode(t *testing.T) {
	path := writeErrorFixture(t)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"validate", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}

	var exitErr *exitCodeError
	if !asExitCodeError(err, &exitErr) {
		t.Fatalf("Execute() error = %T, want *exitCodeError", err)
	}
	if got, want := exitErr.Code(), 2; got != want {
		t.Fatalf("exit code = %d, want %d", got, want)
	}

	out := stdout.String()
	if !strings.Contains(out, "ERROR") {
		t.Fatalf("validate output missing ERROR status: %q", out)
	}
	if !strings.Contains(out, "header.mismatch") {
		t.Fatalf("validate output missing header.mismatch: %q", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestConfigEnvAndOutFlag(t *testing.T) {
	t.Setenv("LVRSRC_FORMAT", "json")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "lvrsrc.yaml")
	if err := os.WriteFile(configPath, []byte("log-level: debug\nstrict: true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	outPath := filepath.Join(tempDir, "dump.json")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{
		"--config", configPath,
		"--out", outPath,
		"dump", fixturePath(t, "config-data.ctl"),
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout should be empty when --out is set, got %q", stdout.String())
	}

	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile(out) error = %v", err)
	}
	var got struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(written, &got); err != nil {
		t.Fatalf("json.Unmarshal(out) error = %v\noutput=%s", err, string(written))
	}
	if got.Kind != "ctl" {
		t.Fatalf("kind = %q, want %q", got.Kind, "ctl")
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRewriteCommandRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "rewritten.ctl")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{
		"rewrite",
		fixturePath(t, "config-data.ctl"),
		"--out", outPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout should be empty when --out is set, got %q", stdout.String())
	}

	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile(out) error = %v", err)
	}

	f, err := lvrsrc.Parse(written, lvrsrc.OpenOptions{Strict: true})
	if err != nil {
		t.Fatalf("Parse(rewritten) error = %v", err)
	}

	if issues := f.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %+v, want none", issues)
	}

	if got, want := f.Kind, lvrsrc.FileKindControl; got != want {
		t.Fatalf("Kind = %v, want %v", got, want)
	}
	if got, want := len(f.Resources()), 28; got != want {
		t.Fatalf("len(Resources()) = %d, want %d", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRewriteCommandLibraryRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "rewritten.llb")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{
		"rewrite",
		llbFixturePath(t, "empty-libfile.llb"),
		"--out", outPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile(out) error = %v", err)
	}

	f, err := lvrsrc.Parse(written, lvrsrc.OpenOptions{Strict: true})
	if err != nil {
		t.Fatalf("Parse(rewritten) error = %v", err)
	}

	if issues := f.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %+v, want none", issues)
	}
	if got, want := f.Kind, lvrsrc.FileKindLibrary; got != want {
		t.Fatalf("Kind = %v, want %v", got, want)
	}
	if got, want := len(f.Resources()), 15; got != want {
		t.Fatalf("len(Resources()) = %d, want %d", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRewriteCommandCanonicalFlagNotImplemented(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "rewritten.ctl")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{
		"rewrite",
		fixturePath(t, "config-data.ctl"),
		"--out", outPath,
		"--canonical",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "canonical") {
		t.Fatalf("Execute() error = %q, want canonical message", err)
	}
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	return corpus.Path(name)
}

func llbFixturePath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(corpus.Dir(), "..", "llb", name)
}

func writeWarningFixture(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile(fixturePath(t, "config-data.ctl"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	f, err := rsrcwire.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	payloadSizeOff := int(f.Header.DataOffset + f.Blocks[0].Sections[0].DataOffset)
	binary.BigEndian.PutUint32(data[payloadSizeOff:], 0)

	path := filepath.Join(t.TempDir(), "warning.ctl")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func writeErrorFixture(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile(fixturePath(t, "config-data.ctl"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	infoOffset := binary.BigEndian.Uint32(data[16:20])
	data[int(infoOffset)+8] ^= 0x01

	path := filepath.Join(t.TempDir(), "error.ctl")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
