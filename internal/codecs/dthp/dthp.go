// Package dthp implements the codec for the "DTHP" resource — Data Types
// for Heap. In LV8.0 and newer, DTHP carries a slice into the VCTP type
// list: a count of TypeDescs used by the heaps plus an index shift telling
// where the slice starts.
//
// pylabview note (LVblock.py:3186): "If there is no count provided, then
// there is no shift." — when TDCount is zero LabVIEW writes only the
// 16-bit count word and omits the IndexShift field entirely. This codec
// preserves that variant exactly: a 2-byte payload containing only a zero
// count round-trips byte-for-byte.
//
// Both fields use pylabview's "U2 plus 2" variable-size encoding: a
// 16-bit big-endian integer where the high bit (0x8000) signals that the
// value continues into the next 16 bits as a 31-bit count. See
// LVmisc.py:336 (readVariableSizeFieldU2p2).
//
// References: pylabview LVblock.py:3177-3278.
//
// See docs/resources/dthp.md for layout and semantics.
package dthp

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "DTHP"

// Value is the decoded form of a DTHP payload.
type Value struct {
	// TDCount is the number of consecutive TypeDescs in VCTP that the
	// heaps reference, starting at IndexShift.
	TDCount uint32
	// IndexShift is the offset into VCTP at which the heap-used slice
	// begins. Only meaningful when TDCount > 0.
	IndexShift uint32
}

// Codec implements codecs.ResourceCodec for the DTHP resource.
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
	count, n, err := readU2p2(payload)
	if err != nil {
		return nil, fmt.Errorf("DTHP: count: %w", err)
	}
	if count == 0 {
		// Per pylabview: when count is zero, IndexShift is omitted.
		// Anything past offset n is unexpected — treat trailing bytes
		// as an error so we surface drift instead of silently dropping
		// data.
		if n != len(payload) {
			return nil, fmt.Errorf("DTHP: trailing %d byte(s) after zero-count payload", len(payload)-n)
		}
		return Value{}, nil
	}
	shift, m, err := readU2p2(payload[n:])
	if err != nil {
		return nil, fmt.Errorf("DTHP: index_shift: %w", err)
	}
	if n+m != len(payload) {
		return nil, fmt.Errorf("DTHP: trailing %d byte(s) after index_shift", len(payload)-n-m)
	}
	return Value{TDCount: count, IndexShift: shift}, nil
}

// Encode serializes a Value (by value or pointer).
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("DTHP: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("DTHP: Encode expected Value or *Value, got %T", value)
	}
	out := writeU2p2(nil, v.TDCount)
	if v.TDCount > 0 {
		out = writeU2p2(out, v.IndexShift)
	}
	return out, nil
}

// Validate reports structural issues. The codec only ships shape checks;
// semantic correctness (e.g. that IndexShift+TDCount fits in the file's
// VCTP) is left to higher-level validators.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	if _, err := (Codec{}).Decode(codecs.Context{}, payload); err != nil {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "dthp.payload.malformed",
			Message:  fmt.Sprintf("DTHP payload could not be parsed: %v", err),
			Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
		}}
	}
	return nil
}

// readU2p2 reads pylabview's variable-size U2p2 encoding from buf:
// a 16-bit big-endian word; if its high bit (0x8000) is set, the low 15
// bits are shifted up and combined with the next 16-bit word to form a
// 31-bit value. Returns (value, bytesConsumed, error).
func readU2p2(buf []byte) (uint32, int, error) {
	if len(buf) < 2 {
		return 0, 0, fmt.Errorf("need at least 2 bytes, got %d", len(buf))
	}
	hi := binary.BigEndian.Uint16(buf[:2])
	if hi&0x8000 == 0 {
		return uint32(hi), 2, nil
	}
	if len(buf) < 4 {
		return 0, 0, fmt.Errorf("extended-form u2p2 needs 4 bytes, got %d", len(buf))
	}
	lo := binary.BigEndian.Uint16(buf[2:4])
	return (uint32(hi&0x7FFF) << 16) | uint32(lo), 4, nil
}

// writeU2p2 appends a variable-size U2p2 encoding of v to dst, choosing
// the short form (2 bytes) when v fits in 15 bits, and the extended form
// (4 bytes with the high bit set) otherwise.
func writeU2p2(dst []byte, v uint32) []byte {
	if v <= 0x7FFF {
		return append(dst, byte(v>>8), byte(v))
	}
	return append(dst,
		byte((v>>24)|0x80), byte(v>>16),
		byte(v>>8), byte(v),
	)
}
