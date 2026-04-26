package fpex

import (
	"bytes"
	"encoding/hex"
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

func TestDecodeZeroCount(t *testing.T) {
	v, err := Codec{}.Decode(codecs.Context{}, []byte{0, 0, 0, 0})
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got := len(v.(Value).Entries); got != 0 {
		t.Fatalf("len(Entries) = %d, want 0", got)
	}
}

func TestDecodeOneEntry(t *testing.T) {
	payload, _ := hex.DecodeString("0000000100000000")
	v, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	val := v.(Value)
	if len(val.Entries) != 1 || val.Entries[0] != 0 {
		t.Fatalf("Entries = %+v, want [0]", val.Entries)
	}
}

func TestDecodeRejectsCountSizeMismatch(t *testing.T) {
	// Count=5 but only 4 bytes of payload follows the count field
	payload := []byte{0, 0, 0, 5, 0, 0, 0, 0}
	if _, err := (Codec{}).Decode(codecs.Context{}, payload); err == nil {
		t.Fatal("Decode of size-mismatched payload returned nil error")
	}
}

func TestEncodeRoundTripCorpusObservedSizes(t *testing.T) {
	cases := []string{
		"00000000",                         // 4 bytes — count 0
		"0000000100000000",                 // 8 bytes — count 1
		"00000003000000000000000000000000", // 16 bytes — count 3
	}
	for _, hexStr := range cases {
		original, _ := hex.DecodeString(hexStr)
		v, err := Codec{}.Decode(codecs.Context{}, original)
		if err != nil {
			t.Fatalf("Decode(%s): %v", hexStr, err)
		}
		back, err := Codec{}.Encode(codecs.Context{}, v)
		if err != nil {
			t.Fatalf("Encode(%s): %v", hexStr, err)
		}
		if !bytes.Equal(back, original) {
			t.Errorf("Encode != original: got %x, want %s", back, hexStr)
		}
	}
}

func TestValidate(t *testing.T) {
	if issues := (Codec{}).Validate(codecs.Context{}, []byte{0, 0, 0, 0}); len(issues) != 0 {
		t.Errorf("Validate(zero count) issues = %+v, want none", issues)
	}
	issues := Codec{}.Validate(codecs.Context{}, []byte{0, 0, 0, 5})
	if len(issues) == 0 {
		t.Fatal("Validate(size mismatch) returned no issues")
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
	nonZero := 0
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
				for _, ent := range v.(Value).Entries {
					if ent != 0 {
						nonZero++
					}
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
		t.Skip("no FPEx sections in corpus")
	}
	t.Logf("exercised %d FPEx section(s); %d non-zero entries (corpus expectation: 0)", total, nonZero)
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
