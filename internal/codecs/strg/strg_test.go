package strg

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
	if c.FourCC != "STRG" {
		t.Fatalf("FourCC = %q, want STRG", c.FourCC)
	}
	if c.Safety != codecs.SafetyTier2 {
		t.Fatalf("Safety = %v, want SafetyTier2", c.Safety)
	}
}

func buildPayload(text string) []byte {
	out := make([]byte, 4+len(text))
	binary.BigEndian.PutUint32(out[:4], uint32(len(text)))
	copy(out[4:], text)
	return out
}

func TestDecodeRoundTrip(t *testing.T) {
	text := "replaces all spaces"
	payload := buildPayload(text)

	got, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode error = %v", err)
	}
	v := got.(Value)
	if v.Text != text {
		t.Fatalf("Text = %q, want %q", v.Text, text)
	}

	back, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode error = %v", err)
	}
	if !bytes.Equal(back, payload) {
		t.Fatalf("round-trip mismatch:\nin  %x\nout %x", payload, back)
	}
}

func TestDecodeEmpty(t *testing.T) {
	payload := buildPayload("")
	got, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode error = %v", err)
	}
	if got.(Value).Text != "" {
		t.Fatalf("Decode of empty = %q, want empty", got.(Value).Text)
	}
}

func TestDecodeRejectsShortPayload(t *testing.T) {
	_, err := Codec{}.Decode(codecs.Context{}, []byte{0, 0, 0})
	if err == nil {
		t.Fatalf("Decode of short payload did not error")
	}
}

func TestDecodeRejectsOverrunSize(t *testing.T) {
	payload := []byte{0x00, 0x00, 0x00, 0x64} // claims 100 bytes but only has header
	_, err := Codec{}.Decode(codecs.Context{}, payload)
	if err == nil {
		t.Fatalf("Decode with oversized size did not error")
	}
}

func TestEncodeAcceptsPointer(t *testing.T) {
	v := Value{Text: "hello"}
	out, err := Codec{}.Encode(codecs.Context{}, &v)
	if err != nil {
		t.Fatalf("Encode(*Value) error = %v", err)
	}
	if binary.BigEndian.Uint32(out[:4]) != 5 {
		t.Fatalf("size field = %d, want 5", binary.BigEndian.Uint32(out[:4]))
	}
}

func TestEncodeRejectsWrongType(t *testing.T) {
	_, err := Codec{}.Encode(codecs.Context{}, 42)
	if err == nil {
		t.Fatalf("Encode of int did not error")
	}
}

func TestEncodeRejectsNilPointer(t *testing.T) {
	var v *Value
	_, err := Codec{}.Encode(codecs.Context{}, v)
	if err == nil {
		t.Fatalf("Encode of nil *Value did not error")
	}
}

func TestValidateCleanPayload(t *testing.T) {
	payload := buildPayload("A clean description with\r\nline breaks.")
	issues := Codec{}.Validate(codecs.Context{}, payload)
	if len(issues) != 0 {
		t.Fatalf("Validate reported %d issues, want 0: %+v", len(issues), issues)
	}
}

func TestValidateReportsShortPayload(t *testing.T) {
	issues := Codec{}.Validate(codecs.Context{}, []byte{0, 0})
	assertHasCode(t, issues, "strg.payload.short", validate.SeverityError)
}

func TestValidateReportsOverrun(t *testing.T) {
	issues := Codec{}.Validate(codecs.Context{}, []byte{0, 0, 0, 100, 'x', 'y'})
	assertHasCode(t, issues, "strg.size.overruns_payload", validate.SeverityError)
}

func TestValidateReportsTrailingBytes(t *testing.T) {
	// Declared size=3, but 5 bytes of text+trailing → 2 bytes trailing.
	payload := append(buildPayload("abc"), 0x00, 0x00)
	issues := Codec{}.Validate(codecs.Context{}, payload)
	assertHasCode(t, issues, "strg.size.trailing_bytes", validate.SeverityWarning)
}

func TestValidateReportsControlChars(t *testing.T) {
	payload := buildPayload("clean\x01text")
	issues := Codec{}.Validate(codecs.Context{}, payload)
	assertHasCode(t, issues, "strg.text.control_chars", validate.SeverityWarning)
}

// TestCorpusRoundTrip decodes, re-encodes, and validates every STRG section
// in the test corpus. All 4 samples must round-trip byte-for-byte and
// validate clean.
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
					t.Fatalf("%s STRG id=%d Decode: %v", e.Name(), section.Index, err)
				}
				back, err := Codec{}.Encode(codecs.Context{}, got)
				if err != nil {
					t.Fatalf("%s STRG id=%d Encode: %v", e.Name(), section.Index, err)
				}
				if !bytes.Equal(back, section.Payload) {
					t.Fatalf("%s STRG id=%d round-trip mismatch", e.Name(), section.Index)
				}
				if issues := (Codec{}).Validate(codecs.Context{}, section.Payload); len(issues) != 0 {
					t.Fatalf("%s STRG id=%d Validate issues: %+v", e.Name(), section.Index, issues)
				}
			}
		}
	}

	if total == 0 {
		t.Fatalf("no STRG sections found in corpus; test is not exercising anything")
	}
	t.Logf("round-tripped %d STRG sections", total)
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
