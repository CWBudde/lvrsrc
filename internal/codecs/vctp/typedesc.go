package vctp

import (
	"encoding/binary"
	"fmt"
)

// FullType is LabVIEW's TD_FULL_TYPE enum, identifying the kind of a
// type descriptor. Values mirror pylabview LVdatatype.py:47-123.
type FullType uint8

// TD_FULL_TYPE constants. The list here is intentionally narrow — only
// the codes the demo renders by name. Anything else falls back to
// `Type(0xNN)` via FullType.String. Add entries as needed when the
// type-tree view grows.
const (
	FullTypeVoid          FullType = 0x00
	FullTypeNumInt8       FullType = 0x01
	FullTypeNumInt16      FullType = 0x02
	FullTypeNumInt32      FullType = 0x03
	FullTypeNumInt64      FullType = 0x04
	FullTypeNumUInt8      FullType = 0x05
	FullTypeNumUInt16     FullType = 0x06
	FullTypeNumUInt32     FullType = 0x07
	FullTypeNumUInt64     FullType = 0x08
	FullTypeNumFloat32    FullType = 0x09
	FullTypeNumFloat64    FullType = 0x0A
	FullTypeNumFloatExt   FullType = 0x0B
	FullTypeNumComplex64  FullType = 0x0C
	FullTypeNumComplex128 FullType = 0x0D
	FullTypeBooleanU16    FullType = 0x20
	FullTypeBoolean       FullType = 0x21
	FullTypeString        FullType = 0x30
	FullTypeString2       FullType = 0x31
	FullTypePath          FullType = 0x32
	FullTypePicture       FullType = 0x33
	FullTypeCString       FullType = 0x34
	FullTypePasString     FullType = 0x35
	FullTypeArray         FullType = 0x40
	FullTypeCluster       FullType = 0x50
	FullTypeLVVariant     FullType = 0x53
	FullTypeMeasureData   FullType = 0x54
	FullTypeRefnum        FullType = 0x70
	FullTypeFunction      FullType = 0xF0
	FullTypeTypeDef       FullType = 0xF1
	FullTypePolyVI        FullType = 0xF2
)

// fullTypeNames is a sparse name table for human-friendly String output.
var fullTypeNames = map[FullType]string{
	FullTypeVoid:          "Void",
	FullTypeNumInt8:       "NumInt8",
	FullTypeNumInt16:      "NumInt16",
	FullTypeNumInt32:      "NumInt32",
	FullTypeNumInt64:      "NumInt64",
	FullTypeNumUInt8:      "NumUInt8",
	FullTypeNumUInt16:     "NumUInt16",
	FullTypeNumUInt32:     "NumUInt32",
	FullTypeNumUInt64:     "NumUInt64",
	FullTypeNumFloat32:    "NumFloat32",
	FullTypeNumFloat64:    "NumFloat64",
	FullTypeNumFloatExt:   "NumFloatExt",
	FullTypeNumComplex64:  "NumComplex64",
	FullTypeNumComplex128: "NumComplex128",
	FullTypeBooleanU16:    "BooleanU16",
	FullTypeBoolean:       "Boolean",
	FullTypeString:        "String",
	FullTypeString2:       "String2",
	FullTypePath:          "Path",
	FullTypePicture:       "Picture",
	FullTypeCString:       "CString",
	FullTypePasString:     "PasString",
	FullTypeArray:         "Array",
	FullTypeCluster:       "Cluster",
	FullTypeLVVariant:     "LVVariant",
	FullTypeMeasureData:   "MeasureData",
	FullTypeRefnum:        "Refnum",
	FullTypeFunction:      "Function",
	FullTypeTypeDef:       "TypeDef",
	FullTypePolyVI:        "PolyVI",
}

// String returns the human-readable type name, falling back to a hex
// `Type(0xNN)` form for codes that are not in the documented set.
func (t FullType) String() string {
	if name, ok := fullTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("Type(%#x)", uint8(t))
}

// HasLabelFlag is the TYPEDESC_FLAGS.HasLabel bit (pylabview
// LVdatatype.py:162). When set, the type descriptor's body ends with a
// trailing label stored as a Pascal-style string (1-byte length + N bytes)
// possibly preceded by alignment padding.
const HasLabelFlag uint8 = 1 << 6

// TypeDescriptor is a partially-decoded VCTP type-descriptor record.
// Per Phase 8.1 scope this surfaces the header (FullType, Flags) plus
// the trailing Label (when HasLabel is set) and the raw inner bytes
// between the header and the label. Type-specific decoding (Cluster
// children, Function parameters, Refnum class, …) is intentionally
// deferred — callers needing those can re-parse Inner against a
// type-specific decoder later.
type TypeDescriptor struct {
	// Index is the 0-based position in the flat type-descriptor list.
	// Top-level lookups (CONP TypeID, DTHP IndexShift) use 1-based
	// IDs that match Index+1.
	Index int
	// FullType is the TD_FULL_TYPE value at offset 3 of the record.
	FullType FullType
	// Flags is the TYPEDESC_FLAGS byte at offset 2.
	Flags uint8
	// HasLabel mirrors `Flags & HasLabelFlag != 0` for callers that
	// don't want to import the constant.
	HasLabel bool
	// Label is the trailing label string when HasLabel is set, decoded
	// from the Pascal-string at the tail of the record. Encoding is
	// preserved as []byte → string with no charset normalisation.
	Label string
	// Inner is the raw bytes between the 4-byte header and the label
	// (or the end of the record when HasLabel is unset). Type-specific
	// parsers consume this slice.
	Inner []byte
	// Length is the wire byte count of this record, including the
	// 4-byte header and any label bytes.
	Length int
}

// ParseInner parses the inflated VCTP payload (see Value.Inflated) into
// the flat type-descriptor list and the trailing top-types list.
//
//   - The flat list begins with a 4-byte big-endian count followed by
//     each typedesc record.
//   - Each record starts with a U2p2 length, a 1-byte flags byte, and a
//     1-byte type code.
//   - When the HasLabel flag is set, the record ends with a Pascal-string
//     label (1-byte length + bytes), possibly preceded by alignment
//     padding. ParseInner peels the label off the tail and stores
//     everything in between as Inner.
//   - After all records, the top-types list is `[U2p2 count] +
//     count × U2p2 indices`.
//
// References: pylabview `TypeDescListBase.parseRSRCTypeDescList` /
// `parseRSRCTopTypesList` (LVblock.py:5728-5750), `TDObject.parseRSRCDataHeader`
// (LVdatatype.py:451-455).
func ParseInner(inflated []byte) ([]TypeDescriptor, []uint32, error) {
	if len(inflated) < 4 {
		return nil, nil, fmt.Errorf("vctp: inflated payload too short for typedesc count: %d bytes", len(inflated))
	}
	count := binary.BigEndian.Uint32(inflated[:4])
	pos := 4

	descs := make([]TypeDescriptor, 0, count)
	for i := uint32(0); i < count; i++ {
		if pos+4 > len(inflated) {
			return nil, nil, fmt.Errorf("vctp: typedesc[%d] header truncated at offset %d", i, pos)
		}
		objLen, hdrBytes, err := readU2p2(inflated[pos:])
		if err != nil {
			return nil, nil, fmt.Errorf("vctp: typedesc[%d] length: %w", i, err)
		}
		recordStart := pos
		pos += hdrBytes
		if pos+2 > len(inflated) {
			return nil, nil, fmt.Errorf("vctp: typedesc[%d] flags/type truncated at offset %d", i, pos)
		}
		flags := inflated[pos]
		fullType := FullType(inflated[pos+1])
		pos += 2

		bodyEnd := recordStart + objLen
		if bodyEnd > len(inflated) {
			return nil, nil, fmt.Errorf("vctp: typedesc[%d] declared length %d overruns payload (record start %d, payload %d)", i, objLen, recordStart, len(inflated))
		}

		td := TypeDescriptor{
			Index:    int(i),
			FullType: fullType,
			Flags:    flags,
			HasLabel: flags&HasLabelFlag != 0,
			Length:   objLen,
		}

		body := inflated[pos:bodyEnd]
		if td.HasLabel {
			label, inner, err := splitTrailingLabel(body)
			if err != nil {
				return nil, nil, fmt.Errorf("vctp: typedesc[%d] label: %w", i, err)
			}
			td.Label = string(label)
			td.Inner = append([]byte(nil), inner...)
		} else {
			td.Inner = append([]byte(nil), body...)
		}

		descs = append(descs, td)
		pos = bodyEnd
	}

	// Top-types list.
	if pos >= len(inflated) {
		return descs, nil, nil
	}
	topCount, n, err := readU2p2(inflated[pos:])
	if err != nil {
		return nil, nil, fmt.Errorf("vctp: top types count: %w", err)
	}
	pos += n
	tops := make([]uint32, 0, topCount)
	for i := 0; i < topCount; i++ {
		val, n2, err := readU2p2(inflated[pos:])
		if err != nil {
			return nil, nil, fmt.Errorf("vctp: top type[%d]: %w", i, err)
		}
		tops = append(tops, uint32(val))
		pos += n2
	}
	return descs, tops, nil
}

// readU2p2 reads pylabview's variable-size U2p2 encoding from buf.
// Returns (value, bytes consumed, error).
func readU2p2(buf []byte) (int, int, error) {
	if len(buf) < 2 {
		return 0, 0, fmt.Errorf("u2p2 needs 2 bytes, have %d", len(buf))
	}
	hi := binary.BigEndian.Uint16(buf[:2])
	if hi&0x8000 == 0 {
		return int(hi), 2, nil
	}
	if len(buf) < 4 {
		return 0, 0, fmt.Errorf("u2p2 extended needs 4 bytes, have %d", len(buf))
	}
	lo := binary.BigEndian.Uint16(buf[2:4])
	return int((uint32(hi&0x7FFF) << 16) | uint32(lo)), 4, nil
}

// splitTrailingLabel peels the label off the tail of a typedesc body. The
// label is a Pascal-string (1-byte length + N bytes). pylabview tolerates
// up to 256 bytes of body before the label and one trailing zero pad
// byte; we follow the same `validLabelLength` heuristic.
func splitTrailingLabel(body []byte) (label, inner []byte, err error) {
	if len(body) == 0 {
		return nil, body, nil
	}
	// Drop one trailing zero byte if present (pylabview's ending_zeros
	// stripping).
	stripped := body
	hadZero := false
	if stripped[len(stripped)-1] == 0 {
		stripped = stripped[:len(stripped)-1]
		hadZero = true
	}
	// Search the last 256 bytes for a position where labelLen + 1
	// remaining bytes make a valid label.
	start := 0
	if len(stripped) > 256 {
		start = len(stripped) - 256
	}
	for i := start; i < len(stripped); i++ {
		labelLen := int(stripped[i])
		if i+1+labelLen != len(stripped) {
			continue
		}
		if !asciiOrCRLF(stripped[i+1:]) {
			continue
		}
		// Inner is body up to position i, ignoring the trailing zero
		// pad we may have stripped (it is part of the label's
		// alignment, not the inner).
		_ = hadZero
		return stripped[i+1:], body[:i], nil
	}
	// Could not find a valid label position; return the body as Inner
	// and an empty label. pylabview also does this fallback (label = "").
	return nil, body, nil
}

// asciiOrCRLF reports whether b consists only of bytes that are CR/LF/TAB
// or printable ASCII (>= 32). Mirrors pylabview validLabelLength.
func asciiOrCRLF(b []byte) bool {
	for _, c := range b {
		if c == '\r' || c == '\n' || c == '\t' {
			continue
		}
		if c < 32 {
			return false
		}
	}
	return true
}
