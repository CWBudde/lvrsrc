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

func TestRewriteCommandCanonicalRoundTrip(t *testing.T) {
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

	outPath2 := filepath.Join(tempDir, "rewritten-2.ctl")
	cmd = newRootCmd(new(bytes.Buffer), new(bytes.Buffer))
	cmd.SetArgs([]string{
		"rewrite",
		outPath,
		"--out", outPath2,
		"--canonical",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(second) error = %v", err)
	}

	written2, err := os.ReadFile(outPath2)
	if err != nil {
		t.Fatalf("ReadFile(out2) error = %v", err)
	}

	if !bytes.Equal(written, written2) {
		t.Fatal("canonical rewrite is not stable across repeated rewrites")
	}
}

func TestRewriteCommandCanonicalLibraryRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "rewritten.llb")

	cmd := newRootCmd(new(bytes.Buffer), new(bytes.Buffer))
	cmd.SetArgs([]string{
		"rewrite",
		llbFixturePath(t, "empty-libfile.llb"),
		"--out", outPath,
		"--canonical",
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

	outPath2 := filepath.Join(tempDir, "rewritten-2.llb")
	cmd = newRootCmd(new(bytes.Buffer), new(bytes.Buffer))
	cmd.SetArgs([]string{
		"rewrite",
		outPath,
		"--out", outPath2,
		"--canonical",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(second) error = %v", err)
	}

	written2, err := os.ReadFile(outPath2)
	if err != nil {
		t.Fatalf("ReadFile(out2) error = %v", err)
	}

	if !bytes.Equal(written, written2) {
		t.Fatal("canonical library rewrite is not stable across repeated rewrites")
	}
}

func TestRepairCommandRepairsHeaderMismatch(t *testing.T) {
	path := writeErrorFixture(t)
	outPath := filepath.Join(t.TempDir(), "repaired.ctl")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{
		"repair",
		path,
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
		t.Fatalf("Parse(repaired) error = %v", err)
	}
	if issues := f.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %+v, want none", issues)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRepairCommandRefusesUnresolvedTruncatedNameTable(t *testing.T) {
	path := writeMissingNameFixture(t)
	outPath := filepath.Join(t.TempDir(), "repaired.ctl")

	cmd := newRootCmd(new(bytes.Buffer), new(bytes.Buffer))
	cmd.SetArgs([]string{
		"repair",
		path,
		"--out", outPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "missing section name") {
		t.Fatalf("Execute() error = %q, want missing-name refusal", err)
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

func writeMissingNameFixture(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile(fixturePath(t, "config-data.ctl"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	f, err := rsrcwire.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	blockIndex, sectionIndex, ok := firstNamedSection(t, f)
	if !ok {
		t.Fatal("fixture has no named sections")
	}

	blockInfoPos := int(f.Header.InfoOffset + f.BlockInfoList.BlockInfoOffset)
	sectionPos := blockInfoPos + int(f.Blocks[blockIndex].Offset) + sectionIndex*20
	binary.BigEndian.PutUint32(data[sectionPos+4:], f.Header.InfoSize)

	path := filepath.Join(t.TempDir(), "missing-name.ctl")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func firstNamedSection(t *testing.T, f *rsrcwire.File) (int, int, bool) {
	t.Helper()
	for bi, block := range f.Blocks {
		for si, section := range block.Sections {
			if section.NameOffset != ^uint32(0) {
				return bi, si, true
			}
		}
	}
	return 0, 0, false
}

// TestKindLabelCoversAllFileKinds drives every arm of the kindLabel
// switch.
func TestKindLabelCoversAllFileKinds(t *testing.T) {
	cases := []struct {
		kind lvrsrc.FileKind
		want string
	}{
		{lvrsrc.FileKindVI, "VI"},
		{lvrsrc.FileKindControl, "Control"},
		{lvrsrc.FileKindTemplate, "Template"},
		{lvrsrc.FileKindLibrary, "Library"},
		{lvrsrc.FileKindUnknown, "Unknown"},
	}
	for _, tc := range cases {
		if got := kindLabel(tc.kind); got != tc.want {
			t.Errorf("kindLabel(%v) = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

// TestExitCodeErrorMethods covers Error/Unwrap/Code on nil, empty and
// populated *exitCodeError values.
func TestExitCodeErrorMethods(t *testing.T) {
	var nilErr *exitCodeError
	if got := nilErr.Error(); got != "" {
		t.Errorf("nil.Error() = %q, want empty", got)
	}
	if got := nilErr.Unwrap(); got != nil {
		t.Errorf("nil.Unwrap() = %v, want nil", got)
	}
	if got := nilErr.Code(); got != 1 {
		t.Errorf("nil.Code() = %d, want 1", got)
	}

	empty := &exitCodeError{}
	if got := empty.Error(); got != "" {
		t.Errorf("empty.Error() = %q, want empty", got)
	}
	if got := empty.Code(); got != 1 {
		t.Errorf("empty.Code() = %d, want 1", got)
	}

	wrapped := &exitCodeError{code: 42, err: errStub{"boom"}}
	if got := wrapped.Error(); got != "boom" {
		t.Errorf("wrapped.Error() = %q, want boom", got)
	}
	if got := wrapped.Code(); got != 42 {
		t.Errorf("wrapped.Code() = %d, want 42", got)
	}
	if got := wrapped.Unwrap(); got == nil || got.Error() != "boom" {
		t.Errorf("wrapped.Unwrap() = %v, want boom", got)
	}
}

// TestColorizeOnNonTerminal asserts colorize returns the raw string when
// the writer is not a terminal.
func TestColorizeOnNonTerminal(t *testing.T) {
	var buf bytes.Buffer
	if got := colorize(&buf, "hi", colorRed); got != "hi" {
		t.Errorf("colorize(buffer) = %q, want hi", got)
	}
	if got := isTerminalWriter(&buf); got {
		t.Errorf("isTerminalWriter(*bytes.Buffer) = true, want false")
	}
}

// TestIsTerminalWriterRegularFileFalse covers the Stat-but-not-tty path
// (a regular *os.File backed by a temp file is not a TTY).
func TestIsTerminalWriterRegularFileFalse(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "term-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer tmp.Close()
	if got := isTerminalWriter(tmp); got {
		t.Errorf("isTerminalWriter(regular file) = true, want false")
	}
}

// TestDumpCommandTextOutput exercises writeDumpText (the non-JSON dump
// path used to be uncovered).
func TestDumpCommandTextOutput(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"dump", fixturePath(t, "config-data.ctl")})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Names:") {
		t.Fatalf("dump text output missing Names header: %q", out)
	}
	if !strings.Contains(out, "Resources:") {
		t.Fatalf("dump text output missing Resources header: %q", out)
	}
}

type errStub struct{ msg string }

func (e errStub) Error() string { return e.msg }

// TestValidateCommandWarningTextOutput exercises the warning-status arm
// of writeValidateText (text mode, no --json).
func TestValidateCommandWarningTextOutput(t *testing.T) {
	path := writeWarningFixture(t)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"validate", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want warning exit code")
	}

	out := stdout.String()
	if !strings.Contains(out, "WARNING") {
		t.Fatalf("validate output missing WARNING status: %q", out)
	}
}

// TestRepairCommandRefusesCleanFile exercises the no-op branch in
// newRepairCmd (file is already valid → repair refuses).
func TestRepairCommandRefusesCleanFile(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"repair", fixturePath(t, "config-data.ctl"), "--out", filepath.Join(t.TempDir(), "out.ctl")})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() returned nil, want error for already-clean file")
	}
}
