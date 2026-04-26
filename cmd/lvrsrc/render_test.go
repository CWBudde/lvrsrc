package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderCommandFrontPanelSVGToStdout(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"render", fixturePath(t, "format-string.vi"), "--view", "front-panel"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, `<svg`) {
		t.Fatalf("render output missing <svg: %q", out)
	}
	if !strings.Contains(out, `LabVIEW front-panel render`) {
		t.Fatalf("render output missing front-panel title: %q", out)
	}
	if !strings.Contains(out, `lvrsrc-node`) {
		t.Fatalf("render output missing scene node classes: %q", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRenderCommandWritesSVGToOutFile(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "render.svg")
	cmd := newRootCmd(new(bytes.Buffer), new(bytes.Buffer))
	cmd.SetArgs([]string{
		"--out", outPath,
		"render", fixturePath(t, "format-string.vi"),
		"--view", "block-diagram",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile(out) error = %v", err)
	}
	if !strings.Contains(string(written), `<svg`) {
		t.Fatalf("written SVG missing <svg: %q", string(written))
	}
	if !strings.Contains(string(written), `LabVIEW block-diagram render`) {
		t.Fatalf("written SVG missing block-diagram title: %q", string(written))
	}
}

func TestRenderCommandRejectsUnsupportedFormat(t *testing.T) {
	cmd := newRootCmd(new(bytes.Buffer), new(bytes.Buffer))
	cmd.SetArgs([]string{
		"render", fixturePath(t, "format-string.vi"),
		"--view", "front-panel",
		"--format", "png",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want unsupported format error")
	}
	if !strings.Contains(err.Error(), "unsupported render format") {
		t.Fatalf("err = %q, want unsupported-format message", err.Error())
	}
}
