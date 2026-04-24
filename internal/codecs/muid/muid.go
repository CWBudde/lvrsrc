// Package muid implements the codec for the "MUID" resource — the
// Map Unique Identifier, a 4-byte big-endian uint32 that records the
// maximum UID value used by the VI's LoadRefMap. Each object inside the
// VI is assigned a fresh UID on every change, so MUID effectively counts
// edits.
//
// References: pylabview LVblock.py:1272 (MUID class). pylavi treats this
// resource as opaque.
//
// See docs/resources/muid.md for layout and semantics.
package muid

import (
	"encoding/binary"
	"fmt"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/validate"
)

// FourCC is the resource type this codec handles.
const FourCC codecs.FourCC = "MUID"

// payloadSize is the fixed byte length of an MUID payload.
const payloadSize = 4

// Value is the decoded form of an MUID payload.
type Value struct {
	// UID is the maximum object UID used by the VI at save time.
	UID uint32
}

// Codec implements codecs.ResourceCodec for the MUID resource.
type Codec struct{}

// Capability reports the codec's static metadata. MUID is read-only:
// the VI runtime owns UID allocation and we do not invent new values.
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
		return nil, fmt.Errorf("MUID: payload size = %d, want %d", len(payload), payloadSize)
	}
	return Value{UID: binary.BigEndian.Uint32(payload)}, nil
}

// Encode serializes a Value (by value or pointer) into the wire format.
func (Codec) Encode(_ codecs.Context, value any) ([]byte, error) {
	var v Value
	switch tv := value.(type) {
	case Value:
		v = tv
	case *Value:
		if tv == nil {
			return nil, fmt.Errorf("MUID: Encode received nil *Value")
		}
		v = *tv
	default:
		return nil, fmt.Errorf("MUID: Encode expected Value or *Value, got %T", value)
	}
	out := make([]byte, payloadSize)
	binary.BigEndian.PutUint32(out, v.UID)
	return out, nil
}

// Validate reports structural issues.
func (Codec) Validate(_ codecs.Context, payload []byte) []validate.Issue {
	if len(payload) == payloadSize {
		return nil
	}
	return []validate.Issue{{
		Severity: validate.SeverityError,
		Code:     "muid.payload.size",
		Message:  fmt.Sprintf("MUID payload is %d bytes, want %d", len(payload), payloadSize),
		Location: validate.IssueLocation{Area: string(FourCC), BlockType: string(FourCC)},
	}}
}
