package rsrcwire

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
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
			path:          filepath.Join("testdata", "config-data.ctl"),
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
			path:          filepath.Join("testdata", "get-vi-description.vi"),
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
