package vits

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

func encodeEntry(name, variant []byte) []byte {
	buf := make([]byte, 0, 8+len(name)+len(variant))
	tmp := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp, uint32(len(name)))
	buf = append(buf, tmp...)
	buf = append(buf, name...)
	binary.BigEndian.PutUint32(tmp, uint32(len(variant)))
	buf = append(buf, tmp...)
	buf = append(buf, variant...)
	return buf
}

func encodePayload(entries ...struct{ name, variant []byte }) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(len(entries)))
	for _, e := range entries {
		buf = append(buf, encodeEntry(e.name, e.variant)...)
	}
	return buf
}

func TestDecodeEmpty(t *testing.T) {
	v, err := Codec{}.Decode(codecs.Context{}, []byte{0, 0, 0, 0})
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got := len(v.(Value).Entries); got != 0 {
		t.Fatalf("len(Entries) = %d, want 0", got)
	}
}

func TestDecodeSingleEntry(t *testing.T) {
	payload := encodePayload(struct{ name, variant []byte }{
		[]byte("NI.LV.All.SourceOnly"),
		[]byte{0x25, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x01},
	})
	v, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	val := v.(Value)
	if len(val.Entries) != 1 {
		t.Fatalf("len(Entries) = %d, want 1", len(val.Entries))
	}
	if string(val.Entries[0].Name) != "NI.LV.All.SourceOnly" {
		t.Errorf("name = %q, want %q", val.Entries[0].Name, "NI.LV.All.SourceOnly")
	}
	if len(val.Entries[0].Variant) != 8 {
		t.Errorf("len(variant) = %d, want 8", len(val.Entries[0].Variant))
	}
}

func TestDecodeMultipleEntries(t *testing.T) {
	payload := encodePayload(
		struct{ name, variant []byte }{[]byte("A"), []byte{0x01}},
		struct{ name, variant []byte }{[]byte("BB"), []byte{0x02, 0x03}},
		struct{ name, variant []byte }{[]byte("CCC"), []byte{0x04, 0x05, 0x06}},
	)
	v, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	val := v.(Value)
	if len(val.Entries) != 3 {
		t.Fatalf("len(Entries) = %d, want 3", len(val.Entries))
	}
	if string(val.Entries[2].Name) != "CCC" || !bytes.Equal(val.Entries[2].Variant, []byte{4, 5, 6}) {
		t.Errorf("entry 2 = %+v, want CCC/[4,5,6]", val.Entries[2])
	}
}

func TestDecodeRejectsTruncatedNameLen(t *testing.T) {
	payload := []byte{0, 0, 0, 1, 0, 0}
	if _, err := (Codec{}).Decode(codecs.Context{}, payload); err == nil {
		t.Fatal("Decode of truncated name length returned nil error")
	}
}

func TestDecodeRejectsTruncatedNameBytes(t *testing.T) {
	payload := []byte{0, 0, 0, 1, 0, 0, 0, 10, 'A'} // claims 10 bytes, only 1
	if _, err := (Codec{}).Decode(codecs.Context{}, payload); err == nil {
		t.Fatal("Decode of truncated name bytes returned nil error")
	}
}

func TestDecodeRejectsTruncatedVariant(t *testing.T) {
	payload := encodePayload(struct{ name, variant []byte }{[]byte("X"), []byte{1, 2, 3, 4}})
	// Truncate the last variant byte.
	truncated := payload[:len(payload)-1]
	if _, err := (Codec{}).Decode(codecs.Context{}, truncated); err == nil {
		t.Fatal("Decode of truncated variant returned nil error")
	}
}

func TestDecodeRejectsTrailingBytes(t *testing.T) {
	payload := encodePayload(struct{ name, variant []byte }{[]byte("X"), []byte{1, 2}})
	payload = append(payload, 0xFF)
	if _, err := (Codec{}).Decode(codecs.Context{}, payload); err == nil {
		t.Fatal("Decode with trailing byte returned nil error")
	}
}

func TestEncodeRoundTripSingleEntry(t *testing.T) {
	original := encodePayload(struct{ name, variant []byte }{
		[]byte("NI.LV.All.SourceOnly"),
		[]byte{0x25, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x04, 0x00, 0x21, 0x00, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00},
	})
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
	if issues := (Codec{}).Validate(codecs.Context{}, []byte{0, 0, 0, 0}); len(issues) != 0 {
		t.Errorf("Validate(empty) issues = %+v, want none", issues)
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
	totalEntries := 0
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
				totalEntries += len(v.(Value).Entries)
				back, err := Codec{}.Encode(codecs.Context{}, v)
				if err != nil {
					t.Fatalf("%s id=%d Encode: %v", e.Name(), section.Index, err)
				}
				if !bytes.Equal(back, section.Payload) {
					t.Fatalf("%s id=%d round-trip mismatch (len=%d)", e.Name(), section.Index, len(section.Payload))
				}
			}
		}
	}
	if total == 0 {
		t.Skip("no VITS sections in corpus")
	}
	t.Logf("exercised %d VITS section(s) totalling %d tag entries", total, totalEntries)
}
