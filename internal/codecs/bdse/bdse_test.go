package bdse

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

func TestDecodeAndEncodeRoundTrip(t *testing.T) {
	original := []byte{0x00, 0x00, 0x56, 0x78}
	v, err := Codec{}.Decode(codecs.Context{}, original)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	val, ok := v.(Value)
	if !ok {
		t.Fatalf("Decode returned %T, want Value", v)
	}
	if val.Estimate != 0x5678 {
		t.Errorf("Estimate = %#x, want 0x5678", val.Estimate)
	}
	back, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("Encode != original: %x vs %x", back, original)
	}
}

func TestDecodeRejectsWrongSize(t *testing.T) {
	if _, err := (Codec{}).Decode(codecs.Context{}, []byte{1, 2, 3}); err == nil {
		t.Fatal("Decode(3 bytes) returned nil error")
	}
}

func TestValidate(t *testing.T) {
	if issues := (Codec{}).Validate(codecs.Context{}, make([]byte, 4)); len(issues) != 0 {
		t.Errorf("Validate(4 bytes) issues = %+v, want none", issues)
	}
	issues := Codec{}.Validate(codecs.Context{}, make([]byte, 3))
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
		f, err := lvrsrc.Open(filepath.Join(corpus.Dir(), e.Name()), lvrsrc.OpenOptions{})
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
					t.Fatalf("%s id=%d Decode: %v", e.Name(), section.Index, err)
				}
				back, err := Codec{}.Encode(codecs.Context{}, v)
				if err != nil {
					t.Fatalf("%s id=%d Encode: %v", e.Name(), section.Index, err)
				}
				if !bytes.Equal(back, section.Payload) {
					t.Fatalf("%s id=%d round-trip mismatch", e.Name(), section.Index)
				}
			}
		}
	}
	if total == 0 {
		t.Skip("no BDSE sections in corpus")
	}
	t.Logf("exercised %d BDSE section(s)", total)
}

// TestEncodeRejectsBadInput exercises the two error branches of Encode
// (nil typed pointer + wrong concrete type) that the round-trip fixtures
// never hit.
func TestEncodeRejectsBadInput(t *testing.T) {
	if _, err := (Codec{}).Encode(codecs.Context{}, (*Value)(nil)); err == nil {
		t.Errorf("Encode(nil *Value) returned no error")
	}
	if _, err := (Codec{}).Encode(codecs.Context{}, "not a Value"); err == nil {
		t.Errorf("Encode(string) returned no error")
	}
}
