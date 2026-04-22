package validate

import (
	"fmt"
	"sort"

	"github.com/example/lvrsrc/internal/rsrcwire"
)

type Severity = rsrcwire.Severity

const (
	SeverityWarning = rsrcwire.SeverityWarning
	SeverityError   = rsrcwire.SeverityError
)

type IssueLocation = rsrcwire.IssueLocation

type Issue struct {
	Severity Severity
	Code     string
	Message  string
	Location IssueLocation
}

type Options struct {
	FileSize int
}

func File(f *rsrcwire.File, opts Options) []Issue {
	if f == nil {
		return nil
	}

	var issues []Issue
	addIssue := func(issue Issue) {
		for _, existing := range issues {
			if existing.Code == issue.Code && existing.Location == issue.Location {
				return
			}
		}
		issues = append(issues, issue)
	}

	for _, issue := range f.ParseIssues {
		addIssue(Issue{
			Severity: issue.Severity,
			Code:     issue.Code,
			Message:  issue.Message,
			Location: issue.Location,
		})
	}

	if f.Header != f.SecondaryHeader {
		addIssue(Issue{
			Severity: SeverityError,
			Code:     "header.mismatch",
			Message:  "secondary header does not match primary header",
			Location: IssueLocation{
				Area:   "header",
				Offset: f.Header.InfoOffset,
			},
		})
	}

	fileSize := opts.FileSize
	if fileSize <= 0 {
		fileSize = inferredFileSize(f)
	}

	if int64(f.Header.InfoOffset)+int64(f.Header.InfoSize) > int64(fileSize) {
		addIssue(Issue{
			Severity: SeverityError,
			Code:     "header.info.bounds",
			Message:  fmt.Sprintf("info section exceeds file bounds (size=%d)", fileSize),
			Location: IssueLocation{
				Area:   "header",
				Offset: f.Header.InfoOffset,
			},
		})
	}
	if int64(f.Header.DataOffset)+int64(f.Header.DataSize) > int64(fileSize) {
		addIssue(Issue{
			Severity: SeverityError,
			Code:     "header.data.bounds",
			Message:  fmt.Sprintf("data section exceeds file bounds (size=%d)", fileSize),
			Location: IssueLocation{
				Area:   "header",
				Offset: f.Header.DataOffset,
			},
		})
	}
	if f.BlockInfoList.BlockInfoOffset > f.Header.InfoSize {
		addIssue(Issue{
			Severity: SeverityError,
			Code:     "block_info.offset_bounds",
			Message:  "block info offset extends beyond info section",
			Location: IssueLocation{
				Area:   "block_info",
				Offset: f.Header.InfoOffset + f.BlockInfoList.BlockInfoOffset,
			},
		})
	}

	nameOffsets := make(map[uint32]struct{}, len(f.Names))
	for _, name := range f.Names {
		nameOffsets[name.Offset] = struct{}{}
	}

	type region struct {
		start uint32
		end   uint32
		block string
		index int32
	}
	var payloads []region

	for _, block := range f.Blocks {
		if !isPrintableASCII(block.Type) {
			addIssue(Issue{
				Severity: SeverityError,
				Code:     "block.type.non_printable",
				Message:  fmt.Sprintf("block type %q contains non-printable ASCII", block.Type),
				Location: IssueLocation{
					Area:      "block_table",
					BlockType: block.Type,
				},
			})
		}

		if int(block.SectionCountMinusOne)+1 != len(block.Sections) {
			addIssue(Issue{
				Severity: SeverityError,
				Code:     "block.count_mismatch",
				Message:  fmt.Sprintf("block %q count field does not match %d sections", block.Type, len(block.Sections)),
				Location: IssueLocation{
					Area:      "block_table",
					BlockType: block.Type,
				},
			})
		}

		for _, section := range block.Sections {
			if section.NameOffset != ^uint32(0) {
				if _, ok := nameOffsets[section.NameOffset]; !ok {
					addIssue(Issue{
						Severity: SeverityError,
						Code:     "section.name_offset.invalid",
						Message:  fmt.Sprintf("section name offset %d is not present in name table", section.NameOffset),
						Location: IssueLocation{
							Area:         "name_table",
							BlockType:    block.Type,
							SectionIndex: section.Index,
							NameOffset:   section.NameOffset,
						},
					})
				}
			}

			if len(section.Payload) == 0 {
				addIssue(Issue{
					Severity: SeverityWarning,
					Code:     "section.size.zero",
					Message:  fmt.Sprintf("section %q/%d has zero-length payload", block.Type, section.Index),
					Location: IssueLocation{
						Area:         "data",
						Offset:       f.Header.DataOffset + section.DataOffset,
						BlockType:    block.Type,
						SectionIndex: section.Index,
					},
				})
			}

			payloadStart := section.DataOffset
			payloadEnd := payloadStart + 4 + uint32(len(section.Payload))
			payloads = append(payloads, region{
				start: payloadStart,
				end:   payloadEnd,
				block: block.Type,
				index: section.Index,
			})
		}
	}

	sort.Slice(payloads, func(i, j int) bool {
		if payloads[i].start == payloads[j].start {
			return payloads[i].end < payloads[j].end
		}
		return payloads[i].start < payloads[j].start
	})
	for i := 1; i < len(payloads); i++ {
		if payloads[i].start < payloads[i-1].end {
			addIssue(Issue{
				Severity: SeverityError,
				Code:     "section.payload.overlap",
				Message: fmt.Sprintf(
					"section %q/%d overlaps prior payload region",
					payloads[i].block,
					payloads[i].index,
				),
				Location: IssueLocation{
					Area:         "data",
					Offset:       f.Header.DataOffset + payloads[i].start,
					BlockType:    payloads[i].block,
					SectionIndex: payloads[i].index,
				},
			})
		}
	}

	return issues
}

func inferredFileSize(f *rsrcwire.File) int {
	size := int(maxU32(
		f.Header.InfoOffset+f.Header.InfoSize,
		f.Header.DataOffset+f.Header.DataSize,
	))
	for _, name := range f.Names {
		end := f.Header.InfoOffset + name.Offset + uint32(name.Consumed)
		size = maxInt(size, int(end))
	}
	for _, block := range f.Blocks {
		for _, section := range block.Sections {
			end := f.Header.DataOffset + section.DataOffset + 4 + uint32(len(section.Payload))
			size = maxInt(size, int(end))
		}
	}
	size = maxInt(size, int(f.Header.InfoOffset+f.Header.InfoSize))
	size = maxInt(size, int(f.Header.DataOffset+f.Header.DataSize))
	return size
}

func isPrintableASCII(s string) bool {
	if len(s) != 4 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

func maxU32(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
