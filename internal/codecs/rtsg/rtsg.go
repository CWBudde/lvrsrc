// Package rtsg implements the codec for the "RTSG" resource — the
// Runtime Signature GUID, a 16-byte unique identifier LabVIEW writes to
// pin the runtime contract this VI was compiled against.
//
// References: pylabview LVblock.py:5383 (RTSG class). Stored as a single
// 16-byte payload with no internal structure; pylabview round-trips it
// verbatim and exposes it as a hex string in XML exports.
//
// See docs/resources/rtsg.md for layout and semantics.
package rtsg

import (
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "RTSG"

// payloadSize is the fixed byte length of an RTSG payload (a single GUID).
const payloadSize = 16

// Value is the decoded form of an RTSG payload.
type Value struct {
	// GUID is the 16-byte runtime signature, preserved verbatim. The
	// codec does not interpret byte order so the GUID is byte-equivalent
	// to the on-disk representation; callers that need the
	// little-endian Microsoft GUID display layout should reorder bytes
	// 0..3, 4..5, 6..7 themselves.
	GUID [16]byte
}

// Codec implements codecs.ResourceCodec for the RTSG resource.
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
		return nil, fmt.Errorf("RTSG: payload size = %d, want %d", len(payload), payloadSize)
	}
	var v Value
	copy(v.GUID[:], payload)
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
			return nil, fmt.Errorf("RTSG: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("RTSG: Encode expected Value or *Value, got %T", value)
	}
	out := make([]byte, payloadSize)
	copy(out, v.GUID[:])
	return out, nil
}

// Validate reports structural issues.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	if len(payload) == payloadSize {
		return nil
	}
	return []validate.Issue{{
		Severity: validate.SeverityError,
		Code:     "rtsg.payload.size",
		Message:  fmt.Sprintf("RTSG payload is %d bytes, want %d", len(payload), payloadSize),
		Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
	}}
}
