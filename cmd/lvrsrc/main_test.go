package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}
