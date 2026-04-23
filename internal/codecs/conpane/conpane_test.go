package conpane

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/internal/validate"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func TestCapabilities(t *testing.T) {
	cases := []struct {
		name   string
		codec  codecs.ResourceCodec
		fourCC string
	}{
		{"CONP", PointerCodec{}, "CONP"},
		{"CPC2", CountCodec{}, "CPC2"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.codec.Capability()
			if c.FourCC != tc.fourCC {
				t.Fatalf("FourCC = %q, want %q", c.FourCC, tc.fourCC)
			}
			if c.Safety != codecs.SafetyTier2 {
				t.Fatalf("Safety = %v, want SafetyTier2", c.Safety)
			}
		})
	}
}

func TestDecodeEncodeRoundTrip(t *testing.T) {
	cases := []struct {
		name       string
		codec      codecs.ResourceCodec
		payload    []byte
		wantFourCC string
		wantValue  uint16
	}{
		{"CONP", PointerCodec{}, []byte{0x00, 0x01}, "CONP", 1},
		{"CPC2", CountCodec{}, []byte{0x00, 0x04}, "CPC2", 4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.codec.Decode(codecs.Context{}, tc.payload)
			if err != nil {
				t.Fatalf("Decode error = %v", err)
			}
			v := got.(Value)
			if v.FourCC != tc.wantFourCC {
				t.Fatalf("FourCC = %q, want %q", v.FourCC, tc.wantFourCC)
			}
			if v.Value != tc.wantValue {
				t.Fatalf("Value = %d, want %d", v.Value, tc.wantValue)
			}

			back, err := tc.codec.Encode(codecs.Context{}, v)
			if err != nil {
				t.Fatalf("Encode error = %v", err)
			}
			if !bytes.Equal(back, tc.payload) {
				t.Fatalf("round-trip mismatch: got %x want %x", back, tc.payload)
			}
		})
	}
}

func TestEncodeAcceptsPointer(t *testing.T) {
	v := Value{FourCC: "CPC2", Value: 3}
	out, err := CountCodec{}.Encode(codecs.Context{}, &v)
	if err != nil {
		t.Fatalf("Encode(*Value) error = %v", err)
	}
	if !bytes.Equal(out, []byte{0x00, 0x03}) {
		t.Fatalf("Encode(*Value) = %x, want 0003", out)
	}
}

func TestDecodeRejectsShortPayload(t *testing.T) {
	_, err := PointerCodec{}.Decode(codecs.Context{}, []byte{0x00})
	if err == nil {
		t.Fatalf("Decode of short payload did not error")
	}
}

func TestEncodeRejectsWrongType(t *testing.T) {
	_, err := PointerCodec{}.Encode(codecs.Context{}, 42)
	if err == nil {
		t.Fatalf("Encode of int did not error")
	}
}

func TestEncodeRejectsWrongFourCC(t *testing.T) {
	_, err := PointerCodec{}.Encode(codecs.Context{}, Value{FourCC: "CPC2", Value: 1})
	if err == nil {
		t.Fatalf("Encode with wrong FourCC did not error")
	}
}

func TestValidateReportsWrongSize(t *testing.T) {
	issues := PointerCodec{}.Validate(codecs.Context{}, []byte{0x00})
	assertHasCode(t, issues, "conp.payload.size", validate.SeverityError)

	issues = CountCodec{}.Validate(codecs.Context{}, []byte{0x00, 0x01, 0x02})
	assertHasCode(t, issues, "cpc2.payload.size", validate.SeverityError)
}

func TestCorpusRoundTrip(t *testing.T) {
	type testCase struct {
		fourCC string
		codec  codecs.ResourceCodec
	}
	cases := []testCase{
		{fourCC: "CONP", codec: PointerCodec{}},
		{fourCC: "CPC2", codec: CountCodec{}},
	}

	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Fatalf("read corpus dir: %v", err)
	}

	total := 0
	for _, tc := range cases {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := filepath.Ext(e.Name())
			if ext != ".vi" && ext != ".ctl" {
				continue
			}
			path := filepath.Join(corpus.Dir(), e.Name())
			f, err := lvrsrc.Open(path, lvrsrc.OpenOptions{})
			if err != nil {
				t.Fatalf("open %s: %v", e.Name(), err)
			}
			for _, block := range f.Blocks {
				if block.Type != tc.fourCC {
					continue
				}
				for _, section := range block.Sections {
					total++
					got, err := tc.codec.Decode(codecs.Context{}, section.Payload)
					if err != nil {
						t.Fatalf("%s %s id=%d Decode: %v", e.Name(), tc.fourCC, section.Index, err)
					}
					back, err := tc.codec.Encode(codecs.Context{}, got)
					if err != nil {
						t.Fatalf("%s %s id=%d Encode: %v", e.Name(), tc.fourCC, section.Index, err)
					}
					if !bytes.Equal(back, section.Payload) {
						t.Fatalf("%s %s id=%d round-trip mismatch", e.Name(), tc.fourCC, section.Index)
					}
					if issues := tc.codec.Validate(codecs.Context{}, section.Payload); len(issues) != 0 {
						t.Fatalf("%s %s id=%d Validate issues: %+v", e.Name(), tc.fourCC, section.Index, issues)
					}
				}
			}
		}
	}

	if total == 0 {
		t.Fatalf("no connector-pane sections found in corpus; test is not exercising anything")
	}
	t.Logf("round-tripped %d connector-pane sections", total)
}

func assertHasCode(t *testing.T, issues []validate.Issue, code string, sev validate.Severity) {
	t.Helper()
	for _, i := range issues {
		if i.Code == code && i.Severity == sev {
			return
		}
	}
	t.Fatalf("expected issue code=%q severity=%q in: %+v", code, sev, issues)
}
