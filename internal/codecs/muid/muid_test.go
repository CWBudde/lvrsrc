package muid

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

func TestCapability(t *testing.T) {
	c := Codec{}.Capability()
	if c.FourCC != FourCC {
		t.Fatalf("FourCC = %q, want %q", c.FourCC, FourCC)
	}
	if c.Safety != codecs.SafetyTier1 {
		t.Fatalf("Safety = %v, want SafetyTier1", c.Safety)
	}
}

func TestDecodeReadsBigEndianUint32(t *testing.T) {
	got, err := Codec{}.Decode(codecs.Context{}, []byte{0xCA, 0xFE, 0xBA, 0xBE})
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	v, ok := got.(Value)
	if !ok {
		t.Fatalf("Decode returned %T, want Value", got)
	}
	if v.UID != 0xCAFEBABE {
		t.Errorf("UID = %#x, want 0xCAFEBABE", v.UID)
	}
}

func TestDecodeRejectsWrongSize(t *testing.T) {
	for _, size := range []int{0, 1, 2, 3, 5, 8} {
		if _, err := (Codec{}).Decode(codecs.Context{}, make([]byte, size)); err == nil {
			t.Errorf("Decode(%d-byte payload) = nil, want error", size)
		}
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	original := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	raw, err := Codec{}.Decode(codecs.Context{}, original)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	back, err := Codec{}.Encode(codecs.Context{}, raw)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("Encode != original: %x vs %x", back, original)
	}
}

func TestValidate(t *testing.T) {
	if issues := (Codec{}).Validate(codecs.Context{}, []byte{0, 0, 0, 0}); len(issues) != 0 {
		t.Errorf("Validate(4 bytes) issues = %+v, want none", issues)
	}
	issues := Codec{}.Validate(codecs.Context{}, []byte{0, 0, 0})
	if len(issues) == 0 {
		t.Fatal("Validate(3 bytes) returned no issues")
	}
	if issues[0].Severity != validate.SeverityError {
		t.Errorf("severity = %v, want error", issues[0].Severity)
	}
}

func TestCorpusRoundTrip(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Skipf("corpus directory not present: %v", err)
	}
	total := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".vi" && ext != ".ctl" && ext != ".vit" {
			continue
		}
		path := filepath.Join(corpus.Dir(), e.Name())
		f, err := lvrsrc.Open(path, lvrsrc.OpenOptions{})
		if err != nil {
			t.Fatalf("open %s: %v", e.Name(), err)
		}
		for _, block := range f.Blocks {
			if block.Type != string(FourCC) {
				continue
			}
			for _, section := range block.Sections {
				total++
				v, err := Codec{}.Decode(codecs.Context{}, section.Payload)
				if err != nil {
					t.Fatalf("%s %s id=%d Decode: %v", e.Name(), FourCC, section.Index, err)
				}
				back, err := Codec{}.Encode(codecs.Context{}, v)
				if err != nil {
					t.Fatalf("%s %s id=%d Encode: %v", e.Name(), FourCC, section.Index, err)
				}
				if !bytes.Equal(back, section.Payload) {
					t.Fatalf("%s %s id=%d round-trip mismatch", e.Name(), FourCC, section.Index)
				}
			}
		}
	}
	if total == 0 {
		t.Skip("no MUID sections found in corpus")
	}
	t.Logf("exercised %d %s section(s)", total, FourCC)
}
