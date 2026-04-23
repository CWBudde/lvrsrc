package lvvi

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

func mustOpen(t *testing.T, name string) *lvrsrc.File {
	t.Helper()
	f, err := lvrsrc.Open(corpus.Path(name), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open(%s) = %v", name, err)
	}
	return f
}

func TestDecodeKnownResourcesNilFile(t *testing.T) {
	m, issues := DecodeKnownResources(nil)
	if m != nil {
		t.Fatalf("Model = %v, want nil", m)
	}
	if len(issues) != 0 {
		t.Fatalf("Issues = %v, want empty", issues)
	}
}

func TestModelVersionReportsFormatAndAppVersion(t *testing.T) {
	// format-string.vi carries FormatVersion=3 in the header and a single
	// `vers` section whose Text decodes as "25.1.2".
	f := mustOpen(t, "format-string.vi")
	m, issues := DecodeKnownResources(f)
	if m == nil {
		t.Fatalf("Model = nil")
	}
	for _, iss := range issues {
		if iss.Severity == SeverityError {
			t.Fatalf("unexpected decode error %s: %s", iss.Code, iss.Message)
		}
	}

	v, ok := m.Version()
	if !ok {
		t.Fatalf("Version() ok = false, want true")
	}
	if v.FormatVersion != f.Header.FormatVersion {
		t.Fatalf("FormatVersion = %d, want %d", v.FormatVersion, f.Header.FormatVersion)
	}
	if !v.HasApp {
		t.Fatalf("HasApp = false, want true (vers section present)")
	}
	if v.Text != "25.1.2" {
		t.Fatalf("Text = %q, want 25.1.2", v.Text)
	}
}

func TestModelDescriptionReturnsDecodedText(t *testing.T) {
	// The corpus fixture has an STRG payload starting with "replaces all ...".
	f := mustOpen(t, "format-string.vi")
	m, _ := DecodeKnownResources(f)
	desc, ok := m.Description()
	if !ok {
		t.Fatalf("Description() ok = false, want true")
	}
	if !strings.HasPrefix(desc, "replaces all") {
		t.Fatalf("Description = %q, want prefix %q", desc, "replaces all")
	}
}

func TestModelDescriptionAbsentWhenNoSTRG(t *testing.T) {
	// is-float.vi has no STRG block.
	f := mustOpen(t, "is-float.vi")
	m, _ := DecodeKnownResources(f)
	if _, ok := m.Description(); ok {
		t.Fatalf("Description() ok = true on file without STRG")
	}
}

func TestModelVersionHasAppFalseWhenNoVers(t *testing.T) {
	// Construct a synthetic file with no vers section.
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 42},
		Blocks: []lvrsrc.Block{
			{Type: "OPQ ", Sections: []lvrsrc.Section{{Index: 0, Payload: []byte{0}}}},
		},
	}
	m, _ := DecodeKnownResources(f)
	v, ok := m.Version()
	if !ok {
		t.Fatalf("Version() ok = false")
	}
	if v.FormatVersion != 42 {
		t.Fatalf("FormatVersion = %d, want 42", v.FormatVersion)
	}
	if v.HasApp {
		t.Fatalf("HasApp = true, want false (no vers section)")
	}
	if v.Text != "" {
		t.Fatalf("Text = %q, want empty", v.Text)
	}
}

func TestListResourcesIncludesAllSectionsInOrder(t *testing.T) {
	f := mustOpen(t, "format-string.vi")
	m, _ := DecodeKnownResources(f)

	got := m.ListResources()
	// Count sections in f and compare.
	var expected int
	for _, b := range f.Blocks {
		expected += len(b.Sections)
	}
	if len(got) != expected {
		t.Fatalf("len(ListResources) = %d, want %d", len(got), expected)
	}

	// Spot-check: every entry's FourCC matches the corresponding block
	// and section Index. We walk the two in lock-step.
	i := 0
	for _, b := range f.Blocks {
		for _, s := range b.Sections {
			if got[i].FourCC != b.Type {
				t.Fatalf("entry %d FourCC = %q, want %q", i, got[i].FourCC, b.Type)
			}
			if got[i].SectionID != s.Index {
				t.Fatalf("entry %d SectionID = %d, want %d", i, got[i].SectionID, s.Index)
			}
			if got[i].Size != len(s.Payload) {
				t.Fatalf("entry %d Size = %d, want %d", i, got[i].Size, len(s.Payload))
			}
			i++
		}
	}
}

func TestListResourcesMarksKnownCodecs(t *testing.T) {
	f := mustOpen(t, "format-string.vi")
	m, _ := DecodeKnownResources(f)

	var sawDecodedVers, sawDecodedSTRG, sawOpaque bool
	for _, r := range m.ListResources() {
		switch r.FourCC {
		case "vers":
			if !r.Decoded {
				t.Fatalf("vers summary Decoded = false, want true")
			}
			sawDecodedVers = true
		case "STRG":
			if !r.Decoded {
				t.Fatalf("STRG summary Decoded = false, want true")
			}
			sawDecodedSTRG = true
		default:
			if !r.Decoded {
				sawOpaque = true
			}
		}
	}
	if !sawDecodedVers {
		t.Fatalf("expected a vers entry in ListResources")
	}
	if !sawDecodedSTRG {
		t.Fatalf("expected a STRG entry in ListResources")
	}
	if !sawOpaque {
		t.Fatalf("expected at least one opaque (Decoded=false) entry")
	}
}

func TestListResourcesNilReceiverIsSafe(t *testing.T) {
	var m *Model
	if got := m.ListResources(); got != nil {
		t.Fatalf("nil Model ListResources = %v, want nil", got)
	}
	if _, ok := m.Version(); ok {
		t.Fatalf("nil Model Version ok = true, want false")
	}
	if _, ok := m.Description(); ok {
		t.Fatalf("nil Model Description ok = true, want false")
	}
	if got := m.File(); got != nil {
		t.Fatalf("nil Model File = %v, want nil", got)
	}
}

func TestDecodeKnownResourcesFlagsMultipleVersSections(t *testing.T) {
	// module-data--cluster.ctl carries five vers sections (per earlier
	// corpus survey). First one wins; a warning Issue is emitted.
	f := mustOpen(t, "module-data--cluster.ctl")
	m, issues := DecodeKnownResources(f)
	if m == nil {
		t.Fatalf("Model = nil")
	}
	var sawMultiple bool
	for _, iss := range issues {
		if iss.Code == "lvvi.decode.multiple_sections" && iss.Location.BlockType == "vers" {
			sawMultiple = true
			if iss.Severity != SeverityWarning {
				t.Fatalf("multiple-vers issue severity = %v, want warning", iss.Severity)
			}
		}
	}
	if !sawMultiple {
		t.Fatalf("expected multiple-sections warning for vers, got %v", issues)
	}
	// First vers section's Major byte should still populate the Version.
	v, _ := m.Version()
	if !v.HasApp {
		t.Fatalf("HasApp = false despite multiple vers sections")
	}
}

func TestDecodeKnownResourcesReportsDecodeFailure(t *testing.T) {
	// Give the decoder a deliberately-short STRG payload (2 bytes).
	// codecs/strg requires at least 4 → Decode returns an error.
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 3},
		Blocks: []lvrsrc.Block{
			{Type: "STRG", Sections: []lvrsrc.Section{
				{Index: 7, Payload: []byte{0x00, 0x00}},
			}},
		},
	}
	m, issues := DecodeKnownResources(f)
	if m == nil {
		t.Fatalf("Model = nil")
	}
	if _, ok := m.Description(); ok {
		t.Fatalf("Description() ok = true after decode failure")
	}
	var sawFailure bool
	for _, iss := range issues {
		if iss.Code != "lvvi.decode.failed" {
			continue
		}
		if iss.Location.BlockType == "STRG" && iss.Location.SectionIndex == 7 {
			sawFailure = true
			if iss.Severity != SeverityError {
				t.Fatalf("decode failure severity = %v, want error", iss.Severity)
			}
		}
	}
	if !sawFailure {
		t.Fatalf("expected decode failure issue with SectionIndex=7, got %v", issues)
	}
}

func TestModelFileReturnsUnderlyingPointer(t *testing.T) {
	f := &lvrsrc.File{Header: lvrsrc.Header{FormatVersion: 1}}
	m, _ := DecodeKnownResources(f)
	if m.File() != f {
		t.Fatalf("Model.File() = %p, want %p", m.File(), f)
	}
}

// Guard against accidental mutation of section.Payload during decoding.
// DecodeKnownResources must NOT modify input payload bytes.
func TestDecodeKnownResourcesDoesNotMutatePayloads(t *testing.T) {
	text := "hello"
	payload := make([]byte, 4+len(text))
	binary.BigEndian.PutUint32(payload[:4], uint32(len(text)))
	copy(payload[4:], text)
	orig := append([]byte(nil), payload...)

	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 3},
		Blocks: []lvrsrc.Block{
			{Type: "STRG", Sections: []lvrsrc.Section{{Index: 0, Payload: payload}}},
		},
	}
	DecodeKnownResources(f)
	for i := range orig {
		if payload[i] != orig[i] {
			t.Fatalf("payload mutated at byte %d: got 0x%02x, want 0x%02x", i, payload[i], orig[i])
		}
	}
}
