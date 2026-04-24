package hist

import (
	"bytes"
	"encoding/binary"
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

func TestDecodePreservesAllBytes(t *testing.T) {
	payload := make([]byte, payloadSize)
	for i := range payload {
		payload[i] = byte(i)
	}
	v, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	val := v.(Value)
	if !bytes.Equal(val.Raw[:], payload) {
		t.Fatalf("Raw = %x, want %x", val.Raw[:], payload)
	}
}

func TestCountersReadsBigEndianUint32(t *testing.T) {
	payload := make([]byte, payloadSize)
	for i := 0; i < CounterCount; i++ {
		binary.BigEndian.PutUint32(payload[i*4:], uint32(0x10000000+i))
	}
	v, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	got := v.(Value).Counters()
	for i := 0; i < CounterCount; i++ {
		if want := uint32(0x10000000 + i); got[i] != want {
			t.Errorf("Counters[%d] = %#x, want %#x", i, got[i], want)
		}
	}
}

func TestDecodeRejectsWrongSize(t *testing.T) {
	for _, size := range []int{0, 4, 39, 41, 80} {
		if _, err := (Codec{}).Decode(codecs.Context{}, make([]byte, size)); err == nil {
			t.Errorf("Decode(%d-byte payload) returned nil error", size)
		}
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	original := make([]byte, payloadSize)
	for i := range original {
		original[i] = byte((i * 7) & 0xFF)
	}
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

func TestValidate(t *testing.T) {
	if issues := (Codec{}).Validate(codecs.Context{}, make([]byte, payloadSize)); len(issues) != 0 {
		t.Errorf("Validate(%d bytes) issues = %+v, want none", payloadSize, issues)
	}
	issues := Codec{}.Validate(codecs.Context{}, make([]byte, 8))
	if len(issues) == 0 {
		t.Fatal("Validate(8 bytes) returned no issues")
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
		t.Skip("no HIST sections in corpus")
	}
	t.Logf("exercised %d HIST section(s)", total)
}
