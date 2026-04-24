// Package fpse implements the codec for the "FPSE" resource — Front
// Panel Size Estimate, a 4-byte big-endian uint32 that LabVIEW uses as a
// hint for the in-memory size of the front-panel object graph.
//
// References: pylabview LVblock.py:1288 (FPSE class, SingleIntBlock).
//
// See docs/resources/fpse.md for layout and semantics.
package fpse

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "FPSE"

const payloadSize = 4

// Value is the decoded form of an FPSE payload.
type Value struct {
	// Estimate is the LabVIEW-computed front-panel size hint, in bytes.
	Estimate uint32
}

// Codec implements codecs.ResourceCodec for the FPSE resource.
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
		return nil, fmt.Errorf("FPSE: payload size = %d, want %d", len(payload), payloadSize)
	}
	return Value{Estimate: binary.BigEndian.Uint32(payload)}, nil
}

// Encode serializes a Value (by value or pointer).
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("FPSE: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("FPSE: Encode expected Value or *Value, got %T", value)
	}
	out := make([]byte, payloadSize)
	binary.BigEndian.PutUint32(out, v.Estimate)
	return out, nil
}

// Validate reports structural issues.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	if len(payload) == payloadSize {
		return nil
	}
	return []validate.Issue{{
		Severity: validate.SeverityError,
		Code:     "fpse.payload.size",
		Message:  fmt.Sprintf("FPSE payload is %d bytes, want %d", len(payload), payloadSize),
		Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
	}}
}
