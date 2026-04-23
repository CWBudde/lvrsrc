package vctp

import (
	"bytes"
	"compress/zlib"
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

func TestDecodeInflatesPayload(t *testing.T) {
	payload := makePayload(t, []byte("type-desc-pool"))

	got, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode error = %v", err)
	}
	v := got.(Value)
	if v.DeclaredSize != uint32(len("type-desc-pool")) {
		t.Fatalf("DeclaredSize = %d, want %d", v.DeclaredSize, len("type-desc-pool"))
	}
	if string(v.Inflated) != "type-desc-pool" {
		t.Fatalf("Inflated = %q, want %q", string(v.Inflated), "type-desc-pool")
	}
	if len(v.Compressed) == 0 {
		t.Fatalf("Compressed is empty")
	}
	if !bytes.Equal(v.Compressed, payload[4:]) {
		t.Fatalf("Compressed bytes mismatch")
	}

	back, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode error = %v", err)
	}
	if !bytes.Equal(back, payload) {
		t.Fatalf("round-trip mismatch")
	}
}

func TestEncodeCompressesWhenCompressedMissing(t *testing.T) {
	orig := []byte{0x00, 0x01, 0x02, 0x03, 0x7f, 0x80, 0xff}
	payload, err := Codec{}.Encode(codecs.Context{}, Value{
		DeclaredSize: uint32(len(orig)),
		Inflated:     append([]byte(nil), orig...),
	})
	if err != nil {
		t.Fatalf("Encode error = %v", err)
	}

	got, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode error = %v", err)
	}
	v := got.(Value)
	if !bytes.Equal(v.Inflated, orig) {
		t.Fatalf("Inflated = %x, want %x", v.Inflated, orig)
	}
}

func TestValidateReportsShortPayload(t *testing.T) {
	issues := Codec{}.Validate(codecs.Context{}, []byte{0x00, 0x00, 0x00})
	assertHasCode(t, issues, "vctp.payload.short", validate.SeverityError)
}

func TestValidateReportsDeclaredSizeMismatch(t *testing.T) {
	payload := makePayload(t, []byte("abc"))
	binary.BigEndian.PutUint32(payload[:4], 4)

	issues := Codec{}.Validate(codecs.Context{}, payload)
	assertHasCode(t, issues, "vctp.declared_size.mismatch", validate.SeverityError)
}

func TestCorpusRoundTrip(t *testing.T) {
	entries, err := os.ReadDir(corpus.Dir())
	if err != nil {
		t.Fatalf("read corpus dir: %v", err)
	}

	total := 0
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
			if block.Type != string(FourCC) {
				continue
			}
			for _, section := range block.Sections {
				total++
				got, err := Codec{}.Decode(codecs.Context{}, section.Payload)
				if err != nil {
					t.Fatalf("%s VCTP id=%d Decode: %v", e.Name(), section.Index, err)
				}
				back, err := Codec{}.Encode(codecs.Context{}, got)
				if err != nil {
					t.Fatalf("%s VCTP id=%d Encode: %v", e.Name(), section.Index, err)
				}
				if !bytes.Equal(back, section.Payload) {
					t.Fatalf("%s VCTP id=%d round-trip mismatch", e.Name(), section.Index)
				}
				if issues := (Codec{}).Validate(codecs.Context{}, section.Payload); len(issues) != 0 {
					t.Fatalf("%s VCTP id=%d Validate issues: %+v", e.Name(), section.Index, issues)
				}
			}
		}
	}

	if total == 0 {
		t.Fatalf("no VCTP sections found in corpus; test is not exercising anything")
	}
	t.Logf("round-tripped %d VCTP sections", total)
}

func makePayload(t *testing.T, inflated []byte) []byte {
	t.Helper()
	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	if _, err := w.Write(inflated); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}

	out := make([]byte, 4+compressed.Len())
	binary.BigEndian.PutUint32(out[:4], uint32(len(inflated)))
	copy(out[4:], compressed.Bytes())
	return out
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
