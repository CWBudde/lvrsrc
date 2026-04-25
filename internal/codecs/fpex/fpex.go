// Package fpex implements the codec for the "FPEx" resource — Front-
// Panel "Extra" heap-aux block. pylabview does not classify this resource;
// the layout is inferred from corpus uniformity.
//
// The corpus contains FPEx in three sizes (4, 8, 16 bytes), all matching
// the same pattern: a 4-byte big-endian count followed by Count uint32
// entries (always zero in the corpus). The codec therefore decodes the
// resource as `Count + Entries[Count]`, validates that the wire size
// equals 4 + 4*Count, and round-trips every observed payload exactly.
//
// Safety tier: 1 (read-only). The semantics of the entries (always zero
// in the corpus) are not yet documented.
//
// See docs/resources/fpex.md for layout and open questions.
package fpex

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "FPEx"

// Value is the decoded form of an FPEx payload.
type Value struct {
	// Entries are the Count BE-uint32 entries that follow the count
	// field. Always zero in the shipped corpus; the codec preserves
	// non-zero values verbatim if/when they appear.
	Entries []uint32
}

// Codec implements codecs.ResourceCodec for the FPEx resource.
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
		return nil, fmt.Errorf("FPEx: payload too short: %d bytes (need at least 4 for count)", len(payload))
	}
	count := binary.BigEndian.Uint32(payload[:4])
	want := 4 + 4*int(count)
	if len(payload) != want {
		return nil, fmt.Errorf("FPEx: payload size = %d, want %d for count %d", len(payload), want, count)
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
			return nil, fmt.Errorf("FPEx: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("FPEx: Encode expected Value or *Value, got %T", value)
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
			Code:     "fpex.payload.malformed",
			Message:  fmt.Sprintf("FPEx payload could not be parsed: %v", err),
			Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
		}}
	}
	return nil
}
