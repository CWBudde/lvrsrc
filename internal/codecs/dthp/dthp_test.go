package dthp

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

func TestDecodeShortFormCountAndShift(t *testing.T) {
	// 16-bit count = 5, 16-bit shift = 12. Two U2p2 fields, both small.
	payload := []byte{0x00, 0x05, 0x00, 0x0C}
	v, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	val := v.(Value)
	if val.TDCount != 5 {
		t.Errorf("TDCount = %d, want 5", val.TDCount)
	}
	if val.IndexShift != 12 {
		t.Errorf("IndexShift = %d, want 12", val.IndexShift)
	}
}

func TestDecodeZeroCountSkipsShift(t *testing.T) {
	// pylabview LVblock.py:3205 — when tdCount == 0, no shift field is
	// emitted. This codec must reflect that: a 2-byte payload of {0,0}
	// is valid and yields TDCount=0, IndexShift=0.
	payload := []byte{0x00, 0x00}
	v, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	val := v.(Value)
	if val.TDCount != 0 {
		t.Errorf("TDCount = %d, want 0", val.TDCount)
	}
	if val.IndexShift != 0 {
		t.Errorf("IndexShift = %d, want 0", val.IndexShift)
	}
}

func TestDecodeExtendedCount(t *testing.T) {
	// U2p2 extended form: high bit of first short signals 32-bit value.
	// {0x80, 0x01, 0x00, 0x00, 0x00, 0x05, 0x00, 0x07} =>
	//   count: ((0x8001 & 0x7FFF) << 16) | 0x0000 = 0x00010000 = 65536
	//   shift: 0x0005 (still short form because high bit clear), trailing 0x0007 unused
	payload := []byte{0x80, 0x01, 0x00, 0x00, 0x00, 0x05}
	v, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	val := v.(Value)
	if val.TDCount != 0x00010000 {
		t.Errorf("TDCount = %#x, want 0x00010000", val.TDCount)
	}
	if val.IndexShift != 5 {
		t.Errorf("IndexShift = %d, want 5", val.IndexShift)
	}
}

func TestDecodeRejectsTooShort(t *testing.T) {
	for _, payload := range [][]byte{{}, {0x00}} {
		if _, err := (Codec{}).Decode(codecs.Context{}, payload); err == nil {
			t.Errorf("Decode(%d-byte payload) returned nil error", len(payload))
		}
	}
}

func TestEncodeRoundTripShortForm(t *testing.T) {
	original := []byte{0x00, 0x05, 0x00, 0x0C}
	v, err := Codec{}.Decode(codecs.Context{}, original)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	back, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("Encode != original: %x vs %x", back, original)
	}
}

func TestEncodeRoundTripZeroCount(t *testing.T) {
	original := []byte{0x00, 0x00}
	v, err := Codec{}.Decode(codecs.Context{}, original)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	back, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("Encode != original: %x vs %x", back, original)
	}
}

func TestEncodeRoundTripExtendedCount(t *testing.T) {
	original := []byte{0x80, 0x01, 0x00, 0x00, 0x00, 0x05}
	v, err := Codec{}.Decode(codecs.Context{}, original)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	back, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("Encode != original: %x vs %x", back, original)
	}
}

func TestValidateAcceptsCanonicalSizes(t *testing.T) {
	cases := [][]byte{
		{0x00, 0x00},                                     // zero count
		{0x00, 0x05, 0x00, 0x0C},                         // short/short
		{0x80, 0x01, 0x00, 0x00, 0x00, 0x05},             // long count, short shift
		{0x00, 0x05, 0x80, 0x00, 0x00, 0x10},             // short count, long shift
		{0x80, 0x01, 0x00, 0x00, 0x80, 0x00, 0x00, 0x10}, // long/long
	}
	for _, p := range cases {
		if issues := (Codec{}).Validate(codecs.Context{}, p); len(issues) != 0 {
			t.Errorf("Validate(%x) issues = %+v, want none", p, issues)
		}
	}
}

func TestValidateRejectsTooShort(t *testing.T) {
	issues := Codec{}.Validate(codecs.Context{}, []byte{0x00})
	if len(issues) == 0 {
		t.Fatal("Validate(1 byte) returned no issues")
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
					t.Fatalf("%s id=%d round-trip mismatch: got %x want %x",
						e.Name(), section.Index, back, section.Payload)
				}
			}
		}
	}
	if total == 0 {
		t.Skip("no DTHP sections in corpus")
	}
	t.Logf("exercised %d DTHP section(s)", total)
}
