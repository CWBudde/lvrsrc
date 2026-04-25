// Package ftab implements the codec for the "FTAB" resource — Font Table.
//
// FTAB carries the VI's font registry: a header with section-level
// properties, a fixed-width entry table, and a trailing pool of Pascal-
// string font names referenced by per-entry offsets. Bit `0x00010000` of
// the header `Prop1` selects between the 12-byte ("narrow") entry shape
// and the 16-byte ("wide") shape; all corpus samples use the wide shape.
//
// References: pylabview LVblock.py:2892-3075 (FTAB class).
//
// Safety tier: 1 (read-only). Round-trip preserves the on-disk byte
// stream because pylabview's name-emission algorithm — append a fresh
// Pascal-string per entry, never share offsets across entries — exactly
// matches the on-disk layout in every observed sample.
//
// See docs/resources/ftab.md for layout, semantics, and open questions.
package ftab

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "FTAB"

const (
	headerSize       = 8
	narrowEntrySize  = 12
	wideEntrySize    = 16
	wideEntryFlagBit = 0x00010000
	maxFontCount     = 127
)

// Value is the decoded form of an FTAB payload.
type Value struct {
	// Prop1 holds the section-level flags. Bit 0x00010000 toggles wide
	// (16-byte) entries; clearing it selects the legacy 12-byte form.
	// Other bits are preserved verbatim.
	Prop1 uint32
	// Prop3 is a section-level 16-bit value preserved verbatim.
	Prop3 uint16
	// Entries lists the font definitions in their original order.
	Entries []FontEntry
}

// FontEntry is one font definition. The four shared properties (Prop2,
// Prop3, Prop4) are present in both entry widths; the additional Prop5
// (narrow) or Prop6/Prop7/Prop8 (wide) trail depending on Value.Prop1.
type FontEntry struct {
	// Prop2, Prop3, Prop4 are 16-bit attributes whose specific meaning
	// pylabview does not document. Preserved verbatim.
	Prop2, Prop3, Prop4 uint16
	// Prop5 is meaningful only when Value.Prop1 has bit 0x00010000
	// clear (12-byte entries). Otherwise it is ignored on encode.
	Prop5 uint16
	// Prop6, Prop7, Prop8 are meaningful only when Value.Prop1 has bit
	// 0x00010000 set (16-byte entries). Otherwise they are ignored.
	Prop6, Prop7, Prop8 uint16
	// Name is the Pascal-string font name. nil/empty means the entry's
	// nameOffs was zero (no name); a non-nil slice produces a fresh
	// name copy in the on-disk name pool.
	Name []byte
}

// hasWideEntries reports whether Prop1 selects the 16-byte entry layout.
func (v Value) hasWideEntries() bool { return v.Prop1&wideEntryFlagBit != 0 }

// entrySize returns the on-disk entry width implied by Prop1.
func (v Value) entrySize() int {
	if v.hasWideEntries() {
		return wideEntrySize
	}
	return narrowEntrySize
}

// Codec implements codecs.ResourceCodec for the FTAB resource.
type Codec struct{}

// Capability reports the codec's static metadata.
func (Codec) Capability() codecs.Capability {
	return codecs.Capability{
		FourCC:        FourCC,
		ReadVersions:  codecs.VersionRange{Min: 0, Max: 0},
		WriteVersions: codecs.VersionRange{Min: 0, Max: 0},
		Safety:        codecs.SafetyTier1,
	}
}

// Decode parses payload into a Value.
func (Codec) Decode(_ codecs.Context, payload []byte) (any, error) {
	if len(payload) < headerSize {
		return nil, fmt.Errorf("FTAB: payload too short: %d bytes (need at least %d for header)", len(payload), headerSize)
	}
	prop1 := binary.BigEndian.Uint32(payload[0:4])
	prop3 := binary.BigEndian.Uint16(payload[4:6])
	count := binary.BigEndian.Uint16(payload[6:8])
	if int(count) > maxFontCount {
		return nil, fmt.Errorf("FTAB: count %d exceeds limit %d", count, maxFontCount)
	}

	v := Value{Prop1: prop1, Prop3: prop3}
	entrySize := v.entrySize()
	entriesEnd := headerSize + int(count)*entrySize
	if entriesEnd > len(payload) {
		return nil, fmt.Errorf("FTAB: entry table truncated: need %d bytes, have %d", entriesEnd, len(payload))
	}

	v.Entries = make([]FontEntry, count)
	nameOffs := make([]uint32, count)
	wide := v.hasWideEntries()
	for i := range v.Entries {
		base := headerSize + i*entrySize
		nameOffs[i] = binary.BigEndian.Uint32(payload[base : base+4])
		v.Entries[i].Prop2 = binary.BigEndian.Uint16(payload[base+4 : base+6])
		v.Entries[i].Prop3 = binary.BigEndian.Uint16(payload[base+6 : base+8])
		v.Entries[i].Prop4 = binary.BigEndian.Uint16(payload[base+8 : base+10])
		if wide {
			v.Entries[i].Prop6 = binary.BigEndian.Uint16(payload[base+10 : base+12])
			v.Entries[i].Prop7 = binary.BigEndian.Uint16(payload[base+12 : base+14])
			v.Entries[i].Prop8 = binary.BigEndian.Uint16(payload[base+14 : base+16])
		} else {
			v.Entries[i].Prop5 = binary.BigEndian.Uint16(payload[base+10 : base+12])
		}
	}

	// Resolve names. nameOffs of 0 means "no name". Names follow the
	// entry table contiguously; pylabview reads each one at its absolute
	// offset, so we do the same.
	for i, off := range nameOffs {
		if off == 0 {
			continue
		}
		if int(off) >= len(payload) {
			return nil, fmt.Errorf("FTAB: name[%d] offset %d out of bounds (payload size %d)", i, off, len(payload))
		}
		strLen := int(payload[off])
		if int(off)+1+strLen > len(payload) {
			return nil, fmt.Errorf("FTAB: name[%d] of length %d at offset %d overruns payload", i, strLen, off)
		}
		name := make([]byte, strLen)
		copy(name, payload[int(off)+1:int(off)+1+strLen])
		v.Entries[i].Name = name
	}

	return v, nil
}

// Encode serializes a Value (by value or pointer).
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("FTAB: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("FTAB: Encode expected Value or *Value, got %T", value)
	}
	count := len(v.Entries)
	if count > maxFontCount {
		return nil, fmt.Errorf("FTAB: count %d exceeds limit %d", count, maxFontCount)
	}
	entrySize := v.entrySize()
	wide := v.hasWideEntries()

	// Build the names blob first so we know each entry's nameOffs.
	namesStart := headerSize + count*entrySize
	namesData := make([]byte, 0)
	nameOffs := make([]uint32, count)
	for i, entry := range v.Entries {
		if entry.Name == nil {
			nameOffs[i] = 0
			continue
		}
		if len(entry.Name) > 0xFF {
			return nil, fmt.Errorf("FTAB: entry[%d] name is %d bytes, max 255", i, len(entry.Name))
		}
		nameOffs[i] = uint32(namesStart + len(namesData))
		namesData = append(namesData, byte(len(entry.Name)))
		namesData = append(namesData, entry.Name...)
	}

	out := make([]byte, namesStart+len(namesData))
	binary.BigEndian.PutUint32(out[0:4], v.Prop1)
	binary.BigEndian.PutUint16(out[4:6], v.Prop3)
	binary.BigEndian.PutUint16(out[6:8], uint16(count))
	for i, entry := range v.Entries {
		base := headerSize + i*entrySize
		binary.BigEndian.PutUint32(out[base:base+4], nameOffs[i])
		binary.BigEndian.PutUint16(out[base+4:base+6], entry.Prop2)
		binary.BigEndian.PutUint16(out[base+6:base+8], entry.Prop3)
		binary.BigEndian.PutUint16(out[base+8:base+10], entry.Prop4)
		if wide {
			binary.BigEndian.PutUint16(out[base+10:base+12], entry.Prop6)
			binary.BigEndian.PutUint16(out[base+12:base+14], entry.Prop7)
			binary.BigEndian.PutUint16(out[base+14:base+16], entry.Prop8)
		} else {
			binary.BigEndian.PutUint16(out[base+10:base+12], entry.Prop5)
		}
	}
	copy(out[namesStart:], namesData)
	return out, nil
}

// Validate reports structural issues by re-running Decode.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	if _, err := (Codec{}).Decode(codecs.Context{}, payload); err != nil {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "ftab.payload.malformed",
			Message:  fmt.Sprintf("FTAB payload could not be parsed: %v", err),
			Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
		}}
	}
	return nil
}
