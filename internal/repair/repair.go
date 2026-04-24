package repair

import (
	"errors"
	"fmt"

	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

var ErrUnrepairable = errors.New("repair: file is not repairable")

// File applies conservative structural repair heuristics to a leniently parsed
// RSRC file. It never guesses missing payload bytes or unresolved section
// names; when those would be required, it refuses repair.
func File(src *lvrsrc.File) (*lvrsrc.File, []string, error) {
	if src == nil {
		return nil, nil, fmt.Errorf("repair: nil file")
	}

	repaired := src.Clone()
	issues := repaired.Validate()

	var (
		needRewrite      bool
		needNameRebuild  bool
		sawHeaderIssue   bool
		sawOverlapIssue  bool
		sawNameOffsetErr bool
	)

	for _, issue := range issues {
		if issue.Severity != lvrsrc.SeverityError {
			continue
		}

		switch issue.Code {
		case "header.mismatch":
			needRewrite = true
			sawHeaderIssue = true
		case "section.payload.overlap":
			needRewrite = true
			sawOverlapIssue = true
		case "section.name_offset.invalid":
			needRewrite = true
			needNameRebuild = true
			sawNameOffsetErr = true
		default:
			return nil, nil, fmt.Errorf("%w: unsupported error %s", ErrUnrepairable, issue.Code)
		}
	}

	if !needRewrite {
		return nil, nil, fmt.Errorf("%w: no repairable structural errors found", ErrUnrepairable)
	}

	var actions []string
	if needNameRebuild {
		if err := rebuildReferencedNames(repaired); err != nil {
			return nil, nil, fmt.Errorf("%w: %s: %w", ErrUnrepairable, "section.name_offset.invalid", err)
		}
		if sawNameOffsetErr {
			actions = append(actions, "rebuild referenced name table from resolved section names")
		}
	}
	if sawHeaderIssue {
		actions = append(actions, "rewrite headers from parsed structure")
	}
	if sawOverlapIssue {
		actions = append(actions, "recompute section/header offsets from parsed payload tree")
	}

	return repaired, actions, nil
}

func rebuildReferencedNames(f *lvrsrc.File) error {
	type nameRecord struct {
		oldOffset uint32
		value     string
	}

	nameValues := make(map[uint32]string, len(f.Names))
	for _, entry := range f.Names {
		nameValues[entry.Offset] = entry.Value
	}

	var (
		ordered   []nameRecord
		offsetMap = make(map[uint32]uint32, len(f.Names))
		seen      = make(map[uint32]string, len(f.Names))
		nextOff   uint32
	)

	for bi := range f.Blocks {
		for si := range f.Blocks[bi].Sections {
			section := &f.Blocks[bi].Sections[si]
			if section.NameOffset == ^uint32(0) {
				continue
			}

			value := section.Name
			if value == "" {
				value = nameValues[section.NameOffset]
			}
			if value == "" {
				return fmt.Errorf("missing section name for %s/%d", f.Blocks[bi].Type, section.Index)
			}

			if existing, ok := seen[section.NameOffset]; ok {
				if existing != value {
					return fmt.Errorf(
						"conflicting section names for offset %d (%q != %q)",
						section.NameOffset,
						existing,
						value,
					)
				}
				section.Name = value
				section.NameOffset = offsetMap[section.NameOffset]
				continue
			}

			seen[section.NameOffset] = value
			offsetMap[section.NameOffset] = nextOff
			ordered = append(ordered, nameRecord{oldOffset: section.NameOffset, value: value})
			section.Name = value
			section.NameOffset = nextOff
			nextOff += uint32(1 + len(value))
		}
	}

	if len(ordered) == 0 {
		f.Names = nil
		return nil
	}

	f.Names = make([]lvrsrc.NameEntry, len(ordered))
	for i, entry := range ordered {
		f.Names[i] = lvrsrc.NameEntry{
			Offset:   offsetMap[entry.oldOffset],
			Value:    entry.value,
			Consumed: int64(1 + len(entry.value)),
		}
	}

	return nil
}
