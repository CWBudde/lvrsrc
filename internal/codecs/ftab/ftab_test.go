package ftab

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

func TestDecodeRejectsTooShortHeader(t *testing.T) {
	if _, err := (Codec{}).Decode(codecs.Context{}, make([]byte, 5)); err == nil {
		t.Fatal("Decode(5 bytes) returned nil error")
	}
}

func TestDecodeRejectsOversizedCount(t *testing.T) {
	// Header with count = 200 (> 127 limit).
	header := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 200}
	if _, err := (Codec{}).Decode(codecs.Context{}, header); err == nil {
		t.Fatal("Decode of count=200 returned nil error")
	}
}

func TestEncodeRoundTripWideSingleEntry(t *testing.T) {
	v := Value{
		Prop1: wideEntryFlagBit | 0x00000002,
		Prop3: 0x0003,
		Entries: []FontEntry{
			{Prop2: 0x0001, Prop3: 0x00CF, Prop4: 0x00C4, Prop6: 0x0080, Prop7: 0x0080, Prop8: 0x0010, Name: []byte("Tahoma")},
		},
	}
	encoded, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, err := Codec{}.Decode(codecs.Context{}, encoded)
	if err != nil {
		t.Fatalf("Decode of just-encoded: %v", err)
	}
	got := decoded.(Value)
	if got.Prop1 != v.Prop1 || got.Prop3 != v.Prop3 {
		t.Fatalf("header drift: got %+v want %+v", got, v)
	}
	if len(got.Entries) != 1 || string(got.Entries[0].Name) != "Tahoma" {
		t.Fatalf("entries drift: got %+v", got.Entries)
	}
	// Re-encode must be byte-identical (no offset drift).
	again, _ := Codec{}.Encode(codecs.Context{}, got)
	if !bytes.Equal(again, encoded) {
		t.Fatalf("re-encode drift: %x vs %x", again, encoded)
	}
}

func TestEncodeRoundTripNarrowEntry(t *testing.T) {
	v := Value{
		Prop1: 0x00000003, // wide bit cleared
		Prop3: 0x0001,
		Entries: []FontEntry{
			{Prop2: 0x0010, Prop3: 0x0020, Prop4: 0x0030, Prop5: 0x0040, Name: []byte("FixedSys")},
		},
	}
	encoded, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if got, want := len(encoded), headerSize+narrowEntrySize+1+len("FixedSys"); got != want {
		t.Errorf("encoded size = %d, want %d", got, want)
	}
	decoded, err := Codec{}.Decode(codecs.Context{}, encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	got := decoded.(Value)
	if got.hasWideEntries() {
		t.Fatal("decoded value reports wide entries; expected narrow")
	}
	if got.Entries[0].Prop5 != 0x0040 {
		t.Errorf("narrow Prop5 = %#x, want 0x0040", got.Entries[0].Prop5)
	}
}

func TestEncodeRejectsLongName(t *testing.T) {
	v := Value{
		Prop1: wideEntryFlagBit,
		Entries: []FontEntry{
			{Name: make([]byte, 256)},
		},
	}
	if _, err := (Codec{}).Encode(codecs.Context{}, v); err == nil {
		t.Fatal("Encode of 256-byte name returned nil error")
	}
}

func TestValidate(t *testing.T) {
	if issues := (Codec{}).Validate(codecs.Context{}, make([]byte, headerSize)); len(issues) != 0 {
		t.Errorf("Validate(empty table) issues = %+v, want none", issues)
	}
	issues := Codec{}.Validate(codecs.Context{}, []byte{0, 0, 0, 0, 0, 0, 0, 200})
	if len(issues) == 0 {
		t.Fatal("Validate(count=200) returned no issues")
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
					t.Fatalf("%s id=%d round-trip mismatch:\n got %x\nwant %x",
						e.Name(), section.Index, back, section.Payload)
				}
			}
		}
	}
	if total == 0 {
		t.Skip("no FTAB sections in corpus")
	}
	t.Logf("exercised %d FTAB section(s)", total)
}
