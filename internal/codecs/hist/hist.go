// Package hist implements the codec for the "HIST" resource — Changes
// History. pylabview ships only a stub `Block` subclass with no parsing
// (LVblock.py:3078-3085); the field semantics are not publicly
// documented.
//
// The shipped corpus carries HIST as a uniform 40-byte payload, suggesting
// a fixed-size counter array. This codec preserves the bytes verbatim and
// exposes a `Counters()` helper that interprets them as ten big-endian
// uint32 values — useful for callers who want to inspect or compare the
// counters without committing to specific field names.
//
// See docs/resources/hist.md for layout and open questions.
package hist

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "HIST"

// payloadSize is the canonical HIST payload size observed in the corpus.
const payloadSize = 40

// CounterCount is the number of big-endian uint32 counters that
// `payloadSize` divides cleanly into.
const CounterCount = payloadSize / 4

// Value is the decoded form of a HIST payload.
type Value struct {
	// Raw is the 40-byte payload preserved verbatim. Use Counters() for
	// a structured view as 10 big-endian uint32s.
	Raw [payloadSize]byte
}

// Counters returns Raw split into ten big-endian uint32 counters. The
// individual semantics of each slot are not yet documented; the array
// exists so callers can compare counter values across files (e.g. for
// diffs) without re-parsing the bytes by hand.
func (v Value) Counters() [CounterCount]uint32 {
	var out [CounterCount]uint32
	for i := range CounterCount {
		out[i] = binary.BigEndian.Uint32(v.Raw[i*4 : i*4+4])
	}
	return out
}

// Codec implements codecs.ResourceCodec for the HIST resource.
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
	if len(payload) != payloadSize {
		return nil, fmt.Errorf("HIST: payload size = %d, want %d", len(payload), payloadSize)
	}
	var v Value
	copy(v.Raw[:], payload)
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
			return nil, fmt.Errorf("HIST: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("HIST: Encode expected Value or *Value, got %T", value)
	}
	out := make([]byte, payloadSize)
	copy(out, v.Raw[:])
	return out, nil
}

// Validate reports structural issues.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	if len(payload) == payloadSize {
		return nil
	}
	return []validate.Issue{{
		Severity: validate.SeverityError,
		Code:     "hist.payload.size",
		Message:  fmt.Sprintf("HIST payload is %d bytes, want %d", len(payload), payloadSize),
		Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
	}}
}
