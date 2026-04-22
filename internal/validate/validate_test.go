package validate

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/example/lvrsrc/internal/rsrcwire"
)

func TestFileValidSyntheticReturnsNoIssues(t *testing.T) {
	data, _ := buildSyntheticRSRC(t)

	f, err := rsrcwire.ParseWithOptions(data, rsrcwire.ParseOptions{})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}

	issues := File(f, Options{FileSize: len(data)})
	if len(issues) != 0 {
		t.Fatalf("File() issues = %+v, want none", issues)
	}
}

func TestFileReportsHeaderMismatch(t *testing.T) {
	data, layout := buildSyntheticRSRC(t)
	data[layout.secondaryOffset+8] ^= 0x01

	f, err := rsrcwire.ParseWithOptions(data, rsrcwire.ParseOptions{})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}

	issues := File(f, Options{FileSize: len(data)})
	assertHasIssueCode(t, issues, "header.mismatch")
}

func TestFileReportsBlockCountMismatch(t *testing.T) {
	data, _ := buildSyntheticRSRC(t)

	f, err := rsrcwire.ParseWithOptions(data, rsrcwire.ParseOptions{})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}
	f.Blocks[0].SectionCountMinusOne = 0

	issues := File(f, Options{FileSize: len(data)})
	assertHasIssueCode(t, issues, "block.count_mismatch")
}

func TestFileReportsInvalidNameOffset(t *testing.T) {
	data, layout := buildSyntheticRSRC(t)
	binary.BigEndian.PutUint32(data[layout.firstSectionOffset+4:], 0x7fffffff)

	f, err := rsrcwire.ParseWithOptions(data, rsrcwire.ParseOptions{})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}

	issues := File(f, Options{FileSize: len(data)})
	assertHasIssueCode(t, issues, "section.name_offset.invalid")
}

func TestFileReportsBlockInfoOffsetBounds(t *testing.T) {
	data, _ := buildSyntheticRSRC(t)

	f, err := rsrcwire.ParseWithOptions(data, rsrcwire.ParseOptions{})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}
	f.BlockInfoList.BlockInfoOffset = f.Header.InfoSize + 1

	issues := File(f, Options{FileSize: len(data)})
	assertHasIssueCode(t, issues, "block_info.offset_bounds")
}

func TestFileReportsOverlappingPayloads(t *testing.T) {
	data, layout := buildSyntheticRSRC(t)
	binary.BigEndian.PutUint32(data[layout.secondSectionOffset+12:], 0)

	f, err := rsrcwire.ParseWithOptions(data, rsrcwire.ParseOptions{})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}

	issues := File(f, Options{FileSize: len(data)})
	assertHasIssueCode(t, issues, "section.payload.overlap")
}

func TestFileReportsZeroSizePayload(t *testing.T) {
	data, layout := buildSyntheticRSRC(t)
	binary.BigEndian.PutUint32(data[layout.dataOffset:], 0)

	f, err := rsrcwire.ParseWithOptions(data, rsrcwire.ParseOptions{})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}

	issues := File(f, Options{FileSize: len(data)})
	assertHasIssueCode(t, issues, "section.size.zero")
}

func TestFileReportsNonPrintableBlockType(t *testing.T) {
	data, layout := buildSyntheticRSRC(t)
	copy(data[layout.blockHeaderOffset:], []byte{0x01, 'E', 'S', 'T'})

	f, err := rsrcwire.ParseWithOptions(data, rsrcwire.ParseOptions{})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}

	issues := File(f, Options{FileSize: len(data)})
	assertHasIssueCode(t, issues, "block.type.non_printable")
}

func assertHasIssueCode(t *testing.T, issues []Issue, want string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("issue code %q not found in %+v", want, issues)
}

type syntheticLayout struct {
	secondaryOffset     int
	blockHeaderOffset   int
	firstSectionOffset  int
	secondSectionOffset int
	dataOffset          int
}

func buildSyntheticRSRC(t *testing.T) ([]byte, syntheticLayout) {
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
	}

	infoOffset := uint32(headerSize + dataRegion.Len())
	blockTableOffset := uint32(headerSize + listHeaderSize)
	blockOffset := uint32(blockInfoSize + blockHeaderSize)
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
	if err := binary.Write(full, binary.BigEndian, uint32(0)); err != nil {
		t.Fatalf("write datasetInt1: %v", err)
	}
	if err := binary.Write(full, binary.BigEndian, uint32(0)); err != nil {
		t.Fatalf("write datasetInt2: %v", err)
	}
	if err := binary.Write(full, binary.BigEndian, uint32(0)); err != nil {
		t.Fatalf("write datasetInt3: %v", err)
	}
	if err := binary.Write(full, binary.BigEndian, blockTableOffset); err != nil {
		t.Fatalf("write blockInfoOffset: %v", err)
	}
	if err := binary.Write(full, binary.BigEndian, uint32(blockInfoSize+blockHeaderSize+len(sections)*sectionStartSize)); err != nil {
		t.Fatalf("write blockInfoSize: %v", err)
	}
	if err := binary.Write(full, binary.BigEndian, uint32(0)); err != nil {
		t.Fatalf("write block count: %v", err)
	}
	full.WriteString("TEST")
	if err := binary.Write(full, binary.BigEndian, uint32(len(sections)-1)); err != nil {
		t.Fatalf("write block section count: %v", err)
	}
	if err := binary.Write(full, binary.BigEndian, blockOffset); err != nil {
		t.Fatalf("write block offset: %v", err)
	}
	for i, section := range sections {
		if err := binary.Write(full, binary.BigEndian, uint32(section.index)); err != nil {
			t.Fatalf("write section index: %v", err)
		}
		if err := binary.Write(full, binary.BigEndian, section.nameOffset); err != nil {
			t.Fatalf("write section name offset: %v", err)
		}
		if err := binary.Write(full, binary.BigEndian, uint32(0)); err != nil {
			t.Fatalf("write section unknown3: %v", err)
		}
		if err := binary.Write(full, binary.BigEndian, sectionDataOffsets[i]); err != nil {
			t.Fatalf("write section data offset: %v", err)
		}
		if err := binary.Write(full, binary.BigEndian, uint32(0)); err != nil {
			t.Fatalf("write section unknown5: %v", err)
		}
	}
	if _, err := full.Write(names); err != nil {
		t.Fatalf("write names: %v", err)
	}
	if _, err := full.Write(rawTail); err != nil {
		t.Fatalf("write raw tail: %v", err)
	}

	return full.Bytes(), syntheticLayout{
		secondaryOffset:     int(infoOffset),
		blockHeaderOffset:   int(infoOffset) + int(blockTableOffset) + blockInfoSize,
		firstSectionOffset:  int(infoOffset) + int(blockTableOffset) + blockInfoSize + blockHeaderSize,
		secondSectionOffset: int(infoOffset) + int(blockTableOffset) + blockInfoSize + blockHeaderSize + sectionStartSize,
		dataOffset:          headerSize,
	}
}
