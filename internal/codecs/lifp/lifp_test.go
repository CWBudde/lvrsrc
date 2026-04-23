package lifp

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

func TestDecodeHeaderOnlySample(t *testing.T) {
	payload := []byte{
		0x00, 0x01,
		'F', 'P', 'H', 'P',
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x03,
	}

	got, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode error = %v", err)
	}
	v := got.(Value)
	if v.Version != 1 {
		t.Fatalf("Version = %d, want 1", v.Version)
	}
	if v.Marker != "FPHP" {
		t.Fatalf("Marker = %q, want FPHP", v.Marker)
	}
	if v.EntryCount != 0 {
		t.Fatalf("EntryCount = %d, want 0", v.EntryCount)
	}
	if len(v.Entries) != 0 {
		t.Fatalf("len(Entries) = %d, want 0", len(v.Entries))
	}
	if v.Footer != 3 {
		t.Fatalf("Footer = %d, want 3", v.Footer)
	}

	back, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode error = %v", err)
	}
	if !bytes.Equal(back, payload) {
		t.Fatalf("round-trip mismatch")
	}
}

func TestDecodeStructuredSample(t *testing.T) {
	payload := []byte{
		0x00, 0x01, 'F', 'P', 'H', 'P',
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x02, 'T', 'D', 'C', 'C',
		0x00, 0x00, 0x00, 0x01,
		0x0f, 'C', 'o', 'n', 'f', 'i', 'g', ' ', 'D', 'a', 't', 'a', '.', 'c', 't', 'l',
		'P', 'T', 'H', '0', 0x00, 0x00, 0x00, 0x15,
		0x00, 0x01, 0x00, 0x02, 0x00, 0x0f, 'C', 'o', 'n', 'f', 'i', 'g', ' ', 'D', 'a', 't', 'a', '.', 'c', 't', 'l', 0x00, 0x00, 0x00,
		0x02, 0x00, 0x00, 0x42,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x01, 0xdf,
		'P', 'T', 'H', '0', 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x03,
	}

	got, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode error = %v", err)
	}
	v := got.(Value)
	if v.EntryCount != 1 {
		t.Fatalf("EntryCount = %d, want 1", v.EntryCount)
	}
	if len(v.Entries) != 1 {
		t.Fatalf("len(Entries) = %d, want 1", len(v.Entries))
	}
	e := v.Entries[0]
	if e.Kind != 2 {
		t.Fatalf("Entry.Kind = %d, want 2", e.Kind)
	}
	if e.LinkType != "TDCC" {
		t.Fatalf("Entry.LinkType = %q, want TDCC", e.LinkType)
	}
	if e.QualifierCount != 1 {
		t.Fatalf("Entry.QualifierCount = %d, want 1", e.QualifierCount)
	}
	if len(e.Qualifiers) != 1 || e.Qualifiers[0] != "Config Data.ctl" {
		t.Fatalf("Entry.Qualifiers = %+v, want [Config Data.ctl]", e.Qualifiers)
	}
	if e.PrimaryPath.Class != "PTH0" || e.PrimaryPath.DeclaredLen != 0x15 {
		t.Fatalf("PrimaryPath = %+v, want PTH0 len 0x15", e.PrimaryPath)
	}
	if len(e.Tail) == 0 {
		t.Fatalf("Tail is empty, want preserved unknown bytes")
	}
	if e.SecondaryPath == nil || e.SecondaryPath.Class != "PTH0" || e.SecondaryPath.DeclaredLen != 0 {
		t.Fatalf("SecondaryPath = %+v, want zero-length PTH0", e.SecondaryPath)
	}
	if v.Footer != 3 {
		t.Fatalf("Footer = %d, want 3", v.Footer)
	}

	back, err := Codec{}.Encode(codecs.Context{}, v)
	if err != nil {
		t.Fatalf("Encode error = %v", err)
	}
	if !bytes.Equal(back, payload) {
		t.Fatalf("round-trip mismatch")
	}
}

func TestDecodeMultiQualifierSample(t *testing.T) {
	payload := []byte{
		0x00, 0x01, 'F', 'P', 'H', 'P',
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x02, 'T', 'D', 'D', 'C',
		0x00, 0x00, 0x00, 0x02,
		0x12, 'N', 'I', '_', 'D', 'a', 't', 'a', ' ', 'T', 'y', 'p', 'e', '.', 'l', 'v', 'l', 'i', 'b',
		0x0d, 'D', 'a', 't', 'a', ' ', 'T', 'y', 'p', 'e', '.', 'c', 't', 'l',
		'P', 'T', 'H', '0', 0x00, 0x00, 0x00, 0x3d,
		0x00, 0x00, 0x00, 0x05, 0x07, '<', 'v', 'i', 'l', 'i', 'b', '>', 0x07, 'U', 't', 'i', 'l', 'i', 't', 'y',
		0x09, 'D', 'a', 't', 'a', ' ', 'T', 'y', 'p', 'e',
		0x10, 'T', 'y', 'p', 'e', ' ', 'D', 'e', 'f', 'i', 'n', 'i', 't', 'i', 'o', 'n', 's',
		0x0d, 'D', 'a', 't', 'a', ' ', 'T', 'y', 'p', 'e', '.', 'c', 't', 'l',
		0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x42,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x6d,
		0x00,
		'P', 'T', 'H', '0', 0x00, 0x00, 0x00, 0x31,
		0x00, 0x00, 0x00, 0x04, 0x07, '<', 'v', 'i', 'l', 'i', 'b', '>', 0x07, 'U', 't', 'i', 'l', 'i', 't', 'y',
		0x09, 'D', 'a', 't', 'a', ' ', 'T', 'y', 'p', 'e',
		0x12, 'N', 'I', '_', 'D', 'a', 't', 'a', ' ', 'T', 'y', 'p', 'e', '.', 'l', 'v', 'l', 'i', 'b',
		0x00, 0x03,
	}

	got, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode error = %v", err)
	}
	v := got.(Value)
	if len(v.Entries) != 1 {
		t.Fatalf("len(Entries) = %d, want 1", len(v.Entries))
	}
	e := v.Entries[0]
	if e.LinkType != "TDDC" {
		t.Fatalf("Entry.LinkType = %q, want TDDC", e.LinkType)
	}
	if e.QualifierCount != 2 {
		t.Fatalf("Entry.QualifierCount = %d, want 2", e.QualifierCount)
	}
	if len(e.Qualifiers) != 2 || e.Qualifiers[0] != "NI_Data Type.lvlib" || e.Qualifiers[1] != "Data Type.ctl" {
		t.Fatalf("Entry.Qualifiers = %+v", e.Qualifiers)
	}
	if e.PrimaryPath.DeclaredLen != 0x3d {
		t.Fatalf("PrimaryPath.DeclaredLen = %#x, want 0x3d", e.PrimaryPath.DeclaredLen)
	}
	if len(e.Tail) == 0 {
		t.Fatalf("Tail is empty, want preserved unknown bytes")
	}
	if e.SecondaryPath == nil || e.SecondaryPath.DeclaredLen != 0x31 {
		t.Fatalf("SecondaryPath = %+v, want len 0x31", e.SecondaryPath)
	}
	if v.Footer != 3 {
		t.Fatalf("Footer = %d, want 3", v.Footer)
	}
}

func TestValidateReportsShortPayload(t *testing.T) {
	issues := Codec{}.Validate(codecs.Context{}, []byte{0x00, 0x01, 'F'})
	assertHasCode(t, issues, "lifp.payload.short", validate.SeverityError)
}

func TestValidateReportsBadMarker(t *testing.T) {
	payload := []byte{
		0x00, 0x01, 'B', 'A', 'D', '!',
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x03,
	}
	issues := Codec{}.Validate(codecs.Context{}, payload)
	assertHasCode(t, issues, "lifp.marker.invalid", validate.SeverityError)
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
					t.Fatalf("%s LIfp id=%d Decode: %v", e.Name(), section.Index, err)
				}
				back, err := Codec{}.Encode(codecs.Context{}, got)
				if err != nil {
					t.Fatalf("%s LIfp id=%d Encode: %v", e.Name(), section.Index, err)
				}
				if !bytes.Equal(back, section.Payload) {
					t.Fatalf("%s LIfp id=%d round-trip mismatch", e.Name(), section.Index)
				}
				if issues := (Codec{}).Validate(codecs.Context{}, section.Payload); len(issues) != 0 {
					t.Fatalf("%s LIfp id=%d Validate issues: %+v", e.Name(), section.Index, issues)
				}
			}
		}
	}

	if total == 0 {
		t.Fatalf("no LIfp sections found in corpus; test is not exercising anything")
	}
	t.Logf("round-tripped %d LIfp sections", total)
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
