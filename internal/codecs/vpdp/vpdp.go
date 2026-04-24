// Package vpdp implements the codec for the "VPDP" resource — VI Probe-
// Data Pointer / VI Primitive Dependency Flags. pylabview classifies it
// as a stub (`Block`, no parsing). The corpus shows it as a uniform
// 4-byte payload that is always 0x00000000, so we expose it as an opaque
// big-endian uint32 so callers can verify it stayed at the expected
// sentinel and round-trip preserves the value byte-for-byte.
//
// References: pylabview LVblock.py:5055 (VPDP class). pylavi treats this
// resource as opaque.
//
// See docs/resources/vpdp.md for layout and open questions.
package vpdp

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "VPDP"

const payloadSize = 4

// Value is the decoded form of a VPDP payload. The Flags field is
// preserved verbatim; pylabview labels the resource "VI Primitive
// Dependency Flags" but the meaning of individual bits is not yet
// documented.
type Value struct {
	Flags uint32
}

// Codec implements codecs.ResourceCodec for the VPDP resource.
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
		return nil, fmt.Errorf("VPDP: payload size = %d, want %d", len(payload), payloadSize)
	}
	return Value{Flags: binary.BigEndian.Uint32(payload)}, nil
}

// Encode serializes a Value (by value or pointer).
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("VPDP: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("VPDP: Encode expected Value or *Value, got %T", value)
	}
	out := make([]byte, payloadSize)
	binary.BigEndian.PutUint32(out, v.Flags)
	return out, nil
}

// Validate reports structural issues.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	if len(payload) == payloadSize {
		return nil
	}
	return []validate.Issue{{
		Severity: validate.SeverityError,
		Code:     "vpdp.payload.size",
		Message:  fmt.Sprintf("VPDP payload is %d bytes, want %d", len(payload), payloadSize),
		Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
	}}
}
