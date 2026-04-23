package lvmeta

import (
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// lvsrFourCC identifies the LabVIEW save-record block. The VI filename is
// surfaced by the container parser as the LVSR section's Name, so renaming a
// VI is a container-level name-table edit rather than a codec call.
const lvsrFourCC codecs.FourCC = "LVSR"

// maxPascalNameLen is the upper bound on an RSRC name-table Pascal string.
const maxPascalNameLen = 0xFF

// SetName renames the VI by updating its LVSR section's Name (and the backing
// name-table entry) while preserving every other section's name references.
//
// It is a convenience over (Mutator{}).SetName and therefore runs in lenient
// (non-strict) mode.
func SetName(f *lvrsrc.File, name string) error {
	return Mutator{}.SetName(f, name)
}

// SetName renames the VI.
//
// The rename is expressed as a container/name-table mutation, not a
// resource-codec edit:
//
//   - Locate the single LVSR section. Reject missing / ambiguous targets.
//   - If the requested name already appears in f.Names, reuse that entry's
//     offset so the name table stays compact.
//   - Else, if the LVSR section is the only reference to its current
//     NameOffset (and the backing entry exists), update that entry in
//     place, preserving the offset.
//   - Else, append a new NameEntry at an offset beyond the last existing
//     entry and point the section at it. The serializer's compaction path
//     rewrites offsets if the sparse layout no longer fits.
//
// Names longer than 255 bytes are rejected (ErrNameTooLong); no path or
// extension normalization is performed.
func (m Mutator) SetName(f *lvrsrc.File, name string) error {
	if f == nil {
		return &MutationError{FourCC: lvsrFourCC, Cause: ErrNilFile}
	}
	if len(name) > maxPascalNameLen {
		return &MutationError{
			FourCC: lvsrFourCC,
			Cause:  ErrNameTooLong,
			Err:    fmt.Errorf("length %d > %d", len(name), maxPascalNameLen),
		}
	}

	ref, found, locErr := requireSingleSectionByType(f, lvsrFourCC)
	if locErr != nil {
		return &MutationError{FourCC: lvsrFourCC, Cause: ErrTargetAmbiguous, Err: locErr}
	}
	if !found {
		return &MutationError{FourCC: lvsrFourCC, Cause: ErrTargetMissing}
	}

	section := &f.Blocks[ref.BlockIndex].Sections[ref.SectionIndex]

	// No-op: same name, and already anchored to a valid entry (or explicitly
	// has no name). Leaves section and Names slice untouched.
	if section.Name == name && section.NameOffset != noNameOffsetSentinel {
		return nil
	}

	baseline := captureStructuralBaseline(f)
	origSection := *section
	origNames := append([]lvrsrc.NameEntry(nil), f.Names...)
	rollback := func() {
		*section = origSection
		f.Names = origNames
	}

	// 1. Reuse path: another entry already carries this value.
	reused := false
	for _, entry := range f.Names {
		if entry.Value == name {
			section.Name = name
			section.NameOffset = entry.Offset
			reused = true
			break
		}
	}

	// 2. In-place update path: the section owns its offset exclusively and
	//    a backing entry exists.
	inPlace := false
	if !reused &&
		section.NameOffset != noNameOffsetSentinel &&
		countSectionsUsingOffset(f, section.NameOffset) == 1 {
		for i := range f.Names {
			if f.Names[i].Offset == section.NameOffset {
				f.Names[i].Value = name
				f.Names[i].Consumed = int64(1 + len(name))
				section.Name = name
				inPlace = true
				break
			}
		}
	}

	// 3. New entry path: pick an offset beyond every existing entry so the
	//    preserving name-table layout still fits; the serializer compacts
	//    if not.
	if !reused && !inPlace {
		newOffset := nextFreeNameOffset(f.Names)
		f.Names = append(f.Names, lvrsrc.NameEntry{
			Offset:   newOffset,
			Value:    name,
			Consumed: int64(1 + len(name)),
		})
		section.Name = name
		section.NameOffset = newOffset
	}

	if err := m.runStructuralCheck(f, lvsrFourCC, baseline); err != nil {
		rollback()
		return err
	}
	return nil
}

// countSectionsUsingOffset returns the number of sections that reference
// offset via NameOffset. Used to decide whether updating a name entry in
// place is safe.
func countSectionsUsingOffset(f *lvrsrc.File, offset uint32) int {
	n := 0
	for _, block := range f.Blocks {
		for _, s := range block.Sections {
			if s.NameOffset == offset {
				n++
			}
		}
	}
	return n
}

// nextFreeNameOffset returns an offset guaranteed not to collide with any
// existing entry. It picks the byte immediately after the last existing
// entry in the sparse layout; callers can append safely without the
// serializer mistaking it for an overlapping name.
func nextFreeNameOffset(names []lvrsrc.NameEntry) uint32 {
	var max uint32
	for _, e := range names {
		end := e.Offset + uint32(1+len(e.Value))
		if end > max {
			max = end
		}
	}
	return max
}
