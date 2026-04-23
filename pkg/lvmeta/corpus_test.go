package lvmeta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sort"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// corpusFileWithSTRG is a corpus fixture known to contain exactly one STRG
// section with a non-empty description.
const corpusFileWithSTRG = "format-string.vi"

// corpusFileWithoutSTRG is a corpus fixture known not to contain any STRG
// section.
const corpusFileWithoutSTRG = "is-float.vi"

func mustOpenCorpus(t *testing.T, name string) *lvrsrc.File {
	t.Helper()
	f, err := lvrsrc.Open(corpus.Path(name), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("Open(%s) = %v", name, err)
	}
	return f
}

// snapshotBlocks returns a deep snapshot of f.Blocks (payloads copied) keyed
// by (block.Type, section.Index) so callers can detect byte-for-byte
// deviations after a mutation.
type blockKey struct {
	Type  string
	Index int32
}

func snapshotBlocks(f *lvrsrc.File) map[blockKey][]byte {
	out := make(map[blockKey][]byte)
	for _, block := range f.Blocks {
		for _, section := range block.Sections {
			payload := append([]byte(nil), section.Payload...)
			out[blockKey{Type: block.Type, Index: section.Index}] = payload
		}
	}
	return out
}

func TestSetDescriptionCorpusUpdatesExistingSTRGEndToEnd(t *testing.T) {
	f := mustOpenCorpus(t, corpusFileWithSTRG)
	originalBlocks := snapshotBlocks(f)

	const newDesc = "Updated description via lvmeta SetDescription."

	if err := SetDescription(f, newDesc); err != nil {
		t.Fatalf("SetDescription err = %v", err)
	}

	// Assert: untouched (non-STRG) sections are byte-for-byte unchanged.
	for _, block := range f.Blocks {
		if block.Type == "STRG" {
			continue
		}
		for _, section := range block.Sections {
			k := blockKey{Type: block.Type, Index: section.Index}
			want, ok := originalBlocks[k]
			if !ok {
				t.Fatalf("unexpected new section %v (existing sections should not move)", k)
			}
			if !bytes.Equal(section.Payload, want) {
				t.Fatalf("non-STRG section %v payload changed by SetDescription", k)
			}
		}
	}

	// End-to-end: write → parse → assert text.
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo err = %v", err)
	}
	round, err := lvrsrc.Parse(buf.Bytes(), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("re-Parse err = %v", err)
	}
	if got := readSTRG(t, round); got != newDesc {
		t.Fatalf("round-tripped STRG text = %q, want %q", got, newDesc)
	}
	assertNoValidateErrors(t, round)
}

func TestSetDescriptionCorpusCreatesNewSTRGEndToEnd(t *testing.T) {
	f := mustOpenCorpus(t, corpusFileWithoutSTRG)
	originalBlocks := snapshotBlocks(f)

	if blocksOfType(f, "STRG") != 0 {
		t.Fatalf("corpus fixture %s unexpectedly has STRG block(s)", corpusFileWithoutSTRG)
	}

	const newDesc = "Freshly inserted description."

	if err := SetDescription(f, newDesc); err != nil {
		t.Fatalf("SetDescription err = %v", err)
	}
	if blocksOfType(f, "STRG") != 1 {
		t.Fatalf("expected exactly 1 STRG block after creation, got %d", blocksOfType(f, "STRG"))
	}

	// All pre-existing sections are still present, byte-for-byte.
	for k, want := range originalBlocks {
		got, found := findSectionPayload(f, k)
		if !found {
			t.Fatalf("pre-existing section %v lost after SetDescription", k)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("pre-existing section %v payload changed", k)
		}
	}

	// End-to-end: serialize → re-parse → verify.
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo err = %v", err)
	}
	round, err := lvrsrc.Parse(buf.Bytes(), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("re-Parse err = %v", err)
	}
	if got := readSTRG(t, round); got != newDesc {
		t.Fatalf("round-tripped new STRG text = %q, want %q", got, newDesc)
	}
	assertNoValidateErrors(t, round)
}

func TestSetDescriptionCorpusEmptyStringRoundTrips(t *testing.T) {
	f := mustOpenCorpus(t, corpusFileWithSTRG)
	if err := SetDescription(f, ""); err != nil {
		t.Fatalf("SetDescription empty err = %v", err)
	}
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo err = %v", err)
	}
	round, err := lvrsrc.Parse(buf.Bytes(), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("re-Parse err = %v", err)
	}
	if got := readSTRG(t, round); got != "" {
		t.Fatalf("round-tripped empty STRG text = %q, want empty", got)
	}
	assertNoValidateErrors(t, round)
}

func TestSetNameCorpusOpaquePreservation(t *testing.T) {
	// Rename an LVSR-carrying corpus file and assert that every non-LVSR
	// section is byte-for-byte preserved.
	f := mustOpenCorpus(t, corpusFileWithSTRG)
	originalBlocks := snapshotBlocks(f)

	if err := SetName(f, "renamed-by-lvmeta.vi"); err != nil {
		t.Fatalf("SetName err = %v", err)
	}

	for _, block := range f.Blocks {
		if block.Type == "LVSR" {
			continue
		}
		for _, section := range block.Sections {
			k := blockKey{Type: block.Type, Index: section.Index}
			want, ok := originalBlocks[k]
			if !ok {
				t.Fatalf("unexpected new section %v during rename", k)
			}
			if !bytes.Equal(section.Payload, want) {
				t.Fatalf("non-LVSR section %v payload changed during rename", k)
			}
		}
	}
}

func TestSetDescriptionEndToEndRoundTrips(t *testing.T) {
	// Exercise the full pipeline on both branches (update + create) and
	// require both write and validate to be clean.
	cases := []struct {
		name    string
		file    string
		newDesc string
	}{
		{"update existing", corpusFileWithSTRG, "new desc via e2e update"},
		{"create new", corpusFileWithoutSTRG, "new desc via e2e create"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := mustOpenCorpus(t, tc.file)
			if err := SetDescription(f, tc.newDesc); err != nil {
				t.Fatalf("SetDescription err = %v", err)
			}
			var buf bytes.Buffer
			if _, err := f.WriteTo(&buf); err != nil {
				t.Fatalf("WriteTo err = %v", err)
			}
			round, err := lvrsrc.Parse(buf.Bytes(), lvrsrc.OpenOptions{})
			if err != nil {
				t.Fatalf("re-Parse err = %v", err)
			}
			if got := readSTRG(t, round); got != tc.newDesc {
				t.Fatalf("STRG text after round-trip = %q, want %q", got, tc.newDesc)
			}
			assertNoValidateErrors(t, round)
		})
	}
}

func TestSetNameEndToEndRoundTrips(t *testing.T) {
	f := mustOpenCorpus(t, corpusFileWithSTRG)
	const newName = "renamed-e2e.vi"
	if err := SetName(f, newName); err != nil {
		t.Fatalf("SetName err = %v", err)
	}
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo err = %v", err)
	}
	round, err := lvrsrc.Parse(buf.Bytes(), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("re-Parse err = %v", err)
	}
	for _, b := range round.Blocks {
		if b.Type != "LVSR" {
			continue
		}
		if got := b.Sections[0].Name; got != newName {
			t.Fatalf("round-tripped LVSR Name = %q, want %q", got, newName)
		}
	}
	assertNoValidateErrors(t, round)
}

// TestSetNameCompactionPath triggers serializer compaction by renaming a
// section in-place to a longer value that makes the existing sparse
// name-table layout overlap. The serializer must rewrite offsets, and the
// post-edit gate must accept the result.
func TestSetNameCompactionPath(t *testing.T) {
	// Two entries, packed tight: the second starts immediately after the
	// first. Renaming the first to a longer value would overlap the second
	// in the preserving layout, forcing the compaction branch of
	// rsrcwire.serializeNames.
	f := &lvrsrc.File{
		Header:          validHeader(),
		SecondaryHeader: validHeader(),
		Names: []lvrsrc.NameEntry{
			{Offset: 0, Value: "ab", Consumed: 3},
			{Offset: 3, Value: "followup", Consumed: 9},
		},
		Blocks: []lvrsrc.Block{
			{
				Type: "OPQ ",
				Sections: []lvrsrc.Section{
					{Index: 1, NameOffset: 3, Name: "followup", Payload: []byte{0xAA}},
				},
			},
			{
				Type: "LVSR",
				Sections: []lvrsrc.Section{
					{Index: 0, NameOffset: 0, Name: "ab", Payload: []byte{0x00}},
				},
			},
		},
	}

	if err := SetName(f, "a-much-longer-lvsr-name"); err != nil {
		t.Fatalf("SetName err = %v", err)
	}

	// Serialize + re-parse. The resulting file must still resolve both
	// section names correctly.
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo err = %v", err)
	}
	round, err := lvrsrc.Parse(buf.Bytes(), lvrsrc.OpenOptions{})
	if err != nil {
		t.Fatalf("re-Parse err = %v", err)
	}
	names := map[string]string{}
	for _, b := range round.Blocks {
		for _, s := range b.Sections {
			names[b.Type] = s.Name
		}
	}
	if names["LVSR"] != "a-much-longer-lvsr-name" {
		t.Fatalf("LVSR name post-compaction = %q, want %q", names["LVSR"], "a-much-longer-lvsr-name")
	}
	if names["OPQ "] != "followup" {
		t.Fatalf("OPQ  name post-compaction = %q, want followup", names["OPQ "])
	}
}

// TestRunStructuralCheckSkippedWhenBaselineCannotBeCaptured verifies the
// gate is tolerant of synthetic fixtures without valid headers. Capture
// fails up front → the gate is a no-op and other assertions still run.
func TestRunStructuralCheckSkippedWhenBaselineCannotBeCaptured(t *testing.T) {
	// f has no Magic/Type/Creator — serialize will fail immediately, so
	// the baseline cannot be captured.
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Blocks: []lvrsrc.Block{
			{Type: "STRG", Sections: []lvrsrc.Section{{Index: 0, Payload: strgPayload(t, "x")}}},
		},
	}
	if err := SetDescription(f, "y"); err != nil {
		t.Fatalf("SetDescription on synthetic fixture err = %v, want nil (gate should skip)", err)
	}
	if got := string(f.Blocks[0].Sections[0].Payload[4:]); got != "y" {
		t.Fatalf("payload text = %q, want y", got)
	}
}

// TestRunStructuralCheckFailsOnEditInducedError crafts an edit that, after
// serialize+reparse, produces a validator error that was NOT present
// before. The gate must reject it and roll the file back.
func TestRunStructuralCheckFailsOnEditInducedError(t *testing.T) {
	// Base the test on a corpus file so baseline captures successfully.
	f := mustOpenCorpus(t, corpusFileWithSTRG)

	// Manually corrupt the SectionCountMinusOne field of the STRG block's
	// section count AFTER SetDescription — we simulate the gate catching
	// a hypothetical edit-induced invariant break by reproducing the same
	// steps: baseline → mutate → runStructuralCheck.
	baseline := captureStructuralBaseline(f)
	if !baseline.captured {
		t.Skip("baseline capture failed on corpus fixture; cannot exercise gate")
	}

	// Induce a new structural error: empty out a block's section list so
	// the serializer rejects it — an invariant we can't quietly recover
	// from on the write path.
	var victim = -1
	for i, b := range f.Blocks {
		if len(b.Sections) >= 1 {
			victim = i
			break
		}
	}
	if victim < 0 {
		t.Skip("no suitable block to corrupt")
	}
	f.Blocks[victim].Sections = nil

	err := (Mutator{}).runStructuralCheck(f, "TEST", baseline)
	if !errors.Is(err, ErrStructuralValidation) {
		t.Fatalf("runStructuralCheck on corrupted file err = %v, want ErrStructuralValidation", err)
	}
}

// --- helpers --------------------------------------------------------------

func blocksOfType(f *lvrsrc.File, fourCC string) int {
	n := 0
	for _, b := range f.Blocks {
		if b.Type == fourCC {
			n++
		}
	}
	return n
}

func findSectionPayload(f *lvrsrc.File, k blockKey) ([]byte, bool) {
	for _, b := range f.Blocks {
		if b.Type != k.Type {
			continue
		}
		for _, s := range b.Sections {
			if s.Index == k.Index {
				return s.Payload, true
			}
		}
	}
	return nil, false
}

func readSTRG(t *testing.T, f *lvrsrc.File) string {
	t.Helper()
	for _, b := range f.Blocks {
		if b.Type != "STRG" {
			continue
		}
		if len(b.Sections) == 0 {
			t.Fatalf("STRG block has no sections")
		}
		p := b.Sections[0].Payload
		if len(p) < 4 {
			t.Fatalf("STRG payload too short: %d", len(p))
		}
		size := binary.BigEndian.Uint32(p[:4])
		if 4+int(size) > len(p) {
			t.Fatalf("STRG size overruns payload")
		}
		return string(p[4 : 4+size])
	}
	return ""
}

func assertNoValidateErrors(t *testing.T, f *lvrsrc.File) {
	t.Helper()
	issues := f.Validate()
	var codes []string
	for _, iss := range issues {
		if iss.Severity == lvrsrc.SeverityError {
			codes = append(codes, iss.Code+": "+iss.Message)
		}
	}
	if len(codes) > 0 {
		sort.Strings(codes)
		t.Fatalf("round-tripped file has %d validator errors:\n  %v", len(codes), codes)
	}
}

func validHeader() lvrsrc.Header {
	return lvrsrc.Header{
		Magic:         "RSRC\r\n",
		FormatVersion: 3,
		Type:          "LVIN",
		Creator:       "LBVW",
	}
}
