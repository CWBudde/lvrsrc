package vers

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
	if c.FourCC != "vers" {
		t.Fatalf("FourCC = %q, want vers", c.FourCC)
	}
	if c.Safety != codecs.SafetyTier2 {
		t.Fatalf("Safety = %v, want SafetyTier2", c.Safety)
	}
}

func TestDecodeCorpusSample(t *testing.T) {
	// Raw bytes from testdata/corpus/is-float.vi, vers id=4.
	payload := []byte{0x25, 0x12, 0x80, 0x02, 0x00, 0x00, 0x06, '2', '5', '.', '1', '.', '2', 0x00}

	got, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode error = %v", err)
	}
	v, ok := got.(Value)
	if !ok {
		t.Fatalf("Decode returned %T, want Value", got)
	}
	want := Value{Major: 0x25, Minor: 1, Patch: 2, Stage: 0x80, Build: 2, Reserved: 0, Text: "25.1.2"}
	if v != want {
		t.Fatalf("Decode = %+v, want %+v", v, want)
	}
}

func TestDecodeShortPatch(t *testing.T) {
	// Corpus vers id=7/9 shape: "25.0" (size 12).
	payload := []byte{0x25, 0x00, 0x80, 0x00, 0x00, 0x00, 0x04, '2', '5', '.', '0', 0x00}

	got, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode error = %v", err)
	}
	v := got.(Value)
	if v.Text != "25.0" || v.Minor != 0 || v.Patch != 0 {
		t.Fatalf("Decode = %+v, want 25.0 / 0 / 0", v)
	}
}

func TestDecodeRejectsShortPayload(t *testing.T) {
	_, err := Codec{}.Decode(codecs.Context{}, []byte{0, 0, 0})
	if err == nil {
		t.Fatalf("Decode of short payload did not error")
	}
}

func TestDecodeRejectsOverrunTextLength(t *testing.T) {
	// TextLen = 100 but payload is only 14 bytes.
	payload := []byte{0x25, 0x12, 0x80, 0x02, 0x00, 0x00, 100, '2', '5', '.', '1', '.', '2', 0x00}
	_, err := Codec{}.Decode(codecs.Context{}, payload)
	if err == nil {
		t.Fatalf("Decode with overrun text length did not error")
	}
}

func TestEncodeRoundTripFromValue(t *testing.T) {
	v := Value{Major: 0x25, Minor: 1, Patch: 2, Stage: 0x80, Build: 2, Text: "25.1.2"}
	out, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode error = %v", err)
	}
	want := []byte{0x25, 0x12, 0x80, 0x02, 0x00, 0x00, 0x06, '2', '5', '.', '1', '.', '2', 0x00}
	if !bytes.Equal(out, want) {
		t.Fatalf("Encode = %x, want %x", out, want)
	}
}

func TestEncodeAcceptsPointer(t *testing.T) {
	v := Value{Major: 0x25, Text: "25.0"}
	got, err := Codec{}.Encode(codecs.Context{}, &v)
	if err != nil {
		t.Fatalf("Encode(*Value) error = %v", err)
	}
	if len(got) < minPayloadSize {
		t.Fatalf("Encode produced %d bytes, want >= %d", len(got), minPayloadSize)
	}
}

func TestEncodeRejectsWrongType(t *testing.T) {
	_, err := Codec{}.Encode(codecs.Context{}, "not a value")
	if err == nil {
		t.Fatalf("Encode of string did not error")
	}
}

func TestEncodeRejectsOverlongNibble(t *testing.T) {
	_, err := Codec{}.Encode(codecs.Context{}, Value{Minor: 16})
	if err == nil {
		t.Fatalf("Encode with Minor=16 did not error")
	}
}

func TestEncodeRejectsOverlongText(t *testing.T) {
	text := string(bytes.Repeat([]byte{'x'}, 256))
	_, err := Codec{}.Encode(codecs.Context{}, Value{Text: text})
	if err == nil {
		t.Fatalf("Encode with 256-char text did not error")
	}
}

func TestValidateCleanPayload(t *testing.T) {
	payload := []byte{0x25, 0x12, 0x80, 0x02, 0x00, 0x00, 0x06, '2', '5', '.', '1', '.', '2', 0x00}
	issues := Codec{}.Validate(codecs.Context{}, payload)
	if len(issues) != 0 {
		t.Fatalf("Validate reported %d issues, want 0: %+v", len(issues), issues)
	}
}

func TestValidateReportsShortPayload(t *testing.T) {
	issues := Codec{}.Validate(codecs.Context{}, []byte{0, 0, 0})
	assertHasCode(t, issues, "vers.payload.short", validate.SeverityError)
}

func TestValidateReportsOverrunText(t *testing.T) {
	payload := []byte{0x25, 0x12, 0x80, 0x02, 0x00, 0x00, 50, '2', '5', '.', '1', '.', '2', 0x00}
	issues := Codec{}.Validate(codecs.Context{}, payload)
	assertHasCode(t, issues, "vers.text.overruns_payload", validate.SeverityError)
}

func TestValidateReportsMissingTrailer(t *testing.T) {
	payload := []byte{0x25, 0x12, 0x80, 0x02, 0x00, 0x00, 0x06, '2', '5', '.', '1', '.', '2', 0xFF}
	issues := Codec{}.Validate(codecs.Context{}, payload)
	assertHasCode(t, issues, "vers.trailer.missing", validate.SeverityError)
}

func TestValidateReportsReservedNonzero(t *testing.T) {
	payload := []byte{0x25, 0x12, 0x80, 0x02, 0x00, 0x01, 0x06, '2', '5', '.', '1', '.', '2', 0x00}
	issues := Codec{}.Validate(codecs.Context{}, payload)
	assertHasCode(t, issues, "vers.reserved.nonzero", validate.SeverityWarning)
}

func TestValidateReportsUnknownStage(t *testing.T) {
	payload := []byte{0x25, 0x12, 0x40, 0x02, 0x00, 0x00, 0x06, '2', '5', '.', '1', '.', '2', 0x00}
	issues := Codec{}.Validate(codecs.Context{}, payload)
	assertHasCode(t, issues, "vers.stage.unknown", validate.SeverityWarning)
}

func TestValidateReportsTextInconsistency(t *testing.T) {
	// Bytes say 25.1.2 but text claims 99.9.9.
	payload := []byte{0x25, 0x12, 0x80, 0x02, 0x00, 0x00, 0x06, '9', '9', '.', '9', '.', '9', 0x00}
	issues := Codec{}.Validate(codecs.Context{}, payload)
	assertHasCode(t, issues, "vers.text.inconsistent", validate.SeverityWarning)
}

func TestValidateReportsNonAsciiText(t *testing.T) {
	payload := []byte{0x25, 0x12, 0x80, 0x02, 0x00, 0x00, 0x06, '2', '5', '.', '1', 0x01, '2', 0x00}
	issues := Codec{}.Validate(codecs.Context{}, payload)
	assertHasCode(t, issues, "vers.text.nonascii", validate.SeverityWarning)
}

// TestCorpusRoundTrip exercises every vers section in the test corpus: each
// payload must Decode without error, re-Encode byte-for-byte, and Validate
// without issues. This is the highest-confidence check that the codec matches
// the real wire format.
func TestCorpusRoundTrip(t *testing.T) {
	corpusDir := corpus.Dir()
	entries, err := os.ReadDir(corpusDir)
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
		path := filepath.Join(corpusDir, e.Name())
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
					t.Fatalf("%s vers id=%d Decode: %v", e.Name(), section.Index, err)
				}
				back, err := Codec{}.Encode(codecs.Context{}, got)
				if err != nil {
					t.Fatalf("%s vers id=%d Encode: %v", e.Name(), section.Index, err)
				}
				if !bytes.Equal(back, section.Payload) {
					t.Fatalf("%s vers id=%d round-trip mismatch:\nin  %x\nout %x", e.Name(), section.Index, section.Payload, back)
				}
				if issues := (Codec{}).Validate(codecs.Context{}, section.Payload); len(issues) != 0 {
					t.Fatalf("%s vers id=%d Validate reported %d issues: %+v", e.Name(), section.Index, len(issues), issues)
				}
			}
		}
	}

	if total == 0 {
		t.Fatalf("no vers sections found in corpus; test is not exercising anything")
	}
	t.Logf("round-tripped %d vers sections", total)
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
