package lvmeta

import (
	"errors"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// fileWithLVSR returns a file whose only block is LVSR with a single named
// section. The section's Name is backed by a distinct NameEntry in f.Names.
func fileWithLVSR(t *testing.T, lvsrName string) *lvrsrc.File {
	t.Helper()
	const nameOffset uint32 = 0
	return &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Kind:   lvrsrc.FileKindVI,
		Names: []lvrsrc.NameEntry{
			{Offset: nameOffset, Value: lvsrName, Consumed: int64(1 + len(lvsrName))},
		},
		Blocks: []lvrsrc.Block{
			{
				Type: lvsrFourCC,
				Sections: []lvrsrc.Section{
					{Index: 0, NameOffset: nameOffset, Name: lvsrName, Payload: []byte{0x00, 0x01, 0x02, 0x03}},
				},
			},
		},
	}
}

func TestSetNameUpdatesExistingEntryInPlace(t *testing.T) {
	f := fileWithLVSR(t, "Old.vi")

	if err := SetName(f, "New.vi"); err != nil {
		t.Fatalf("SetName err = %v, want nil", err)
	}

	section := f.Blocks[0].Sections[0]
	if section.Name != "New.vi" {
		t.Fatalf("section.Name = %q, want New.vi", section.Name)
	}
	// Offset should be preserved because we updated the entry in place.
	if section.NameOffset != 0 {
		t.Fatalf("section.NameOffset = %d, want 0 (preserved)", section.NameOffset)
	}
	if len(f.Names) != 1 {
		t.Fatalf("len(f.Names) = %d, want 1 (in-place update, no new entry)", len(f.Names))
	}
	if f.Names[0].Value != "New.vi" {
		t.Fatalf("f.Names[0].Value = %q, want New.vi", f.Names[0].Value)
	}
	if f.Names[0].Offset != 0 {
		t.Fatalf("f.Names[0].Offset = %d, want 0 (preserved)", f.Names[0].Offset)
	}
	if f.Names[0].Consumed != int64(1+len("New.vi")) {
		t.Fatalf("f.Names[0].Consumed = %d, want %d", f.Names[0].Consumed, 1+len("New.vi"))
	}
}

func TestSetNameReusesExistingEntryWhenAnotherCarriesIt(t *testing.T) {
	// f has LVSR named "Old" and an OPQ section named "Target". Renaming
	// LVSR to "Target" must reuse the existing name-table entry for "Target"
	// rather than creating a duplicate.
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Names: []lvrsrc.NameEntry{
			{Offset: 0, Value: "Old", Consumed: 4},
			{Offset: 4, Value: "Target", Consumed: 7},
		},
		Blocks: []lvrsrc.Block{
			{
				Type: "OPQ ",
				Sections: []lvrsrc.Section{
					{Index: 1, NameOffset: 4, Name: "Target", Payload: []byte{0xAA}},
				},
			},
			{
				Type: lvsrFourCC,
				Sections: []lvrsrc.Section{
					{Index: 0, NameOffset: 0, Name: "Old", Payload: []byte{0x00}},
				},
			},
		},
	}
	beforeNamesLen := len(f.Names)

	if err := SetName(f, "Target"); err != nil {
		t.Fatalf("SetName err = %v, want nil", err)
	}

	lvsrSection := f.Blocks[1].Sections[0]
	if lvsrSection.Name != "Target" {
		t.Fatalf("LVSR section.Name = %q, want Target", lvsrSection.Name)
	}
	if lvsrSection.NameOffset != 4 {
		t.Fatalf("LVSR section.NameOffset = %d, want 4 (reused)", lvsrSection.NameOffset)
	}
	if len(f.Names) != beforeNamesLen {
		t.Fatalf("len(f.Names) = %d, want %d (no new entry should be added)", len(f.Names), beforeNamesLen)
	}
	// The old "Old" entry remains because no other section references it
	// (the sparse name table is preserved verbatim; the serializer will
	// drop unreachable entries if needed). Ensure we did not modify it.
	if f.Names[0].Value != "Old" || f.Names[0].Offset != 0 {
		t.Fatalf("f.Names[0] mutated: %+v", f.Names[0])
	}
	// Other section's name must be untouched.
	if f.Blocks[0].Sections[0].Name != "Target" || f.Blocks[0].Sections[0].NameOffset != 4 {
		t.Fatalf("OPQ section changed: %+v", f.Blocks[0].Sections[0])
	}
}

func TestSetNameAddsNewEntryWhenCurrentOffsetIsSharedByOthers(t *testing.T) {
	// Two sections share NameOffset 0 → "Shared". Renaming LVSR must
	// preserve the other section's name, which means the LVSR must get
	// a fresh entry rather than edit the shared one in place.
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Names: []lvrsrc.NameEntry{
			{Offset: 0, Value: "Shared", Consumed: 7},
		},
		Blocks: []lvrsrc.Block{
			{
				Type: "OPQ ",
				Sections: []lvrsrc.Section{
					{Index: 1, NameOffset: 0, Name: "Shared", Payload: []byte{0xAA}},
				},
			},
			{
				Type: lvsrFourCC,
				Sections: []lvrsrc.Section{
					{Index: 0, NameOffset: 0, Name: "Shared", Payload: []byte{0x00}},
				},
			},
		},
	}

	if err := SetName(f, "LVSRUnique"); err != nil {
		t.Fatalf("SetName err = %v, want nil", err)
	}

	lvsr := f.Blocks[1].Sections[0]
	opq := f.Blocks[0].Sections[0]

	if lvsr.Name != "LVSRUnique" {
		t.Fatalf("LVSR section.Name = %q, want LVSRUnique", lvsr.Name)
	}
	if lvsr.NameOffset == 0 {
		t.Fatalf("LVSR section.NameOffset = 0, should be a fresh offset (shared offset must not be repurposed)")
	}
	if opq.Name != "Shared" || opq.NameOffset != 0 {
		t.Fatalf("OPQ section changed: %+v", opq)
	}

	var foundLvsrEntry, foundSharedEntry bool
	for _, e := range f.Names {
		if e.Offset == lvsr.NameOffset && e.Value == "LVSRUnique" {
			foundLvsrEntry = true
		}
		if e.Offset == 0 && e.Value == "Shared" {
			foundSharedEntry = true
		}
	}
	if !foundLvsrEntry {
		t.Fatalf("no f.Names entry matches LVSR's new offset/value: %+v", f.Names)
	}
	if !foundSharedEntry {
		t.Fatalf("Shared entry for offset 0 lost: %+v", f.Names)
	}
}

func TestSetNameAddsNewEntryWhenSectionHadNoName(t *testing.T) {
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Blocks: []lvrsrc.Block{
			{
				Type: lvsrFourCC,
				Sections: []lvrsrc.Section{
					{Index: 0, NameOffset: noNameOffsetSentinel, Name: "", Payload: []byte{0x00}},
				},
			},
		},
	}

	if err := SetName(f, "Fresh"); err != nil {
		t.Fatalf("SetName err = %v, want nil", err)
	}

	s := f.Blocks[0].Sections[0]
	if s.Name != "Fresh" {
		t.Fatalf("section.Name = %q, want Fresh", s.Name)
	}
	if s.NameOffset == noNameOffsetSentinel {
		t.Fatalf("section.NameOffset still noName sentinel after rename")
	}
	if len(f.Names) != 1 {
		t.Fatalf("len(f.Names) = %d, want 1", len(f.Names))
	}
	if f.Names[0].Value != "Fresh" || f.Names[0].Offset != s.NameOffset {
		t.Fatalf("f.Names[0] = %+v, does not match section", f.Names[0])
	}
}

func TestSetNameRejectsPascalOverflow(t *testing.T) {
	f := fileWithLVSR(t, "Old")
	long := strings.Repeat("x", 256)

	err := SetName(f, long)
	if !errors.Is(err, ErrNameTooLong) {
		t.Fatalf("err = %v, want ErrNameTooLong", err)
	}
	if f.Blocks[0].Sections[0].Name != "Old" {
		t.Fatalf("section.Name mutated despite error: %q", f.Blocks[0].Sections[0].Name)
	}
}

func TestSetNameTargetMissing(t *testing.T) {
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Blocks: []lvrsrc.Block{{Type: "OPQ ", Sections: []lvrsrc.Section{{Index: 0, Payload: []byte{0}}}}},
	}
	err := SetName(f, "X")
	if !errors.Is(err, ErrTargetMissing) {
		t.Fatalf("err = %v, want ErrTargetMissing", err)
	}
}

func TestSetNameTargetAmbiguous(t *testing.T) {
	f := &lvrsrc.File{
		Header: lvrsrc.Header{FormatVersion: 10},
		Blocks: []lvrsrc.Block{
			{Type: lvsrFourCC, Sections: []lvrsrc.Section{
				{Index: 0, NameOffset: noNameOffsetSentinel, Payload: []byte{0}},
				{Index: 1, NameOffset: noNameOffsetSentinel, Payload: []byte{0}},
			}},
		},
	}
	err := SetName(f, "X")
	if !errors.Is(err, ErrTargetAmbiguous) {
		t.Fatalf("err = %v, want ErrTargetAmbiguous", err)
	}
}

func TestSetNameNilFile(t *testing.T) {
	if err := SetName(nil, "x"); !errors.Is(err, ErrNilFile) {
		t.Fatalf("err = %v, want ErrNilFile", err)
	}
}

func TestSetNameNoOp(t *testing.T) {
	f := fileWithLVSR(t, "Same.vi")
	beforeNames := append([]lvrsrc.NameEntry(nil), f.Names...)
	beforeOffset := f.Blocks[0].Sections[0].NameOffset

	if err := SetName(f, "Same.vi"); err != nil {
		t.Fatalf("SetName err = %v, want nil", err)
	}

	if f.Blocks[0].Sections[0].NameOffset != beforeOffset {
		t.Fatalf("NameOffset changed from %d to %d on no-op rename", beforeOffset, f.Blocks[0].Sections[0].NameOffset)
	}
	if len(f.Names) != len(beforeNames) {
		t.Fatalf("name table grew on no-op rename: %d -> %d", len(beforeNames), len(f.Names))
	}
}

func TestSetNameAcceptsExactly255Bytes(t *testing.T) {
	f := fileWithLVSR(t, "Old")
	boundary := strings.Repeat("y", 255)
	if err := SetName(f, boundary); err != nil {
		t.Fatalf("SetName 255-byte err = %v, want nil", err)
	}
	if f.Blocks[0].Sections[0].Name != boundary {
		t.Fatalf("section name = %q, want 255-byte string", f.Blocks[0].Sections[0].Name)
	}
}
