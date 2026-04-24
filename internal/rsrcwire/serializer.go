package rsrcwire

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/CWBudde/lvrsrc/internal/binaryx"
)

type serializeMode uint8

const (
	serializeModePreserving serializeMode = iota
	serializeModeCanonical
)

// Serialize encodes f back to RSRC wire format while preserving opaque data.
func Serialize(f *File) ([]byte, error) {
	return serialize(f, serializeModePreserving)
}

// SerializeCanonical encodes f using deterministic layout rules while preserving
// the parsed block and section order.
func SerializeCanonical(f *File) ([]byte, error) {
	return serialize(f, serializeModeCanonical)
}

func serialize(f *File, mode serializeMode) ([]byte, error) {
	if f == nil {
		return nil, fmt.Errorf("serialize file: nil file")
	}

	dataRegion, sectionDataOffsets, err := serializeDataRegion(f.Blocks)
	if err != nil {
		return nil, err
	}

	nameTable, nameOffsets, err := serializeNames(f, mode)
	if err != nil {
		return nil, err
	}

	blockInfoOffset := uint32(headerSize + listHeaderSize)
	blockHeadersSize := len(f.Blocks) * blockHeaderSize
	sectionsSize := 0
	for _, block := range f.Blocks {
		if len(block.Sections) == 0 {
			return nil, fmt.Errorf("serialize block %q: empty section list", block.Type)
		}
		sectionsSize += len(block.Sections) * sectionStartSize
	}

	infoOffset := uint32(headerSize + len(dataRegion))
	infoSize := uint32(headerSize + listHeaderSize + blockInfoSize + blockHeadersSize + sectionsSize + len(nameTable) + len(f.RawTail))
	dataOffset := uint32(headerSize)
	dataSize := uint32(len(dataRegion))

	full := bytes.NewBuffer(make([]byte, 0, int(infoOffset+infoSize)))
	if err := writeHeader(full, f.Header, infoOffset, infoSize, dataOffset, dataSize); err != nil {
		return nil, err
	}
	if _, err := full.Write(dataRegion); err != nil {
		return nil, fmt.Errorf("write data region: %w", err)
	}
	if err := writeHeader(full, f.Header, infoOffset, infoSize, dataOffset, dataSize); err != nil {
		return nil, err
	}

	blockInfoSizeValue := infoSize - headerSize - listHeaderSize
	for _, v := range []uint32{
		f.BlockInfoList.DatasetInt1,
		f.BlockInfoList.DatasetInt2,
		f.BlockInfoList.DatasetInt3,
		blockInfoOffset,
		blockInfoSizeValue,
	} {
		if err := binary.Write(full, binary.BigEndian, v); err != nil {
			return nil, fmt.Errorf("write block info list: %w", err)
		}
	}

	blockCountMinusOne := uint32(len(f.Blocks) - 1)
	if err := binary.Write(full, binary.BigEndian, blockCountMinusOne); err != nil {
		return nil, fmt.Errorf("write block count: %w", err)
	}

	nextBlockOffset := uint32(blockInfoSize + blockHeadersSize)
	for i, block := range f.Blocks {
		if len(block.Type) != 4 {
			return nil, fmt.Errorf("serialize block %d: type %q must be 4 bytes", i, block.Type)
		}
		if _, err := full.WriteString(block.Type); err != nil {
			return nil, fmt.Errorf("write block %d type: %w", i, err)
		}
		if err := binary.Write(full, binary.BigEndian, uint32(len(block.Sections)-1)); err != nil {
			return nil, fmt.Errorf("write block %d section count: %w", i, err)
		}
		if err := binary.Write(full, binary.BigEndian, nextBlockOffset); err != nil {
			return nil, fmt.Errorf("write block %d offset: %w", i, err)
		}

		nextBlockOffset += uint32(len(block.Sections) * sectionStartSize)
	}

	for bi, block := range f.Blocks {
		for si, section := range block.Sections {
			nameOffset := section.NameOffset
			if nameOffset != noNameOffset {
				var ok bool
				nameOffset, ok = nameOffsets[section.NameOffset]
				if !ok {
					return nil, fmt.Errorf("serialize block %d section %d: missing name offset %d", bi, si, section.NameOffset)
				}
			}

			for _, v := range []uint32{
				uint32(section.Index),
				nameOffset,
				section.Unknown3,
				sectionDataOffsets[bi][si],
				section.Unknown5,
			} {
				if err := binary.Write(full, binary.BigEndian, v); err != nil {
					return nil, fmt.Errorf("write block %d section %d: %w", bi, si, err)
				}
			}
		}
	}

	if _, err := full.Write(nameTable); err != nil {
		return nil, fmt.Errorf("write name table: %w", err)
	}
	if _, err := full.Write(f.RawTail); err != nil {
		return nil, fmt.Errorf("write raw tail: %w", err)
	}

	return full.Bytes(), nil
}

func serializeDataRegion(blocks []Block) ([]byte, [][]uint32, error) {
	buf := bytes.NewBuffer(nil)
	offsets := make([][]uint32, len(blocks))

	for bi, block := range blocks {
		offsets[bi] = make([]uint32, len(block.Sections))
		for si, section := range block.Sections {
			offsets[bi][si] = uint32(buf.Len())

			if err := binary.Write(buf, binary.BigEndian, uint32(len(section.Payload))); err != nil {
				return nil, nil, fmt.Errorf("serialize block %d section %d size: %w", bi, si, err)
			}
			if _, err := buf.Write(section.Payload); err != nil {
				return nil, nil, fmt.Errorf("serialize block %d section %d payload: %w", bi, si, err)
			}
			for buf.Len()%4 != 0 {
				if err := buf.WriteByte(0); err != nil {
					return nil, nil, fmt.Errorf("serialize block %d section %d padding: %w", bi, si, err)
				}
			}
		}
	}

	return buf.Bytes(), offsets, nil
}

func serializeNames(f *File, mode serializeMode) ([]byte, map[uint32]uint32, error) {
	if mode == serializeModeCanonical {
		return serializeNamesCanonical(f)
	}
	return serializeNamesPreserving(f)
}

func serializeNamesPreserving(f *File) ([]byte, map[uint32]uint32, error) {
	type nameValue struct {
		offset uint32
		value  string
	}

	values := make(map[uint32]string, len(f.Names))
	for _, entry := range f.Names {
		values[entry.Offset] = entry.Value
	}

	for bi, block := range f.Blocks {
		for si, section := range block.Sections {
			if section.NameOffset == noNameOffset {
				continue
			}

			value := section.Name
			if value == "" {
				var ok bool
				value, ok = values[section.NameOffset]
				if !ok {
					return nil, nil, fmt.Errorf("serialize block %d section %d: missing name for offset %d", bi, si, section.NameOffset)
				}
			}

			if existing, ok := values[section.NameOffset]; ok && existing != value {
				return nil, nil, fmt.Errorf(
					"serialize block %d section %d: conflicting names for offset %d (%q != %q)",
					bi,
					si,
					section.NameOffset,
					existing,
					value,
				)
			}
			values[section.NameOffset] = value
		}
	}

	if len(values) == 0 {
		return nil, map[uint32]uint32{}, nil
	}

	ordered := make([]nameValue, 0, len(values))
	for offset, value := range values {
		ordered = append(ordered, nameValue{offset: offset, value: value})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].offset < ordered[j].offset })

	offsetMap := make(map[uint32]uint32, len(ordered))
	preserveOffsets := true
	var lastEnd uint32
	for _, entry := range ordered {
		if entry.offset < lastEnd {
			preserveOffsets = false
			break
		}
		lastEnd = entry.offset + uint32(1+len(entry.value))
	}

	if preserveOffsets {
		table := make([]byte, lastEnd)
		writer := binaryx.NewWriter(sliceWriterAt(table), binary.BigEndian)
		for _, entry := range ordered {
			if _, err := writer.WritePascalString(int64(entry.offset), entry.value); err != nil {
				return nil, nil, fmt.Errorf("serialize name at offset %d: %w", entry.offset, err)
			}
			offsetMap[entry.offset] = entry.offset
		}
		return table, offsetMap, nil
	}

	var offset uint32
	table := bytes.NewBuffer(nil)
	for _, entry := range ordered {
		if err := table.WriteByte(byte(len(entry.value))); err != nil {
			return nil, nil, fmt.Errorf("serialize compacted name at offset %d: %w", offset, err)
		}
		if _, err := table.WriteString(entry.value); err != nil {
			return nil, nil, fmt.Errorf("serialize compacted name at offset %d: %w", offset, err)
		}
		offsetMap[entry.offset] = offset
		offset += uint32(1 + len(entry.value))
	}

	return table.Bytes(), offsetMap, nil
}

func serializeNamesCanonical(f *File) ([]byte, map[uint32]uint32, error) {
	values := make(map[uint32]string, len(f.Names))
	for _, entry := range f.Names {
		values[entry.Offset] = entry.Value
	}

	orderedOffsets := make([]uint32, 0, len(f.Names))
	seenOffsets := make(map[uint32]struct{}, len(f.Names))
	canonicalValues := make(map[uint32]string, len(f.Names))

	for bi, block := range f.Blocks {
		for si, section := range block.Sections {
			if section.NameOffset == noNameOffset {
				continue
			}

			value := section.Name
			if value == "" {
				var ok bool
				value, ok = values[section.NameOffset]
				if !ok {
					return nil, nil, fmt.Errorf("serialize block %d section %d: missing name for offset %d", bi, si, section.NameOffset)
				}
			}

			if existing, ok := canonicalValues[section.NameOffset]; ok && existing != value {
				return nil, nil, fmt.Errorf(
					"serialize block %d section %d: conflicting names for offset %d (%q != %q)",
					bi,
					si,
					section.NameOffset,
					existing,
					value,
				)
			}
			canonicalValues[section.NameOffset] = value

			if _, ok := seenOffsets[section.NameOffset]; ok {
				continue
			}
			seenOffsets[section.NameOffset] = struct{}{}
			orderedOffsets = append(orderedOffsets, section.NameOffset)
		}
	}

	if len(orderedOffsets) == 0 {
		return nil, map[uint32]uint32{}, nil
	}

	table := bytes.NewBuffer(make([]byte, 0, len(orderedOffsets)*8))
	offsetMap := make(map[uint32]uint32, len(orderedOffsets))
	var nextOffset uint32

	for _, originalOffset := range orderedOffsets {
		value := canonicalValues[originalOffset]
		if len(value) > 255 {
			return nil, nil, fmt.Errorf("serialize canonical name at offset %d: string too long (%d)", originalOffset, len(value))
		}
		if err := table.WriteByte(byte(len(value))); err != nil {
			return nil, nil, fmt.Errorf("serialize canonical name at offset %d: %w", originalOffset, err)
		}
		if _, err := table.WriteString(value); err != nil {
			return nil, nil, fmt.Errorf("serialize canonical name at offset %d: %w", originalOffset, err)
		}
		offsetMap[originalOffset] = nextOffset
		nextOffset += uint32(1 + len(value))
	}

	return table.Bytes(), offsetMap, nil
}

func writeHeader(buf *bytes.Buffer, header Header, infoOffset, infoSize, dataOffset, dataSize uint32) error {
	if len(header.Magic) != 6 {
		return fmt.Errorf("serialize header: magic %q must be 6 bytes", header.Magic)
	}
	if len(header.Type) != 4 {
		return fmt.Errorf("serialize header: type %q must be 4 bytes", header.Type)
	}
	if len(header.Creator) != 4 {
		return fmt.Errorf("serialize header: creator %q must be 4 bytes", header.Creator)
	}

	if _, err := buf.WriteString(header.Magic); err != nil {
		return fmt.Errorf("write header magic: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, header.FormatVersion); err != nil {
		return fmt.Errorf("write header version: %w", err)
	}
	if _, err := buf.WriteString(header.Type); err != nil {
		return fmt.Errorf("write header type: %w", err)
	}
	if _, err := buf.WriteString(header.Creator); err != nil {
		return fmt.Errorf("write header creator: %w", err)
	}
	for _, v := range []uint32{infoOffset, infoSize, dataOffset, dataSize} {
		if err := binary.Write(buf, binary.BigEndian, v); err != nil {
			return fmt.Errorf("write header offsets: %w", err)
		}
	}

	return nil
}

type sliceWriterAt []byte

func (w sliceWriterAt) WriteAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, fmt.Errorf("negative offset %d", off)
	}
	if off > int64(len(w)) || int64(len(p)) > int64(len(w))-off {
		return 0, fmt.Errorf("short write at offset %d size %d", off, len(p))
	}

	return copy(w[off:], p), nil
}
