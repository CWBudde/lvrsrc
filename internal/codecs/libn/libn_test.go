package libn

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

func TestDecodeEmptyList(t *testing.T) {
	v, err := Codec{}.Decode(codecs.Context{}, []byte{0, 0, 0, 0})
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got := len(v.(Value).Names); got != 0 {
		t.Fatalf("len(Names) = %d, want 0", got)
	}
}

func TestDecodeSingleEntry(t *testing.T) {
	// count=1, name="Tools.lvlib" (11 bytes)
	payload := append([]byte{0, 0, 0, 1, 11}, []byte("Tools.lvlib")...)
	v, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	val := v.(Value)
	if len(val.Names) != 1 {
		t.Fatalf("len(Names) = %d, want 1", len(val.Names))
	}
	if string(val.Names[0]) != "Tools.lvlib" {
		t.Errorf("Names[0] = %q, want %q", val.Names[0], "Tools.lvlib")
	}
}

func TestDecodeMultipleEntries(t *testing.T) {
	payload := []byte{0, 0, 0, 2, 4, 'A', 'B', 'C', 'D', 3, 'X', 'Y', 'Z'}
	v, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	val := v.(Value)
	if len(val.Names) != 2 {
		t.Fatalf("len(Names) = %d, want 2", len(val.Names))
	}
	if string(val.Names[0]) != "ABCD" || string(val.Names[1]) != "XYZ" {
		t.Errorf("Names = %q, want [ABCD XYZ]", val.Names)
	}
}

func TestDecodeRejectsTruncatedCount(t *testing.T) {
	if _, err := (Codec{}).Decode(codecs.Context{}, []byte{0, 0, 1}); err == nil {
		t.Fatal("Decode(3 bytes) returned nil error")
	}
}

func TestDecodeRejectsTruncatedName(t *testing.T) {
	// count=1, length=10, but only 5 bytes of payload
	payload := []byte{0, 0, 0, 1, 10, 'A', 'B', 'C', 'D', 'E'}
	if _, err := (Codec{}).Decode(codecs.Context{}, payload); err == nil {
		t.Fatal("Decode of truncated name returned nil error")
	}
}

func TestDecodeRejectsTrailingBytes(t *testing.T) {
	// count=1, name="A", then a trailing byte
	payload := []byte{0, 0, 0, 1, 1, 'A', 0xFF}
	if _, err := (Codec{}).Decode(codecs.Context{}, payload); err == nil {
		t.Fatal("Decode with trailing byte returned nil error")
	}
}

func TestEncodeRoundTripSingleEntry(t *testing.T) {
	original := append([]byte{0, 0, 0, 1, 11}, []byte("Tools.lvlib")...)
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

func TestEncodeRejectsLongName(t *testing.T) {
	// 256 bytes is one over the Pascal-string limit.
	name := make([]byte, 256)
	v := Value{Names: [][]byte{name}}
	if _, err := (Codec{}).Encode(codecs.Context{}, v); err == nil {
		t.Fatal("Encode of 256-byte name returned nil error")
	}
}

func TestValidate(t *testing.T) {
	if issues := (Codec{}).Validate(codecs.Context{}, []byte{0, 0, 0, 0}); len(issues) != 0 {
		t.Errorf("Validate(empty list) issues = %+v, want none", issues)
	}
	issues := Codec{}.Validate(codecs.Context{}, []byte{0, 0, 0, 1})
	if len(issues) == 0 {
		t.Fatal("Validate(truncated) returned no issues")
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
					t.Fatalf("%s id=%d Decode: %v (payload %x)", e.Name(), section.Index, err, section.Payload)
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
		t.Skip("no LIBN sections in corpus")
	}
	t.Logf("exercised %d LIBN section(s)", total)
}
