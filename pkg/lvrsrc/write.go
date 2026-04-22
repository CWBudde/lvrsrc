package lvrsrc

import (
	"bytes"
	"io"
	"os"

	"github.com/example/lvrsrc/internal/rsrcwire"
	iv "github.com/example/lvrsrc/internal/validate"
)

type Severity = iv.Severity

const (
	SeverityWarning = iv.SeverityWarning
	SeverityError   = iv.SeverityError
)

type (
	IssueLocation = iv.IssueLocation
	Issue         = iv.Issue
)

func (f *File) WriteTo(w io.Writer) error {
	if f == nil {
		return nil
	}

	data, err := rsrcwire.Serialize(toWireFile(f))
	if err != nil {
		return err
	}

	_, err = io.Copy(w, bytes.NewReader(data))
	return err
}

func (f *File) WriteToFile(path string) error {
	if f == nil {
		return nil
	}

	data, err := rsrcwire.Serialize(toWireFile(f))
	if err != nil {
		return err
	}

	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	}

	return os.WriteFile(path, data, mode)
}

func (f *File) Validate() []Issue {
	if f == nil {
		return nil
	}

	return iv.File(toWireFile(f), iv.Options{})
}

func toWireFile(src *File) *rsrcwire.File {
	if src == nil {
		return nil
	}

	dst := &rsrcwire.File{
		Header: rsrcwire.Header{
			Magic:         src.Header.Magic,
			FormatVersion: src.Header.FormatVersion,
			Type:          src.Header.Type,
			Creator:       src.Header.Creator,
			InfoOffset:    src.Header.InfoOffset,
			InfoSize:      src.Header.InfoSize,
			DataOffset:    src.Header.DataOffset,
			DataSize:      src.Header.DataSize,
		},
		SecondaryHeader: rsrcwire.Header{
			Magic:         src.SecondaryHeader.Magic,
			FormatVersion: src.SecondaryHeader.FormatVersion,
			Type:          src.SecondaryHeader.Type,
			Creator:       src.SecondaryHeader.Creator,
			InfoOffset:    src.SecondaryHeader.InfoOffset,
			InfoSize:      src.SecondaryHeader.InfoSize,
			DataOffset:    src.SecondaryHeader.DataOffset,
			DataSize:      src.SecondaryHeader.DataSize,
		},
		BlockInfoList: rsrcwire.BlockInfoList{
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
		dst.Names = make([]rsrcwire.NameEntry, len(src.Names))
		for i, name := range src.Names {
			dst.Names[i] = rsrcwire.NameEntry{
				Offset:   name.Offset,
				Value:    name.Value,
				Consumed: name.Consumed,
			}
		}
	}

	if len(src.Blocks) > 0 {
		dst.Blocks = make([]rsrcwire.Block, len(src.Blocks))
		for i, block := range src.Blocks {
			dst.Blocks[i] = rsrcwire.Block{
				Type:                 block.Type,
				SectionCountMinusOne: block.SectionCountMinusOne,
				Offset:               block.Offset,
			}
			if len(block.Sections) > 0 {
				dst.Blocks[i].Sections = make([]rsrcwire.Section, len(block.Sections))
				for j, section := range block.Sections {
					dst.Blocks[i].Sections[j] = rsrcwire.Section{
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
