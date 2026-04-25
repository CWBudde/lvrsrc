// Package bdex implements the codec for the "BDEx" resource — Block-
// Diagram "Extra" heap-aux block. Sibling of FPEx with the same wire
// shape: a 4-byte big-endian count followed by Count uint32 entries.
//
// The corpus contains BDEx in three sizes (4, 8, 12 bytes), all matching
// the same pattern with zero entries. The codec validates the wire
// layout and round-trips byte-for-byte.
//
// Safety tier: 1 (read-only). Entry semantics are not yet documented;
// pylabview does not classify this resource.
//
// See docs/resources/bdex.md for layout and open questions.
package bdex

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "BDEx"

// Value is the decoded form of a BDEx payload.
type Value struct {
	// Entries are the Count BE-uint32 entries that follow the count
	// field.
	Entries []uint32
}

// Codec implements codecs.ResourceCodec for the BDEx resource.
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
	if len(payload) < 4 {
		return nil, fmt.Errorf("BDEx: payload too short: %d bytes (need at least 4 for count)", len(payload))
	}
	count := binary.BigEndian.Uint32(payload[:4])
	want := 4 + 4*int(count)
	if len(payload) != want {
		return nil, fmt.Errorf("BDEx: payload size = %d, want %d for count %d", len(payload), want, count)
	}
	entries := make([]uint32, count)
	for i := range entries {
		entries[i] = binary.BigEndian.Uint32(payload[4+i*4 : 4+i*4+4])
	}
	return Value{Entries: entries}, nil
}

// Encode serializes a Value (by value or pointer).
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("BDEx: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("BDEx: Encode expected Value or *Value, got %T", value)
	}
	out := make([]byte, 4+4*len(v.Entries))
	binary.BigEndian.PutUint32(out[:4], uint32(len(v.Entries)))
	for i, e := range v.Entries {
		binary.BigEndian.PutUint32(out[4+i*4:], e)
	}
	return out, nil
}

// Validate reports structural issues by re-running Decode.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	if _, err := (Codec{}).Decode(codecs.Context{}, payload); err != nil {
		return []validate.Issue{{
			Severity: validate.SeverityError,
			Code:     "bdex.payload.malformed",
			Message:  fmt.Sprintf("BDEx payload could not be parsed: %v", err),
			Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
		}}
	}
	return nil
}
