package rsrcwire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"

	"github.com/example/lvrsrc/internal/binaryx"
)

const (
	headerSize       = 32
	listHeaderSize   = 20
	blockInfoSize    = 4
	blockHeaderSize  = 12
	sectionStartSize = 20

	noNameOffset = ^uint32(0)
)

type FileKind string

const (
	FileKindUnknown  FileKind = "unknown"
	FileKindVI       FileKind = "vi"
	FileKindControl  FileKind = "ctl"
	FileKindTemplate FileKind = "vit"
	FileKindLibrary  FileKind = "llb"
)

type CompressionKind string

const (
	CompressionKindUnknown CompressionKind = "unknown"
)

type Severity string

const (
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type IssueLocation struct {
	Area         string
	Offset       uint32
	BlockType    string
	SectionIndex int32
	NameOffset   uint32
}

type ParseIssue struct {
	Severity Severity
	Code     string
	Message  string
	Location IssueLocation
}

type ParseOptions struct {
	Strict bool
}

type File struct {
	Header          Header
	SecondaryHeader Header
	BlockInfoList   BlockInfoList
	Blocks          []Block
	Names           []NameEntry
	RawTail         []byte
	Kind            FileKind
	Compression     CompressionKind
	ParseIssues     []ParseIssue
}

type Header struct {
	Magic         string
	FormatVersion uint16
	Type          string
	Creator       string
	InfoOffset    uint32
	InfoSize      uint32
	DataOffset    uint32
	DataSize      uint32
}

type BlockInfoList struct {
	DatasetInt1     uint32
	DatasetInt2     uint32
	DatasetInt3     uint32
	BlockInfoOffset uint32
	BlockInfoSize   uint32
}

type Block struct {
	Type                 string
	SectionCountMinusOne uint32
	Offset               uint32
	Sections             []Section
}

type Section struct {
	Index      int32
	NameOffset uint32
	Unknown3   uint32
	DataOffset uint32
	Unknown5   uint32
	Name       string
	Payload    []byte
}

type NameEntry struct {
	Offset   uint32
	Value    string
	Consumed int64
}

func Parse(data []byte) (*File, error) {
	return ParseWithOptions(data, ParseOptions{Strict: true})
}

func ParseWithOptions(data []byte, opts ParseOptions) (*File, error) {
	r := binaryx.NewReader(data, binary.BigEndian)
	state := parseState{strict: opts.Strict}

	header, err := parseHeader(r, 0)
	if err != nil {
		return nil, err
	}
	if err := validateHeaderBounds(len(data), header); err != nil {
		return nil, err
	}

	secondary, err := parseHeader(r, int64(header.InfoOffset))
	if err != nil {
		return nil, fmt.Errorf("parse secondary header: %w", err)
	}
	if header != secondary {
		if err := state.addIssue(ParseIssue{
			Severity: SeverityError,
			Code:     "header.mismatch",
			Message:  "secondary header mismatch",
			Location: IssueLocation{
				Area:   "header",
				Offset: header.InfoOffset,
			},
		}); err != nil {
			return nil, err
		}
	}

	blockInfoList, blockInfoPos, err := parseBlockInfoList(r, header)
	if err != nil {
		return nil, err
	}

	blockCountMinusOne, err := r.U32(int64(blockInfoPos))
	if err != nil {
		return nil, fmt.Errorf("parse block count: %w", err)
	}
	blockCount := int(blockCountMinusOne) + 1

	blockHeadersPos := int64(blockInfoPos) + blockInfoSize
	blocks := make([]Block, 0, blockCount)
	namesStart := blockHeadersPos + int64(blockCount)*blockHeaderSize
	nameOffsets := make(map[uint32]struct{})

	for i := 0; i < blockCount; i++ {
		block, blockNamesStart, err := parseBlock(r, header, blockInfoPos, blockHeadersPos+int64(i)*blockHeaderSize)
		if err != nil {
			return nil, fmt.Errorf("parse block %d: %w", i, err)
		}
		if blockNamesStart > namesStart {
			namesStart = blockNamesStart
		}
		for _, section := range block.Sections {
			if section.NameOffset != noNameOffset {
				nameOffsets[section.NameOffset] = struct{}{}
			}
		}
		blocks = append(blocks, block)
	}

	infoEnd := int64(header.InfoOffset) + int64(header.InfoSize)
	if namesStart > infoEnd {
		return nil, fmt.Errorf("name table starts beyond info section")
	}

	names, namesByOffset, rawTailStart, err := parseNames(r, namesStart, infoEnd, nameOffsets, &state)
	if err != nil {
		return nil, err
	}
	for bi := range blocks {
		for si := range blocks[bi].Sections {
			if name, ok := namesByOffset[blocks[bi].Sections[si].NameOffset]; ok {
				blocks[bi].Sections[si].Name = name
			}
		}
	}

	var rawTail []byte
	if rawTailStart < infoEnd {
		rawTail, err = r.Bytes(rawTailStart, int(infoEnd-rawTailStart))
		if err != nil {
			return nil, fmt.Errorf("parse raw tail: %w", err)
		}
	}

	return &File{
		Header:          header,
		SecondaryHeader: secondary,
		BlockInfoList:   blockInfoList,
		Blocks:          blocks,
		Names:           names,
		RawTail:         rawTail,
		Kind:            detectFileKind(header.Type),
		Compression:     detectCompressionKind(blocks),
		ParseIssues:     append([]ParseIssue(nil), state.issues...),
	}, nil
}

func ParseHeader(data []byte) (Header, error) {
	return parseHeader(binaryx.NewReader(data, binary.BigEndian), 0)
}

func parseHeader(r *binaryx.Reader, off int64) (Header, error) {
	magicBytes, err := r.Bytes(off, 6)
	if err != nil {
		return Header{}, fmt.Errorf("read header magic: %w", err)
	}

	formatVersion, err := r.U16(off + 6)
	if err != nil {
		return Header{}, fmt.Errorf("read header format version: %w", err)
	}

	typeBytes, err := r.Bytes(off+8, 4)
	if err != nil {
		return Header{}, fmt.Errorf("read header type: %w", err)
	}

	creatorBytes, err := r.Bytes(off+12, 4)
	if err != nil {
		return Header{}, fmt.Errorf("read header creator: %w", err)
	}

	infoOffset, err := r.U32(off + 16)
	if err != nil {
		return Header{}, fmt.Errorf("read header info offset: %w", err)
	}

	infoSize, err := r.U32(off + 20)
	if err != nil {
		return Header{}, fmt.Errorf("read header info size: %w", err)
	}

	dataOffset, err := r.U32(off + 24)
	if err != nil {
		return Header{}, fmt.Errorf("read header data offset: %w", err)
	}

	dataSize, err := r.U32(off + 28)
	if err != nil {
		return Header{}, fmt.Errorf("read header data size: %w", err)
	}

	header := Header{
		Magic:         string(magicBytes),
		FormatVersion: formatVersion,
		Type:          string(typeBytes),
		Creator:       string(creatorBytes),
		InfoOffset:    infoOffset,
		InfoSize:      infoSize,
		DataOffset:    dataOffset,
		DataSize:      dataSize,
	}

	if header.Magic != "RSRC\r\n" && header.Magic != "RSRC\x00\x00" {
		return Header{}, fmt.Errorf("unexpected header magic %q", header.Magic)
	}

	return header, nil
}

func parseBlockInfoList(r *binaryx.Reader, header Header) (BlockInfoList, uint32, error) {
	base := int64(header.InfoOffset) + headerSize

	datasetInt1, err := r.U32(base)
	if err != nil {
		return BlockInfoList{}, 0, fmt.Errorf("read block info list dataset_int1: %w", err)
	}
	datasetInt2, err := r.U32(base + 4)
	if err != nil {
		return BlockInfoList{}, 0, fmt.Errorf("read block info list dataset_int2: %w", err)
	}
	datasetInt3, err := r.U32(base + 8)
	if err != nil {
		return BlockInfoList{}, 0, fmt.Errorf("read block info list dataset_int3: %w", err)
	}
	blockInfoOffset, err := r.U32(base + 12)
	if err != nil {
		return BlockInfoList{}, 0, fmt.Errorf("read block info list offset: %w", err)
	}
	blockInfoSize, err := r.U32(base + 16)
	if err != nil {
		return BlockInfoList{}, 0, fmt.Errorf("read block info list size: %w", err)
	}

	pos := header.InfoOffset + blockInfoOffset
	if int64(pos) > int64(header.InfoOffset)+int64(header.InfoSize) {
		return BlockInfoList{}, 0, fmt.Errorf("block info offset beyond info section")
	}

	return BlockInfoList{
		DatasetInt1:     datasetInt1,
		DatasetInt2:     datasetInt2,
		DatasetInt3:     datasetInt3,
		BlockInfoOffset: blockInfoOffset,
		BlockInfoSize:   blockInfoSize,
	}, pos, nil
}

func parseBlock(r *binaryx.Reader, header Header, blockInfoPos uint32, headerPos int64) (Block, int64, error) {
	typeBytes, err := r.Bytes(headerPos, 4)
	if err != nil {
		return Block{}, 0, fmt.Errorf("read block type: %w", err)
	}

	countMinusOne, err := r.U32(headerPos + 4)
	if err != nil {
		return Block{}, 0, fmt.Errorf("read block section count: %w", err)
	}

	offset, err := r.U32(headerPos + 8)
	if err != nil {
		return Block{}, 0, fmt.Errorf("read block offset: %w", err)
	}

	sectionCount := int(countMinusOne) + 1
	sections := make([]Section, 0, sectionCount)
	startPos := int64(blockInfoPos) + int64(offset)
	namesStart := startPos + int64(sectionCount)*sectionStartSize

	for i := 0; i < sectionCount; i++ {
		section, err := parseSection(r, header, startPos+int64(i)*sectionStartSize)
		if err != nil {
			return Block{}, 0, fmt.Errorf("read section %d: %w", i, err)
		}
		sections = append(sections, section)
	}

	return Block{
		Type:                 string(typeBytes),
		SectionCountMinusOne: countMinusOne,
		Offset:               offset,
		Sections:             sections,
	}, namesStart, nil
}

func parseSection(r *binaryx.Reader, header Header, off int64) (Section, error) {
	index, err := r.U32(off)
	if err != nil {
		return Section{}, fmt.Errorf("read section index: %w", err)
	}
	nameOffset, err := r.U32(off + 4)
	if err != nil {
		return Section{}, fmt.Errorf("read section name offset: %w", err)
	}
	unknown3, err := r.U32(off + 8)
	if err != nil {
		return Section{}, fmt.Errorf("read section unknown3: %w", err)
	}
	dataOffset, err := r.U32(off + 12)
	if err != nil {
		return Section{}, fmt.Errorf("read section data offset: %w", err)
	}
	unknown5, err := r.U32(off + 16)
	if err != nil {
		return Section{}, fmt.Errorf("read section unknown5: %w", err)
	}

	payloadSizePos := int64(header.DataOffset) + int64(dataOffset)
	payloadSize, err := r.U32(payloadSizePos)
	if err != nil {
		return Section{}, fmt.Errorf("read section payload size: %w", err)
	}
	payload, err := r.Bytes(payloadSizePos+4, int(payloadSize))
	if err != nil {
		return Section{}, fmt.Errorf("read section payload: %w", err)
	}

	return Section{
		Index:      int32(index),
		NameOffset: nameOffset,
		Unknown3:   unknown3,
		DataOffset: dataOffset,
		Unknown5:   unknown5,
		Payload:    payload,
	}, nil
}

func parseNames(r *binaryx.Reader, namesStart, infoEnd int64, offsets map[uint32]struct{}, state *parseState) ([]NameEntry, map[uint32]string, int64, error) {
	if len(offsets) == 0 {
		return nil, map[uint32]string{}, namesStart, nil
	}

	orderedOffsets := make([]uint32, 0, len(offsets))
	for offset := range offsets {
		orderedOffsets = append(orderedOffsets, offset)
	}
	sort.Slice(orderedOffsets, func(i, j int) bool { return orderedOffsets[i] < orderedOffsets[j] })

	names := make([]NameEntry, 0, len(orderedOffsets))
	namesByOffset := make(map[uint32]string, len(orderedOffsets))
	rawTailStart := namesStart

	for _, offset := range orderedOffsets {
		namePos := namesStart + int64(offset)
		if namePos >= infoEnd {
			if err := state.addIssue(ParseIssue{
				Severity: SeverityError,
				Code:     "section.name_offset.invalid",
				Message:  fmt.Sprintf("name offset %d beyond info section", offset),
				Location: IssueLocation{
					Area:       "name_table",
					Offset:     uint32(namesStart),
					NameOffset: offset,
				},
			}); err != nil {
				return nil, nil, 0, err
			}
			continue
		}

		value, consumed, err := r.PascalString(namePos)
		if err != nil {
			if issueErr := state.addIssue(ParseIssue{
				Severity: SeverityError,
				Code:     "section.name_offset.invalid",
				Message:  fmt.Sprintf("parse name at offset %d: %v", offset, err),
				Location: IssueLocation{
					Area:       "name_table",
					Offset:     uint32(namePos),
					NameOffset: offset,
				},
			}); issueErr != nil {
				return nil, nil, 0, issueErr
			}
			continue
		}

		endPos := namePos + consumed
		if endPos > infoEnd {
			if err := state.addIssue(ParseIssue{
				Severity: SeverityError,
				Code:     "section.name_offset.invalid",
				Message:  fmt.Sprintf("name at offset %d exceeds info section", offset),
				Location: IssueLocation{
					Area:       "name_table",
					Offset:     uint32(namePos),
					NameOffset: offset,
				},
			}); err != nil {
				return nil, nil, 0, err
			}
			continue
		}

		names = append(names, NameEntry{
			Offset:   offset,
			Value:    value,
			Consumed: consumed,
		})
		namesByOffset[offset] = value
		if endPos > rawTailStart {
			rawTailStart = endPos
		}
	}

	return names, namesByOffset, rawTailStart, nil
}

func validateHeaderBounds(size int, header Header) error {
	if int64(header.InfoOffset)+int64(header.InfoSize) > int64(size) {
		return fmt.Errorf("info section exceeds file bounds")
	}
	if int64(header.DataOffset)+int64(header.DataSize) > int64(size) {
		return fmt.Errorf("data section exceeds file bounds")
	}
	if header.InfoOffset < headerSize {
		return fmt.Errorf("info offset %d before minimum header size", header.InfoOffset)
	}
	if header.DataOffset < headerSize {
		return fmt.Errorf("data offset %d before minimum header size", header.DataOffset)
	}
	return nil
}

func detectFileKind(headerType string) FileKind {
	switch headerType {
	case "LVIN":
		return FileKindVI
	case "LVCC":
		return FileKindControl
	case "sVIN":
		return FileKindTemplate
	case "LVAR":
		return FileKindLibrary
	default:
		return FileKindUnknown
	}
}

func detectCompressionKind(_ []Block) CompressionKind {
	return CompressionKindUnknown
}

type parseState struct {
	strict bool
	issues []ParseIssue
}

func (s *parseState) addIssue(issue ParseIssue) error {
	if s.strict {
		return errors.New(issue.Message)
	}

	s.issues = append(s.issues, issue)
	return nil
}
