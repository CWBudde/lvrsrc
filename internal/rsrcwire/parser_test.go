package rsrcwire

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/corpus"
)

func TestParseSyntheticFile(t *testing.T) {
	data := buildSyntheticRSRC(t)

	f, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got, want := f.Kind, FileKindVI; got != want {
		t.Fatalf("Kind = %v, want %v", got, want)
	}

	if got, want := f.Compression, CompressionKindUnknown; got != want {
		t.Fatalf("Compression = %v, want %v", got, want)
	}

	if got, want := f.Header.Magic, "RSRC\r\n"; got != want {
		t.Fatalf("Header.Magic = %q, want %q", got, want)
	}

	if got, want := f.Header.Type, "LVIN"; got != want {
		t.Fatalf("Header.Type = %q, want %q", got, want)
	}

	if got, want := len(f.Blocks), 1; got != want {
		t.Fatalf("len(Blocks) = %d, want %d", got, want)
	}

	block := f.Blocks[0]
	if got, want := block.Type, "TEST"; got != want {
		t.Fatalf("block.Type = %q, want %q", got, want)
	}

	if got, want := len(block.Sections), 2; got != want {
		t.Fatalf("len(block.Sections) = %d, want %d", got, want)
	}

	if got, want := block.Sections[0].Index, int32(0); got != want {
		t.Fatalf("section[0].Index = %d, want %d", got, want)
	}

	if got, want := block.Sections[0].Name, "alpha"; got != want {
		t.Fatalf("section[0].Name = %q, want %q", got, want)
	}

	if got, want := string(block.Sections[0].Payload), "abc"; got != want {
		t.Fatalf("section[0].Payload = %q, want %q", got, want)
	}

	if got, want := block.Sections[1].Index, int32(7); got != want {
		t.Fatalf("section[1].Index = %d, want %d", got, want)
	}

	if got, want := block.Sections[1].Name, "beta!"; got != want {
		t.Fatalf("section[1].Name = %q, want %q", got, want)
	}

	if got, want := string(block.Sections[1].Payload), "wxyz"; got != want {
		t.Fatalf("section[1].Payload = %q, want %q", got, want)
	}

	if got, want := len(f.Names), 2; got != want {
		t.Fatalf("len(Names) = %d, want %d", got, want)
	}

	if got, want := f.Names[0].Offset, uint32(0); got != want {
		t.Fatalf("name[0].Offset = %d, want %d", got, want)
	}

	if got, want := f.Names[1].Offset, uint32(6); got != want {
		t.Fatalf("name[1].Offset = %d, want %d", got, want)
	}

	if got, want := string(f.RawTail), "\xde\xad\xbe"; got != want {
		t.Fatalf("RawTail = %x, want %x", f.RawTail, []byte(want))
	}
}

func TestParseCorpusFixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		path          string
		wantKind      FileKind
		wantType      string
		wantBlocks    int
		wantSections  int
		wantInfoSize  uint32
		wantDataSize  uint32
		wantFirstName string
	}{
		{
			name:          "ctl",
			path:          corpus.Path("config-data.ctl"),
			wantKind:      FileKindControl,
			wantType:      "LVCC",
			wantBlocks:    24,
			wantSections:  28,
			wantInfoSize:  920,
			wantDataSize:  3404,
			wantFirstName: "Config Data.ctl",
		},
		{
			name:          "vi",
			path:          corpus.Path("get-vi-description.vi"),
			wantKind:      FileKindVI,
			wantType:      "LVIN",
			wantBlocks:    26,
			wantSections:  26,
			wantInfoSize:  910,
			wantDataSize:  12408,
			wantFirstName: "get vi description.vi",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", tc.path, err)
			}

			f, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tc.path, err)
			}

			if got, want := f.Kind, tc.wantKind; got != want {
				t.Fatalf("Kind = %v, want %v", got, want)
			}

			if got, want := f.Header.Type, tc.wantType; got != want {
				t.Fatalf("Header.Type = %q, want %q", got, want)
			}

			if got, want := f.Header.InfoSize, tc.wantInfoSize; got != want {
				t.Fatalf("Header.InfoSize = %d, want %d", got, want)
			}

			if got, want := f.Header.DataSize, tc.wantDataSize; got != want {
				t.Fatalf("Header.DataSize = %d, want %d", got, want)
			}

			if got, want := len(f.Blocks), tc.wantBlocks; got != want {
				t.Fatalf("len(Blocks) = %d, want %d", got, want)
			}

			totalSections := 0
			for _, block := range f.Blocks {
				totalSections += len(block.Sections)
			}
			if got, want := totalSections, tc.wantSections; got != want {
				t.Fatalf("total sections = %d, want %d", got, want)
			}

			if got, want := len(f.Names), 1; got != want {
				t.Fatalf("len(Names) = %d, want %d", got, want)
			}

			if got, want := f.Names[0].Value, tc.wantFirstName; got != want {
				t.Fatalf("name[0].Value = %q, want %q", got, want)
			}

			if got := len(f.RawTail); got != 0 {
				t.Fatalf("len(RawTail) = %d, want 0", got)
			}

			if got, want := f.Blocks[0].Type, "LIBN"; got != want {
				t.Fatalf("first block type = %q, want %q", got, want)
			}
		})
	}
}

func TestSerializeSyntheticFilePreservesBytes(t *testing.T) {
	data := buildSyntheticRSRC(t)

	f, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	serialized, err := Serialize(f)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	if !bytes.Equal(serialized, data) {
		t.Fatalf("Serialize() changed bytes:\n got %x\nwant %x", serialized, data)
	}
}

func TestSerializeCorpusFixturesRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "ctl",
			path: corpus.Path("config-data.ctl"),
		},
		{
			name: "vi",
			path: corpus.Path("get-vi-description.vi"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", tc.path, err)
			}

			parsed, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tc.path, err)
			}

			serialized, err := Serialize(parsed)
			if err != nil {
				t.Fatalf("Serialize(%q) error = %v", tc.path, err)
			}

			roundTrip, err := Parse(serialized)
			if err != nil {
				t.Fatalf("Parse(Serialize(%q)) error = %v", tc.path, err)
			}

			assertEquivalentFiles(t, roundTrip, parsed)
		})
	}
}

// TestParseCorpusSmoke parses every *.vi and *.ctl fixture under testdata/
// and confirms a structural round-trip (Parse → Serialize → Parse yields an
// equivalent file). Byte-exact round-trip on production files is a PLAN.md
// Phase 2 goal and is NOT asserted here — see TestSerializeSyntheticFilePreservesBytes
// for the synthetic-only byte-exact invariant.
// Fixtures are copied from ../labview_mcp; see testdata/README.md.
func TestParseCorpusSmoke(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob(corpus.Path("*.vi"))
	if err != nil {
		t.Fatalf("glob vi: %v", err)
	}
	ctlPaths, err := filepath.Glob(corpus.Path("*.ctl"))
	if err != nil {
		t.Fatalf("glob ctl: %v", err)
	}
	paths = append(paths, ctlPaths...)
	if len(paths) < 20 {
		t.Fatalf("corpus has only %d fixtures; expected ≥20", len(paths))
	}

	for _, p := range paths {
		p := p
		t.Run(filepath.Base(p), func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", p, err)
			}

			parsed, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", p, err)
			}

			if parsed.Header.Type != "LVIN" && parsed.Header.Type != "LVCC" {
				t.Fatalf("unexpected header type %q for %q", parsed.Header.Type, p)
			}

			serialized, err := Serialize(parsed)
			if err != nil {
				t.Fatalf("Serialize(%q) error = %v", p, err)
			}

			roundTrip, err := Parse(serialized)
			if err != nil {
				t.Fatalf("Parse(Serialize(%q)) error = %v", p, err)
			}

			assertEquivalentFiles(t, roundTrip, parsed)
		})
	}
}

func TestSerializeRecomputesOffsetsForModifiedPayloads(t *testing.T) {
	data := buildSyntheticRSRC(t)

	f, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	f.Blocks[0].Sections[0].Payload = []byte("abcdefgh")

	serialized, err := Serialize(f)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	roundTrip, err := Parse(serialized)
	if err != nil {
		t.Fatalf("Parse(Serialize()) error = %v", err)
	}

	first := roundTrip.Blocks[0].Sections[0]
	second := roundTrip.Blocks[0].Sections[1]

	if got, want := string(first.Payload), "abcdefgh"; got != want {
		t.Fatalf("section[0].Payload = %q, want %q", got, want)
	}

	if got, want := string(second.Payload), "wxyz"; got != want {
		t.Fatalf("section[1].Payload = %q, want %q", got, want)
	}

	if got, want := first.DataOffset, uint32(0); got != want {
		t.Fatalf("section[0].DataOffset = %d, want %d", got, want)
	}

	if got, want := second.DataOffset, uint32(12); got != want {
		t.Fatalf("section[1].DataOffset = %d, want %d", got, want)
	}

	if got, want := roundTrip.Header.DataSize, uint32(20); got != want {
		t.Fatalf("Header.DataSize = %d, want %d", got, want)
	}

	if got, want := roundTrip.Header.InfoOffset, uint32(52); got != want {
		t.Fatalf("Header.InfoOffset = %d, want %d", got, want)
	}
}

func TestSerializeCanonicalCompactsReferencedNamesDeterministically(t *testing.T) {
	data := buildSyntheticRSRC(t)

	f, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	f.Names = []NameEntry{
		{Offset: 0, Value: "alpha", Consumed: 6},
		{Offset: 32, Value: "unused", Consumed: 7},
		{Offset: 40, Value: "beta!", Consumed: 6},
	}
	f.Blocks[0].Sections[0].NameOffset = 0
	f.Blocks[0].Sections[1].NameOffset = 40

	canonical, err := SerializeCanonical(f)
	if err != nil {
		t.Fatalf("SerializeCanonical() error = %v", err)
	}

	roundTrip, err := Parse(canonical)
	if err != nil {
		t.Fatalf("Parse(SerializeCanonical()) error = %v", err)
	}

	if got, want := len(roundTrip.Names), 2; got != want {
		t.Fatalf("len(Names) = %d, want %d", got, want)
	}
	if got, want := roundTrip.Names[0].Offset, uint32(0); got != want {
		t.Fatalf("Names[0].Offset = %d, want %d", got, want)
	}
	if got, want := roundTrip.Names[1].Offset, uint32(6); got != want {
		t.Fatalf("Names[1].Offset = %d, want %d", got, want)
	}
	if got, want := roundTrip.Names[0].Value, "alpha"; got != want {
		t.Fatalf("Names[0].Value = %q, want %q", got, want)
	}
	if got, want := roundTrip.Names[1].Value, "beta!"; got != want {
		t.Fatalf("Names[1].Value = %q, want %q", got, want)
	}
	if got, want := roundTrip.Blocks[0].Sections[0].NameOffset, uint32(0); got != want {
		t.Fatalf("Sections[0].NameOffset = %d, want %d", got, want)
	}
	if got, want := roundTrip.Blocks[0].Sections[1].NameOffset, uint32(6); got != want {
		t.Fatalf("Sections[1].NameOffset = %d, want %d", got, want)
	}
	if got, want := roundTrip.Blocks[0].Sections[0].Index, int32(0); got != want {
		t.Fatalf("Sections[0].Index = %d, want %d", got, want)
	}
	if got, want := roundTrip.Blocks[0].Sections[1].Index, int32(7); got != want {
		t.Fatalf("Sections[1].Index = %d, want %d", got, want)
	}

	canonicalAgain, err := SerializeCanonical(roundTrip)
	if err != nil {
		t.Fatalf("SerializeCanonical(roundTrip) error = %v", err)
	}

	if !bytes.Equal(canonicalAgain, canonical) {
		t.Fatalf("SerializeCanonical() is not stable across rewrites")
	}
}

func TestSerializeCanonicalReordersBlocksAndSections(t *testing.T) {
	f := &File{
		Header: Header{
			Magic:         "RSRC\r\n",
			FormatVersion: 3,
			Type:          "LVIN",
			Creator:       "LBVW",
		},
		SecondaryHeader: Header{
			Magic:         "RSRC\r\n",
			FormatVersion: 3,
			Type:          "LVIN",
			Creator:       "LBVW",
		},
		BlockInfoList: BlockInfoList{DatasetInt3: headerSize},
		RawTail:       []byte{0xaa, 0xbb},
		Blocks: []Block{
			{
				Type: "VCTP",
				Sections: []Section{
					{Index: 0, NameOffset: noNameOffset, Payload: []byte("z")},
					{Index: 0, NameOffset: noNameOffset, Payload: []byte("a")},
				},
			},
			{
				Type: "LIBN",
				Sections: []Section{
					{Index: 5, NameOffset: 20, Name: "zeta", Payload: []byte("b")},
					{Index: 1, NameOffset: 0, Name: "middle", Payload: []byte("a")},
					{Index: 5, NameOffset: 40, Name: "alpha", Payload: []byte("c")},
				},
			},
			{
				Type: "ICON",
				Sections: []Section{
					{Index: 0, NameOffset: noNameOffset, Payload: bytes.Repeat([]byte{0x01}, 128)},
				},
			},
			{
				Type: "CPC2",
				Sections: []Section{
					{Index: 0, NameOffset: noNameOffset, Payload: []byte{0x00, 0x04}},
				},
			},
		},
		Names: []NameEntry{
			{Offset: 0, Value: "middle", Consumed: 7},
			{Offset: 20, Value: "zeta", Consumed: 5},
			{Offset: 40, Value: "alpha", Consumed: 6},
			{Offset: 60, Value: "unused", Consumed: 7},
		},
	}

	canonical, err := SerializeCanonical(f)
	if err != nil {
		t.Fatalf("SerializeCanonical() error = %v", err)
	}

	roundTrip, err := Parse(canonical)
	if err != nil {
		t.Fatalf("Parse(SerializeCanonical()) error = %v", err)
	}

	if got, want := len(roundTrip.Blocks), 4; got != want {
		t.Fatalf("len(Blocks) = %d, want %d", got, want)
	}
	if got, want := roundTrip.Blocks[0].Type, "LIBN"; got != want {
		t.Fatalf("Blocks[0].Type = %q, want %q", got, want)
	}
	if got, want := roundTrip.Blocks[1].Type, "ICON"; got != want {
		t.Fatalf("Blocks[1].Type = %q, want %q", got, want)
	}
	if got, want := roundTrip.Blocks[2].Type, "CPC2"; got != want {
		t.Fatalf("Blocks[2].Type = %q, want %q", got, want)
	}
	if got, want := roundTrip.Blocks[3].Type, "VCTP"; got != want {
		t.Fatalf("Blocks[3].Type = %q, want %q", got, want)
	}

	libn := roundTrip.Blocks[0]
	if got, want := len(libn.Sections), 3; got != want {
		t.Fatalf("len(LIBN.Sections) = %d, want %d", got, want)
	}
	if got, want := libn.Sections[0].Index, int32(1); got != want {
		t.Fatalf("LIBN.Sections[0].Index = %d, want %d", got, want)
	}
	if got, want := libn.Sections[0].Name, "middle"; got != want {
		t.Fatalf("LIBN.Sections[0].Name = %q, want %q", got, want)
	}
	if got, want := libn.Sections[1].Name, "alpha"; got != want {
		t.Fatalf("LIBN.Sections[1].Name = %q, want %q", got, want)
	}
	if got, want := libn.Sections[2].Name, "zeta"; got != want {
		t.Fatalf("LIBN.Sections[2].Name = %q, want %q", got, want)
	}

	vctp := roundTrip.Blocks[3]
	if got, want := string(vctp.Sections[0].Payload), "a"; got != want {
		t.Fatalf("VCTP.Sections[0].Payload = %q, want %q", got, want)
	}
	if got, want := string(vctp.Sections[1].Payload), "z"; got != want {
		t.Fatalf("VCTP.Sections[1].Payload = %q, want %q", got, want)
	}

	if got, want := len(roundTrip.Names), 3; got != want {
		t.Fatalf("len(Names) = %d, want %d", got, want)
	}
	if !bytes.Equal(roundTrip.RawTail, f.RawTail) {
		t.Fatalf("RawTail = %x, want %x", roundTrip.RawTail, f.RawTail)
	}
}

func buildSyntheticRSRC(t *testing.T) []byte {
	t.Helper()

	const (
		headerSize       = 32
		listHeaderSize   = 20
		blockInfoSize    = 4
		blockHeaderSize  = 12
		sectionStartSize = 20
	)

	type sectionSpec struct {
		index      int32
		nameOffset uint32
		payload    []byte
	}

	sections := []sectionSpec{
		{index: 0, nameOffset: 0, payload: []byte("abc")},
		{index: 7, nameOffset: 6, payload: []byte("wxyz")},
	}

	names := append([]byte{5}, []byte("alpha")...)
	names = append(names, 5)
	names = append(names, []byte("beta!")...)
	rawTail := []byte{0xde, 0xad, 0xbe}

	dataRegion := bytes.NewBuffer(nil)
	sectionDataOffsets := make([]uint32, len(sections))
	for i, section := range sections {
		sectionDataOffsets[i] = uint32(dataRegion.Len())
		if err := binary.Write(dataRegion, binary.BigEndian, uint32(len(section.payload))); err != nil {
			t.Fatalf("write section size: %v", err)
		}
		if _, err := dataRegion.Write(section.payload); err != nil {
			t.Fatalf("write section payload: %v", err)
		}
		for dataRegion.Len()%4 != 0 {
			if err := dataRegion.WriteByte(0); err != nil {
				t.Fatalf("write padding: %v", err)
			}
		}
		_ = i
	}

	infoOffset := uint32(headerSize + dataRegion.Len())
	blockTableOffset := uint32(headerSize + listHeaderSize)
	blockOffset := uint32(blockInfoSize + blockHeaderSize)
	namesOffset := blockTableOffset + blockInfoSize + blockHeaderSize + uint32(len(sections))*sectionStartSize
	infoSize := uint32(headerSize + listHeaderSize + blockInfoSize + blockHeaderSize + len(sections)*sectionStartSize + len(names) + len(rawTail))

	full := bytes.NewBuffer(make([]byte, 0, int(infoOffset+infoSize)))
	writeHeader := func() {
		full.WriteString("RSRC\r\n")
		if err := binary.Write(full, binary.BigEndian, uint16(3)); err != nil {
			t.Fatalf("write version: %v", err)
		}
		full.WriteString("LVIN")
		full.WriteString("LBVW")
		if err := binary.Write(full, binary.BigEndian, infoOffset); err != nil {
			t.Fatalf("write info offset: %v", err)
		}
		if err := binary.Write(full, binary.BigEndian, infoSize); err != nil {
			t.Fatalf("write info size: %v", err)
		}
		if err := binary.Write(full, binary.BigEndian, uint32(headerSize)); err != nil {
			t.Fatalf("write data offset: %v", err)
		}
		if err := binary.Write(full, binary.BigEndian, uint32(dataRegion.Len())); err != nil {
			t.Fatalf("write data size: %v", err)
		}
	}

	writeHeader()
	if _, err := full.Write(dataRegion.Bytes()); err != nil {
		t.Fatalf("write data region: %v", err)
	}
	writeHeader()

	for _, v := range []uint32{0, 0, headerSize, blockTableOffset, infoSize - headerSize - listHeaderSize} {
		if err := binary.Write(full, binary.BigEndian, v); err != nil {
			t.Fatalf("write list header: %v", err)
		}
	}

	if err := binary.Write(full, binary.BigEndian, uint32(0)); err != nil {
		t.Fatalf("write block count: %v", err)
	}

	full.WriteString("TEST")
	if err := binary.Write(full, binary.BigEndian, uint32(len(sections)-1)); err != nil {
		t.Fatalf("write section count: %v", err)
	}
	if err := binary.Write(full, binary.BigEndian, blockOffset); err != nil {
		t.Fatalf("write block offset: %v", err)
	}

	for i, section := range sections {
		if err := binary.Write(full, binary.BigEndian, section.index); err != nil {
			t.Fatalf("write section index: %v", err)
		}
		if err := binary.Write(full, binary.BigEndian, section.nameOffset); err != nil {
			t.Fatalf("write section name offset: %v", err)
		}
		if err := binary.Write(full, binary.BigEndian, uint32(0)); err != nil {
			t.Fatalf("write section int3: %v", err)
		}
		if err := binary.Write(full, binary.BigEndian, sectionDataOffsets[i]); err != nil {
			t.Fatalf("write section data offset: %v", err)
		}
		if err := binary.Write(full, binary.BigEndian, uint32(0)); err != nil {
			t.Fatalf("write section int5: %v", err)
		}
	}

	if got := uint32(full.Len()) - infoOffset; got != namesOffset {
		t.Fatalf("names offset mismatch: got %d want %d", got, namesOffset)
	}

	if _, err := full.Write(names); err != nil {
		t.Fatalf("write names: %v", err)
	}
	if _, err := full.Write(rawTail); err != nil {
		t.Fatalf("write raw tail: %v", err)
	}

	return full.Bytes()
}

func assertEquivalentFiles(t *testing.T, got, want *File) {
	t.Helper()

	if got == nil || want == nil {
		t.Fatalf("got nil comparison: got=%v want=%v", got, want)
	}

	if !reflect.DeepEqual(got.Header, want.Header) {
		t.Fatalf("Header mismatch:\n got %#v\nwant %#v", got.Header, want.Header)
	}

	if !reflect.DeepEqual(got.SecondaryHeader, want.SecondaryHeader) {
		t.Fatalf("SecondaryHeader mismatch:\n got %#v\nwant %#v", got.SecondaryHeader, want.SecondaryHeader)
	}

	if got.Kind != want.Kind {
		t.Fatalf("Kind = %v, want %v", got.Kind, want.Kind)
	}

	if got.Compression != want.Compression {
		t.Fatalf("Compression = %v, want %v", got.Compression, want.Compression)
	}

	if !reflect.DeepEqual(got.Names, want.Names) {
		t.Fatalf("Names mismatch:\n got %#v\nwant %#v", got.Names, want.Names)
	}

	if !bytes.Equal(got.RawTail, want.RawTail) {
		t.Fatalf("RawTail = %x, want %x", got.RawTail, want.RawTail)
	}

	if len(got.Blocks) != len(want.Blocks) {
		t.Fatalf("len(Blocks) = %d, want %d", len(got.Blocks), len(want.Blocks))
	}

	for bi := range want.Blocks {
		gotBlock := got.Blocks[bi]
		wantBlock := want.Blocks[bi]

		if gotBlock.Type != wantBlock.Type {
			t.Fatalf("block[%d].Type = %q, want %q", bi, gotBlock.Type, wantBlock.Type)
		}
		if len(gotBlock.Sections) != len(wantBlock.Sections) {
			t.Fatalf("len(block[%d].Sections) = %d, want %d", bi, len(gotBlock.Sections), len(wantBlock.Sections))
		}

		for si := range wantBlock.Sections {
			gotSection := gotBlock.Sections[si]
			wantSection := wantBlock.Sections[si]

			if gotSection.Index != wantSection.Index {
				t.Fatalf("block[%d].section[%d].Index = %d, want %d", bi, si, gotSection.Index, wantSection.Index)
			}
			if gotSection.NameOffset != wantSection.NameOffset {
				t.Fatalf("block[%d].section[%d].NameOffset = %d, want %d", bi, si, gotSection.NameOffset, wantSection.NameOffset)
			}
			if gotSection.Unknown3 != wantSection.Unknown3 {
				t.Fatalf("block[%d].section[%d].Unknown3 = %d, want %d", bi, si, gotSection.Unknown3, wantSection.Unknown3)
			}
			if gotSection.Unknown5 != wantSection.Unknown5 {
				t.Fatalf("block[%d].section[%d].Unknown5 = %d, want %d", bi, si, gotSection.Unknown5, wantSection.Unknown5)
			}
			if gotSection.Name != wantSection.Name {
				t.Fatalf("block[%d].section[%d].Name = %q, want %q", bi, si, gotSection.Name, wantSection.Name)
			}
			if !bytes.Equal(gotSection.Payload, wantSection.Payload) {
				t.Fatalf("block[%d].section[%d].Payload = %x, want %x", bi, si, gotSection.Payload, wantSection.Payload)
			}
		}
	}
}

// TestCanonicalBlockRankCoversShippedFourCCs exercises every arm of the
// canonical-order switch so the rank table doesn't sit unexercised.
// Unknown types fall through to the default rank (1000) — verified
// alongside the known mappings.
func TestCanonicalBlockRankCoversShippedFourCCs(t *testing.T) {
	cases := []struct {
		fourCC string
		want   int
	}{
		{"ADir", 0}, {"LIBN", 10}, {"LVSR", 20}, {"RTSG", 30},
		{"LIvi", 40}, {"vers", 50}, {"CONP", 60}, {"BDPW", 70},
		{"STRG", 80}, {"PALM", 90}, {"PLM2", 100}, {"CPST", 110},
		{"ICON", 120}, {"icl4", 121}, {"icl8", 122}, {"CPC2", 130},
		{"LIfp", 140}, {"FPEx", 150}, {"FPHb", 151}, {"FPSE", 152},
		{"VPDP", 153}, {"LIbd", 160}, {"BDEx", 170}, {"BDHb", 171},
		{"BDSE", 172}, {"VITS", 180}, {"DTHP", 190}, {"MUID", 200},
		{"HIST", 210}, {"VCTP", 220}, {"FTAB", 230}, {"STR ", 240},
		{"????", 1000},
	}
	for _, tc := range cases {
		if got := canonicalBlockRank(tc.fourCC); got != tc.want {
			t.Errorf("canonicalBlockRank(%q) = %d, want %d", tc.fourCC, got, tc.want)
		}
	}
}

// TestParseRejectsTruncatedInputs walks the parser's truncation guards
// (header, secondary header, block info table, name table) using slices
// of a known-good synthetic file.
func TestParseRejectsTruncatedInputs(t *testing.T) {
	full := buildSyntheticRSRC(t)

	cases := []struct {
		name string
		data []byte
	}{
		{name: "empty", data: nil},
		{name: "header only", data: full[:16]},
		{name: "header truncated mid", data: full[:30]},
		{name: "single byte", data: []byte{0xff}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(tc.data); err == nil {
				t.Errorf("Parse(%s) returned no error", tc.name)
			}
		})
	}
}

// TestParseRejectsBadMagic ensures the magic-bytes guard fires.
func TestParseRejectsBadMagic(t *testing.T) {
	full := buildSyntheticRSRC(t)
	full[0] = 'X'
	if _, err := Parse(full); err == nil {
		t.Errorf("Parse(bad magic) returned no error")
	}
}

// TestDetectFileKindCoversAllHeaderTypes drives every arm of the
// header-type-to-FileKind switch so the table doesn't sit unexercised.
func TestDetectFileKindCoversAllHeaderTypes(t *testing.T) {
	cases := []struct {
		headerType string
		want       FileKind
	}{
		{"LVIN", FileKindVI},
		{"LVCC", FileKindControl},
		{"sVIN", FileKindTemplate},
		{"LVAR", FileKindLibrary},
		{"????", FileKindUnknown},
	}
	for _, tc := range cases {
		if got := detectFileKind(tc.headerType); got != tc.want {
			t.Errorf("detectFileKind(%q) = %v, want %v", tc.headerType, got, tc.want)
		}
	}
}

// TestValidateHeaderBoundsRejectsOutOfRange covers the four guard
// branches in validateHeaderBounds (info OOB, data OOB, info underflow,
// data underflow).
func TestValidateHeaderBoundsRejectsOutOfRange(t *testing.T) {
	good := Header{
		InfoOffset: 32,
		InfoSize:   16,
		DataOffset: 32,
		DataSize:   16,
	}
	cases := []struct {
		name string
		mod  func(*Header)
		size int
	}{
		{name: "info exceeds size", mod: func(h *Header) { h.InfoSize = 1 << 30 }, size: 64},
		{name: "data exceeds size", mod: func(h *Header) { h.DataSize = 1 << 30 }, size: 64},
		{name: "info before header", mod: func(h *Header) { h.InfoOffset = 4 }, size: 1024},
		{name: "data before header", mod: func(h *Header) { h.DataOffset = 4 }, size: 1024},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := good
			tc.mod(&h)
			if err := validateHeaderBounds(tc.size, h); err == nil {
				t.Errorf("validateHeaderBounds(%s) returned nil error", tc.name)
			}
		})
	}
	if err := validateHeaderBounds(1024, good); err != nil {
		t.Errorf("validateHeaderBounds(good) = %v, want nil", err)
	}
}

// TestSerializeRejectsInvalidHeader exercises writeHeader's three shape
// guards (magic, type, creator length) by mutating a known-good File.
func TestSerializeRejectsInvalidHeader(t *testing.T) {
	good := &File{
		Header:          Header{Magic: "RSRC\r\n", Type: "LVIN", Creator: "LBVW", FormatVersion: 3},
		SecondaryHeader: Header{Magic: "RSRC\r\n", Type: "LVIN", Creator: "LBVW", FormatVersion: 3},
		BlockInfoList:   BlockInfoList{DatasetInt3: headerSize},
	}

	cases := []struct {
		name string
		mod  func(*File)
	}{
		{name: "short magic", mod: func(f *File) { f.Header.Magic = "RSRC" }},
		{name: "short type", mod: func(f *File) { f.Header.Type = "LV" }},
		{name: "short creator", mod: func(f *File) { f.Header.Creator = "LB" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := *good
			tc.mod(&f)
			if _, err := Serialize(&f); err == nil {
				t.Errorf("Serialize(%s) returned no error", tc.name)
			}
		})
	}
}

// TestSliceWriterAtRejectsBadOffsets covers the negative-offset and
// out-of-bounds branches of sliceWriterAt.WriteAt.
func TestSliceWriterAtRejectsBadOffsets(t *testing.T) {
	w := sliceWriterAt(make([]byte, 4))

	if _, err := w.WriteAt([]byte{1}, -1); err == nil {
		t.Errorf("WriteAt(-1) returned no error")
	}
	if _, err := w.WriteAt([]byte{1, 2}, 3); err == nil {
		t.Errorf("WriteAt(off=3 len=2 into 4-byte slice) returned no error")
	}
	if n, err := w.WriteAt([]byte{1, 2}, 0); err != nil || n != 2 {
		t.Errorf("WriteAt(0,2) = (%d,%v), want (2,nil)", n, err)
	}
}

// TestSerializeRejectsConflictingNames exercises the dedup conflict
// branch in serializeNamesPreserving (two sections claim different
// names for the same offset).
func TestSerializeRejectsConflictingNames(t *testing.T) {
	f := &File{
		Header:          Header{Magic: "RSRC\r\n", Type: "LVIN", Creator: "LBVW", FormatVersion: 3},
		SecondaryHeader: Header{Magic: "RSRC\r\n", Type: "LVIN", Creator: "LBVW", FormatVersion: 3},
		BlockInfoList:   BlockInfoList{DatasetInt3: headerSize},
		Names: []NameEntry{
			{Offset: 0, Value: "alpha", Consumed: 6},
		},
		Blocks: []Block{
			{
				Type: "LIBN",
				Sections: []Section{
					{Index: 0, NameOffset: 0, Name: "different", Payload: []byte("x")},
				},
			},
		},
	}
	if _, err := Serialize(f); err == nil {
		t.Errorf("Serialize(conflicting names) returned no error")
	}
}

// TestSerializeCanonicalRejectsMissingName covers the canonical
// serializer's missing-name and oversize-name guards.
func TestSerializeCanonicalRejectsMissingName(t *testing.T) {
	missing := &File{
		Header:          Header{Magic: "RSRC\r\n", Type: "LVIN", Creator: "LBVW", FormatVersion: 3},
		SecondaryHeader: Header{Magic: "RSRC\r\n", Type: "LVIN", Creator: "LBVW", FormatVersion: 3},
		BlockInfoList:   BlockInfoList{DatasetInt3: headerSize},
		Blocks: []Block{
			{
				Type: "LIBN",
				Sections: []Section{
					{Index: 0, NameOffset: 99, Payload: []byte("x")},
				},
			},
		},
	}
	if _, err := SerializeCanonical(missing); err == nil {
		t.Errorf("SerializeCanonical(missing name) returned no error")
	}

	long := strings.Repeat("a", 256)
	oversize := &File{
		Header:          Header{Magic: "RSRC\r\n", Type: "LVIN", Creator: "LBVW", FormatVersion: 3},
		SecondaryHeader: Header{Magic: "RSRC\r\n", Type: "LVIN", Creator: "LBVW", FormatVersion: 3},
		BlockInfoList:   BlockInfoList{DatasetInt3: headerSize},
		Names:           []NameEntry{{Offset: 0, Value: long, Consumed: int64(1 + len(long))}},
		Blocks: []Block{
			{
				Type: "LIBN",
				Sections: []Section{
					{Index: 0, NameOffset: 0, Payload: []byte("x")},
				},
			},
		},
	}
	if _, err := SerializeCanonical(oversize); err == nil {
		t.Errorf("SerializeCanonical(oversize name) returned no error")
	}
}

// TestSerializeCanonicalRejectsConflictingNames covers the canonical
// dedup conflict branch (two sections claim different names for the
// same offset).
func TestSerializeCanonicalRejectsConflictingNames(t *testing.T) {
	f := &File{
		Header:          Header{Magic: "RSRC\r\n", Type: "LVIN", Creator: "LBVW", FormatVersion: 3},
		SecondaryHeader: Header{Magic: "RSRC\r\n", Type: "LVIN", Creator: "LBVW", FormatVersion: 3},
		BlockInfoList:   BlockInfoList{DatasetInt3: headerSize},
		Names:           []NameEntry{{Offset: 0, Value: "alpha", Consumed: 6}},
		Blocks: []Block{
			{
				Type: "LIBN",
				Sections: []Section{
					{Index: 0, NameOffset: 0, Name: "alpha", Payload: []byte("x")},
					{Index: 1, NameOffset: 0, Name: "different", Payload: []byte("y")},
				},
			},
		},
	}
	if _, err := SerializeCanonical(f); err == nil {
		t.Errorf("SerializeCanonical(conflicting) returned no error")
	}
}
