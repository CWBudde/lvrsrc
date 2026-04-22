package lvrsrc

import "encoding/json"

type jsonFile struct {
	Header          jsonHeader        `json:"header"`
	SecondaryHeader jsonHeader        `json:"secondaryHeader"`
	BlockInfoList   jsonBlockInfoList `json:"blockInfoList"`
	Blocks          []jsonBlock       `json:"blocks"`
	Names           []jsonNameEntry   `json:"names"`
	RawTail         []byte            `json:"rawTail"`
	Kind            string            `json:"kind"`
	Compression     string            `json:"compression"`
}

type jsonHeader struct {
	Magic         string `json:"magic"`
	FormatVersion uint16 `json:"formatVersion"`
	Type          string `json:"type"`
	Creator       string `json:"creator"`
	InfoOffset    uint32 `json:"infoOffset"`
	InfoSize      uint32 `json:"infoSize"`
	DataOffset    uint32 `json:"dataOffset"`
	DataSize      uint32 `json:"dataSize"`
}

type jsonBlockInfoList struct {
	DatasetInt1     uint32 `json:"datasetInt1"`
	DatasetInt2     uint32 `json:"datasetInt2"`
	DatasetInt3     uint32 `json:"datasetInt3"`
	BlockInfoOffset uint32 `json:"blockInfoOffset"`
	BlockInfoSize   uint32 `json:"blockInfoSize"`
}

type jsonBlock struct {
	Type         string        `json:"type"`
	SectionCount int           `json:"sectionCount"`
	Offset       uint32        `json:"offset"`
	Sections     []jsonSection `json:"sections"`
}

type jsonSection struct {
	Index      int32  `json:"index"`
	NameOffset uint32 `json:"nameOffset"`
	Unknown3   uint32 `json:"unknown3"`
	DataOffset uint32 `json:"dataOffset"`
	Unknown5   uint32 `json:"unknown5"`
	Name       string `json:"name"`
	Payload    []byte `json:"payload"`
}

type jsonNameEntry struct {
	Offset   uint32 `json:"offset"`
	Value    string `json:"value"`
	Consumed int64  `json:"consumed"`
}

func (f *File) MarshalJSON() ([]byte, error) {
	if f == nil {
		return []byte("null"), nil
	}

	out := jsonFile{
		Header: jsonHeader{
			Magic:         f.Header.Magic,
			FormatVersion: f.Header.FormatVersion,
			Type:          f.Header.Type,
			Creator:       f.Header.Creator,
			InfoOffset:    f.Header.InfoOffset,
			InfoSize:      f.Header.InfoSize,
			DataOffset:    f.Header.DataOffset,
			DataSize:      f.Header.DataSize,
		},
		SecondaryHeader: jsonHeader{
			Magic:         f.SecondaryHeader.Magic,
			FormatVersion: f.SecondaryHeader.FormatVersion,
			Type:          f.SecondaryHeader.Type,
			Creator:       f.SecondaryHeader.Creator,
			InfoOffset:    f.SecondaryHeader.InfoOffset,
			InfoSize:      f.SecondaryHeader.InfoSize,
			DataOffset:    f.SecondaryHeader.DataOffset,
			DataSize:      f.SecondaryHeader.DataSize,
		},
		BlockInfoList: jsonBlockInfoList{
			DatasetInt1:     f.BlockInfoList.DatasetInt1,
			DatasetInt2:     f.BlockInfoList.DatasetInt2,
			DatasetInt3:     f.BlockInfoList.DatasetInt3,
			BlockInfoOffset: f.BlockInfoList.BlockInfoOffset,
			BlockInfoSize:   f.BlockInfoList.BlockInfoSize,
		},
		RawTail:     f.RawTail,
		Kind:        string(f.Kind),
		Compression: string(f.Compression),
	}

	if len(f.Blocks) > 0 {
		out.Blocks = make([]jsonBlock, len(f.Blocks))
		for i, block := range f.Blocks {
			out.Blocks[i] = jsonBlock{
				Type:         block.Type,
				SectionCount: len(block.Sections),
				Offset:       block.Offset,
			}
			if len(block.Sections) > 0 {
				out.Blocks[i].Sections = make([]jsonSection, len(block.Sections))
				for j, section := range block.Sections {
					out.Blocks[i].Sections[j] = jsonSection{
						Index:      section.Index,
						NameOffset: section.NameOffset,
						Unknown3:   section.Unknown3,
						DataOffset: section.DataOffset,
						Unknown5:   section.Unknown5,
						Name:       section.Name,
						Payload:    section.Payload,
					}
				}
			}
		}
	}

	if len(f.Names) > 0 {
		out.Names = make([]jsonNameEntry, len(f.Names))
		for i, name := range f.Names {
			out.Names[i] = jsonNameEntry{
				Offset:   name.Offset,
				Value:    name.Value,
				Consumed: name.Consumed,
			}
		}
	}

	return json.Marshal(out)
}
