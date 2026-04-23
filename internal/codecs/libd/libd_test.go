package libd

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
		'B', 'D', 'H', 'P',
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
	if v.Marker != "BDHP" {
		t.Fatalf("Marker = %q, want BDHP", v.Marker)
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
		0x00, 0x01, 'B', 'D', 'H', 'P',
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x02, 'I', 'U', 'V', 'I',
		0x00, 0x00, 0x00, 0x01,
		0x18, 'A', 'p', 'p', 'l', 'i', 'c', 'a', 't', 'i', 'o', 'n', ' ', 'D', 'i', 'r', 'e', 'c', 't', 'o', 'r', 'y', '.', 'v', 'i',
		0x00, 'P', 'T', 'H', '0', 0x00, 0x00, 0x00, 0x36,
		0x00, 0x00, 0x00, 0x04, 0x07, '<', 'v', 'i', 'l', 'i', 'b', '>', 0x07, 'U', 't', 'i', 'l', 'i', 't', 'y',
		0x08, 'f', 'i', 'l', 'e', '.', 'l', 'l', 'b',
		0x18, 'A', 'p', 'p', 'l', 'i', 'c', 'a', 't', 'i', 'o', 'n', ' ', 'D', 'i', 'r', 'e', 'c', 't', 'o', 'r', 'y', '.', 'v', 'i',
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x42,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x29,
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
	if e.LinkType != "IUVI" {
		t.Fatalf("Entry.LinkType = %q, want IUVI", e.LinkType)
	}
	if e.QualifierCount != 1 {
		t.Fatalf("Entry.QualifierCount = %d, want 1", e.QualifierCount)
	}
	if len(e.Qualifiers) != 1 || e.Qualifiers[0] != "Application Directory.vi" {
		t.Fatalf("Entry.Qualifiers = %+v, want [Application Directory.vi]", e.Qualifiers)
	}
	if e.PrimaryPath.Class != "PTH0" || e.PrimaryPath.DeclaredLen != 0x36 {
		t.Fatalf("PrimaryPath = %+v, want PTH0 len 0x36", e.PrimaryPath)
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

func TestDecodeMultiEntrySample(t *testing.T) {
	payload := []byte{
		0x00, 0x01, 'B', 'D', 'H', 'P',
		0x00, 0x00, 0x00, 0x02,
		0x00, 0x02, 'T', 'D', 'C', 'C',
		0x00, 0x00, 0x00, 0x01,
		0x11, 'R', 'e', 'f', 'e', 'r', 'e', 'n', 'c', 'e', 'T', 'y', 'p', 'e', '.', 'c', 't', 'l',
		'P', 'T', 'H', '0', 0x00, 0x00, 0x00, 0x21,
		0x00, 0x01, 0x00, 0x04, 0x00, 0x00, 0x08, 'T', 'y', 'p', 'e', 'D', 'e', 'f', 's', 0x11, 'R', 'e', 'f', 'e', 'r', 'e', 'n', 'c', 'e', 'T', 'y', 'p', 'e', '.', 'c', 't', 'l',
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x42,
		0x00, 0x00, 0x08, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0xe7,
		'P', 'T', 'H', '0', 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x02, 'T', 'D', 'C', 'C',
		0x00, 0x00, 0x00, 0x01,
		0x11, 'R', 'e', 'f', 'e', 'r', 'e', 'n', 'c', 'e', 'I', 't', 'e', 'm', '.', 'c', 't', 'l',
		'P', 'T', 'H', '0', 0x00, 0x00, 0x00, 0x21,
		0x00, 0x01, 0x00, 0x04, 0x00, 0x00, 0x08, 'T', 'y', 'p', 'e', 'D', 'e', 'f', 's', 0x11, 'R', 'e', 'f', 'e', 'r', 'e', 'n', 'c', 'e', 'I', 't', 'e', 'm', '.', 'c', 't', 'l',
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x42,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0xbc,
		'P', 'T', 'H', '0', 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x03,
	}

	got, err := Codec{}.Decode(codecs.Context{}, payload)
	if err != nil {
		t.Fatalf("Decode error = %v", err)
	}
	v := got.(Value)
	if v.EntryCount != 2 {
		t.Fatalf("EntryCount = %d, want 2", v.EntryCount)
	}
	if len(v.Entries) != 2 {
		t.Fatalf("len(Entries) = %d, want 2", len(v.Entries))
	}
	if v.Entries[0].LinkType != "TDCC" || v.Entries[1].LinkType != "TDCC" {
		t.Fatalf("LinkTypes = [%q %q], want [TDCC TDCC]", v.Entries[0].LinkType, v.Entries[1].LinkType)
	}
	if v.Entries[0].SecondaryPath == nil || v.Entries[1].SecondaryPath == nil {
		t.Fatalf("expected both entries to have secondary paths: %+v", v.Entries)
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

func TestValidateReportsShortPayload(t *testing.T) {
	issues := Codec{}.Validate(codecs.Context{}, []byte{0x00, 0x01, 'B'})
	assertHasCode(t, issues, "libd.payload.short", validate.SeverityError)
}

func TestValidateReportsBadMarker(t *testing.T) {
	payload := []byte{
		0x00, 0x01, 'B', 'A', 'D', '!',
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x03,
	}
	issues := Codec{}.Validate(codecs.Context{}, payload)
	assertHasCode(t, issues, "libd.marker.invalid", validate.SeverityError)
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
					t.Fatalf("%s LIbd id=%d Decode: %v", e.Name(), section.Index, err)
				}
				back, err := Codec{}.Encode(codecs.Context{}, got)
				if err != nil {
					t.Fatalf("%s LIbd id=%d Encode: %v", e.Name(), section.Index, err)
				}
				if !bytes.Equal(back, section.Payload) {
					t.Fatalf("%s LIbd id=%d round-trip mismatch", e.Name(), section.Index)
				}
				if issues := (Codec{}).Validate(codecs.Context{}, section.Payload); len(issues) != 0 {
					t.Fatalf("%s LIbd id=%d Validate issues: %+v", e.Name(), section.Index, issues)
				}
			}
		}
	}

	if total == 0 {
		t.Fatalf("no LIbd sections found in corpus; test is not exercising anything")
	}
	t.Logf("round-tripped %d LIbd sections", total)
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
