package lvrsrc

import (
	"os"

	"github.com/CWBudde/lvrsrc/internal/rsrcwire"
)

type FileKind = rsrcwire.FileKind

const (
	FileKindUnknown  = rsrcwire.FileKindUnknown
	FileKindVI       = rsrcwire.FileKindVI
	FileKindControl  = rsrcwire.FileKindControl
	FileKindTemplate = rsrcwire.FileKindTemplate
	FileKindLibrary  = rsrcwire.FileKindLibrary
)

type CompressionKind = rsrcwire.CompressionKind

const (
	CompressionKindUnknown = rsrcwire.CompressionKindUnknown
)

// OpenOptions configures lenient vs. strict parsing behavior.
type OpenOptions struct {
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

type ResourceRef struct {
	Type string
	ID   int32
	Name string
	Size int
}

func Open(path string, opts OpenOptions) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data, opts)
}

func Parse(data []byte, opts OpenOptions) (*File, error) {
	wireFile, err := rsrcwire.ParseWithOptions(data, rsrcwire.ParseOptions{Strict: opts.Strict})
	if err != nil {
		return nil, err
	}
	return fromWireFile(wireFile), nil
}

func (f *File) Resources() []ResourceRef {
	if f == nil {
		return nil
	}

	total := 0
	for _, block := range f.Blocks {
		total += len(block.Sections)
	}

	refs := make([]ResourceRef, 0, total)
	for _, block := range f.Blocks {
		for _, section := range block.Sections {
			refs = append(refs, ResourceRef{
				Type: block.Type,
				ID:   section.Index,
				Name: section.Name,
				Size: len(section.Payload),
			})
		}
	}
	return refs
}

func (f *File) Clone() *File {
	if f == nil {
		return nil
	}

	clone := &File{
		Header:          f.Header,
		SecondaryHeader: f.SecondaryHeader,
		BlockInfoList:   f.BlockInfoList,
		Kind:            f.Kind,
		Compression:     f.Compression,
		RawTail:         append([]byte(nil), f.RawTail...),
	}

	if len(f.Names) > 0 {
		clone.Names = append([]NameEntry(nil), f.Names...)
	}

	if len(f.Blocks) > 0 {
		clone.Blocks = make([]Block, len(f.Blocks))
		for i, block := range f.Blocks {
			clone.Blocks[i] = Block{
				Type:                 block.Type,
				SectionCountMinusOne: block.SectionCountMinusOne,
				Offset:               block.Offset,
			}
			if len(block.Sections) > 0 {
				clone.Blocks[i].Sections = make([]Section, len(block.Sections))
				for j, section := range block.Sections {
					clone.Blocks[i].Sections[j] = Section{
						Index:      section.Index,
						NameOffset: section.NameOffset,
						Unknown3:   section.Unknown3,
						DataOffset: section.DataOffset,
						Unknown5:   section.Unknown5,
						Name:       section.Name,
						Payload:    append([]byte(nil), section.Payload...),
					}
				}
			}
		}
	}

	return clone
}

func fromWireFile(src *rsrcwire.File) *File {
	dst := &File{
		Header: Header{
			Magic:         src.Header.Magic,
			FormatVersion: src.Header.FormatVersion,
			Type:          src.Header.Type,
			Creator:       src.Header.Creator,
			InfoOffset:    src.Header.InfoOffset,
			InfoSize:      src.Header.InfoSize,
			DataOffset:    src.Header.DataOffset,
			DataSize:      src.Header.DataSize,
		},
		SecondaryHeader: Header{
			Magic:         src.SecondaryHeader.Magic,
			FormatVersion: src.SecondaryHeader.FormatVersion,
			Type:          src.SecondaryHeader.Type,
			Creator:       src.SecondaryHeader.Creator,
			InfoOffset:    src.SecondaryHeader.InfoOffset,
			InfoSize:      src.SecondaryHeader.InfoSize,
			DataOffset:    src.SecondaryHeader.DataOffset,
			DataSize:      src.SecondaryHeader.DataSize,
		},
		BlockInfoList: BlockInfoList{
			DatasetInt1:     src.BlockInfoList.DatasetInt1,
			DatasetInt2:     src.BlockInfoList.DatasetInt2,
			DatasetInt3:     src.BlockInfoList.DatasetInt3,
			BlockInfoOffset: src.BlockInfoList.BlockInfoOffset,
			BlockInfoSize:   src.BlockInfoList.BlockInfoSize,
		},
		Kind:        src.Kind,
		Compression: src.Compression,
		RawTail:     append([]byte(nil), src.RawTail...),
	}

	if len(src.Names) > 0 {
		dst.Names = make([]NameEntry, len(src.Names))
		for i, name := range src.Names {
			dst.Names[i] = NameEntry{
				Offset:   name.Offset,
				Value:    name.Value,
				Consumed: name.Consumed,
			}
		}
	}

	if len(src.Blocks) > 0 {
		dst.Blocks = make([]Block, len(src.Blocks))
		for i, block := range src.Blocks {
			dst.Blocks[i] = Block{
				Type:                 block.Type,
				SectionCountMinusOne: block.SectionCountMinusOne,
				Offset:               block.Offset,
			}
			if len(block.Sections) > 0 {
				dst.Blocks[i].Sections = make([]Section, len(block.Sections))
				for j, section := range block.Sections {
					dst.Blocks[i].Sections[j] = Section{
						Index:      section.Index,
						NameOffset: section.NameOffset,
						Unknown3:   section.Unknown3,
						DataOffset: section.DataOffset,
						Unknown5:   section.Unknown5,
						Name:       section.Name,
						Payload:    append([]byte(nil), section.Payload...),
					}
				}
			}
		}
	}

	return dst
}
