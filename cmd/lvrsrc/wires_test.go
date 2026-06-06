package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWiresCommandTextDecodesLeftwardChain(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs([]string{"wires", fixturePath(t, "Numeric42_left_auto_8px_up.vi")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{
		"Block-diagram wire chunks: 1",
		"raw=06 08 00 01 01 00 10 10 9c 18",
		"mode=auto-chain",
		"leftward-chain: up=true verticalPixels=8 horizontalSeed=0x9c",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("wires text output missing %q\n--- got ---\n%s", want, out)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestWiresCommandJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	cmd := newRootCmd(stdout, new(bytes.Buffer))
	cmd.SetArgs([]string{"wires", "--json", fixturePath(t, "Numeric42_8px_down.vi")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var report wireReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, stdout.String())
	}
	if len(report.Wires) != 1 {
		t.Fatalf("Wires = %d, want 1", len(report.Wires))
	}
	w := report.Wires[0]
	if w.Raw != "04 08 00 00 41 08" {
		t.Errorf("Raw = %q, want 04 08 00 00 41 08", w.Raw)
	}
	if w.ChainAuto == nil || w.ChainAuto.YStep != 8 || w.ChainAuto.SourceAnchorX != 65 {
		t.Errorf("ChainAuto = %+v, want yStep=8 sourceAnchorX=65", w.ChainAuto)
	}
}

func TestWiresCommandTreeEndpoints(t *testing.T) {
	stdout := new(bytes.Buffer)
	cmd := newRootCmd(stdout, new(bytes.Buffer))
	cmd.SetArgs([]string{"wires", fixturePath(t, "Numeric42ThreeIndicatorsY.vi")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "mode=tree") {
		t.Errorf("expected tree mode, got:\n%s", out)
	}
	if !strings.Contains(out, "tree-endpoints:") {
		t.Errorf("expected tree-endpoints line, got:\n%s", out)
	}
}

func TestWiresCommandNoBlockDiagram(t *testing.T) {
	stdout := new(bytes.Buffer)
	cmd := newRootCmd(stdout, new(bytes.Buffer))
	cmd.SetArgs([]string{"wires", fixturePath(t, "action.ctl")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Block-diagram wire chunks: 0") {
		t.Errorf("expected zero wire chunks for control, got:\n%s", stdout.String())
	}
}
